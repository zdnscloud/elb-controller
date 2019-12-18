package lbctrl

import (
	"context"
	"fmt"
	"time"

	"github.com/zdnscloud/elb-controller/driver"

	"github.com/zdnscloud/gok8s/client"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

const (
	ZcloudLBVIPAnnotationKey    = "lb.zcloud.cn/vip"
	ZcloudLBMethodAnnotationKey = "lb.zcloud.cn/method"

	k8sGetRetries       = 5
	k8sGetRetryInterval = 3 * time.Second
)

var ServiceNoNeedHandleError = fmt.Errorf("not loadbalancer service or has none zcloud lb annotation")

func genLBConfigByService(c client.Client, svc *corev1.Service, clusterName string, nodeIpMap map[string]string) (driver.Config, error) {
	result := driver.Config{
		K8sCluster:   clusterName,
		K8sNamespace: svc.Namespace,
		K8sService:   svc.Name,
		VIP:          getLBConfigVIP(svc),
		Method:       getLBConfigMethod(svc),
		Services:     []driver.Service{},
	}

	endpoints := &corev1.Endpoints{}
	for i := 0; i < k8sGetRetries; i++ {
		if err := c.Get(context.TODO(), types.NamespacedName{Namespace: svc.Namespace, Name: svc.Name}, endpoints); err != nil {
			if apierrors.IsNotFound(err) {
				i += 1
				time.Sleep(k8sGetRetryInterval)
			} else {
				return result, err
			}
		}
	}

	svcNodesIP, err := getServiceNodesIP(nodeIpMap, endpoints)
	if err != nil {
		return result, err
	}

	for _, port := range svc.Spec.Ports {
		lbService := driver.Service{
			Port:         port.Port,
			BackendPort:  port.NodePort,
			BackendHosts: svcNodesIP,
			Protocol:     getLBConfigProtocol(port.Protocol),
		}
		result.Services = append(result.Services, lbService)
	}
	return result, nil
}

func genLBConfigByEndpoins(c client.Client, eps *corev1.Endpoints, clusterName string, nodeIpMap map[string]string) (driver.Config, error) {
	result := driver.Config{
		K8sCluster:   clusterName,
		K8sNamespace: eps.Namespace,
		K8sService:   eps.Name,
		Services:     []driver.Service{},
	}

	svc := &corev1.Service{}
	if err := c.Get(context.TODO(), types.NamespacedName{Namespace: eps.Namespace, Name: eps.Name}, svc); err != nil {
		return result, err
	}

	result.VIP = getLBConfigVIP(svc)
	result.Method = getLBConfigMethod(svc)

	if !isServiceNeedHandle(svc) {
		return result, ServiceNoNeedHandleError
	}

	svcNodesIP, err := getServiceNodesIP(nodeIpMap, eps)
	if err != nil {
		return result, err
	}

	for _, port := range svc.Spec.Ports {
		lbService := driver.Service{
			Port:         port.Port,
			BackendPort:  port.NodePort,
			BackendHosts: svcNodesIP,
			Protocol:     getLBConfigProtocol(port.Protocol),
		}
		result.Services = append(result.Services, lbService)
	}
	return result, nil
}

func getServiceNodesIP(nodeIpMap map[string]string, eps *corev1.Endpoints) ([]string, error) {
	if len(eps.Subsets) == 0 {
		return nil, nil
	}
	nodes := make(map[string]bool)
	for _, addr := range eps.Subsets[0].Addresses {
		nodes[*addr.NodeName] = true
	}
	for _, addr := range eps.Subsets[0].NotReadyAddresses {
		nodes[*addr.NodeName] = true
	}

	ips := make([]string, 0)
	for key := range nodes {
		ips = append(ips, nodeIpMap[key])
	}
	return ips, nil
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

func getLBConfigProtocol(svcProtoco corev1.Protocol) driver.Protocol {
	switch svcProtoco {
	case corev1.ProtocolUDP:
		return driver.ProtocolUDP
	default:
		return driver.ProtocolTCP
	}
}

func getLBConfigVIP(svc *corev1.Service) string {
	v, ok := svc.Annotations[ZcloudLBVIPAnnotationKey]
	if ok && v != "" {
		return v
	}
	return ""
}

func getLBConfigMethod(svc *corev1.Service) driver.LoadBalanceMethod {
	v, ok := svc.Annotations[ZcloudLBMethodAnnotationKey]
	if ok && v != "" {
		switch v {
		case "rr":
			return driver.LBMethodRoundRobin
		case "lc":
			return driver.LBMethodLeastConnections
		case "hash":
			return driver.LBMethodHash
		}
	}
	return ""
}

func isServiceNeedHandle(svc *corev1.Service) bool {
	return hasZcloudLBAnnotation(svc) && isLoadBalancerService(svc)
}

func hasZcloudLBAnnotation(svc *corev1.Service) bool {
	if _, ok := svc.Annotations[ZcloudLBVIPAnnotationKey]; ok {
		return true
	}

	if _, ok := svc.Annotations[ZcloudLBVIPAnnotationKey]; ok {
		return true
	}
	return false
}

func isLoadBalancerService(svc *corev1.Service) bool {
	return svc.Spec.Type == corev1.ServiceTypeLoadBalancer
}
