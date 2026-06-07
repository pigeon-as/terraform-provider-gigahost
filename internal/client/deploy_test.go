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
