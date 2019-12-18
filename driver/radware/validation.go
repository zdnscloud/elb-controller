package radware

import (
	"fmt"
	"net"

	"github.com/zdnscloud/elb-controller/driver"
)

const maxIDLength = 255

func validateConfig(c driver.Config) error {
	if err := validateConfigOption(c); err != nil {
		return fmt.Errorf("driver config validate failed %s", err.Error())
	}
	if err := validateGenIdLength(c); err != nil {
		return fmt.Errorf("driver config validate failed %s", err.Error())
	}
	return nil
}

func validateGenIdLength(c driver.Config) error {
	for _, s := range c.Services {
		vsID := genVsID(s, c)
		if len(vsID) > maxIDLength {
			return fmt.Errorf("gen vsid %s exceed max id length %v", vsID, maxIDLength)
		}
		for rsID := range getRsmap(c, s) {
			if len(rsID) > maxIDLength {
				return fmt.Errorf("gen rsid %s exceed max id length %v", rsID, maxIDLength)
			}
		}
	}
	return nil
}

func validateConfigOption(c driver.Config) error {
	if c.K8sCluster == "" {
		return fmt.Errorf("K8sCluster field empty")
	}
	if c.K8sNamespace == "" {
		return fmt.Errorf("K8sNamespace field empty")
	}
	if c.K8sService == "" {
		return fmt.Errorf("K8sService field empty")
	}
	if !isIPv4(c.VIP) {
		return fmt.Errorf("VIP field %s isn't an ipv4 address", c.VIP)
	}
	return validateConfigServices(c)
}

func validateConfigServices(c driver.Config) error {
	if len(c.Services) == 0 {
		return nil
	}
	for _, s := range c.Services {
		if err := validateConfigService(s); err != nil {
			return err
		}
	}
	return nil
}

func validateConfigService(s driver.Service) error {
	if !isPort(s.Port) {
		return fmt.Errorf("service port %v isn't legal", s.Port)
	}
	if !isPort(s.BackendPort) {
		return fmt.Errorf("service backendPort %v isn't legal", s.Port)
	}
	for _, h := range s.BackendHosts {
		if !isIPv4(h) {
			return fmt.Errorf("service backendhost %s isn't an ipv4 address", h)
		}
	}
	if s.Protocol == "" {
		return fmt.Errorf("service Protocol is empty")
	}
	return nil
}

func isIPv4(input string) bool {
	ip := net.ParseIP(input)
	return ip != nil && ip.To4() != nil
}

func isPort(p int32) bool {
	return p > 0 && p <= 65535
}
