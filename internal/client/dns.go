package client

import (
	"context"
	"net/http"
)

type DnsZone struct {
	ZoneID           string `json:"zone_id"`
	ZoneName         string `json:"zone_name"`
	ZoneType         string `json:"zone_type"`
	ZoneActive       string `json:"zone_active"`
	ZoneProtected    string `json:"zone_protected"`
	ZoneIsRegistered string `json:"zone_is_registered"`
	DomainRegistrar  string `json:"domain_registrar"`
	DomainStatus     string `json:"domain_status"`
	DomainExpiryDate string `json:"domain_expiry_date"`
	DomainAutoRenew  string `json:"domain_auto_renew"`
	ExternalDNS      string `json:"external_dns"`
	RecordCount      string `json:"record_count"`
	ZoneUpdated      string `json:"zone_updated"`
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
