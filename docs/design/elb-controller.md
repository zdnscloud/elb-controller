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
* endpoints create event：
    * 根据endpoints的namespace和name获取service，并判断svc是否需要处理
        * 判断service DeletionTime不为空，创建elb delete任务，加入任务队列；否则创建elb create任务，加入任务队列
* service update event：
    * 判断svc是否需要处理
        * 若service DeletionTime不为空，创建elb service delete任务，加入任务队列
        * 判断更新前后svc的annotation或spec是否有更新，若annotation和spec均无更新，直接返回
            * 创建elb update任务，加入任务队列
* endpoints update event：
    * 根据endpoints的namespace和name获取service，并判断svc是否需要处理
        * 判断endpoints的Subsets是否有更新，若无更新直接返回
        * 创建elb update任务，加入任务队列
* node create event：
    * 将node name和ip加入至elb-controller缓存中
* node delete event：
    * 删除elb-controller缓存中对应node信息
> service是否需要处理的判断标准：1.为LoadBalancer类型svc 2.该svc有zcloud lb vip annotation（lb.zcloud.cn/vip）
### elb task处理
elb-controller会起一个线程，读取elb的任务队列，并调用api更新外部负载均衡设备的配置
* task分类及处理逻辑：
    * create task
        * 调用lb driver的Create接口创建相应的lb配置
        * 更新svc status（设置status中的lb vip）
        * 为svc和svc的endpoints添加finalizer
    * update task
        * 调用lb driver的Update接口更新对应的lb配置
        * 根据更新前后配置判断是否需要更新svc status（lb vip）
    * delete task
        * 调用lb driver的Delete接口删除对应的lb配置
        * 移除svc和svc endpoints上的finalizer
         > finalizer存在的意义是为了保证elb-controller可以完全清除掉负载均衡器上的相关所有配置；不设置finalizer情况下controller会因为获取不到service的endpoints导致无法删除负载均衡器上的realserver配置
* 错误处理：
    * 若task执行失败，会记录日志，若未达到最大失败次数，会增加失败计数后再次将该task加入任务队列
    * 若达到最大失败次数（5次），则会丢弃此task，防止反复执行占用cpu
### elb-controller启动
* 根据启动参数（elb api地址，用户名，密码）初始化elb-controller对象，并向api-server list node，初始化elb-controller对象内的node name和ip缓存map
* 启动k8s事件监听线程
    * 首次启动会list集群中所有svc，若svc需要处理，创建elb create任务，加入任务队列
* 启动任务处理的线程
    > k8s事件监听和elb任务处理同时进行，不存在LoadBalancer svc数量超过任务队列长度导致阻塞的问题
### loadbalance driver
目前实现了radware的适配驱动，并支持radware整机HA部署模式
* HA逻辑
RadwareDriver对象包含ha双机的client对象，在执行task时，先获取当前master角色的client，再进行task配置处理
> 若启动elb-controller时没有填写backup server地址，则相当于单机模式
* driver client处理逻辑
    * 在执行操作（创建、更新、删除）前，先get 检查资源是否存在，是否需要更新，若资源已存在且不需要进行更新，直接跳过
    > 该逻辑是为规避radware 相关配置api调用过于频繁可能会导致配置错乱的bug
## annotation设计
1. 指定负载均衡算法(可选)
    * key：lb.zdns.cn/method
    * value: rr(轮询)、lc(最小连接)、hash(源ip hash)
2. 服务vip（必填）
    * key：lb.zdns.cn/vip
    * key：ipv4 address
## 目录结构
* cmd:main入口函数
* lbctrl:controller模块，k8s事件监听及任务队列管理
* driver:外部负载均衡器driver实现，目前实现了radware的适配支持，主要提供对l4负载策略的配置接口（create、update、delete）
## todo
* 优化radware driver逻辑，提高效率，增加更多的异常处理
* 添加k8s event告警通知
* 支持L7负载（即支持Ingress）
