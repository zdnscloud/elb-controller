package types

type RealServer struct {
	// IpAddr:realserver ip
	IpAddr string `json:"IpAddr"`
	// State:keep 2(enable)
	State int `json:"State"`
	// Type:keep 1(local)
	Type int `json:"Type"`
}

type RealServerList struct {
	RSTable []RealServer `json:"SlbNewCfgEnhRealServerTable"`
}

type RealServerPort struct {
	// RealPort:realserver service port
	RealPort int32 `json:"RealPort"`
}

type RealServerPortList struct {
	RSPortTable []RealServerPort `json:"SlbNewCfgEnhRealServPortTable"`
}

type ServerGroup struct {
	Metric           int    `json:"Metric,omitempty"`
	HealthCheckLayer int    `json:"HealthCheckLayer,omitempty"`
	AddServer        string `json:"AddServer,omitempty"`
	RemoveServer     string `json:"RemoveServer,omitempty"`
}

type ServerGroupList struct {
	SGTable []ServerGroup `json:"SlbNewCfgEnhGroupTable"`
}

type GroupServer struct {
	RealServerGroupIndex string `json:"RealServerGroupIndex"`
	Index                string `json:"Index"`
}

type GroupServerList struct {
	GSTable []GroupServer `json:"SlbEnhGroupRealServersTable"`
}

type VirtualServer struct {
	VirtServerIpAddress string `json:"VirtServerIpAddress"`
	VirtServerState     int    `json:"VirtServerState"`
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

type VirtualServiceList struct {
	VSTable []VirtualService `json:"SlbNewCfgEnhVirtServicesTable"`
}

type VirtualServicePart2 struct {
	XForwardedFor int `json:"XForwardedFor"`
}

type VirtualServicePart2List struct {
	VSTable []VirtualServicePart2 `json:"SlbNewCfgEnhVirtServicesSixthPartTable"`
}

type VirtualServicePart3 struct {
	RealGroup         string `json:"RealGroup"`
	PersistentTimeOut int    `json:"PersistentTimeOut"`
	ProxyIpMode       int    `json:"ProxyIpMode"`
}

type VirtualServicePart3List struct {
	VSTable []VirtualServicePart3 `json:"SlbNewCfgEnhVirtServicesSeventhPartTable"`
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
