package provider

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/joakimhellum/terraform-provider-gigahost/internal/client"
)

func TestAccDNSRecordResource_basic(t *testing.T) {
	zoneName := testAccZoneName()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDNSRecordDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDNSRecordResourceConfig(zoneName, "203.0.113.10"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("gigahost_dns_record.test", "record_name", "test"),
					resource.TestCheckResourceAttr("gigahost_dns_record.test", "record_type", "A"),
					resource.TestCheckResourceAttr("gigahost_dns_record.test", "record_value", "203.0.113.10"),
					resource.TestCheckResourceAttr("gigahost_dns_record.test", "record_ttl", "3600"),
					resource.TestCheckResourceAttrSet("gigahost_dns_record.test", "record_id"),
					resource.TestCheckResourceAttrPair("gigahost_dns_record.test", "zone_id", "gigahost_dns_zone.test", "zone_id"),
				),
			},
			{
				Config: testAccDNSRecordResourceConfig(zoneName, "203.0.113.20"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("gigahost_dns_record.test", "record_value", "203.0.113.20"),
					resource.TestCheckResourceAttrSet("gigahost_dns_record.test", "record_id"),
				),
			},
			{
				ResourceName:                         "gigahost_dns_record.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "record_id",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["gigahost_dns_record.test"]
					if !ok {
						return "", fmt.Errorf("resource not found in state")
					}
					return rs.Primary.Attributes["zone_id"] + "/" + rs.Primary.Attributes["record_id"], nil
				},
			},
		},
	})
}

func testAccDNSRecordResourceConfig(zoneName, value string) string {
	return fmt.Sprintf(`
resource "gigahost_dns_zone" "test" {
  zone_name = %q
}

resource "gigahost_dns_record" "test" {
  zone_id      = gigahost_dns_zone.test.zone_id
  record_name  = "test"
  record_type  = "A"
  record_value = %q
}
`, zoneName, value)
}

func testAccCheckDNSRecordDestroy(s *terraform.State) error {
	c, err := client.NewClient(&client.Config{
		Address: os.Getenv("GIGAHOST_BASE_URL"),
		Token:   os.Getenv("GIGAHOST_API_TOKEN"),
	})
	if err != nil {
		return err
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gigahost_dns_record" {
			continue
		}
		record, err := c.GetRecord(context.Background(), rs.Primary.Attributes["zone_id"], rs.Primary.Attributes["record_id"])
		if err != nil {
			// The parent zone is destroyed alongside the record, so listing its
			// records may fail; that still means the record is gone.
			continue
		}
		if record != nil {
			return fmt.Errorf("dns record %s still exists", rs.Primary.Attributes["record_id"])
		}
	}
	return nil
}
