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

	t.Run("defaults the address", func(t *testing.T) {
		c, err := NewClient(&Config{Token: "x"})
		if err != nil {
			t.Fatalf("NewClient: %v", err)
		}
		if c.baseURL.String() != DefaultAddress {
			t.Errorf("baseURL = %q, want %q", c.baseURL.String(), DefaultAddress)
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
