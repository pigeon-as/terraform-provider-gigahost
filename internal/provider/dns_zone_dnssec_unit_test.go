package provider

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/pigeon-as/terraform-provider-gigahost/internal/client"
)

// These tests drive the resource CRUD methods against an httptest mock server.
// They never reach the real Gigahost API.

type recordedRequest struct {
	Method, Path, Body string
}

type mockAPI struct {
	mu       sync.Mutex
	requests []recordedRequest
}

func (m *mockAPI) last() recordedRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.requests[len(m.requests)-1]
}

func newMockClient(t *testing.T, handler func(w http.ResponseWriter, r *http.Request)) (*client.Client, *mockAPI) {
	t.Helper()
	api := &mockAPI{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		r.Body = io.NopCloser(strings.NewReader(string(body)))
		api.mu.Lock()
		api.requests = append(api.requests, recordedRequest{r.Method, r.URL.Path, string(body)})
		api.mu.Unlock()
		handler(w, r)
	}))
	t.Cleanup(srv.Close)
	c, err := client.NewClient(&client.Config{Address: srv.URL, Token: "test-token", HTTPClient: srv.Client()})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c, api
}

func schemaOf(t *testing.T, r resource.Resource) resource.SchemaResponse {
	t.Helper()
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if diags := resp.Schema.ValidateImplementation(context.Background()); diags.HasError() {
		t.Fatalf("invalid schema: %v", diags)
	}
	return resp
}

func stringSet(vals ...string) tftypes.Value {
	elems := make([]tftypes.Value, len(vals))
	for i, v := range vals {
		elems[i] = tftypes.NewValue(tftypes.String, v)
	}
	return tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, elems)
}

func TestNameserversResourceCreateReadUpdateDelete(t *testing.T) {
	ctx := context.Background()
	c, api := newMockClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/dns/zones/195/nameservers":
			_, _ = io.WriteString(w, `{"meta":{"status":200}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/dns/zones":
			_, _ = io.WriteString(w, `{"meta":{"status":200},"data":[{"zone_id":"195","zone_name":"example.com"}]}`)
		case r.Method == http.MethodGet && r.URL.Path == "/dns/zones/195/records":
			_, _ = io.WriteString(w, `{"meta":{"status":200},"data":[
				{"record_id":"1","record_name":"@","record_type":"NS","record_value":"ns1.example.net.","record_ttl":3600},
				{"record_id":"2","record_name":"@","record_type":"NS","record_value":"ns3.example.net","record_ttl":3600}]}`)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	})

	r := &dnsZoneNameserversResource{client: c}
	sch := schemaOf(t, r).Schema
	objType := sch.Type().TerraformType(ctx)
	val := func(ns tftypes.Value) tftypes.Value {
		return tftypes.NewValue(objType, map[string]tftypes.Value{
			"zone_id":     tftypes.NewValue(tftypes.String, "195"),
			"nameservers": ns,
		})
	}

	plan := tfsdk.Plan{Schema: sch, Raw: val(stringSet("ns1.example.net", "ns2.example.net"))}
	createResp := resource.CreateResponse{State: tfsdk.State{Schema: sch, Raw: tftypes.NewValue(objType, nil)}}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, &createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", createResp.Diagnostics)
	}
	if got := api.last(); got.Method != http.MethodPut || !strings.Contains(got.Body, `"nameservers":["ns1.example.net","ns2.example.net"]`) {
		// set ordering is not guaranteed
		if !strings.Contains(got.Body, "ns1.example.net") || !strings.Contains(got.Body, "ns2.example.net") {
			t.Fatalf("unexpected create request: %+v", got)
		}
	}

	// Read: apex NS records differ from state, so drift is reported.
	readResp := resource.ReadResponse{State: createResp.State}
	r.Read(ctx, resource.ReadRequest{State: createResp.State}, &readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", readResp.Diagnostics)
	}
	var model dnsZoneNameserversResourceModel
	readResp.State.Get(ctx, &model)
	var got []string
	model.Nameservers.ElementsAs(ctx, &got, false)
	if !sameNameservers(got, []string{"ns1.example.net", "ns3.example.net"}) {
		t.Fatalf("drifted nameservers = %v", got)
	}

	// Update restores the configured nameservers.
	upd := resource.UpdateResponse{State: readResp.State}
	r.Update(ctx, resource.UpdateRequest{Plan: plan, State: readResp.State}, &upd)
	if upd.Diagnostics.HasError() {
		t.Fatalf("Update: %v", upd.Diagnostics)
	}

	// Delete makes no API call.
	n := len(api.requests)
	r.Delete(ctx, resource.DeleteRequest{State: upd.State}, &resource.DeleteResponse{})
	if len(api.requests) != n {
		t.Fatalf("Delete made an API call: %+v", api.last())
	}
}

func TestSameNameservers(t *testing.T) {
	if !sameNameservers([]string{"NS1.Example.com."}, []string{"ns1.example.com"}) {
		t.Error("expected case and trailing dot to be ignored")
	}
	if sameNameservers([]string{"a", "b"}, []string{"a", "c"}) {
		t.Error("expected different sets to differ")
	}
}

func TestDNSSECResourceLifecycle(t *testing.T) {
	ctx := context.Background()
	enabled := false
	c, api := newMockClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/dns/zones/195/dnssec":
			enabled = strings.Contains(readBody(r), `"enable":1`)
			_, _ = io.WriteString(w, `{"meta":{"status":200}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/dns/zones/195/ds-records":
			if !enabled {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = io.WriteString(w, `{"meta":{"status":400,"message":"DNSSEC is not enabled for this domain."},"data":[]}`)
				return
			}
			_, _ = io.WriteString(w, `{"meta":{"status":200},"data":{"ds_records":"12345 13 2 ABCD"}}`)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	})
	r := &dnsZoneDNSSECResource{client: c}
	sch := schemaOf(t, r).Schema
	objType := sch.Type().TerraformType(ctx)
	plan := tfsdk.Plan{Schema: sch, Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
		"zone_id":    tftypes.NewValue(tftypes.String, "195"),
		"ds_records": tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
	})}
	createResp := resource.CreateResponse{State: tfsdk.State{Schema: sch, Raw: tftypes.NewValue(objType, nil)}}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, &createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", createResp.Diagnostics)
	}
	var m dnsZoneDNSSECResourceModel
	createResp.State.Get(ctx, &m)
	if m.DSRecords != types.StringValue("12345 13 2 ABCD") {
		t.Fatalf("ds_records = %v", m.DSRecords)
	}

	// Disabled out of band: the resource disappears from state.
	enabled = false
	readResp := resource.ReadResponse{State: createResp.State}
	r.Read(ctx, resource.ReadRequest{State: createResp.State}, &readResp)
	if readResp.Diagnostics.HasError() || !readResp.State.Raw.IsNull() {
		t.Fatalf("expected the resource to be removed: %v", readResp.Diagnostics)
	}

	enabled = true
	r.Delete(ctx, resource.DeleteRequest{State: createResp.State}, &resource.DeleteResponse{})
	if got := api.last(); got.Method != http.MethodPut || !strings.Contains(got.Body, `"enable":0`) {
		t.Fatalf("unexpected delete request: %+v", got)
	}
}

func readBody(r *http.Request) string {
	b, _ := io.ReadAll(r.Body)
	return string(b)
}

func TestDSRecordsResourceCreateAndDelete(t *testing.T) {
	ctx := context.Background()
	c, api := newMockClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"meta":{"status":200}}`)
	})
	r := &dnsZoneDSRecordsResource{client: c}
	sch := schemaOf(t, r).Schema
	objType := sch.Type().TerraformType(ctx)
	dsType := tftypes.Set{ElementType: tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"key_tag": tftypes.Number, "algorithm": tftypes.Number, "digest_type": tftypes.Number,
		"digest": tftypes.String, "public_key": tftypes.String, "key_type": tftypes.Number,
	}}}.ElementType
	rec := tftypes.NewValue(dsType, map[string]tftypes.Value{
		"key_tag":     tftypes.NewValue(tftypes.Number, 12345),
		"algorithm":   tftypes.NewValue(tftypes.Number, 13),
		"digest_type": tftypes.NewValue(tftypes.Number, 2),
		"digest":      tftypes.NewValue(tftypes.String, "ABCD"),
		"public_key":  tftypes.NewValue(tftypes.String, nil),
		"key_type":    tftypes.NewValue(tftypes.Number, 257),
	})
	plan := tfsdk.Plan{Schema: sch, Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
		"zone_id":    tftypes.NewValue(tftypes.String, "195"),
		"ds_records": tftypes.NewValue(tftypes.Set{ElementType: dsType}, []tftypes.Value{rec}),
	})}

	createResp := resource.CreateResponse{State: tfsdk.State{Schema: sch, Raw: tftypes.NewValue(objType, nil)}}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, &createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", createResp.Diagnostics)
	}
	got := api.last()
	if got.Method != http.MethodPost || got.Path != "/dns/zones/195/ds-records/external" ||
		!strings.Contains(got.Body, `"keyTag":12345`) || !strings.Contains(got.Body, `"alg":13`) ||
		!strings.Contains(got.Body, `"digestType":2`) || !strings.Contains(got.Body, `"keyType":257`) {
		t.Fatalf("unexpected request: %+v", got)
	}

	r.Delete(ctx, resource.DeleteRequest{State: createResp.State}, &resource.DeleteResponse{})
	if got := api.last(); got.Method != http.MethodPut || got.Path != "/dns/zones/195/dnssec" || !strings.Contains(got.Body, `"enable":0`) {
		t.Fatalf("unexpected delete request: %+v", got)
	}
}
