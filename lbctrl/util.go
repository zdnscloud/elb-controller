package lbctrl

import (
	"context"
	"fmt"

	"github.com/zdnscloud/elb-controller/driver"

	"github.com/zdnscloud/gok8s/client"
	corev1 "k8s.io/api/core/v1"
)

const (
	ZcloudLBVIPAnnotationKey    = "lb.zcloud.cn/vip"
	ZcloudLBMethodAnnotationKey = "lb.zcloud.cn/method"
)

func genLBConfig(svc *corev1.Service, ep *corev1.Endpoints, clusterName string, nodeIpMap map[string]string) driver.Config {
	result := driver.Config{
		K8sCluster:   clusterName,
		K8sNamespace: ep.Namespace,
		K8sService:   ep.Name,
		Services:     []driver.Service{},
		VIP:          svc.Annotations[ZcloudLBVIPAnnotationKey],
		Method:       getLBConfigMethod(svc),
	}

	for _, port := range svc.Spec.Ports {
		lbService := driver.Service{
			Port:         port.Port,
			BackendPort:  port.NodePort,
			BackendHosts: getServiceNodesIP(nodeIpMap, ep),
			Protocol:     getLBConfigProtocol(port.Protocol),
		}
		result.Services = append(result.Services, lbService)
	}
	return result
}

func getServiceNodesIP(nodeIpMap map[string]string, ep *corev1.Endpoints) []string {
	if len(ep.Subsets) == 0 {
		return nil
	}
	nodes := make(map[string]bool)
	for _, addr := range ep.Subsets[0].Addresses {
		nodes[*addr.NodeName] = true
	}
	for _, addr := range ep.Subsets[0].NotReadyAddresses {
		nodes[*addr.NodeName] = true
	}

	ips := make([]string, 0)
	for key := range nodes {
		ips = append(ips, nodeIpMap[key])
	}
	return ips
}

func getLBConfigProtocol(p corev1.Protocol) driver.Protocol {
	if p == corev1.ProtocolUDP {
		return driver.ProtocolUDP
	}
	return driver.ProtocolTCP
}

func getNodeIPMap(c client.Client) (map[string]string, error) {
	nl := &corev1.NodeList{}
	if err := c.List(context.TODO(), &client.ListOptions{}, nl); err != nil {
		return nil, err
	}

	nodes := make(map[string]string)
	for _, n := range nl.Items {
		for _, addr := range n.Status.Addresses {
			if addr.Type == corev1.NodeInternalIP {
				nodes[n.Name] = addr.Address
			}
		}
	}
	return nodes, nil
}

func getLBConfigMethod(svc *corev1.Service) driver.LoadBalanceMethod {
	switch svc.Annotations[ZcloudLBMethodAnnotationKey] {
	case "lc":
		return driver.LBMethodLeastConnections
	case "hash":
		return driver.LBMethodHash
	default:
		return driver.LBMethodRoundRobin
	}
}

func isServiceNeedHandle(svc *corev1.Service) bool {
	_, ok := svc.Annotations[ZcloudLBVIPAnnotationKey]
	if isLoadBalancerService(svc) && ok {
		return true
	}
	return false
}

func isLoadBalancerService(svc *corev1.Service) bool {
	return svc.Spec.Type == corev1.ServiceTypeLoadBalancer
}

func genObjNamespacedName(namespace, name string) string {
	return fmt.Sprintf("{namespace:%s, name:%s}", namespace, name)
}
