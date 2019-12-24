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
    * 判断是否需要处理：LoadBalancer类型服务，且有zcloud lb vip annotation
        * 判断service删除时间是否为空，若不为空，创建elb service delete任务，加入任务队列
        * 若service删除时间为空则创建elb service create任务，加入任务队列
* 监听到service更新事件后：
    * 判断是否需要处理
        * 判断service删除时间是否为空，若不为空，创建elb service delete任务，加入任务队列
        * 若service删除时间为空，则创建elb service update任务，加入任务队列
* 监听到service删除事件后：
    * 判断是否需要处理
        * 创建elb service delete任务，并加入任务队列
* 监听到endpoint事件后：
    * 向api-server请求同namespace同名service，并判断是否需要处理
        * 创建elb service update任务，加入任务队列
* 监听到node事件：
    * node创建事件：将node name和ip加入至elb-controller nodes属性中（map）
    * node删除事件，删除elb-controller nodes属性中对应项
### elb task处理
elb-controller会起一个线程，读取elb的任务队列，并调用api更新外部负载均衡设备的配置
* task分类：
    * create task
    * update task
    * delete task:任务执行完成后会自动移除service上的zcloud lb finalizer
    > finalizer存在的意义是为了保证elb-controller可以完全清除掉负载均衡器上的相关所有配置；不设置finalizer情况下controller会因为获取不到service的endpoints导致无法删除负载均衡器上的realserver配置
* 错误处理：
    * 若task执行失败，会记录日志，若未达到最大失败次数，会增加失败计数后再次将该task加入任务队列
    * 若达到最大失败次数（3次），则会丢弃此task，防止反复执行占用cpu
### elb-controller启动
* 根据启动参数（elb api地址，用户名，密码）初始化elb-controller对象，并向api-server list node，初始化elb-controller对象内的node name和ip缓存map
* 启动k8s事件监听线程：首次启动后会list集群所有service，筛选出符合条件的service，并根据该service的endpoint及node列表，生成相应的elb配置，向elb查询该service配置是否存在以及是否需要更新
    * 若elb上缺失配置，则创建elb service create任务，加入任务队列
    * 若elb上配置与期望配置不一致，则创建elb service update任务，并加入任务队列
    * 若存在有删除时间不为空的service（符合条件），则创建elb service delete任务，加入任务队列
* 启动任务处理的线程
### loadbalance driver
目前实现了radware的适配驱动，并支持负载均衡器双机ha部署
* ha逻辑
RadwareDriver对象包含ha双机的client对象，在执行task前，先获取当前master角色的client，再执行task
> 若启动elb-controller时没有填写backup server地址，则相当于单机模式
## annotation设计
功能划分：
1. 指定负载均衡算法：轮询、最小连接、源ip hash
2. 服务vip（必需项）
## 目录结构
* cmd:main入口函数
* lbctrl:controller模块，k8s事件监听及任务队列管理
* driver:外部负载均衡器driver实现，目前实现了radware的适配支持，主要提供对l4负载策略的配置接口（create、update、delete）
## todo
* 支持外部负载均衡ha部署模式自动切换功能
* 支持L7负载（即支持Ingress）：需考虑外部负载设备场景下Ingress的对象限制，如果Ingress的后端service类型是ClusterIP则外部负载设备实际是无法访问到该service的
