package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pigeon-as/terraform-provider-gigahost/internal/client"
)

func newWaitTestResource(t *testing.T, handler http.HandlerFunc) *serverResource {
	t.Helper()

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

func deployStatusHandler(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"meta":{},"data":` + body + `}`))
	}
}

func TestWaitForServerReady(t *testing.T) {
	r := newWaitTestResource(t, deployStatusHandler(
		`{"servers":[{"order_id":"777","srv_id":"12345","ip":"185.199.2.3","status":"ready"}],"all_ready":"1"}`,
	))

	server, err := r.waitForServer(context.Background(), 777)
	if err != nil {
		t.Fatalf("waitForServer: %v", err)
	}
	if server == nil || int64(server.SrvID) != 12345 {
		t.Fatalf("server = %+v, want srv_id 12345", server)
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
	if server == nil || int64(server.SrvID) != 12345 {
		t.Fatalf("server = %+v, want the failed server returned alongside the error", server)
	}
}

func TestWaitForServerTimeoutWithoutServer(t *testing.T) {
	r := newWaitTestResource(t, deployStatusHandler(`{"servers":[],"all_ready":"0"}`))

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	server, err := r.waitForServer(ctx, 777)
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("err = %v, want timeout error", err)
	}
	if server != nil {
		t.Fatalf("server = %+v, want nil when the order was never seen", server)
	}
}
