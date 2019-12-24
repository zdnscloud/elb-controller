# User-guide
## 前提
* zcloud k8s集群
* radware负载均衡设备或虚拟机
## 部署
修改deploy.yml中的启动参数后通过kubectl进行部署，需要修改的参数如下：
* -master:radware master设备管理地址
* -backup:radware backup设备管理地址（可选，仅ha场景下需要）
* -user:radware设备管理用户
* -password:radware密码
* -cluster:k8s集群名称
`kubectl apply -f ../deploy/deploy.yml`
## 使用
* annoation
    1. lb.zcloud.cn/vip:指定负载均衡设备上虚拟服务的服务ip
    2. lb.zcloud.cn/method:指定负载均衡算法，目前支持rr（轮询）、lc（最小连接）、hash（源ip hash）
> vip必须指定，若无vip annoation，controller会忽略该service；负载均衡算法默认为rr，可不指定
* finalizer
创建LoadBalancer service建议配置finalizer（为了在删除时不残留负载均衡配置），如下：
```yaml
finalizers:
    - lb.zcloud.cn/protect
```
* externalTrafficPolicy
LoadBalancer service externalTrafficPolicy属性默认为Cluster，建议修改为Local，减少请求进入集群后的snat环节
* service示例如下：
```yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    lb.zcloud.cn/method: "rr"
    lb.zcloud.cn/vip: 192.168.135.111
  name: lb-test1
  finalizers:
    - lb.zcloud.cn/protect
spec:
  ports:
    - name: tcp-9090
      port: 9090
      protocol: TCP
      targetPort: 9090
  selector:
    app: lb-test
  sessionAffinity: None
  externalTrafficPolicy: Local
  type: LoadBalancer
```