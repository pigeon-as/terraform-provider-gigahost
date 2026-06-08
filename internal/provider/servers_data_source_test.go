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
  rescue       = true
  name         = "tf-acc-servers-ds"
}

data "gigahost_servers" "all" {
  depends_on = [gigahost_server.test]
}

data "gigahost_server" "test" {
  srv_id     = gigahost_server.test.server_id
  depends_on = [gigahost_server.test]
}
`
}

func TestAccServersDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccServersDataSourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// gigahost_servers (list) returns at least one server.
					resource.TestMatchResourceAttr("data.gigahost_servers.all", "servers.#", regexp.MustCompile(`^[1-9]`)),
					// gigahost_server (singular) found the deployed server and decoded its fields.
					resource.TestCheckResourceAttrPair("data.gigahost_server.test", "srv_id", "gigahost_server.test", "server_id"),
					resource.TestCheckResourceAttrPair("data.gigahost_server.test", "srv_primary_ip", "gigahost_server.test", "ipv4"),
					resource.TestCheckResourceAttrSet("data.gigahost_server.test", "srv_type"),
					resource.TestCheckResourceAttrSet("data.gigahost_server.test", "srv_name"),
					resource.TestMatchResourceAttr("data.gigahost_server.test", "ips.#", regexp.MustCompile(`^[1-9]`)),
				),
			},
		},
	})
}
