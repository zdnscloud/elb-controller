package radware

import (
	"fmt"

	"github.com/zdnscloud/elb-controller/driver"
	"github.com/zdnscloud/elb-controller/driver/radware/types"
)

type radwareConfig struct {
	RealServers    map[string]*types.RealServer
	RealServerPort *types.RealServerPort
	VsID           string
	ServerGroup    *types.ServerGroup
	VirtualServer  *types.VirtualServer
	VirtualService *types.VirtualService
}

type updateRadwareConfig struct {
	old radwareConfig
	new radwareConfig
}

func isSameRsport(c updateRadwareConfig) bool {
	return c.old.RealServerPort.RealPort == c.new.RealServerPort.RealPort
}

func getRadwareConfigs(config driver.Config) []radwareConfig {
	result := []radwareConfig{}

	for _, s := range config.Services {
		c := radwareConfig{}
		c.RealServers = getRsmap(config, s)
		c.RealServerPort = getRsport(s)
		c.VsID = genVsID(s, config)
		c.ServerGroup = getServerGroup(s, config)
		c.VirtualServer = getVirtualServer(config)
		c.VirtualService = getVirtualService(s)
		result = append(result, c)
	}

	return result
}

func getToDeleteRdConfigs(old, new []radwareConfig) []radwareConfig {
	result := []radwareConfig{}
	for _, oldc := range old {
		var found bool
		for _, newc := range new {
			if oldc.VsID == newc.VsID {
				found = true
				break
			}
		}
		if !found {
			result = append(result, oldc)
		}
	}
	return result
}

func getToAddRdConfigs(old, new []radwareConfig) []radwareConfig {
	result := []radwareConfig{}
	for _, newc := range new {
		var found bool
		for _, oldc := range old {
			if oldc.VsID == newc.VsID {
				found = true
				break
			}
		}
		if !found {
			result = append(result, newc)
		}
	}
	return result
}

func getUpdateRdConfigs(old, new []radwareConfig) []updateRadwareConfig {
	result := []updateRadwareConfig{}
	for _, newc := range new {
		for _, oldc := range old {
			if oldc.VsID == newc.VsID {
				result = append(result, updateRadwareConfig{
					old: oldc,
					new: newc,
				})
				break
			}
		}
	}
	return result
}

func getToDeleteRsmap(old, new radwareConfig) map[string]*types.RealServer {
	result := map[string]*types.RealServer{}
	for k, v := range old.RealServers {
		if _, ok := new.RealServers[k]; !ok {
			result[k] = v
		}
	}
	return result
}

func getToAddRsmap(old, new radwareConfig) map[string]*types.RealServer {
	result := map[string]*types.RealServer{}
	for k, v := range new.RealServers {
		if _, ok := old.RealServers[k]; !ok {
			result[k] = v
		}
	}
	return result
}

func getRsmap(cfg driver.Config, s driver.Service) map[string]*types.RealServer {
	result := map[string]*types.RealServer{}
	for _, h := range s.BackendHosts {
		id := genRealServerID(h, s.BackendPort, s.Protocol, cfg)
		rs := &types.RealServer{
			IpAddr: h,
			State:  2,
			Type:   1,
		}
		result[id] = rs
	}
	return result
}

func getRsport(s driver.Service) *types.RealServerPort {
	return &types.RealServerPort{
		RealPort: s.BackendPort,
	}
}

func genRealServerID(nodeIP string, nodePort int32, protocol driver.Protocol, cfg driver.Config) string {
	return fmt.Sprintf("%s_%s_%s_%s_%s_%v", cfg.K8sCluster, cfg.K8sNamespace, cfg.K8sService, nodeIP, protocol, nodePort)
}

func getServerGroup(s driver.Service, c driver.Config) *types.ServerGroup {
	result := &types.ServerGroup{
		Metric:   1,     // default use round robin
		HealthID: "tcp", // default use tcp healthcheck
	}
	switch c.Method {
	case driver.LBMethodLeastConnections:
		result.Metric = 2
	case driver.LBMethodHash:
		result.Metric = 4
	}

	if s.Protocol == driver.ProtocolUDP {
		result.HealthID = "udp"
	}
	return result
}

func genVsID(service driver.Service, cfg driver.Config) string {
	return fmt.Sprintf("%s_%s_%s_%s_%s_%v", cfg.K8sCluster, cfg.K8sNamespace, cfg.K8sService, cfg.VIP, service.Protocol, service.Port)
}

func getVirtualServer(c driver.Config) *types.VirtualServer {
	return &types.VirtualServer{
		VirtServerIpAddress: c.VIP,
		VirtServerState:     2,
	}
}

func getVirtualService(s driver.Service) *types.VirtualService {
	result := &types.VirtualService{
		UDPBalance: 3, // default tcp service type
		VirtPort:   s.Port,
		RealPort:   s.BackendPort,
		DBind:      3,
		PBind:      2,
	}
	if s.Protocol == driver.ProtocolUDP {
		result.UDPBalance = 2 // set virtual service type udp
	}
	return result
}
