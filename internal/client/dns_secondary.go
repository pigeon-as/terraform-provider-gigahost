package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

type createSecondaryZoneRequest struct {
	ZoneName string   `json:"zone_name"`
	Masters  []string `json:"masters"`
}

// CreateSecondaryZone creates a secondary (slave) zone that is transferred
// from the given master servers. It uses POST /dns/zones/secondary, which is
// not part of the public API documentation but is used by the Gigahost panel.
func (c *Client) CreateSecondaryZone(ctx context.Context, zoneName string, masters []string) (*DnsZone, error) {
	if len(masters) == 0 {
		return nil, fmt.Errorf("gigahost: at least one master is required")
	}
	req, err := c.newRequest(ctx, http.MethodPost, "dns/zones/secondary", nil, createSecondaryZoneRequest{ZoneName: zoneName, Masters: masters})
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

// SplitMasters splits the zone_masters field of a zone into individual hosts.
func SplitMasters(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == ';' || r == ' ' || r == '\n' || r == '\t'
	})
}
