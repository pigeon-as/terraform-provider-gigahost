package client

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"strconv"
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
	OSID              string           `json:"os_id"`
	OS                ServerOS         `json:"os"`
	IPs               []ServerIP       `json:"ips"`
	SrvFeatureBackups flexBool         `json:"srv_feature_backups"`
	Order             ServerOrder      `json:"order"`
	Datacenter        ServerDatacenter `json:"datacenter"`
}

type ServerOS struct {
	OSID      flexInt64 `json:"os_id"`
	OSName    string    `json:"os_name"`
	OSRelease string    `json:"os_release"`
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

type ServerHdd struct {
	HddID           flexInt64 `json:"hdd_id"`
	HddType         string    `json:"hdd_type"`
	HddSize         flexInt64 `json:"hdd_size"`
	HddManufacturer string    `json:"hdd_manufacturer"`
	HddModel        string    `json:"hdd_model"`
}

type ServerDetail struct {
	Server
	SrvDateCreated string      `json:"srv_date_created"`
	SrvBw          flexInt64   `json:"srv_bw"`
	SrvBwType      string      `json:"srv_bw_type"`
	Hdds           []ServerHdd `json:"hdds"`
}

func (c *Client) GetServer(ctx context.Context, id string) (*ServerDetail, error) {
	// An empty or non-numeric id would change the request path.
	if _, err := strconv.ParseInt(id, 10, 64); err != nil {
		return nil, fmt.Errorf("gigahost: invalid server id %q", id)
	}

	req, err := c.newRequest(ctx, http.MethodGet, path.Join("servers", id), nil, nil)
	if err != nil {
		return nil, err
	}

	var servers []ServerDetail
	if err := c.sendRequest(req, &servers); err != nil {
		return nil, err
	}
	if len(servers) != 1 {
		return nil, fmt.Errorf("gigahost: server %s: expected one server in the response, got %d", id, len(servers))
	}
	return &servers[0], nil
}
