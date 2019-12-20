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

func (c *ServerGroupClient) Reconcile(id string, sg *types.ServerGroup) error {
	exist, err := c.get(id)
	if err != nil {
		if err == ResourceNotFoundError {
			return c.create(id, sg)
		}
		return err
	}

	if isServerGroupEqual(exist, sg) {
		return nil
	}
	return c.update(id, sg)
}

func (c *ServerGroupClient) AddServer(id, rsID string) error {
	return c.update(id, types.NewAddServerServerGroup(rsID))
}

func (c *ServerGroupClient) RemoveServer(id, rsID string) error {
	return c.update(id, types.NewRemoveServerServerGroup(rsID))
}

func (c *ServerGroupClient) create(id string, obj *types.ServerGroup) error {
	return create(c.genUrl(id), c.token, obj)
}

func (c *ServerGroupClient) update(id string, obj *types.ServerGroup) error {
	return update(c.genUrl(id), c.token, obj)
}

func (c *ServerGroupClient) Delete(id string) error {
	return delete(c.genUrl(id), c.token)
}

func (c *ServerGroupClient) get(id string) (*types.ServerGroup, error) {
	list := &types.ServerGroupList{}
	if err := get(c.genUrl(id), c.token, list); err != nil {
		return nil, err
	}
	if len(list.SGTable) == 0 {
		return nil, ResourceNotFoundError
	}
	return &list.SGTable[0], nil
}

func isServerGroupEqual(s1, s2 *types.ServerGroup) bool {
	if s1 == nil || s2 == nil {
		return false
	}
	return s1.Metric == s2.Metric && s1.HealthID == s2.HealthID
}

func (c *ServerGroupClient) genUrl(id string) string {
	return fmt.Sprintf("%s%s%s%s", reqUrlPrefix, c.server, serverGroupPath, id)
}
