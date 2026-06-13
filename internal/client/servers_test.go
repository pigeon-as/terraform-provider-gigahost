package client

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

func TestGetServer(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/servers/17617" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(`{"meta":{},"data":[{` +
			`"srv_id":"17617","srv_name":"srv17617.gigahost.no","srv_status":true,` +
			`"srv_status_install":true,"srv_cores":"2","srv_ram":4,` +
			`"srv_date_created":"1781214032","srv_bw":"20000","srv_bw_type":"quota",` +
			`"bw_used":0,"bw_used_in":0,"bw_used_out":0,` +
			`"hdds":[{"hdd_id":"22731","hdd_type":"SSD","hdd_size":"40","hdd_space_used":"0","hdd_manufacturer":"Proxmox","hdd_model":""}],` +
			`"order":{"order_id":"31053","order_status":"active"}}]}`))
	})

	s, err := c.GetServer(context.Background(), "17617")
	if err != nil {
		t.Fatalf("GetServer: %v", err)
	}
	if s.SrvID != "17617" || !bool(s.SrvStatus) || !bool(s.SrvStatusInstall) {
		t.Fatalf("unexpected base fields: %+v", s.Server)
	}
	if int64(s.SrvCores) != 2 || int64(s.SrvRAM) != 4 {
		t.Fatalf("cores/ram = %d/%d, want 2/4 across mixed JSON types", s.SrvCores, s.SrvRAM)
	}
	if s.SrvDateCreated != "1781214032" || int64(s.SrvBw) != 20000 || s.SrvBwType != "quota" {
		t.Fatalf("detail fields = %q/%d/%q", s.SrvDateCreated, s.SrvBw, s.SrvBwType)
	}
	if len(s.Hdds) != 1 || int64(s.Hdds[0].HddSize) != 40 || s.Hdds[0].HddType != "SSD" {
		t.Fatalf("hdds = %+v", s.Hdds)
	}
	if s.Order.OrderStatus != "active" {
		t.Fatalf("order status = %q", s.Order.OrderStatus)
	}
}

func TestGetServerInvalidID(t *testing.T) {
	calls := 0
	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.WriteHeader(http.StatusOK)
	})

	for _, id := range []string{"", "../account", "abc"} {
		if _, err := c.GetServer(context.Background(), id); err == nil {
			t.Fatalf("id %q: expected an error", id)
		}
	}
	if calls != 0 {
		t.Fatalf("calls = %d, want no requests for invalid ids", calls)
	}
}

func TestGetServerMultipleResults(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"meta":{},"data":[{"srv_id":"1"},{"srv_id":"2"}]}`))
	})

	if _, err := c.GetServer(context.Background(), "1"); err == nil {
		t.Fatal("expected an error for a multi-element response")
	}
}

func TestGetServerNotFound(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"meta":{"status":404,"message":"404 Not Found"},"data":[]}`))
	})

	_, err := c.GetServer(context.Background(), "99999999")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestGetServerForbidden(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"meta":{"status":403,"message":"You do not have permission for this operation."},"data":[]}`))
	})

	var apiErr *Error
	_, err := c.GetServer(context.Background(), "17617")
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusForbidden {
		t.Fatalf("err = %v, want a 403 *Error", err)
	}
}

func TestGetServerEmptyResponse(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"meta":{},"data":[]}`))
	})

	if _, err := c.GetServer(context.Background(), "17617"); err == nil {
		t.Fatal("expected an error for an empty data array")
	}
}
