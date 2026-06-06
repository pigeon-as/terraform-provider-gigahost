package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDNSRecordsDataSource_basic(t *testing.T) {
	zoneName := testAccZoneName()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDNSRecordDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDNSRecordsDataSourceConfig(zoneName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.gigahost_dns_records.test", "records.#"),
					resource.TestCheckResourceAttrPair("data.gigahost_dns_records.test", "zone_id", "gigahost_dns_zone.test", "zone_id"),
					resource.TestCheckTypeSetElemNestedAttrs("data.gigahost_dns_records.test", "records.*", map[string]string{
						"record_name":  "datasourcetest",
						"record_type":  "A",
						"record_value": "203.0.113.30",
						"record_ttl":   "3600",
					}),
				),
			},
		},
	})
}

func testAccDNSRecordsDataSourceConfig(zoneName string) string {
	return fmt.Sprintf(`
resource "gigahost_dns_zone" "test" {
  zone_name = %q
}

resource "gigahost_dns_record" "test" {
  zone_id      = gigahost_dns_zone.test.zone_id
  record_name  = "datasourcetest"
  record_type  = "A"
  record_value = "203.0.113.30"
}

data "gigahost_dns_records" "test" {
  zone_id    = gigahost_dns_zone.test.zone_id
  depends_on = [gigahost_dns_record.test]
}
`, zoneName)
}
