package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testAccServersDataSourceConfig() string {
	return fmt.Sprintf(`
resource "gigahost_server" "test" {
  product_name = %q
  region_name  = "Sandefjord"
  os_name      = "Ubuntu 24.04 LTS"
  srv_name     = "tf-acc-servers-ds"
  timeouts     = { create = "15m" }
}

data "gigahost_servers" "all" {
  depends_on = [gigahost_server.test]
}

data "gigahost_server" "by_id" {
  srv_id     = gigahost_server.test.srv_id
  depends_on = [gigahost_server.test]
}

data "gigahost_server" "by_name" {
  srv_name   = gigahost_server.test.srv_name
  depends_on = [gigahost_server.test]
}
`, testAccServerProduct())
}

func TestAccServersDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckServerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccServersDataSourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// gigahost_servers (list) returns at least one server.
					resource.TestMatchResourceAttr("data.gigahost_servers.all", "servers.#", regexp.MustCompile(`^[1-9]`)),
					resource.TestCheckResourceAttrPair("data.gigahost_server.by_id", "srv_id", "gigahost_server.test", "srv_id"),
					resource.TestCheckResourceAttrPair("data.gigahost_server.by_id", "srv_primary_ip", "gigahost_server.test", "srv_primary_ip"),
					resource.TestCheckResourceAttrSet("data.gigahost_server.by_id", "srv_type"),
					resource.TestCheckResourceAttrSet("data.gigahost_server.by_id", "srv_cores"),
					resource.TestCheckResourceAttrSet("data.gigahost_server.by_id", "srv_ram"),
					resource.TestCheckResourceAttrSet("data.gigahost_server.by_id", "os.os_id"),
					resource.TestMatchResourceAttr("data.gigahost_server.by_id", "ips.#", regexp.MustCompile(`^[1-9]`)),
					resource.TestCheckResourceAttrSet("data.gigahost_server.by_id", "ips.0.ip_address"),
					resource.TestCheckResourceAttrSet("data.gigahost_server.by_id", "srv_date_created"),
					resource.TestCheckResourceAttrSet("data.gigahost_server.by_id", "srv_bw"),
					resource.TestMatchResourceAttr("data.gigahost_server.by_id", "hdds.#", regexp.MustCompile(`^[1-9]`)),
					resource.TestCheckResourceAttrSet("data.gigahost_server.by_id", "hdds.0.hdd_size"),
					resource.TestCheckResourceAttrPair("data.gigahost_server.by_name", "srv_id", "gigahost_server.test", "srv_id"),
					resource.TestCheckResourceAttr("data.gigahost_server.by_name", "srv_name", "tf-acc-servers-ds"),
					resource.TestCheckResourceAttrSet("data.gigahost_server.by_name", "srv_date_created"),
				),
			},
		},
	})
}
