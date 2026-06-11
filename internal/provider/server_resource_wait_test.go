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

	oldPoll, oldConfirm := serverDeployPollInterval, serverListConfirmDelay
	serverDeployPollInterval = time.Millisecond
	serverListConfirmDelay = time.Millisecond
	t.Cleanup(func() { serverDeployPollInterval, serverListConfirmDelay = oldPoll, oldConfirm })

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

func waitTestHandler(statusBody, serversBody string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/deploy/status"):
			_, _ = w.Write([]byte(`{"meta":{},"data":` + statusBody + `}`))
		case strings.HasPrefix(r.URL.Path, "/servers"):
			_, _ = w.Write([]byte(`{"meta":{},"data":` + serversBody + `}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}
}

func TestWaitForServerReady(t *testing.T) {
	r := newWaitTestResource(t, waitTestHandler(
		`{"servers":[{"order_id":"777","srv_id":"12345","ip":"185.199.2.3","status":"ready","password":"hunter2"}],"all_ready":"1"}`,
		`[]`,
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
	r := newWaitTestResource(t, waitTestHandler(
		`{"servers":[{"order_id":"777","srv_id":"12345","status":"failed"}],"all_ready":"0"}`,
		`[]`,
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
	r := newWaitTestResource(t, waitTestHandler(`{"servers":[],"all_ready":"0"}`, `[]`))

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

func TestWaitForServerListFallbackReady(t *testing.T) {
	r := newWaitTestResource(t, waitTestHandler(
		`{"servers":[],"all_ready":"0"}`,
		`[{"srv_id":"12345","srv_status":"1","srv_status_install":"0","srv_primary_ip":"185.199.2.3","ips":[{"ip_id":"2","ip_address":"2a01:db8::1","ip_v4v6":"ipv6"}],"order":{"order_id":"777","order_status":"active"}}]`,
	))

	server, err := r.waitForServer(context.Background(), 777)
	if err != nil {
		t.Fatalf("waitForServer: %v", err)
	}
	if server == nil || server.SrvID != "12345" || server.IPv6 != "2a01:db8::1" {
		t.Fatalf("server = %+v, want srv_id 12345 adopted from the server list", server)
	}
	if server.Password != "" {
		t.Fatalf("password = %q, want empty on the list path", server.Password)
	}
}

func TestWaitForServerListFallbackInstallingKeepsWaiting(t *testing.T) {
	r := newWaitTestResource(t, waitTestHandler(
		`{"servers":[],"all_ready":"0"}`,
		`[{"srv_id":"12345","srv_status":"1","srv_status_install":"1","srv_primary_ip":"185.199.2.3","order":{"order_id":"777","order_status":"active"}}]`,
	))

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	server, err := r.waitForServer(ctx, 777)
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("err = %v, want timeout error while installing", err)
	}
	if server == nil || server.SrvID != "12345" {
		t.Fatalf("server = %+v, want the installing server's id preserved for the error path", server)
	}
}

func TestFindServerByIDConfirmsAbsence(t *testing.T) {
	var mu sync.Mutex
	listCalls := 0
	r := newWaitTestResource(t, func(w http.ResponseWriter, req *http.Request) {
		if !strings.HasPrefix(req.URL.Path, "/servers") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		mu.Lock()
		listCalls++
		n := listCalls
		mu.Unlock()
		if n < 3 {
			_, _ = w.Write([]byte(`{"meta":{},"data":[]}`))
			return
		}
		_, _ = w.Write([]byte(`{"meta":{},"data":[{"srv_id":"12345","order":{"order_id":"777","order_status":"active"}}]}`))
	})

	found, err := r.findServerByID(context.Background(), "12345")
	if err != nil {
		t.Fatalf("findServerByID: %v", err)
	}
	if found == nil || found.SrvID != "12345" {
		t.Fatalf("found = %+v, want the server after a transient list gap", found)
	}
}

func TestFindServerByIDAbsentAfterConfirmation(t *testing.T) {
	var mu sync.Mutex
	listCalls := 0
	r := newWaitTestResource(t, func(w http.ResponseWriter, req *http.Request) {
		mu.Lock()
		listCalls++
		mu.Unlock()
		_, _ = w.Write([]byte(`{"meta":{},"data":[]}`))
	})

	found, err := r.findServerByID(context.Background(), "12345")
	if err != nil {
		t.Fatalf("findServerByID: %v", err)
	}
	if found != nil {
		t.Fatalf("found = %+v, want nil for a confirmed absence", found)
	}
	mu.Lock()
	defer mu.Unlock()
	if listCalls != 5 {
		t.Fatalf("list calls = %d, want 5 confirming reads", listCalls)
	}
}

func TestWaitForServerDisappearedFails(t *testing.T) {
	var mu sync.Mutex
	listCalls := 0
	r := newWaitTestResource(t, func(w http.ResponseWriter, req *http.Request) {
		switch {
		case strings.HasPrefix(req.URL.Path, "/deploy/status"):
			_, _ = w.Write([]byte(`{"meta":{},"data":{"servers":[],"all_ready":"0"}}`))
		case strings.HasPrefix(req.URL.Path, "/servers"):
			mu.Lock()
			listCalls++
			n := listCalls
			mu.Unlock()
			if n == 1 {
				_, _ = w.Write([]byte(`{"meta":{},"data":[{"srv_id":"12345","srv_status":"1","srv_status_install":"1","srv_primary_ip":"185.199.2.3","order":{"order_id":"777","order_status":"active"}}]}`))
				return
			}
			_, _ = w.Write([]byte(`{"meta":{},"data":[]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	server, err := r.waitForServer(context.Background(), 777)
	if err == nil || !strings.Contains(err.Error(), "disappeared") {
		t.Fatalf("err = %v, want disappeared error", err)
	}
	if server == nil || server.SrvID != "12345" {
		t.Fatalf("server = %+v, want the vanished server's id preserved for cleanup", server)
	}
}
