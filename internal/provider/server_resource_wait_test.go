package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pigeon-as/terraform-provider-gigahost/internal/client"
)

func newWaitTestResource(t *testing.T, handler http.HandlerFunc) *serverResource {
	t.Helper()

	oldPoll := serverDeployPollInterval
	serverDeployPollInterval = time.Millisecond
	t.Cleanup(func() { serverDeployPollInterval = oldPoll })

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	c, err := client.NewClient(&client.Config{
		Address:    srv.URL,
		Token:      "test-token",
		HTTPClient: srv.Client(),
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return &serverResource{client: c}
}

func deployStatusHandler(statusBody string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/deploy/status") {
			_, _ = w.Write([]byte(`{"meta":{},"data":` + statusBody + `}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}
}

func TestWaitForServerReady(t *testing.T) {
	r := newWaitTestResource(t, deployStatusHandler(
		`{"servers":[{"order_id":"777","srv_id":"12345","ip":"185.199.2.3","status":"ready","password":"hunter2"}],"all_ready":"1"}`,
	))

	server, err := r.waitForServer(context.Background(), 777)
	if err != nil {
		t.Fatalf("waitForServer: %v", err)
	}
	if server == nil || server.SrvID != "12345" || server.Password != "hunter2" {
		t.Fatalf("server = %+v, want srv_id 12345 with password", server)
	}
}

func TestWaitForServerTerminalStatusReturnsServer(t *testing.T) {
	r := newWaitTestResource(t, deployStatusHandler(
		`{"servers":[{"order_id":"777","srv_id":"12345","status":"failed"}],"all_ready":"0"}`,
	))

	server, err := r.waitForServer(context.Background(), 777)
	if err == nil || !strings.Contains(err.Error(), `status "failed"`) {
		t.Fatalf("err = %v, want terminal status error", err)
	}
	if server == nil || server.SrvID != "12345" {
		t.Fatalf("server = %+v, want the failed server returned alongside the error", server)
	}
}

func TestWaitForServerTimeoutWithoutServer(t *testing.T) {
	r := newWaitTestResource(t, deployStatusHandler(`{"servers":[],"all_ready":"0"}`))

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	server, err := r.waitForServer(ctx, 777)
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("err = %v, want timeout error", err)
	}
	if server != nil {
		t.Fatalf("server = %+v, want nil when the order was never seen", server)
	}
}

func TestWaitForServerKeepsPasswordAcrossSightings(t *testing.T) {
	var mu sync.Mutex
	calls := 0
	r := newWaitTestResource(t, func(w http.ResponseWriter, req *http.Request) {
		if !strings.HasPrefix(req.URL.Path, "/deploy/status") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		mu.Lock()
		calls++
		n := calls
		mu.Unlock()
		if n == 1 {
			_, _ = w.Write([]byte(`{"meta":{},"data":{"servers":[{"order_id":"777","srv_id":"12345","status":"installing","password":"hunter2"}],"all_ready":"0"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"meta":{},"data":{"servers":[{"order_id":"777","srv_id":"12345","ip":"185.199.2.3","status":"ready","password":""}],"all_ready":"1"}}`))
	})

	server, err := r.waitForServer(context.Background(), 777)
	if err != nil {
		t.Fatalf("waitForServer: %v", err)
	}
	if server == nil || server.Password != "hunter2" {
		t.Fatalf("server = %+v, want the install-time password kept at ready", server)
	}
}

func TestWaitForServerReadyWithoutIDKeepsWaiting(t *testing.T) {
	r := newWaitTestResource(t, deployStatusHandler(
		`{"servers":[{"order_id":"777","srv_id":0,"ip":"185.199.2.3","status":"ready","password":"secret"}],"all_ready":"1"}`,
	))

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	server, err := r.waitForServer(ctx, 777)
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("err = %v, want a timeout: a ready status without an id is not usable", err)
	}
	if server == nil || server.SrvID != "" || server.IP != "185.199.2.3" {
		t.Fatalf("server = %+v, want the partial sighting preserved with an empty id", server)
	}
}
