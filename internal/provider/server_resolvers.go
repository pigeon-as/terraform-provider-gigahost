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
		if strings.EqualFold(r.RegionName, region) || strings.EqualFold(r.RegionNameShort, region) {
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

func resolveOS(catalog []client.OSCatalogEntry, distro, version string) (int64, error) {
	var matches []client.OSCatalogEntry
	for _, e := range catalog {
		if osMatches(e, distro, version) {
			matches = append(matches, e)
		}
	}

	switch len(matches) {
	case 0:
		return 0, fmt.Errorf("no OS image found for distro %q version %q", distro, version)
	case 1:
		id, err := strconv.ParseInt(matches[0].OS.OsID, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("OS %q has an unparseable id %q: %w", matches[0].OS.OsName, matches[0].OS.OsID, err)
		}
		return id, nil
	default:
		names := make([]string, 0, len(matches))
		for _, m := range matches {
			names = append(names, m.OS.OsName)
		}
		return 0, fmt.Errorf("%d OS images match distro %q version %q (%s); narrow os_version", len(matches), distro, version, strings.Join(names, ", "))
	}
}

func osMatches(e client.OSCatalogEntry, distro, version string) bool {
	if distro != "" && !strings.EqualFold(e.Distro.DistName, distro) && !strings.EqualFold(e.Distro.DistValue, distro) {
		return false
	}
	if version != "" &&
		!strings.Contains(strings.ToLower(e.OS.OsName), strings.ToLower(version)) &&
		!strings.EqualFold(e.OS.OsDist, version) {
		return false
	}
	return true
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
	if a == b {
		return true
	}
	ai, errA := strconv.ParseInt(a, 10, 64)
	bi, errB := strconv.ParseInt(b, 10, 64)
	return errA == nil && errB == nil && ai == bi
}
