package client

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"strings"
)

type flexBool bool

func (b *flexBool) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), `"`)
	*b = flexBool(s == "1" || s == "true")
	return nil
}

type DnsZone struct {
	ZoneID        string   `json:"zone_id"`
	ZoneName      string   `json:"zone_name"`
	ZoneType      string   `json:"zone_type"`
	ZoneActive    flexBool `json:"zone_active"`
	ZoneProtected flexBool `json:"zone_protected"`
	ExternalDNS   flexBool `json:"external_dns"`
}

type createZoneRequest struct {
	ZoneName string `json:"zone_name"`
	ZoneType string `json:"zone_type"`
}

func (c *Client) ListDnsZones(ctx context.Context) ([]DnsZone, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "dns/zones", nil)
	if err != nil {
		return nil, err
	}

	var zones []DnsZone
	if err := c.sendRequest(req, &zones); err != nil {
		return nil, err
	}
	return zones, nil
}

func (c *Client) GetZone(ctx context.Context, id string) (*DnsZone, error) {
	zones, err := c.ListDnsZones(ctx)
	if err != nil {
		return nil, err
	}
	for i := range zones {
		if zones[i].ZoneID == id {
			return &zones[i], nil
		}
	}
	return nil, nil
}

func (c *Client) CreateZone(ctx context.Context, zoneName, zoneType string) (*DnsZone, error) {
	req, err := c.newRequest(ctx, http.MethodPost, "dns/zones", createZoneRequest{ZoneName: zoneName, ZoneType: zoneType})
	if err != nil {
		return nil, err
	}
	if err := c.sendRequest(req, nil); err != nil {
		return nil, err
	}

	zones, err := c.ListDnsZones(ctx)
	if err != nil {
		return nil, err
	}
	for i := range zones {
		if strings.EqualFold(zones[i].ZoneName, zoneName) {
			return &zones[i], nil
		}
	}
	return nil, fmt.Errorf("zone %q was created but could not be found on the account", zoneName)
}

func (c *Client) DeleteZone(ctx context.Context, id string) error {
	req, err := c.newRequest(ctx, http.MethodDelete, path.Join("dns", "zones", id), nil)
	if err != nil {
		return err
	}
	return c.sendRequest(req, nil)
}
