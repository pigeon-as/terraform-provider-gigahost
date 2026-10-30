package client

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"testing"
)

func TestUpdateNameservers(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/dns/zones/195/nameservers" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body updateNameserversRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if !reflect.DeepEqual(body.Nameservers, []string{"ns1.example.com", "ns2.example.com"}) {
			t.Fatalf("body = %+v", body)
		}
		_, _ = io.WriteString(w, `{"meta":{"status":200,"message":"Nameservers updated successfully."}}`)
	})

	if err := c.UpdateNameservers(context.Background(), "195", []string{"ns1.example.com", "ns2.example.com"}); err != nil {
		t.Fatalf("UpdateNameservers: %v", err)
	}
	if err := c.UpdateNameservers(context.Background(), "195", []string{"ns1.example.com"}); err == nil {
		t.Error("expected an error for fewer than two nameservers")
	}
	if err := c.UpdateNameservers(context.Background(), "abc", []string{"a", "b"}); err == nil {
		t.Error("expected an error for a non-numeric zone id")
	}
}

func TestListApexNameservers(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/dns/zones/195/records" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = io.WriteString(w, `{"meta":{"status":200},"data":[
			{"record_id":"1","record_name":"@","record_type":"NS","record_value":"ns1.example.com","record_ttl":3600},
			{"record_id":"2","record_name":"sub","record_type":"NS","record_value":"ns.sub.example.com","record_ttl":3600},
			{"record_id":"3","record_name":"@","record_type":"A","record_value":"192.0.2.1","record_ttl":3600},
			{"record_id":"4","record_name":"@","record_type":"NS","record_value":"ns2.example.com","record_ttl":3600}]}`)
	})

	got, err := c.ListApexNameservers(context.Background(), "195")
	if err != nil {
		t.Fatalf("ListApexNameservers: %v", err)
	}
	if want := []string{"ns1.example.com", "ns2.example.com"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestSetDNSSEC(t *testing.T) {
	for _, enable := range []bool{true, false} {
		want := 0
		if enable {
			want = 1
		}
		c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut || r.URL.Path != "/dns/zones/195/dnssec" {
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if body["enable"] != float64(want) {
				t.Fatalf("body = %v, want enable=%d", body, want)
			}
			_, _ = io.WriteString(w, `{"meta":{"status":200,"message":"ok"}}`)
		})
		if err := c.SetDNSSEC(context.Background(), "195", enable); err != nil {
			t.Fatalf("SetDNSSEC(%v): %v", enable, err)
		}
	}
}

func TestSetDNSSECForbidden(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, `{"meta":{"status":403,"message":"DNSSEC can only be enabled for registered domains."}}`)
	})
	err := c.SetDNSSEC(context.Background(), "195", true)
	var apiErr *Error
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusForbidden {
		t.Fatalf("err = %v, want 403 *Error", err)
	}
}

func TestGetDSRecords(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/dns/zones/195/ds-records" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_, _ = io.WriteString(w, `{"meta":{"status":200},"data":{"ds_records":"12345 13 2 ABCDEF"}}`)
	})
	got, err := c.GetDSRecords(context.Background(), "195")
	if err != nil {
		t.Fatalf("GetDSRecords: %v", err)
	}
	if got != "12345 13 2 ABCDEF" {
		t.Fatalf("got %q", got)
	}
}

func TestGetDSRecordsNotEnabled(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"meta":{"status":400,"message":"DNSSEC is not enabled for this domain."},"data":[]}`)
	})
	if _, err := c.GetDSRecords(context.Background(), "195"); !errors.Is(err, ErrDNSSECNotEnabled) {
		t.Fatalf("err = %v, want ErrDNSSECNotEnabled", err)
	}
}

func TestSubmitExternalDSRecords(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/dns/zones/195/ds-records/external" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		raw, _ := io.ReadAll(r.Body)
		var body map[string][]map[string]any
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		recs := body["ds_records"]
		if len(recs) != 1 || recs[0]["keyTag"] != float64(12345) || recs[0]["alg"] != float64(13) ||
			recs[0]["digestType"] != float64(2) || recs[0]["digest"] != "ABCD" {
			t.Fatalf("body = %s", raw)
		}
		if _, ok := recs[0]["publicKey"]; ok {
			t.Fatalf("unset publicKey should be omitted: %s", raw)
		}
		if _, ok := recs[0]["keyType"]; ok {
			t.Fatalf("unset keyType should be omitted: %s", raw)
		}
		_, _ = io.WriteString(w, `{"meta":{"status":200,"message":"DS records submitted successfully."}}`)
	})

	err := c.SubmitExternalDSRecords(context.Background(), "195", []DSRecord{{KeyTag: 12345, Algorithm: 13, DigestType: 2, Digest: "ABCD"}})
	if err != nil {
		t.Fatalf("SubmitExternalDSRecords: %v", err)
	}
	if err := c.SubmitExternalDSRecords(context.Background(), "195", nil); err == nil {
		t.Error("expected an error for an empty record set")
	}
}

func TestCreateSecondaryZone(t *testing.T) {
	var created bool
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/dns/zones/secondary":
			var body createSecondaryZoneRequest
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if body.ZoneName != "example.com" || !reflect.DeepEqual(body.Masters, []string{"192.0.2.1", "192.0.2.2"}) {
				t.Fatalf("body = %+v", body)
			}
			created = true
			_, _ = io.WriteString(w, `{"meta":{"status":200}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/dns/zones":
			_, _ = io.WriteString(w, `{"meta":{"status":200},"data":[{"zone_id":"7","zone_name":"example.com","zone_type":"SLAVE","zone_masters":"192.0.2.1,192.0.2.2"}]}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	z, err := c.CreateSecondaryZone(context.Background(), "example.com", []string{"192.0.2.1", "192.0.2.2"})
	if err != nil {
		t.Fatalf("CreateSecondaryZone: %v", err)
	}
	if !created || z.ZoneID != "7" || z.ZoneType != "SLAVE" {
		t.Fatalf("zone = %+v", z)
	}
	if got := SplitMasters(z.ZoneMasters); !reflect.DeepEqual(got, []string{"192.0.2.1", "192.0.2.2"}) {
		t.Fatalf("SplitMasters = %v", got)
	}
	if _, err := c.CreateSecondaryZone(context.Background(), "example.com", nil); err == nil {
		t.Error("expected an error without masters")
	}
}
