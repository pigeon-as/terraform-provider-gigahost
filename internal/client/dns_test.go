package client

import (
	"encoding/json"
	"testing"
)

func TestFlexBool(t *testing.T) {
	cases := map[string]bool{
		`"1"`:     true,
		`"0"`:     false,
		`"true"`:  true,
		`"false"`: false,
		`""`:      false,
		`1`:       true,
		`true`:    true,
		`null`:    false,
	}
	for input, want := range cases {
		var b flexBool
		if err := b.UnmarshalJSON([]byte(input)); err != nil {
			t.Fatalf("UnmarshalJSON(%s): %v", input, err)
		}
		if bool(b) != want {
			t.Errorf("flexBool(%s) = %v, want %v", input, bool(b), want)
		}
	}
}

func TestNormalizeRecordValue(t *testing.T) {
	cases := []struct {
		recordType string
		value      string
		want       string
	}{
		{"AAAA", "2001:0db8:0000:0000:0000:0000:0000:0001", "2001:db8::1"},
		{"AAAA", "2001:db8::1", "2001:db8::1"},
		{"aaaa", "2001:0DB8::1", "2001:db8::1"},
		{"A", "203.0.113.10", "203.0.113.10"},
		{"CNAME", "Target.Example.Com", "Target.Example.Com"},
		{"AAAA", "not-an-ip", "not-an-ip"},
	}
	for _, tc := range cases {
		if got := NormalizeRecordValue(tc.recordType, tc.value); got != tc.want {
			t.Errorf("NormalizeRecordValue(%q, %q) = %q, want %q", tc.recordType, tc.value, got, tc.want)
		}
	}
}

func TestFlexInt64(t *testing.T) {
	cases := map[string]int64{
		`"10"`:   10,
		`10`:     10,
		`"3600"`: 3600,
		`3600`:   3600,
		`"0"`:    0,
		`""`:     0,
		`null`:   0,
	}
	for input, want := range cases {
		var n flexInt64
		if err := n.UnmarshalJSON([]byte(input)); err != nil {
			t.Fatalf("UnmarshalJSON(%s): %v", input, err)
		}
		if int64(n) != want {
			t.Errorf("flexInt64(%s) = %d, want %d", input, int64(n), want)
		}
	}

	var n flexInt64
	if err := n.UnmarshalJSON([]byte(`"abc"`)); err == nil {
		t.Error("expected an error for a non-numeric value")
	}
}

func TestDnsRecordPriorityDecode(t *testing.T) {
	cases := []struct {
		body    string
		wantNil bool
		want    int64
	}{
		{`{"record_priority": null}`, true, 0},
		{`{}`, true, 0},
		{`{"record_priority": "10"}`, false, 10},
		{`{"record_priority": 10}`, false, 10},
	}
	for _, tc := range cases {
		var record DnsRecord
		if err := json.Unmarshal([]byte(tc.body), &record); err != nil {
			t.Fatalf("Unmarshal(%s): %v", tc.body, err)
		}
		switch {
		case tc.wantNil && record.RecordPriority != nil:
			t.Errorf("%s: priority = %d, want nil", tc.body, int64(*record.RecordPriority))
		case !tc.wantNil && record.RecordPriority == nil:
			t.Errorf("%s: priority = nil, want %d", tc.body, tc.want)
		case !tc.wantNil && int64(*record.RecordPriority) != tc.want:
			t.Errorf("%s: priority = %d, want %d", tc.body, int64(*record.RecordPriority), tc.want)
		}
	}
}
