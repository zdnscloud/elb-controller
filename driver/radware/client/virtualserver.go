package client

import (
	"fmt"

	"github.com/zdnscloud/elb-controller/driver/radware/types"
)

const (
	virtualServerPath = "/config/SlbNewCfgEnhVirtServerTable/"
)

type VirtualServerClient struct {
	token  string
	server string
}

func NewVirtualServerClient(token, serverAddr string) *VirtualServerClient {
	return &VirtualServerClient{
		token:  token,
		server: serverAddr,
	}
}

func (c *VirtualServerClient) Create(id string, obj *types.VirtualServer) error {
	return create(c.genUrl(id), c.token, obj)
}

func (c *VirtualServerClient) Update(id string, obj *types.VirtualServer) error {
	return update(c.genUrl(id), c.token, obj)
}

func (c *VirtualServerClient) Delete(id string) error {
	return delete(c.genUrl(id), c.token)
}

func (c *VirtualServerClient) Get(id string) (*types.VirtualServer, error) {
	list := &types.VirtualServerList{}
	if err := get(c.genUrl(id), c.token, list); err != nil {
		return nil, err
	}
	if len(list.VSTable) == 0 {
		return nil, ResourceNotFoundError
	}
	return &list.VSTable[0], nil
}

func (c *VirtualServerClient) genUrl(id string) string {
	return fmt.Sprintf("%s%s%s%s", reqUrlPrefix, c.server, virtualServerPath, id)
}
