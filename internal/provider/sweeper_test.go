package provider

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/pigeon-as/terraform-provider-gigahost/internal/client"
)

func TestMain(m *testing.M) {
	resource.TestMain(m)
}

func sweepClient() (*client.Client, error) {
	return client.NewClient(&client.Config{
		Address: os.Getenv("GIGAHOST_BASE_URL"),
		Token:   os.Getenv("GIGAHOST_API_TOKEN"),
	})
}

func init() {
	resource.AddTestSweepers("gigahost_dns_zone", &resource.Sweeper{
		Name: "gigahost_dns_zone",
		F: func(string) error {
			c, err := sweepClient()
			if err != nil {
				return err
			}
			zones, err := c.ListDnsZones(context.Background())
			if err != nil {
				return err
			}
			for _, z := range zones {
				if !strings.HasPrefix(z.ZoneName, "tf-acc-") {
					continue
				}
				if err := c.DeleteZone(context.Background(), z.ZoneID); err != nil {
					return err
				}
			}
			return nil
		},
	})

	resource.AddTestSweepers("gigahost_ssh_key", &resource.Sweeper{
		Name: "gigahost_ssh_key",
		F: func(string) error {
			c, err := sweepClient()
			if err != nil {
				return err
			}
			keys, err := c.ListSSHKeys(context.Background())
			if err != nil {
				return err
			}
			for _, k := range keys {
				if !strings.HasPrefix(k.KeyName, "tf-acc-") {
					continue
				}
				if err := c.DeleteSSHKey(context.Background(), k.KeyID); err != nil {
					return err
				}
			}
			return nil
		},
	})
}
