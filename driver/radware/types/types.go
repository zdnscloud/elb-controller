package types

import (
	"encoding/json"
)

type RealServer struct {
	// IpAddr:realserver ip
	IpAddr string `json:"IpAddr"`
	// State:keep 2(enable)
	State int `json:"State"`
	// Type:keep 1(local)
	Type int `json:"Type"`
}

func (rs *RealServer) ToJson() string {
	b, _ := json.Marshal(rs)
	return string(b)
}

type RealServerList struct {
	RSTable []RealServer `json:"SlbNewCfgEnhRealServerTable"`
}

type RealServerPort struct {
	// RealPort:realserver service port
	RealPort int32 `json:"RealPort"`
}

func (rsp *RealServerPort) ToJson() string {
	b, _ := json.Marshal(rsp)
	return string(b)
}

type RealServerPortList struct {
	RSPortTable []RealServerPort `json:"SlbNewCfgEnhRealServPortTable"`
}

type ServerGroup struct {
	Metric       int    `json:"Metric,omitempty"`
	HealthID     string `json:"HealthID,omitempty"`
	AddServer    string `json:"AddServer,omitempty"`
	RemoveServer string `json:"RemoveServer,omitempty"`
}

func (sg *ServerGroup) ToJson() string {
	b, _ := json.Marshal(sg)
	return string(b)
}

type ServerGroupList struct {
	SGTable []ServerGroup `json:"SlbNewCfgEnhGroupTable"`
}

type GroupServer struct {
	RealServerGroupIndex string `json:"RealServerGroupIndex"`
	Index                string `json:"Index"`
}

func (g *GroupServer) ToJson() string {
	b, _ := json.Marshal(g)
	return string(b)
}

type GroupServerList struct {
	GSTable []GroupServer `json:"SlbEnhGroupRealServersTable"`
}

type VirtualServer struct {
	VirtServerIpAddress string `json:"VirtServerIpAddress"`
	VirtServerState     int    `json:"VirtServerState"`
}

func (v *VirtualServer) ToJson() string {
	b, _ := json.Marshal(v)
	return string(b)
}

type VirtualServerList struct {
	VSTable []VirtualServer `json:"SlbNewCfgEnhVirtServerTable"`
}

type VirtualService struct {
	UDPBalance int32 `json:"UDPBalance"`
	VirtPort   int32 `json:"VirtPort"`
	RealPort   int32 `json:"RealPort"`
	DBind      int   `json:"DBind"`
	PBind      int   `json:"PBind"`
}

func (v *VirtualService) ToJson() string {
	b, _ := json.Marshal(v)
	return string(b)
}

type VirtualServiceList struct {
	VSTable []VirtualService `json:"SlbNewCfgEnhVirtServicesTable"`
}

type VirtualServiceRealGroup struct {
	RealGroup         string `json:"RealGroup"`
	PersistentTimeOut int    `json:"PersistentTimeOut"`
	ProxyIpMode       int    `json:"ProxyIpMode"`
}

func (v *VirtualServiceRealGroup) ToJson() string {
	b, _ := json.Marshal(v)
	return string(b)
}

type VirtualServiceRealGroupList struct {
	VSTable []VirtualServiceRealGroup `json:"SlbNewCfgEnhVirtServicesSeventhPartTable"`
}

func NewAddServerServerGroup(id string) *ServerGroup {
	return &ServerGroup{
		AddServer: id,
	}
}

func NewRemoveServerServerGroup(id string) *ServerGroup {
	return &ServerGroup{
		RemoveServer: id,
	}
}

func NewVirtualServiceRealGroup(id string) *VirtualServiceRealGroup {
	return &VirtualServiceRealGroup{
		RealGroup:         id,
		PersistentTimeOut: 10,
		ProxyIpMode:       1,
	}
}
