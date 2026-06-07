package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccServerCatalogDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "gigahost_server_catalog" "test" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.gigahost_server_catalog.test", "currency", "NOK"),
					resource.TestCheckResourceAttrSet("data.gigahost_server_catalog.test", "tiers.0.group_name"),
					resource.TestCheckResourceAttrSet("data.gigahost_server_catalog.test", "tiers.0.products.0.product_id"),
					resource.TestCheckResourceAttrSet("data.gigahost_server_catalog.test", "tiers.0.products.0.rate_hourly"),
					resource.TestCheckResourceAttrSet("data.gigahost_server_catalog.test", "regions.0.region_id"),
				),
			},
		},
	})
}
