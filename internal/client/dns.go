package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
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

type DnsRecord struct {
	RecordID       string `json:"record_id"`
	RecordName     string `json:"record_name"`
	RecordType     string `json:"record_type"`
	RecordValue    string `json:"record_value"`
	RecordTTL      int64  `json:"record_ttl"`
	RecordPriority *int64 `json:"record_priority"`
}

type RecordRequest struct {
	RecordName     string `json:"record_name"`
	RecordType     string `json:"record_type"`
	RecordValue    string `json:"record_value"`
	RecordTTL      int64  `json:"record_ttl"`
	RecordPriority *int64 `json:"record_priority,omitempty"`
}

func (c *Client) ListRecords(ctx context.Context, zoneID string) ([]DnsRecord, error) {
	req, err := c.newRequest(ctx, http.MethodGet, path.Join("dns", "zones", zoneID, "records"), nil)
	if err != nil {
		return nil, err
	}

	var records []DnsRecord
	if err := c.sendRequest(req, &records); err != nil {
		return nil, err
	}
	return records, nil
}

func (c *Client) GetRecord(ctx context.Context, zoneID, recordID string) (*DnsRecord, error) {
	records, err := c.ListRecords(ctx, zoneID)
	if err != nil {
		return nil, err
	}
	for i := range records {
		if records[i].RecordID == recordID {
			return &records[i], nil
		}
	}
	return nil, nil
}

func (c *Client) CreateRecord(ctx context.Context, zoneID string, body RecordRequest) (*DnsRecord, error) {
	req, err := c.newRequest(ctx, http.MethodPost, path.Join("dns", "zones", zoneID, "records"), body)
	if err != nil {
		return nil, err
	}
	if err := c.sendRequest(req, nil); err != nil {
		return nil, err
	}
	return c.resolveRecord(ctx, zoneID, body)
}

func (c *Client) UpdateRecord(ctx context.Context, zoneID, recordID string, body RecordRequest) (*DnsRecord, error) {
	req, err := c.newRequest(ctx, http.MethodPut, path.Join("dns", "zones", zoneID, "records", recordID), body)
	if err != nil {
		return nil, err
	}
	if err := c.sendRequest(req, nil); err != nil {
		return nil, err
	}
	return c.resolveRecord(ctx, zoneID, body)
}

func (c *Client) DeleteRecord(ctx context.Context, zoneID, recordID, name, recordType, value string) error {
	query := url.Values{}
	query.Set("name", name)
	query.Set("type", recordType)
	query.Set("value", value)
	endpoint := path.Join("dns", "zones", zoneID, "records", recordID) + "?" + query.Encode()

	req, err := c.newRequest(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return err
	}
	return c.sendRequest(req, nil)
}

func (c *Client) resolveRecord(ctx context.Context, zoneID string, body RecordRequest) (*DnsRecord, error) {
	records, err := c.ListRecords(ctx, zoneID)
	if err != nil {
		return nil, err
	}
	for i := range records {
		if records[i].RecordName == body.RecordName &&
			strings.EqualFold(records[i].RecordType, body.RecordType) &&
			records[i].RecordValue == body.RecordValue {
			return &records[i], nil
		}
	}
	return nil, fmt.Errorf("record %s %q was written but could not be found in zone %s", body.RecordType, body.RecordName, zoneID)
}
