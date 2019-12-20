package client

import (
	"fmt"

	"github.com/zdnscloud/elb-controller/driver/radware/types"
)

const (
	virtualServicePath          = "/config/SlbNewCfgEnhVirtServicesTable/"
	virtualServiceRealGroupPath = "/config/SlbNewCfgEnhVirtServicesSeventhPartTable/"
)

type VirtualServiceClient struct {
	token  string
	server string
}

func NewVirtualServiceClient(token, serverAddr string) *VirtualServiceClient {
	return &VirtualServiceClient{
		token:  token,
		server: serverAddr,
	}
}

func (c *VirtualServiceClient) Reconcile(id string, vs *types.VirtualService) error {
	exist, err := c.get(id)
	if err != nil {
		if err == ResourceNotFoundError {
			return c.create(id, vs)
		}
		return err
	}

	existGroup, err := c.getRealGroup(id)
	if err != nil {
		return err
	}

	if !isVirtualServiceRealGroupEqual(existGroup, types.NewVirtualServiceRealGroup(id)) {
		if err := c.setRealGroup(id); err != nil {
			return err
		}
	}
	if isVirtualServiceEqual(exist, vs) {
		return nil
	}
	return c.update(id, vs)
}

func (c *VirtualServiceClient) Delete(id string) error {
	return delete(c.genUrl(id), c.token)
}

func (c *VirtualServiceClient) create(id string, obj *types.VirtualService) error {
	if err := create(c.genUrl(id), c.token, obj); err != nil {
		return err
	}
	return c.setRealGroup(id)
}

func (c *VirtualServiceClient) update(id string, obj *types.VirtualService) error {
	return update(c.genUrl(id), c.token, obj)
}

func (c *VirtualServiceClient) setRealGroup(id string) error {
	return update(c.genRealGroupUrl(id), c.token, types.NewVirtualServiceRealGroup(id))
}

func (c *VirtualServiceClient) get(id string) (*types.VirtualService, error) {
	list := &types.VirtualServiceList{}
	if err := get(c.genUrl(id), c.token, list); err != nil {
		return nil, err
	}
	if len(list.VSTable) == 0 {
		return nil, ResourceNotFoundError
	}
	return &list.VSTable[0], nil
}

func (c *VirtualServiceClient) getRealGroup(id string) (*types.VirtualServiceRealGroup, error) {
	list := &types.VirtualServiceRealGroupList{}
	if err := get(c.genRealGroupUrl(id), c.token, list); err != nil {
		return nil, err
	}
	if len(list.VSTable) == 0 {
		return nil, ResourceNotFoundError
	}
	return &list.VSTable[0], nil
}

func isVirtualServiceEqual(v1, v2 *types.VirtualService) bool {
	if v1 == nil || v2 == nil {
		return false
	}
	return v1.UDPBalance == v2.UDPBalance && v1.VirtPort == v2.VirtPort && v1.RealPort == v2.RealPort && v1.DBind == v2.DBind && v1.PBind == v2.PBind
}

func isVirtualServiceRealGroupEqual(g1, g2 *types.VirtualServiceRealGroup) bool {
	if g1 == nil || g2 == nil {
		return false
	}
	return g1.RealGroup == g2.RealGroup && g1.ProxyIpMode == g2.ProxyIpMode && g1.PersistentTimeOut == g2.PersistentTimeOut
}

func (c *VirtualServiceClient) genUrl(id string) string {
	return fmt.Sprintf("%s%s%s%s/1", reqUrlPrefix, c.server, virtualServicePath, id)
}

func (c *VirtualServiceClient) genRealGroupUrl(id string) string {
	return fmt.Sprintf("%s%s%s%s/1", reqUrlPrefix, c.server, virtualServiceRealGroupPath, id)
}
