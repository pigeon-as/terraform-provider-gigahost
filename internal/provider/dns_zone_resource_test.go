package provider

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/pigeon-as/terraform-provider-gigahost/internal/client"
)

func testAccZoneName() string {
	domain := os.Getenv("GIGAHOST_TEST_DOMAIN")
	if domain == "" {
		domain = "gigahost-tf-acceptance-tests.com"
	}
	return acctest.RandomWithPrefix("tf-acc") + "." + domain
}

func TestAccDNSZoneResource_basic(t *testing.T) {
	zoneName := testAccZoneName()
	renamedZone := testAccZoneName()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDNSZoneDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDNSZoneResourceConfig(zoneName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("gigahost_dns_zone.test", "zone_name", zoneName),
					resource.TestCheckResourceAttr("gigahost_dns_zone.test", "zone_type", "NATIVE"),
					resource.TestCheckResourceAttrSet("gigahost_dns_zone.test", "zone_id"),
					resource.TestCheckResourceAttrSet("gigahost_dns_zone.test", "zone_active"),
				),
			},
			{
				Config: testAccDNSZoneResourceConfig(renamedZone),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("gigahost_dns_zone.test", plancheck.ResourceActionReplace),
					},
				},
				Check: resource.TestCheckResourceAttr("gigahost_dns_zone.test", "zone_name", renamedZone),
			},
			{
				ResourceName:                         "gigahost_dns_zone.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "zone_id",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["gigahost_dns_zone.test"]
					if !ok {
						return "", fmt.Errorf("resource not found in state")
					}
					return rs.Primary.Attributes["zone_id"], nil
				},
			},
		},
	})
}

func testAccDNSZoneResourceConfig(zoneName string) string {
	return fmt.Sprintf(`
resource "gigahost_dns_zone" "test" {
  zone_name = %q
}
`, zoneName)
}

func testAccCheckDNSZoneDestroy(s *terraform.State) error {
	c, err := client.NewClient(&client.Config{
		Address: os.Getenv("GIGAHOST_BASE_URL"),
		Token:   os.Getenv("GIGAHOST_API_TOKEN"),
	})
	if err != nil {
		return err
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gigahost_dns_zone" {
			continue
		}
		_, err := c.GetZone(context.Background(), rs.Primary.Attributes["zone_id"])
		if err == nil {
			return fmt.Errorf("dns zone %s still exists", rs.Primary.Attributes["zone_id"])
		}
		if !errors.Is(err, client.ErrNotFound) {
			return err
		}
	}
	return nil
}
