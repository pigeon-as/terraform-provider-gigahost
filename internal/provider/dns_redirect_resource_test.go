package provider

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/pigeon-as/terraform-provider-gigahost/internal/client"
)

func testAccDNSRedirectConfig(zoneName, source, targetURL string) string {
	return fmt.Sprintf(`
resource "gigahost_dns_zone" "test" {
  zone_name = %q
}

resource "gigahost_dns_redirect" "test" {
  zone_id    = gigahost_dns_zone.test.zone_id
  source     = %q
  target_url = %q
}
`, zoneName, source, targetURL)
}

func testAccDNSRedirectImportID(s *terraform.State) (string, error) {
	rs, ok := s.RootModule().Resources["gigahost_dns_redirect.test"]
	if !ok {
		return "", fmt.Errorf("resource not found in state")
	}
	return rs.Primary.Attributes["zone_id"] + "/" + rs.Primary.Attributes["source"], nil
}

func TestAccDNSRedirectResource_basic(t *testing.T) {
	zoneName := testAccZoneName()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDNSRedirectDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDNSRedirectConfig(zoneName, "@", "https://example.com/one"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("gigahost_dns_redirect.test", "source", "@"),
					resource.TestCheckResourceAttr("gigahost_dns_redirect.test", "target_url", "https://example.com/one"),
					resource.TestCheckResourceAttr("gigahost_dns_redirect.test", "enabled", "true"),
					resource.TestCheckResourceAttrSet("gigahost_dns_redirect.test", "zone_id"),
				),
			},
			{
				Config: testAccDNSRedirectConfig(zoneName, "@", "https://example.com/updated"),
				Check:  resource.TestCheckResourceAttr("gigahost_dns_redirect.test", "target_url", "https://example.com/updated"),
			},
			{
				Config: testAccDNSRedirectConfig(zoneName, "blog", "https://example.com/updated"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("gigahost_dns_redirect.test", plancheck.ResourceActionReplace),
					},
				},
				Check: resource.TestCheckResourceAttr("gigahost_dns_redirect.test", "source", "blog"),
			},
			{
				ResourceName:                         "gigahost_dns_redirect.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "source",
				ImportStateIdFunc:                    testAccDNSRedirectImportID,
			},
		},
	})
}

func testAccCheckDNSRedirectDestroy(s *terraform.State) error {
	c, err := sweepClient()
	if err != nil {
		return err
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gigahost_dns_redirect" {
			continue
		}
		_, err := c.GetRedirect(context.Background(), rs.Primary.Attributes["zone_id"], rs.Primary.Attributes["source"])
		if err == nil {
			return fmt.Errorf("dns redirect %s still exists", rs.Primary.Attributes["source"])
		}
		if !errors.Is(err, client.ErrNotFound) {
			return err
		}
	}
	return nil
}
