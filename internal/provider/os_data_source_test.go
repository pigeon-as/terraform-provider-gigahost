package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccOSDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "gigahost_os" "test" {
  distro  = "Ubuntu"
  version = "24.04"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.gigahost_os.test", "os_id"),
					resource.TestCheckResourceAttr("data.gigahost_os.test", "os_name", "Ubuntu 24.04 LTS"),
					resource.TestCheckResourceAttr("data.gigahost_os.test", "os_dist", "noble"),
				),
			},
		},
	})
}
