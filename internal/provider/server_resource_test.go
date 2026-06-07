package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
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

func TestAccServerResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccServerResourceConfig("tf-acc-server"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("gigahost_server.test", "product_id", "7955"),
					resource.TestCheckResourceAttr("gigahost_server.test", "price_id", "4054"),
					resource.TestCheckResourceAttr("gigahost_server.test", "region_id", "1"),
					resource.TestCheckResourceAttrSet("gigahost_server.test", "server_id"),
					resource.TestCheckResourceAttrSet("gigahost_server.test", "ipv4"),
					resource.TestCheckResourceAttr("gigahost_server.test", "name", "tf-acc-server"),
				),
			},
			{
				Config: testAccServerResourceConfig("tf-acc-renamed"),
				Check:  resource.TestCheckResourceAttr("gigahost_server.test", "name", "tf-acc-renamed"),
			},
		},
	})
}
