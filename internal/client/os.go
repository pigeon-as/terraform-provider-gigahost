package client

import (
	"context"
	"net/http"
	"path"
)

type Distro struct {
	DistID    string `json:"dist_id"`
	DistName  string `json:"dist_name"`
	DistValue string `json:"dist_value"`
}

type OS struct {
	OsID      string `json:"os_id"`
	DistID    string `json:"dist_id"`
	OsName    string `json:"os_name"`
	OsRelease string `json:"os_release"`
	OsDist    string `json:"os_dist"`
	OsArch    string `json:"os_arch"`
	OsMinRAM  string `json:"os_minram"`

	OsCustomPartition flexBool `json:"os_custom_partition"`
	OsSingleDiskOnly  flexBool `json:"os_single_disk_only"`
	OsSupportRAID     flexBool `json:"os_support_raid"`
	OsDedicatedOnly   flexBool `json:"os_dedicated_only"`
}

type OSCatalogEntry struct {
	Distro Distro
	OS     OS
}

func (c *Client) GetDistros(ctx context.Context) ([]Distro, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "reinstall/distro", nil, nil)
	if err != nil {
		return nil, err
	}

	var distros []Distro
	if err := c.sendRequest(req, &distros); err != nil {
		return nil, err
	}
	return distros, nil
}

func (c *Client) GetDistroVersions(ctx context.Context, distID string) ([]OS, error) {
	req, err := c.newRequest(ctx, http.MethodGet, path.Join("reinstall", "distro", distID), nil, nil)
	if err != nil {
		return nil, err
	}

	var versions []OS
	if err := c.sendRequest(req, &versions); err != nil {
		return nil, err
	}
	return versions, nil
}

func (c *Client) GetOSCatalog(ctx context.Context) ([]OSCatalogEntry, error) {
	distros, err := c.GetDistros(ctx)
	if err != nil {
		return nil, err
	}

	var catalog []OSCatalogEntry
	for _, d := range distros {
		versions, err := c.GetDistroVersions(ctx, d.DistID)
		if err != nil {
			return nil, err
		}
		for _, osImage := range versions {
			catalog = append(catalog, OSCatalogEntry{Distro: d, OS: osImage})
		}
	}
	return catalog, nil
}
