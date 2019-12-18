package client

import (
	"fmt"

	"github.com/zdnscloud/elb-controller/driver/radware/types"
)

const (
	serverGroupPath = "/config/SlbNewCfgEnhGroupTable/"
)

type ServerGroupClient struct {
	token  string
	server string
}

func NewServerGroupClient(token, serverAddr string) *ServerGroupClient {
	return &ServerGroupClient{
		token:  token,
		server: serverAddr,
	}
}

func (c *ServerGroupClient) Create(id string, obj *types.ServerGroup) error {
	return create(c.genUrl(id), c.token, obj)
}

func (c *ServerGroupClient) Update(id string, obj *types.ServerGroup) error {
	return update(c.genUrl(id), c.token, obj)
}

func (c *ServerGroupClient) Delete(id string) error {
	return delete(c.genUrl(id), c.token)
}

func (c *ServerGroupClient) Get(id string) (*types.ServerGroup, error) {
	list := &types.ServerGroupList{}
	if err := get(c.genUrl(id), c.token, list); err != nil {
		return nil, err
	}
	if len(list.SGTable) == 0 {
		return nil, ResourceNotFoundError
	}
	return &list.SGTable[0], nil
}

func (c *ServerGroupClient) genUrl(id string) string {
	return fmt.Sprintf("%s%s%s%s", reqUrlPrefix, c.server, serverGroupPath, id)
}
