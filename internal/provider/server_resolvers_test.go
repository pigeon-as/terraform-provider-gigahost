package provider

import (
	"testing"

	"github.com/pigeon-as/terraform-provider-gigahost/internal/client"
)

func testResolverCatalog() *client.DeployCatalog {
	return &client.DeployCatalog{
		Tiers: []client.DeployTier{
			{Products: []client.DeployProduct{
				{ProductID: 7955, PriceID: 4054, ProductName: "KVM Value VPS 4GB", RegionIDs: []int64{1}},
				{ProductID: 7956, PriceID: 4055, ProductName: "KVM Value VPS 8GB", RegionIDs: []int64{1, 2}},
			}},
		},
		Regions: []client.DeployRegion{
			{RegionID: "1", RegionName: "Sandefjord", RegionNameShort: "DC1", RegionActive: true},
			{RegionID: "2", RegionName: "Oslo", RegionNameShort: "DC2", RegionActive: true},
			{RegionID: "3", RegionName: "Bergen", RegionNameShort: "DC3", RegionActive: false},
		},
	}
}

func TestResolveProduct(t *testing.T) {
	catalog := testResolverCatalog()
	tests := []struct {
		name      string
		product   string
		wantID    int64
		wantPrice int64
		wantErr   bool
	}{
		{"exact", "KVM Value VPS 4GB", 7955, 4054, false},
		{"case insensitive", "kvm value vps 4gb", 7955, 4054, false},
		{"not found", "Nonexistent", 0, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, price, err := resolveProduct(catalog, tt.product)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && (id != tt.wantID || price != tt.wantPrice) {
				t.Errorf("got (%d, %d), want (%d, %d)", id, price, tt.wantID, tt.wantPrice)
			}
		})
	}
}

func TestResolveProductAmbiguous(t *testing.T) {
	catalog := &client.DeployCatalog{
		Tiers: []client.DeployTier{
			{Products: []client.DeployProduct{
				{ProductID: 1, PriceID: 10, ProductName: "Duplicate"},
				{ProductID: 2, PriceID: 20, ProductName: "Duplicate"},
			}},
		},
	}
	if _, _, err := resolveProduct(catalog, "Duplicate"); err == nil {
		t.Error("expected an error for an ambiguous product name")
	}
}

func TestResolveRegion(t *testing.T) {
	catalog := testResolverCatalog()
	tests := []struct {
		name    string
		region  string
		wantID  int64
		wantErr bool
	}{
		{"by name", "Sandefjord", 1, false},
		{"by short name", "DC1", 1, false},
		{"case insensitive", "sandefjord", 1, false},
		{"not found", "Nowhere", 0, true},
		{"inactive", "Bergen", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := resolveRegion(catalog, tt.region)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && id != tt.wantID {
				t.Errorf("got %d, want %d", id, tt.wantID)
			}
		})
	}
}

func testResolverOSCatalog() []client.OSCatalogEntry {
	return []client.OSCatalogEntry{
		{Distro: client.Distro{DistName: "Ubuntu", DistValue: "ubuntu"}, OS: client.OS{OsID: "100", OsName: "Ubuntu 24.04 LTS", OsDist: "noble"}},
		{Distro: client.Distro{DistName: "Ubuntu", DistValue: "ubuntu"}, OS: client.OS{OsID: "101", OsName: "Ubuntu 22.04 LTS", OsDist: "jammy"}},
		{Distro: client.Distro{DistName: "Debian", DistValue: "debian"}, OS: client.OS{OsID: "200", OsName: "Debian 12", OsDist: "bookworm"}},
	}
}

func TestProductOffersRegion(t *testing.T) {
	catalog := testResolverCatalog()
	if !productOffersRegion(catalog, 7955, 1) {
		t.Error("product 7955 should offer region 1")
	}
	if productOffersRegion(catalog, 7955, 2) {
		t.Error("product 7955 should not offer region 2")
	}
}

func TestResolveOS(t *testing.T) {
	catalog := testResolverOSCatalog()
	tests := []struct {
		name    string
		distro  string
		version string
		wantID  int64
		wantErr bool
	}{
		{"name and version substring", "Ubuntu", "24.04", 100, false},
		{"slug and codename", "ubuntu", "noble", 100, false},
		{"distro not found", "Arch", "1", 0, true},
		{"version not found", "Ubuntu", "18.04", 0, true},
		{"ambiguous version substring", "Ubuntu", "2", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := resolveOS(catalog, tt.distro, tt.version)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && id != tt.wantID {
				t.Errorf("got %d, want %d", id, tt.wantID)
			}
		})
	}
}
