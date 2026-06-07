package client

import (
	"context"
	"io"
	"net/http"
	"testing"
)

func TestGetOSCatalog(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/reinstall/distro":
			_, _ = io.WriteString(w, `{"meta":{"status":200},"data":[{"dist_id":"3","dist_name":"Ubuntu","dist_value":"ubuntu"},{"dist_id":"2","dist_name":"Debian","dist_value":"debian"}]}`)
		case "/reinstall/distro/3":
			_, _ = io.WriteString(w, `{"meta":{"status":200},"data":[{"os_id":"102","dist_id":"3","os_name":"Ubuntu 24.04 LTS","os_release":"ubuntu","os_dist":"noble","os_arch":"amd64","os_custom_partition":"1","os_single_disk_only":"1","os_support_raid":"1","os_dedicated_only":"0","os_minram":"4","os_template":"","os_slow_install":"0"}]}`)
		case "/reinstall/distro/2":
			_, _ = io.WriteString(w, `{"meta":{"status":200},"data":[{"os_id":"95","dist_id":"2","os_name":"Debian 12","os_release":"debian","os_dist":"bookworm","os_arch":"amd64","os_custom_partition":"1","os_single_disk_only":"0","os_support_raid":"1","os_dedicated_only":"0","os_minram":"2","os_template":"","os_slow_install":"0"}]}`)
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	})

	catalog, err := c.GetOSCatalog(context.Background())
	if err != nil {
		t.Fatalf("GetOSCatalog: %v", err)
	}
	if len(catalog) != 2 {
		t.Fatalf("expected 2 OS entries, got %d", len(catalog))
	}

	var found bool
	for _, e := range catalog {
		if e.OS.OsID == "102" {
			found = true
			if e.OS.OsName != "Ubuntu 24.04 LTS" {
				t.Errorf("OsName = %q, want %q", e.OS.OsName, "Ubuntu 24.04 LTS")
			}
			if e.Distro.DistName != "Ubuntu" {
				t.Errorf("DistName = %q, want %q", e.Distro.DistName, "Ubuntu")
			}
			if !e.OS.OsSupportRAID {
				t.Errorf("OsSupportRAID = false, want true")
			}
			if e.OS.OsDedicatedOnly {
				t.Errorf("OsDedicatedOnly = true, want false")
			}
		}
	}
	if !found {
		t.Error("Ubuntu 24.04 (os_id 102) not found in catalog")
	}
}
