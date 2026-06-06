package client

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	c, err := NewClient(&Config{
		Address:    srv.URL,
		Token:      "test-token",
		HTTPClient: srv.Client(),
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

func TestNewClient(t *testing.T) {
	t.Run("requires a token", func(t *testing.T) {
		if _, err := NewClient(&Config{Token: ""}); err == nil {
			t.Fatal("expected an error when the token is empty")
		}
	})

	t.Run("defaults the address with a trailing slash", func(t *testing.T) {
		c, err := NewClient(&Config{Token: "x"})
		if err != nil {
			t.Fatalf("NewClient: %v", err)
		}
		// The base path is normalized with a trailing slash so relative request
		// paths resolve against it (the go-tfe pattern).
		want := DefaultAddress + "/"
		if c.baseURL.String() != want {
			t.Errorf("baseURL = %q, want %q", c.baseURL.String(), want)
		}
	})
}

func TestGetAccount(t *testing.T) {
	const body = `{
		"meta": {"status": 200, "status_message": "200 OK"},
		"data": {
			"cust_id": "1111",
			"cust_name": "Example AS",
			"cust_billing_email": "billing@example.com"
		}
	}`

	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/account" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer test-token")
		}
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Errorf("Accept = %q", got)
		}
		_, _ = io.WriteString(w, body)
	})

	account, err := c.GetAccount(context.Background())
	if err != nil {
		t.Fatalf("GetAccount: %v", err)
	}
	if account.CustID != "1111" {
		t.Errorf("CustID = %q, want %q", account.CustID, "1111")
	}
	if account.CustName != "Example AS" {
		t.Errorf("CustName = %q, want %q", account.CustName, "Example AS")
	}
	if account.CustBillingEmail != "billing@example.com" {
		t.Errorf("CustBillingEmail = %q", account.CustBillingEmail)
	}
}

func TestGetAccountError(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, `{"meta": {"status": 403, "status_message": "403 Forbidden", "message": "API key does not permit this operation."}}`)
	})

	_, err := c.GetAccount(context.Background())
	if err == nil {
		t.Fatal("expected an error")
	}

	var apiErr *Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *Error, got %T", err)
	}
	if apiErr.StatusCode != http.StatusForbidden {
		t.Errorf("StatusCode = %d, want %d", apiErr.StatusCode, http.StatusForbidden)
	}
	if apiErr.Message != "API key does not permit this operation." {
		t.Errorf("Message = %q", apiErr.Message)
	}
}

func TestErrorMessageFallsBackToStatusText(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = io.WriteString(w, "<html><body>502 Bad Gateway</body></html>")
	})

	_, err := c.GetAccount(context.Background())
	var apiErr *Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *Error, got %T", err)
	}
	if apiErr.Message != http.StatusText(http.StatusBadGateway) {
		t.Errorf("Message = %q, want %q", apiErr.Message, http.StatusText(http.StatusBadGateway))
	}
}

func TestGetAccountNullData(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"meta": {"status": 200}, "data": null}`)
	})

	account, err := c.GetAccount(context.Background())
	if err != nil {
		t.Fatalf("GetAccount: %v", err)
	}
	if account.CustID != "" {
		t.Errorf("expected zero-value account, got %+v", account)
	}
}

func TestListDnsZonesEmpty(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"meta": {"status": 200}, "data": []}`)
	})

	zones, err := c.ListDnsZones(context.Background())
	if err != nil {
		t.Fatalf("ListDnsZones: %v", err)
	}
	if len(zones) != 0 {
		t.Errorf("expected 0 zones, got %d", len(zones))
	}
}

func TestGetZoneNotFound(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"meta": {"status": 200}, "data": [{"zone_id": "1", "zone_name": "a.example.com"}]}`)
	})

	_, err := c.GetZone(context.Background(), "999")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestCreateRecordResolve(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			// Create returns meta-only; the client resolves from the list.
			_, _ = io.WriteString(w, `{"meta": {"status": 201}, "data": []}`)
		default:
			_, _ = io.WriteString(w, `{"meta": {"status": 200}, "data": [{"record_id": "r1", "record_name": "www", "record_type": "A", "record_value": "1.2.3.4", "record_ttl": 3600}]}`)
		}
	})

	record, err := c.CreateRecord(context.Background(), "z1", RecordRequest{
		RecordName:  "www",
		RecordType:  "A",
		RecordValue: "1.2.3.4",
		RecordTTL:   3600,
	})
	if err != nil {
		t.Fatalf("CreateRecord: %v", err)
	}
	if record.RecordID != "r1" {
		t.Errorf("RecordID = %q, want %q", record.RecordID, "r1")
	}
}

func TestDeleteRecordQuery(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		q := r.URL.Query()
		if q.Get("name") != "www" || q.Get("type") != "A" || q.Get("value") != "1.2.3.4" {
			t.Errorf("query = %q, want name=www&type=A&value=1.2.3.4", r.URL.RawQuery)
		}
		_, _ = io.WriteString(w, `{"meta": {"status": 200}, "data": []}`)
	})

	if err := c.DeleteRecord(context.Background(), "z1", "r1", "www", "A", "1.2.3.4"); err != nil {
		t.Fatalf("DeleteRecord: %v", err)
	}
}
