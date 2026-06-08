package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testAccServersDataSourceConfig() string {
	return `
resource "gigahost_server" "test" {
  product_name = "KVM Value VPS 4GB"
  region       = "Sandefjord"
  os_distro    = "Ubuntu"
  os_version   = "24.04"
  name         = "tf-acc-servers-ds"
}

data "gigahost_servers" "all" {
  depends_on = [gigahost_server.test]
}

data "gigahost_server" "by_id" {
  srv_id     = gigahost_server.test.server_id
  depends_on = [gigahost_server.test]
}

data "gigahost_server" "by_name" {
  srv_name   = gigahost_server.test.name
  depends_on = [gigahost_server.test]
}
`
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
					resource.TestCheckResourceAttrPair("data.gigahost_server.by_id", "srv_id", "gigahost_server.test", "server_id"),
					resource.TestCheckResourceAttrPair("data.gigahost_server.by_id", "srv_primary_ip", "gigahost_server.test", "ipv4"),
					resource.TestCheckResourceAttrSet("data.gigahost_server.by_id", "srv_type"),
					resource.TestCheckResourceAttrSet("data.gigahost_server.by_id", "srv_cores"),
					resource.TestCheckResourceAttrSet("data.gigahost_server.by_id", "srv_ram"),
					resource.TestCheckResourceAttrSet("data.gigahost_server.by_id", "os.os_id"),
					resource.TestMatchResourceAttr("data.gigahost_server.by_id", "ips.#", regexp.MustCompile(`^[1-9]`)),
					resource.TestCheckResourceAttrSet("data.gigahost_server.by_id", "ips.0.ip_address"),
					resource.TestCheckResourceAttrPair("data.gigahost_server.by_name", "srv_id", "gigahost_server.test", "server_id"),
					resource.TestCheckResourceAttr("data.gigahost_server.by_name", "srv_name", "tf-acc-servers-ds"),
				),
			},
		},
	})
}
