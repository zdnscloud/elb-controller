package client

import (
	"fmt"

	"github.com/zdnscloud/elb-controller/driver/radware/types"
)

const (
	virtualServicePath      = "/config/SlbNewCfgEnhVirtServicesTable/"
	virtualServicePart2Path = "/config/SlbNewCfgEnhVirtServicesSixthPartTable/"
	virtualServicePart3Path = "/config/SlbNewCfgEnhVirtServicesSeventhPartTable/"
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

func (c *VirtualServiceClient) Create(id string, obj *types.VirtualService) error {
	if err := create(c.genUrl(id), c.token, obj); err != nil {
		return err
	}
	if err := c.setXForwarde(id); err != nil {
		return err
	}
	return c.setRSGroup(id, id)
}

func (c *VirtualServiceClient) Update(id string, obj *types.VirtualService) error {
	return update(c.genUrl(id), c.token, obj)
}

func (c *VirtualServiceClient) setXForwarde(id string) error {
	return update(c.genPart2Url(id), c.token, &types.VirtualServicePart2{
		XForwardedFor: 1,
	})
}

func (c *VirtualServiceClient) setRSGroup(id string, sgID string) error {
	return update(c.genPart3Url(id), c.token, &types.VirtualServicePart3{
		RealGroup:         sgID,
		PersistentTimeOut: 10,
		ProxyIpMode:       1,
	})
}

func (c *VirtualServiceClient) Delete(id string) error {
	return delete(c.genUrl(id), c.token)
}

func (c *VirtualServiceClient) Get(id string) (*types.VirtualService, error) {
	list := &types.VirtualServiceList{}
	if err := get(c.genUrl(id), c.token, list); err != nil {
		return nil, err
	}
	if len(list.VSTable) == 0 {
		return nil, ResourceNotFoundError
	}
	return &list.VSTable[0], nil
}

func (c *VirtualServiceClient) genUrl(id string) string {
	return fmt.Sprintf("%s%s%s%s/1", reqUrlPrefix, c.server, virtualServicePath, id)
}

func (c *VirtualServiceClient) genPart2Url(id string) string {
	return fmt.Sprintf("%s%s%s%s/1", reqUrlPrefix, c.server, virtualServicePart2Path, id)
}

func (c *VirtualServiceClient) genPart3Url(id string) string {
	return fmt.Sprintf("%s%s%s%s/1", reqUrlPrefix, c.server, virtualServicePart3Path, id)
}
