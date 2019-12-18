package client

import (
	"encoding/base64"
	"fmt"
)

const (
	reqUrlPrefix = "https://"

	applyActionPath       = "/config?action=apply"
	saveActionPath        = "/config?action=save"
	revertActionPath      = "/config?action=revert"
	revertApplyActionPath = "/config?action=revertapply"
)

var ResourceNotFoundError = fmt.Errorf("resource not found")

type Client struct {
	token          string
	server         string
	realServer     *RealServerClient
	realServerPort *RealServerPortClient
	serverGroup    *ServerGroupClient
	virtualServer  *VirtualServerClient
	virtualService *VirtualServiceClient
}

func New(user, password, serverAddr string) *Client {
	token := genBasicAuthToken(user, password)
	return &Client{
		token:          token,
		server:         serverAddr,
		realServer:     NewRealServerClient(token, serverAddr),
		realServerPort: NewRealServerPortClient(token, serverAddr),
		serverGroup:    NewServerGroupClient(token, serverAddr),
		virtualServer:  NewVirtualServerClient(token, serverAddr),
		virtualService: NewVirtualServiceClient(token, serverAddr),
	}
}

func genBasicAuthToken(user, password string) string {
	t := fmt.Sprintf("%s:%s", user, password)
	return base64.StdEncoding.EncodeToString([]byte(t))
}

func (c *Client) Rs() *RealServerClient {
	return c.realServer
}

func (c *Client) Rsp() *RealServerPortClient {
	return c.realServerPort
}

func (c *Client) Sg() *ServerGroupClient {
	return c.serverGroup
}

func (c *Client) VServer() *VirtualServerClient {
	return c.virtualServer
}

func (c *Client) VService() *VirtualServiceClient {
	return c.virtualService
}

func (c *Client) ApplyAndSave() error {
	if err := c.Apply(); err != nil {
		return err
	}
	return c.Save()
}

func (c *Client) Apply() error {
	url := fmt.Sprintf("%s%s%s", reqUrlPrefix, c.server, applyActionPath)
	return actionWithRetry(url, c.token)
}

func (c *Client) Save() error {
	url := fmt.Sprintf("%s%s%s", reqUrlPrefix, c.server, saveActionPath)
	return actionWithRetry(url, c.token)
}

func (c *Client) RollBack() error {
	if err := c.revert(); err != nil {
		return err
	}
	return c.revertApply()
}

func (c *Client) revert() error {
	url := fmt.Sprintf("%s%s%s", reqUrlPrefix, c.server, revertActionPath)
	return actionWithRetry(url, c.token)
}

func (c *Client) revertApply() error {
	url := fmt.Sprintf("%s%s%s", reqUrlPrefix, c.server, revertApplyActionPath)
	return actionWithRetry(url, c.token)
}
