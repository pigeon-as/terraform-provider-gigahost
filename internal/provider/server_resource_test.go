package provider

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func testAccServerProduct() string {
	if v := os.Getenv("GIGAHOST_TEST_PRODUCT"); v != "" {
		return v
	}
	return "KVM Performance VPS 4GB"
}

func testAccServerResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "gigahost_server" "test" {
  product_name = %q
  region_name  = "Sandefjord"
  rescue       = true
  srv_name     = %q
  timeouts     = { create = "15m" }
}
`, testAccServerProduct(), name)
}

func testAccServerResourceOSConfig(name string) string {
	return fmt.Sprintf(`
resource "gigahost_server" "test" {
  product_name = %q
  region_name  = "Sandefjord"
  os_name      = "Ubuntu 24.04 LTS"
  srv_name     = %q
  timeouts     = { create = "15m" }
}
`, testAccServerProduct(), name)
}

func TestAccServerResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckServerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccServerResourceConfig("tf-acc-server"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("gigahost_server.test", "product_id"),
					resource.TestCheckResourceAttrSet("gigahost_server.test", "price_id"),
					resource.TestCheckResourceAttrSet("gigahost_server.test", "region_id"),
					resource.TestCheckResourceAttrSet("gigahost_server.test", "srv_id"),
					resource.TestCheckResourceAttrSet("gigahost_server.test", "order_id"),
					resource.TestCheckResourceAttrSet("gigahost_server.test", "srv_primary_ip"),
					resource.TestCheckResourceAttrSet("gigahost_server.test", "rate_hourly"),
					resource.TestCheckResourceAttrSet("gigahost_server.test", "monthly_cap"),
					resource.TestCheckResourceAttr("gigahost_server.test", "srv_name", "tf-acc-server"),
				),
			},
			{
				Config: testAccServerResourceConfig("tf-acc-renamed"),
				Check:  resource.TestCheckResourceAttr("gigahost_server.test", "srv_name", "tf-acc-renamed"),
			},
			{
				ResourceName:                         "gigahost_server.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "srv_id",
				ImportStateIdFunc:                    testAccServerImportID,
				ImportStateVerifyIgnore: []string{
					"srv_name", "password", "price_id", "order_id",
					"order_number", "rate_hourly", "monthly_cap", "currency", "ips",
				},
			},
		},
	})
}

func testAccServerImportID(s *terraform.State) (string, error) {
	rs, ok := s.RootModule().Resources["gigahost_server.test"]
	if !ok {
		return "", fmt.Errorf("resource not found in state")
	}
	return rs.Primary.Attributes["srv_id"], nil
}

func TestAccServerResource_osInstall(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckServerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccServerResourceOSConfig("tf-acc-server-os"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("gigahost_server.test", "os_id"),
					resource.TestCheckResourceAttrSet("gigahost_server.test", "srv_id"),
					resource.TestCheckResourceAttrSet("gigahost_server.test", "srv_primary_ip"),
					resource.TestCheckResourceAttr("gigahost_server.test", "os_name", "Ubuntu 24.04 LTS"),
					resource.TestCheckResourceAttr("gigahost_server.test", "srv_name", "tf-acc-server-os"),
				),
			},
			{
				ResourceName:                         "gigahost_server.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "srv_id",
				ImportStateIdFunc:                    testAccServerImportID,
				ImportStateVerifyIgnore: []string{
					"srv_name", "os_name", "os_dist", "password",
					"price_id", "order_id", "order_number", "rate_hourly", "monthly_cap", "currency",
					"ips", "os", "srv_status_install", "srv_status",
				},
			},
		},
	})
}

func testAccServerResourceBareMetalConfig(name string) string {
	return fmt.Sprintf(`
resource "gigahost_server" "test" {
  product_name = "Intro - Intel Core i3 4GB"
  region_name  = "Sandefjord"
  rescue       = true
  srv_name     = %q
  timeouts     = { create = "15m" }
}
`, name)
}

func TestAccServerResource_bareMetal(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckServerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccServerResourceBareMetalConfig("tf-acc-baremetal"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("gigahost_server.test", "srv_id"),
					resource.TestCheckResourceAttrSet("gigahost_server.test", "srv_primary_ip"),
					resource.TestCheckResourceAttrSet("gigahost_server.test", "product_id"),
					resource.TestCheckResourceAttrSet("gigahost_server.test", "srv_type"),
				),
			},
		},
	})
}

func testAccCheckServerDestroy(s *terraform.State) error {
	c, err := sweepClient()
	if err != nil {
		return err
	}

	servers, err := c.ListServers(context.Background())
	if err != nil {
		return err
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gigahost_server" {
			continue
		}
		id := rs.Primary.Attributes["srv_id"]
		for _, srv := range servers {
			if srv.SrvID != id {
				continue
			}
			if !strings.EqualFold(srv.Order.OrderStatus, "cancelled") {
				return fmt.Errorf("server %s still present (order_status=%q)", id, srv.Order.OrderStatus)
			}
		}
	}
	return nil
}
