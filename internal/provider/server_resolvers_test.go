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
		{"case insensitive", "sandefjord", 1, false},
		{"short name no longer matches", "DC1", 0, true},
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
		{Distro: client.Distro{DistName: "Ubuntu", DistValue: "ubuntu"}, OS: client.OS{OSID: "100", OSName: "Ubuntu 24.04 LTS", OSDist: "noble"}},
		{Distro: client.Distro{DistName: "Ubuntu", DistValue: "ubuntu"}, OS: client.OS{OSID: "101", OSName: "Ubuntu 22.04 LTS", OSDist: "jammy"}},
		{Distro: client.Distro{DistName: "Debian", DistValue: "debian"}, OS: client.OS{OSID: "200", OSName: "Debian 12", OSDist: "bookworm"}},
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
		os      string
		wantID  int64
		wantErr bool
	}{
		{"by os_name", "Ubuntu 24.04 LTS", 100, false},
		{"by codename", "noble", 100, false},
		{"case insensitive name", "ubuntu 24.04 lts", 100, false},
		{"case insensitive codename", "NOBLE", 100, false},
		{"not found", "Ubuntu 18.04 LTS", 0, true},
		{"partial name does not match", "24.04", 0, true},
		{"distro alone does not match", "Ubuntu", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := resolveOS(catalog, tt.os)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && id != tt.wantID {
				t.Errorf("got %d, want %d", id, tt.wantID)
			}
		})
	}
}
