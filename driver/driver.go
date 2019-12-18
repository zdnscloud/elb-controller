package driver

import (
	"encoding/json"
)

type Protocol string
type LoadBalanceMethod string

const (
	ProtocolTCP Protocol = "tcp"
	ProtocolUDP Protocol = "udp"

	LBMethodRoundRobin       LoadBalanceMethod = "rr"
	LBMethodLeastConnections LoadBalanceMethod = "lc"
	LBMethodHash             LoadBalanceMethod = "hash"
)

type Driver interface {
	Create(Config) error
	Update(old, new Config) error
	Delete(Config) error
	Version() string
}

type Config struct {
	K8sCluster   string            `json:"k8sCluster"`
	K8sNamespace string            `json:"k8sNamespace"`
	K8sService   string            `json:"k8sService"`
	VIP          string            `json:"vip"`
	Method       LoadBalanceMethod `json:"method"`
	Services     []Service         `json:"services"`
}

type Service struct {
	Port         int32    `json:"port"`
	BackendPort  int32    `json:"backendPort"`
	BackendHosts []string `json:"backendHosts"`
	Protocol     Protocol `json:"protocol"`
}

func (c Config) ToJson() string {
	b, _ := json.Marshal(c)
	return string(b)
}
