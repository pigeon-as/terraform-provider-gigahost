package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDNSZoneDataSource_basic(t *testing.T) {
	zoneName := testAccZoneName()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDNSZoneDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDNSZoneDataSourceConfig(zoneName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.gigahost_dns_zone.test", "zone_name", zoneName),
					resource.TestCheckResourceAttrPair("data.gigahost_dns_zone.test", "zone_id", "gigahost_dns_zone.test", "zone_id"),
					resource.TestCheckResourceAttr("data.gigahost_dns_zone.test", "zone_type", "NATIVE"),
				),
			},
		},
	})
}

func testAccDNSZoneDataSourceConfig(zoneName string) string {
	return fmt.Sprintf(`
resource "gigahost_dns_zone" "test" {
  zone_name = %q
}

data "gigahost_dns_zone" "test" {
  zone_name = gigahost_dns_zone.test.zone_name
}
`, zoneName)
}
