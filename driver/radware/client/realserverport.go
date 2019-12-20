package client

import (
	"fmt"

	"github.com/zdnscloud/elb-controller/driver/radware/types"
)

const (
	realServerPortPath = "/config/SlbNewCfgEnhRealServPortTable/"
)

type RealServerPortClient struct {
	token  string
	server string
}

func NewRealServerPortClient(token, serverAddr string) *RealServerPortClient {
	return &RealServerPortClient{
		token:  token,
		server: serverAddr,
	}
}

func (c *RealServerPortClient) Reconcile(id string, p *types.RealServerPort) error {
	exist, err := c.get(id)
	if err != nil {
		if err == ResourceNotFoundError {
			return c.create(id, p)
		}
		return err
	}
	if isRealServerPortEqual(exist, p) {
		return nil
	}
	return c.update(id, p)
}

func (c *RealServerPortClient) create(id string, p *types.RealServerPort) error {
	return create(c.genUrl(id), c.token, p)
}

func (c *RealServerPortClient) update(id string, p *types.RealServerPort) error {
	return update(c.genUrl(id), c.token, p)
}

func (c *RealServerPortClient) Delete(id string) error {
	_, err := c.get(id)
	if err != nil {
		if err == ResourceNotFoundError {
			return nil
		}
		return err
	}
	return delete(c.genUrl(id), c.token)
}

func (c *RealServerPortClient) get(id string) (*types.RealServerPort, error) {
	list := &types.RealServerPortList{}
	if err := get(c.genUrl(id), c.token, list); err != nil {
		return nil, err
	}
	if len(list.RSPortTable) == 0 {
		return nil, ResourceNotFoundError
	}
	return &list.RSPortTable[0], nil
}

func isRealServerPortEqual(p1, p2 *types.RealServerPort) bool {
	if p1 == nil || p2 == nil {
		return false
	}
	return p1.RealPort == p2.RealPort
}

func (c *RealServerPortClient) genUrl(id string) string {
	return fmt.Sprintf("%s%s%s%s/1", reqUrlPrefix, c.server, realServerPortPath, id)
}
