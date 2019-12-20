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

func (c *RealServerClient) Reconcile(id string, rs *types.RealServer) error {
	exist, err := c.get(id)
	if err != nil {
		if err == ResourceNotFoundError {
			return c.create(id, rs)
		}
		return err
	}
	if isRealServerEqual(exist, rs) {
		return nil
	}
	return c.update(id, rs)
}

func (c *RealServerClient) create(id string, rs *types.RealServer) error {
	return create(c.genUrl(id), c.token, rs)
}

func (c *RealServerClient) update(id string, rs *types.RealServer) error {
	return update(c.genUrl(id), c.token, rs)
}

func (c *RealServerClient) Delete(id string) error {
	return delete(c.genUrl(id), c.token)
}

func (c *RealServerClient) get(id string) (*types.RealServer, error) {
	rss := &types.RealServerList{}
	if err := get(c.genUrl(id), c.token, rss); err != nil {
		return nil, err
	}
	if len(rss.RSTable) == 0 {
		return nil, ResourceNotFoundError
	}
	return &rss.RSTable[0], nil
}

func isRealServerEqual(rs1, rs2 *types.RealServer) bool {
	if rs1 == nil || rs2 == nil {
		return false
	}
	return rs1.IpAddr == rs2.IpAddr && rs1.State == rs2.State && rs1.Type == rs2.Type
}

func (c *RealServerClient) genUrl(id string) string {
	return fmt.Sprintf("%s%s%s%s", reqUrlPrefix, c.server, realServerPath, id)
}
