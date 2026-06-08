package client

import (
	"context"
	"io"
	"net/http"
	"testing"
)

func TestGetDeployStatusUsesQueryParam(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/deploy/status" {
			t.Errorf("path = %q, want %q (ids must be a query param, not a path segment)", r.URL.Path, "/deploy/status")
		}
		if got := r.URL.Query().Get("ids"); got != "30625,30626" {
			t.Errorf("ids = %q, want %q", got, "30625,30626")
		}
		// The PHP API string-encodes some numeric ids (order_number, srv_id); flexInt64 must absorb that.
		_, _ = io.WriteString(w, `{"meta": {"status": 200}, "data": {"servers": [{"order_id": 30625, "order_number": "12502", "srv_id": "17393", "status": "ready"}], "all_ready": true}}`)
	})

	status, err := c.GetDeployStatus(context.Background(), []int64{30625, 30626})
	if err != nil {
		t.Fatalf("GetDeployStatus: %v", err)
	}
	if !status.AllReady {
		t.Error("AllReady = false, want true")
	}
	if len(status.Servers) != 1 || status.Servers[0].SrvID != 17393 {
		t.Errorf("unexpected servers: %+v", status.Servers)
	}
}

func TestGetDeployCatalogParsesNumbers(t *testing.T) {
	// The catalog encodes numeric fields as integers/floats (verified against the live API).
	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"meta":{"status":200},"data":{"tiers":[{"group_id":57,"products":[{"product_id":7955,"price_id":4054,"product_name":"KVM Value VPS 4GB","rate_hourly":0.12618,"rate_monthly":99,"region_ids":[1]}]}],"regions":[{"region_id":"1","region_name":"Sandefjord","region_active":true}],"currency":"NOK"}}`)
	})

	catalog, err := c.GetDeployCatalog(context.Background())
	if err != nil {
		t.Fatalf("GetDeployCatalog: %v", err)
	}
	if len(catalog.Tiers) != 1 || len(catalog.Tiers[0].Products) != 1 {
		t.Fatalf("unexpected catalog: %+v", catalog)
	}
	p := catalog.Tiers[0].Products[0]
	if p.ProductID != 7955 || p.PriceID != 4054 || p.RateMonthly != 99 || p.RateHourly != 0.12618 {
		t.Errorf("unexpected product: %+v", p)
	}
	if len(p.RegionIDs) != 1 || p.RegionIDs[0] != 1 {
		t.Errorf("region_ids = %v, want [1]", p.RegionIDs)
	}
}

func TestDeployParsesNumbers(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"meta":{"status":200},"data":{"order_ids":[30743],"order_numbers":[12601],"rate_hourly":0.12618,"monthly_cap":99,"currency":"NOK"}}`)
	})

	result, err := c.Deploy(context.Background(), DeployInput{ProductID: 7955, PriceID: 4054, RegionID: 1, Rescue: true})
	if err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	if len(result.OrderIDs) != 1 || result.OrderIDs[0] != 30743 {
		t.Errorf("order_ids = %v, want [30743]", result.OrderIDs)
	}
	if len(result.OrderNumbers) != 1 || result.OrderNumbers[0] != 12601 || result.MonthlyCap != 99 || result.Currency != "NOK" {
		t.Errorf("unexpected result: %+v", result)
	}
}
