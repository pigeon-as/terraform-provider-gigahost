package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path"
	"strconv"
	"strings"
)

// ErrDNSSECNotEnabled is returned by GetDSRecords when DNSSEC is not enabled
// for the zone.
var ErrDNSSECNotEnabled = errors.New("gigahost: DNSSEC is not enabled for this domain")

type updateNameserversRequest struct {
	Nameservers []string `json:"nameservers"`
}

type updateDNSSECRequest struct {
	Enable int `json:"enable"`
}

// DSRecord is an external DS record submitted to the registry.
type DSRecord struct {
	KeyTag     int64  `json:"keyTag"`
	Algorithm  int64  `json:"alg"`
	DigestType int64  `json:"digestType"`
	Digest     string `json:"digest"`
	PublicKey  string `json:"publicKey,omitempty"`
	KeyType    *int64 `json:"keyType,omitempty"`
}

type submitExternalDSRecordsRequest struct {
	DSRecords []DSRecord `json:"ds_records"`
}

func validateZoneID(zoneID string) error {
	if _, err := strconv.ParseInt(zoneID, 10, 64); err != nil {
		return fmt.Errorf("gigahost: invalid zone id %q", zoneID)
	}
	return nil
}

// UpdateNameservers sets the registry nameservers of a registered domain.
func (c *Client) UpdateNameservers(ctx context.Context, zoneID string, nameservers []string) error {
	if err := validateZoneID(zoneID); err != nil {
		return err
	}
	if len(nameservers) < 2 {
		return errors.New("gigahost: at least two nameservers are required")
	}
	req, err := c.newRequest(ctx, http.MethodPut, path.Join("dns", "zones", zoneID, "nameservers"), nil, updateNameserversRequest{Nameservers: nameservers})
	if err != nil {
		return err
	}
	return c.sendRequest(req, nil)
}

// ListApexNameservers returns the values of the NS records at the zone apex.
func (c *Client) ListApexNameservers(ctx context.Context, zoneID string) ([]string, error) {
	records, err := c.ListRecords(ctx, zoneID)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, r := range records {
		if strings.EqualFold(r.RecordType, "NS") && r.RecordName == "@" {
			out = append(out, r.RecordValue)
		}
	}
	return out, nil
}

// SetDNSSEC enables or disables DNSSEC for a registered domain.
func (c *Client) SetDNSSEC(ctx context.Context, zoneID string, enable bool) error {
	if err := validateZoneID(zoneID); err != nil {
		return err
	}
	body := updateDNSSECRequest{}
	if enable {
		body.Enable = 1
	}
	req, err := c.newRequest(ctx, http.MethodPut, path.Join("dns", "zones", zoneID, "dnssec"), nil, body)
	if err != nil {
		return err
	}
	return c.sendRequest(req, nil)
}

// GetDSRecords returns the DS records text reported for the zone. It returns
// ErrDNSSECNotEnabled when DNSSEC is not enabled.
func (c *Client) GetDSRecords(ctx context.Context, zoneID string) (string, error) {
	if err := validateZoneID(zoneID); err != nil {
		return "", err
	}
	req, err := c.newRequest(ctx, http.MethodGet, path.Join("dns", "zones", zoneID, "ds-records"), nil, nil)
	if err != nil {
		return "", err
	}
	var data struct {
		DSRecords json.RawMessage `json:"ds_records"`
	}
	if err := c.sendRequest(req, &data); err != nil {
		var apiErr *Error
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusBadRequest && strings.Contains(strings.ToLower(apiErr.Message), "dnssec is not enabled") {
			return "", ErrDNSSECNotEnabled
		}
		return "", err
	}
	var s string
	if err := json.Unmarshal(data.DSRecords, &s); err == nil {
		return s, nil
	}
	return string(data.DSRecords), nil
}

// SubmitExternalDSRecords submits DS records for an externally hosted domain.
// The submitted set replaces all existing DS records.
func (c *Client) SubmitExternalDSRecords(ctx context.Context, zoneID string, records []DSRecord) error {
	if err := validateZoneID(zoneID); err != nil {
		return err
	}
	if len(records) == 0 {
		return errors.New("gigahost: at least one DS record is required")
	}
	req, err := c.newRequest(ctx, http.MethodPost, path.Join("dns", "zones", zoneID, "ds-records", "external"), nil, submitExternalDSRecordsRequest{DSRecords: records})
	if err != nil {
		return err
	}
	return c.sendRequest(req, nil)
}
