package provider

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pigeon-as/terraform-provider-gigahost/internal/client"
)

func resolveProduct(catalog *client.DeployCatalog, productName string) (productID, priceID int64, err error) {
	var matches []client.DeployProduct
	for _, t := range catalog.Tiers {
		for _, p := range t.Products {
			if strings.EqualFold(p.ProductName, productName) {
				matches = append(matches, p)
			}
		}
	}

	switch len(matches) {
	case 0:
		return 0, 0, fmt.Errorf("no product named %q in the catalog", productName)
	case 1:
		return matches[0].ProductID, matches[0].PriceID, nil
	default:
		return 0, 0, fmt.Errorf("%d products named %q in the catalog", len(matches), productName)
	}
}

func resolveRegion(catalog *client.DeployCatalog, region string) (int64, error) {
	var matches []client.DeployRegion
	for _, r := range catalog.Regions {
		if strings.EqualFold(r.RegionName, region) {
			matches = append(matches, r)
		}
	}

	switch len(matches) {
	case 0:
		return 0, fmt.Errorf("no region found named %q", region)
	case 1:
		if !bool(matches[0].RegionActive) {
			return 0, fmt.Errorf("region %q is not active", region)
		}
		id, err := strconv.ParseInt(matches[0].RegionID, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("region %q has an unparseable id %q: %w", region, matches[0].RegionID, err)
		}
		return id, nil
	default:
		return 0, fmt.Errorf("%d regions match %q", len(matches), region)
	}
}

// findOS looks up a deployable OS image by its catalog name or release
// codename — the os_name (e.g. "Ubuntu 24.04 LTS") or os_dist (e.g. "noble")
// the API actually returns, matched exactly.
func findOS(catalog []client.OSCatalogEntry, os string) (*client.OSCatalogEntry, error) {
	var matches []client.OSCatalogEntry
	for _, e := range catalog {
		if strings.EqualFold(e.OS.OSName, os) || strings.EqualFold(e.OS.OSDist, os) {
			matches = append(matches, e)
		}
	}

	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("no OS image named %q in the catalog (use the os_name like %q or the codename like %q)", os, "Ubuntu 24.04 LTS", "noble")
	case 1:
		return &matches[0], nil
	default:
		names := make([]string, 0, len(matches))
		for _, m := range matches {
			names = append(names, m.OS.OSName)
		}
		return nil, fmt.Errorf("%d OS images match %q (%s)", len(matches), os, strings.Join(names, ", "))
	}
}

func resolveOS(catalog []client.OSCatalogEntry, os string) (int64, error) {
	e, err := findOS(catalog, os)
	if err != nil {
		return 0, err
	}
	id, err := strconv.ParseInt(e.OS.OSID, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("OS %q has an unparseable id %q: %w", e.OS.OSName, e.OS.OSID, err)
	}
	return id, nil
}

func productOffersRegion(catalog *client.DeployCatalog, productID, regionID int64) bool {
	for _, t := range catalog.Tiers {
		for _, p := range t.Products {
			if p.ProductID == productID {
				for _, id := range p.RegionIDs {
					if id == regionID {
						return true
					}
				}
				return false
			}
		}
	}
	return false
}

func equalID(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	if a == b {
		return true
	}
	ai, errA := strconv.ParseInt(a, 10, 64)
	bi, errB := strconv.ParseInt(b, 10, 64)
	return errA == nil && errB == nil && ai == bi
}
