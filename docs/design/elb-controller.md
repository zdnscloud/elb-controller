# 外部负载均衡控制器
## 概要
通过在k8s集群内部署插件的方式，支持外部负载均衡器与k8s LoadBalancer类型service的自动联动配置
## 动机和目标
在大多数k8s私有云场景下，用户都会有专业的负载均衡设备，用于k8s服务的对外发布，负载均衡的配置需要管理员手动完成，需要满足该场景下，k8s对外服务在外部负载均衡器上的自动发布

当创建、更新及删除LoadBalancer类型service时，可以自动调用外部负载均衡器相关api完成负载均衡策略的的创建、更新和删除；支持基本的外部负载均衡设备的高可用部署方案（双机HA）
## 架构
```
                               +-------------------------------------+
                               |                                     |
                               |         +----------------+          |
                               |         |                |          |
                               |         | kube-apiserver |          |
                               |         |                |          |
                               |         +-------+--------+          |
                               |                 ^                   |
                               |                 |                   |
                               |                 |                   |
                               |                 |                   |
                               |                 v                   |
+--------------------+         |         +-------+--------+          |
|                    |         |         |                |          |
| external loadblance+<----------------->+ elb-controller |          |
|                    |         |         |                |          |
+--------------------+         |         +----------------+          |
                               |                                     |
                               |                                     |
                               |                                     |
                               +-------------------------------------+

```
## 原理
### LoadBalancer service
k8s LoadBalancer类型service对象，供与外部负载设备联动使用，实际是在是NodePort类型基础上增加了service status（所有类型service都有status字段，但该字段目前仅有LoadBalancer一个子属性）
所以k8s与外部负载均衡设备联动的原理图如下：
```
                                                             +------------------------+
                                                             |                        |
                                                             |                        |
                                                             |     +-------------+    |
                                                             |     |             |    |
                                                             |  +->+worker1:30010|    |
                                                             |  |  |             |    |
                                                             |  |  +-------------+    |
                                                             |  |                     |
                                                             |  |                     |
                                                             |  |                     |
                                                             |  |  +-------------+    |
+-------------+            +--------------------+            |  |  |             |    |
|             |            |                    |            |  +->+worker2:30010|    |
|   client    +----------->+ external loadblance+---------------+  |             |    |
|             |            |                    |            |  |  +-------------+    |
+-------------+            +--------------------+            |  |                     |
                                                             |  |                     |
                                                             |  |                     |
                                                             |  |  +-------------+    |
                                                             |  |  |             |    |
                                                             |  +->+worker3:30010|    |
                                                             |     |             |    |
                                                             |     +-------------+    |
                                                             |                        |
                                                             +------------------------+

```
即客户端请求到达elb后，elb再将请求负载给后端的nodeport上，因nodeport默认会进行snat，将请求的源ip替换为node ip（原因是为了实现任意node的nodeport上都可以路由service的请求），为了避免node层的snat性能损耗和网络延迟（elb会做一层snat），需要在创建LoadBalancer类型service的时候，将service中的`externalTrafficPolicy`属性设置为`Local`，同时elb只将流量负载给service pod所在节点的nodeport上
当service pod发生漂移后，elb-controller需要感知service真实的nodeport（可以接受请求的node的nodeport），并更新elb上的虚拟server的server group配置

### k8s事件监听
elb-controller一方面监听k8s service、endpoint及node事件，并根据如下规则执行相应操作:
* 监听到service创建事件后：
    判断是否需要处理（LoadBalancer类型，且有zcloud elb-controller annotation），若满足条件，创建elb service create任务，并加入任务队列
* 监听到service更新事件后：
    判断是否需要处理（LoadBalancer类型，且有zcloud elb-controller annotation），若满足条件，创建elb service update任务，并加入任务队列
* 监听到service删除事件后：
    判断是否需要处理（LoadBalancer类型，且有zcloud elb-controller annotation），若满足条件，创建elb service delete任务，并加入任务队列
* 监听到endpoint事件后：
    判断是否需要处理（LoadBalancer类型service的endpoint事件，且该service拥有zcloud elb-controller annotation），若满足条件，创建elb service update任务，加入任务队列
* 监听到node事件：
    根据监听到的node事件，更新elb-controller中缓存的nodename和ip对应关系表
> 创建elb task前，会为service添加finalizer，elb task执行成功或失败计数已满task丢弃的时候会删除此service的finalizer，即service在被elb-controller处理完之前不允许删除
### elb task处理
elb-controller会起一个线程，读取elb的任务队列，并调用api更新外部负载均衡设备的配置
下发配置时先向主elb下发配置，若失败，则向备elb进行下发，若两者都失败，则将task重新加入任务队列
* task分类：
    * create task
    * update task
    * delete task
* 错误处理：
每个task有失败次数，若达到一定次数，则会丢弃此task，防止反复执行占用cpu，同时会记录错误日志，方便问题排查
> create和update任务结束后，会更新LoadBalancer service的status，补充eip信息
> elb配置下发均为强制覆盖，若有同名配置项，会被覆盖，仅做warn日志记录
### elb-controller启动
* 启动参数：主备elb api地址，用户名，密码（或token等）
* 首次启动后会list集群所有service，筛选出符合条件的service，并根据该service的endpoint及node列表，生成相应的elb配置，向elb查询该service配置是否存在以及是否需要更新
    * 若elb上缺失配置，则创建elb service create任务，加入任务队列
    * 若elb上配置与期望配置不一致，则创建elb service update任务，并加入任务队列
* 启动k8s事件监听的线程和任务处理的线程

## annotation设计
功能划分：
1. 指定负载均衡算法：轮询、最小连接、源ip hash
2. 源ip snat配置：是否进行snat（默认开启）
3. 会话保持配置
4. 服务vip（必需项）

## todo
* 支持L7负载（即支持Ingress）：需考虑外部负载设备场景下Ingress的对象限制，如果Ingress的后端service类型是ClusterIP则外部负载设备实际是无法访问到该service的
