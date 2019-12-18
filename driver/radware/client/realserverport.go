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

func (c *RealServerPortClient) Create(id string, p *types.RealServerPort) error {
	return create(c.genUrl(id), c.token, p)
}

func (c *RealServerPortClient) Update(id string, p *types.RealServerPort) error {
	return update(c.genUrl(id), c.token, p)
}

func (c *RealServerPortClient) Delete(id string) error {
	return delete(c.genUrl(id), c.token)
}

func (c *RealServerPortClient) Get(id string) (*types.RealServerPort, error) {
	list := &types.RealServerPortList{}
	if err := get(c.genUrl(id), c.token, list); err != nil {
		return nil, err
	}
	if len(list.RSPortTable) == 0 {
		return nil, ResourceNotFoundError
	}
	return &list.RSPortTable[0], nil
}

func (c *RealServerPortClient) genUrl(id string) string {
	return fmt.Sprintf("%s%s%s%s/1", reqUrlPrefix, c.server, realServerPortPath, id)
}
