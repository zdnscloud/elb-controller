package client

import (
	"fmt"

	"github.com/zdnscloud/elb-controller/driver/radware/types"
)

const (
	realServerPath = "/config/SlbNewCfgEnhRealServerTable/"
)

type RealServerClient struct {
	token  string
	server string
}

func NewRealServerClient(token, serverAddr string) *RealServerClient {
	return &RealServerClient{
		token:  token,
		server: serverAddr,
	}
}

func (c *RealServerClient) Create(id string, rs *types.RealServer) error {
	return create(c.genUrl(id), c.token, rs)
}

func (c *RealServerClient) Update(id string, rs *types.RealServer) error {
	return update(c.genUrl(id), c.token, rs)
}

func (c *RealServerClient) Delete(id string) error {
	return delete(c.genUrl(id), c.token)
}

func (c *RealServerClient) Get(id string) (*types.RealServer, error) {
	rss := &types.RealServerList{}
	if err := get(c.genUrl(id), c.token, rss); err != nil {
		return nil, err
	}
	if len(rss.RSTable) == 0 {
		return nil, ResourceNotFoundError
	}
	return &rss.RSTable[0], nil
}

func (c *RealServerClient) genUrl(id string) string {
	return fmt.Sprintf("%s%s%s%s", reqUrlPrefix, c.server, realServerPath, id)
}
