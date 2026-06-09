package provider

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func testAccServerResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "gigahost_server" "test" {
  product_name = "KVM Value VPS 4GB"
  region       = "Sandefjord"
  rescue       = true
  name         = %q
}
`, name)
}

func testAccServerResourceOSConfig(name string) string {
	return fmt.Sprintf(`
resource "gigahost_server" "test" {
  product_name = "KVM Value VPS 4GB"
  region       = "Sandefjord"
  os_distro    = "Ubuntu"
  os_version   = "24.04"
  name         = %q
}
`, name)
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
					resource.TestCheckResourceAttrSet("gigahost_server.test", "server_id"),
					resource.TestCheckResourceAttrSet("gigahost_server.test", "order_id"),
					resource.TestCheckResourceAttrSet("gigahost_server.test", "ipv4"),
					resource.TestCheckResourceAttrSet("gigahost_server.test", "rate_hourly"),
					resource.TestCheckResourceAttrSet("gigahost_server.test", "monthly_cap"),
					resource.TestCheckResourceAttr("gigahost_server.test", "name", "tf-acc-server"),
				),
			},
			{
				Config: testAccServerResourceConfig("tf-acc-renamed"),
				Check:  resource.TestCheckResourceAttr("gigahost_server.test", "name", "tf-acc-renamed"),
			},
			{
				ResourceName:                         "gigahost_server.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "server_id",
				ImportStateIdFunc:                    testAccServerImportID,
				ImportStateVerifyIgnore: []string{
					"name", "ipv6", "root_password", "price_id", "order_id",
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
	return rs.Primary.Attributes["server_id"], nil
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
					resource.TestCheckResourceAttrSet("gigahost_server.test", "server_id"),
					resource.TestCheckResourceAttrSet("gigahost_server.test", "ipv4"),
					resource.TestCheckResourceAttr("gigahost_server.test", "os_distro", "Ubuntu"),
					resource.TestCheckResourceAttr("gigahost_server.test", "os_version", "24.04"),
					resource.TestCheckResourceAttr("gigahost_server.test", "name", "tf-acc-server-os"),
				),
			},
			{
				ResourceName:                         "gigahost_server.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "server_id",
				ImportStateIdFunc:                    testAccServerImportID,
				ImportStateVerifyIgnore: []string{
					"name", "ipv6", "os_distro", "os_version", "root_password",
					"price_id", "order_id", "order_number", "rate_hourly", "monthly_cap", "currency",
					"ips", "os", "installing", "running",
				},
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
		id := rs.Primary.Attributes["server_id"]
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
