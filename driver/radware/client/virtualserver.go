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

func (c *VirtualServerClient) Reconcile(id string, vs *types.VirtualServer) error {
	exist, err := c.get(id)
	if err != nil {
		if err == ResourceNotFoundError {
			return c.create(id, vs)
		}
		return err
	}
	if isVirtualServerEqual(exist, vs) {
		return nil
	}
	return c.update(id, vs)
}

func (c *VirtualServerClient) create(id string, obj *types.VirtualServer) error {
	return create(c.genUrl(id), c.token, obj)
}

func (c *VirtualServerClient) update(id string, obj *types.VirtualServer) error {
	return update(c.genUrl(id), c.token, obj)
}

func (c *VirtualServerClient) Delete(id string) error {
	_, err := c.get(id)
	if err != nil {
		if err == ResourceNotFoundError {
			return nil
		}
		return err
	}
	return delete(c.genUrl(id), c.token)
}

func (c *VirtualServerClient) get(id string) (*types.VirtualServer, error) {
	list := &types.VirtualServerList{}
	if err := get(c.genUrl(id), c.token, list); err != nil {
		return nil, err
	}
	if len(list.VSTable) == 0 {
		return nil, ResourceNotFoundError
	}
	return &list.VSTable[0], nil
}

func isVirtualServerEqual(v1, v2 *types.VirtualServer) bool {
	if v1 == nil || v2 == nil {
		return false
	}
	return v1.VirtServerIpAddress == v2.VirtServerIpAddress && v1.VirtServerState == v2.VirtServerState
}

func (c *VirtualServerClient) genUrl(id string) string {
	return fmt.Sprintf("%s%s%s%s", reqUrlPrefix, c.server, virtualServerPath, id)
}
