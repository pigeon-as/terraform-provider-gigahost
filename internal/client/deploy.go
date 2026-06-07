package client

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type DeployCatalog struct {
	Tiers    []DeployTier   `json:"tiers"`
	Regions  []DeployRegion `json:"regions"`
	Currency string         `json:"currency"`
}

type DeployTier struct {
	GroupID   int64           `json:"group_id"`
	GroupName string          `json:"group_name"`
	Products  []DeployProduct `json:"products"`
}

type DeployProduct struct {
	ProductID   int64   `json:"product_id"`
	ProductHash string  `json:"product_hash"`
	ProductName string  `json:"product_name"`
	VMCores     string  `json:"vm_cores"`
	VMMemory    string  `json:"vm_memory"`
	VMStorage   string  `json:"vm_storage"`
	VMBw        string  `json:"vm_bw"`
	VMBwType    string  `json:"vm_bw_type"`
	PriceID     int64   `json:"price_id"`
	RateHourly  float64 `json:"rate_hourly"`
	RateMonthly int64   `json:"rate_monthly"`
	RegionIDs   []int64 `json:"region_ids"`
}

type DeployRegion struct {
	RegionID        string   `json:"region_id"`
	RegionName      string   `json:"region_name"`
	RegionNameShort string   `json:"region_name_short"`
	RegionCountry   string   `json:"region_country"`
	RegionActive    flexBool `json:"region_active"`
}

func (c *Client) GetDeployCatalog(ctx context.Context) (*DeployCatalog, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "deploy/servers", nil, nil)
	if err != nil {
		return nil, err
	}

	var catalog DeployCatalog
	if err := c.sendRequest(req, &catalog); err != nil {
		return nil, err
	}
	return &catalog, nil
}

type DeployInput struct {
	ProductID int64
	PriceID   int64
	RegionID  int64
	OSID      *int64
	IsoID     *int64
	Rescue    bool
	Hostname  string
	SSHKeys   []int64
	Backups   bool
}

type deployRequest struct {
	Pid       int64    `json:"pid"`
	PriceID   int64    `json:"price_id"`
	RegionID  int64    `json:"region_id"`
	Quantity  int64    `json:"quantity"`
	OSID      *int64   `json:"os_id,omitempty"`
	IsoID     *int64   `json:"iso_id,omitempty"`
	Rescue    *int64   `json:"rescue,omitempty"`
	Hostnames []string `json:"hostnames,omitempty"`
	SSHKeys   []int64  `json:"ssh_keys,omitempty"`
	Backups   *int64   `json:"backups,omitempty"`
}

type DeployResult struct {
	OrderIDs     []int64 `json:"order_ids"`
	OrderNumbers []int64 `json:"order_numbers"`
	RateHourly   float64 `json:"rate_hourly"`
	MonthlyCap   int64   `json:"monthly_cap"`
	Currency     string  `json:"currency"`
}

func (c *Client) Deploy(ctx context.Context, in DeployInput) (*DeployResult, error) {
	body := deployRequest{
		Pid:      in.ProductID,
		PriceID:  in.PriceID,
		RegionID: in.RegionID,
		Quantity: 1,
		OSID:     in.OSID,
		IsoID:    in.IsoID,
		SSHKeys:  in.SSHKeys,
	}
	if in.Rescue {
		v := int64(1)
		body.Rescue = &v
	}
	if in.Backups {
		v := int64(1)
		body.Backups = &v
	}
	if in.Hostname != "" {
		body.Hostnames = []string{in.Hostname}
	}

	req, err := c.newRequest(ctx, http.MethodPost, "deploy/servers", nil, body)
	if err != nil {
		return nil, err
	}

	var result DeployResult
	if err := c.sendRequest(req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

type DeployStatusServer struct {
	OrderID     flexInt64 `json:"order_id"`
	OrderNumber flexInt64 `json:"order_number"`
	Hostname    string    `json:"hostname"`
	SrvID       flexInt64 `json:"srv_id"`
	IP          string    `json:"ip"`
	IPv6        string    `json:"ipv6"`
	Status      string    `json:"status"`
	Password    string    `json:"password"`
}

type DeployStatus struct {
	Servers  []DeployStatusServer `json:"servers"`
	AllReady flexBool             `json:"all_ready"`
}

func (c *Client) GetDeployStatus(ctx context.Context, orderIDs []int64) (*DeployStatus, error) {
	parts := make([]string, len(orderIDs))
	for i, id := range orderIDs {
		parts[i] = strconv.FormatInt(id, 10)
	}

	query := url.Values{"ids": {strings.Join(parts, ",")}}
	req, err := c.newRequest(ctx, http.MethodGet, "deploy/status", query, nil)
	if err != nil {
		return nil, err
	}

	var status DeployStatus
	if err := c.sendRequest(req, &status); err != nil {
		return nil, err
	}
	return &status, nil
}
