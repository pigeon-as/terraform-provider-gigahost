package client

import (
	"context"
	"net/http"
)

type Server struct {
	SrvID             string           `json:"srv_id"`
	SrvName           string           `json:"srv_name"`
	SrvHostname       string           `json:"srv_hostname"`
	SrvStatus         flexBool         `json:"srv_status"`
	SrvStatusRescue   flexBool         `json:"srv_status_rescue"`
	SrvStatusInstall  flexBool         `json:"srv_status_install"`
	SrvSuspended      flexBool         `json:"srv_suspended"`
	SrvLocation       string           `json:"srv_location"`
	SrvType           string           `json:"srv_type"`
	SrvVpsType        string           `json:"srv_vps_type"`
	SrvPrimaryIP      string           `json:"srv_primary_ip"`
	SrvCores          flexInt64        `json:"srv_cores"`
	SrvRAM            flexInt64        `json:"srv_ram"`
	ProductID         string           `json:"product_id"`
	OsID              string           `json:"os_id"`
	OS                ServerOS         `json:"os"`
	IPs               []ServerIP       `json:"ips"`
	SrvFeatureBackups flexBool         `json:"srv_feature_backups"`
	Order             ServerOrder      `json:"order"`
	Datacenter        ServerDatacenter `json:"datacenter"`
}

type ServerOS struct {
	OsID      flexInt64 `json:"os_id"`
	OsName    string    `json:"os_name"`
	OsRelease string    `json:"os_release"`
}

type ServerIP struct {
	IPID      flexInt64 `json:"ip_id"`
	IPAddress string    `json:"ip_address"`
	IPv4v6    string    `json:"ip_v4v6"`
	IPReverse string    `json:"ip_reverse"`
	IPType    string    `json:"ip_type"`
	IPNetmask string    `json:"ip_netmask"`
	IPGateway string    `json:"ip_gateway"`
}

type ServerOrder struct {
	OrderID     string `json:"order_id"`
	OrderNumber string `json:"order_number"`
	OrderStatus string `json:"order_status"`
	ProductID   string `json:"product_id"`
	ProductName string `json:"product_name"`
}

type ServerDatacenter struct {
	RegionID   string `json:"region_id"`
	RegionName string `json:"region_name"`
}

func (c *Client) ListServers(ctx context.Context) ([]Server, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "servers", nil, nil)
	if err != nil {
		return nil, err
	}

	var servers []Server
	if err := c.sendRequest(req, &servers); err != nil {
		return nil, err
	}
	return servers, nil
}
