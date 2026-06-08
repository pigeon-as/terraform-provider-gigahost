package client

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
)

type flexBool bool

func (b *flexBool) UnmarshalJSON(data []byte) error {
	switch s := strings.Trim(string(data), `"`); s {
	case "1", "true":
		*b = true
	case "0", "false", "", "null":
		*b = false
	default:
		return fmt.Errorf("cannot parse %q as a boolean", s)
	}
	return nil
}

type flexInt64 int64

func (n *flexInt64) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), `"`)
	if s == "" || s == "null" {
		*n = 0
		return nil
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return err
	}
	*n = flexInt64(v)
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
	req, err := c.newRequest(ctx, http.MethodGet, "dns/zones", nil, nil)
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
	return nil, ErrNotFound
}

func (c *Client) CreateZone(ctx context.Context, zoneName, zoneType string) (*DnsZone, error) {
	req, err := c.newRequest(ctx, http.MethodPost, "dns/zones", nil, createZoneRequest{ZoneName: zoneName, ZoneType: zoneType})
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
	req, err := c.newRequest(ctx, http.MethodDelete, path.Join("dns", "zones", id), nil, nil)
	if err != nil {
		return err
	}
	return c.sendRequest(req, nil)
}

type DnsRecord struct {
	RecordID       string     `json:"record_id"`
	RecordName     string     `json:"record_name"`
	RecordType     string     `json:"record_type"`
	RecordValue    string     `json:"record_value"`
	RecordTTL      flexInt64  `json:"record_ttl"`
	RecordPriority *flexInt64 `json:"record_priority"`
}

type RecordRequest struct {
	RecordName     string `json:"record_name"`
	RecordType     string `json:"record_type"`
	RecordValue    string `json:"record_value"`
	RecordTTL      int64  `json:"record_ttl"`
	RecordPriority *int64 `json:"record_priority,omitempty"`
}

func (c *Client) ListRecords(ctx context.Context, zoneID string) ([]DnsRecord, error) {
	req, err := c.newRequest(ctx, http.MethodGet, path.Join("dns", "zones", zoneID, "records"), nil, nil)
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
	return nil, ErrNotFound
}

func (c *Client) CreateRecord(ctx context.Context, zoneID string, body RecordRequest) (*DnsRecord, error) {
	req, err := c.newRequest(ctx, http.MethodPost, path.Join("dns", "zones", zoneID, "records"), nil, body)
	if err != nil {
		return nil, err
	}
	if err := c.sendRequest(req, nil); err != nil {
		return nil, err
	}
	return c.resolveRecord(ctx, zoneID, body)
}

func (c *Client) UpdateRecord(ctx context.Context, zoneID, recordID string, body RecordRequest) (*DnsRecord, error) {
	req, err := c.newRequest(ctx, http.MethodPut, path.Join("dns", "zones", zoneID, "records", recordID), nil, body)
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

	req, err := c.newRequest(ctx, http.MethodDelete, path.Join("dns", "zones", zoneID, "records", recordID), query, nil)
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
	want := NormalizeRecordValue(body.RecordType, body.RecordValue)
	for i := range records {
		if records[i].RecordName == body.RecordName &&
			strings.EqualFold(records[i].RecordType, body.RecordType) &&
			records[i].RecordValue == want {
			return &records[i], nil
		}
	}
	return nil, fmt.Errorf("record %s %q was written but could not be found in zone %s", body.RecordType, body.RecordName, zoneID)
}

func NormalizeRecordValue(recordType, value string) string {
	if strings.EqualFold(recordType, "AAAA") {
		if ip := net.ParseIP(value); ip != nil {
			return ip.String()
		}
	}
	return value
}

type DnsRedirect struct {
	Source    string   `json:"source"`
	TargetURL string   `json:"target_url"`
	Enabled   flexBool `json:"enabled"`
}

type redirectRequest struct {
	Source    string `json:"source"`
	TargetURL string `json:"target_url"`
}

func (c *Client) ListRedirects(ctx context.Context, zoneID string) ([]DnsRedirect, error) {
	req, err := c.newRequest(ctx, http.MethodGet, path.Join("dns", "zones", zoneID, "redirect"), nil, nil)
	if err != nil {
		return nil, err
	}

	var redirects []DnsRedirect
	if err := c.sendRequest(req, &redirects); err != nil {
		return nil, err
	}
	return redirects, nil
}

func (c *Client) GetRedirect(ctx context.Context, zoneID, source string) (*DnsRedirect, error) {
	redirects, err := c.ListRedirects(ctx, zoneID)
	if err != nil {
		return nil, err
	}
	for i := range redirects {
		if redirects[i].Source == source {
			return &redirects[i], nil
		}
	}
	return nil, ErrNotFound
}

func (c *Client) CreateRedirect(ctx context.Context, zoneID, source, targetURL string) (*DnsRedirect, error) {
	body := redirectRequest{Source: source, TargetURL: targetURL}
	req, err := c.newRequest(ctx, http.MethodPost, path.Join("dns", "zones", zoneID, "redirect"), nil, body)
	if err != nil {
		return nil, err
	}
	if err := c.sendRequest(req, nil); err != nil {
		return nil, err
	}
	return c.GetRedirect(ctx, zoneID, source)
}

func (c *Client) UpdateRedirect(ctx context.Context, zoneID, source, targetURL string) (*DnsRedirect, error) {
	body := redirectRequest{Source: source, TargetURL: targetURL}
	req, err := c.newRequest(ctx, http.MethodPut, path.Join("dns", "zones", zoneID, "redirect"), nil, body)
	if err != nil {
		return nil, err
	}
	if err := c.sendRequest(req, nil); err != nil {
		return nil, err
	}
	return c.GetRedirect(ctx, zoneID, source)
}

func (c *Client) DeleteRedirect(ctx context.Context, zoneID, source string) error {
	query := url.Values{}
	query.Set("source", source)

	req, err := c.newRequest(ctx, http.MethodDelete, path.Join("dns", "zones", zoneID, "redirect"), query, nil)
	if err != nil {
		return err
	}
	return c.sendRequest(req, nil)
}
