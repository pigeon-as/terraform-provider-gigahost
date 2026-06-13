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
				Config: `data "gigahost_os" "by_name" {
  os_name = "Ubuntu 24.04 LTS"
}

data "gigahost_os" "by_codename" {
  os_dist = "noble"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.gigahost_os.by_name", "os_id"),
					resource.TestCheckResourceAttr("data.gigahost_os.by_name", "os_dist", "noble"),
					resource.TestCheckResourceAttrSet("data.gigahost_os.by_codename", "os_id"),
					resource.TestCheckResourceAttr("data.gigahost_os.by_codename", "os_name", "Ubuntu 24.04 LTS"),
				),
			},
		},
	})
}
