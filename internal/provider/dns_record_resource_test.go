package provider

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/pigeon-as/terraform-provider-gigahost/internal/client"
)

func testAccDNSRecordConfig(zoneName, recordBody string) string {
	return fmt.Sprintf(`
resource "gigahost_dns_zone" "test" {
  zone_name = %q
}

resource "gigahost_dns_record" "test" {
  zone_id = gigahost_dns_zone.test.zone_id
%s
}
`, zoneName, recordBody)
}

func testAccDNSRecordImportID(s *terraform.State) (string, error) {
	rs, ok := s.RootModule().Resources["gigahost_dns_record.test"]
	if !ok {
		return "", fmt.Errorf("resource not found in state")
	}
	return rs.Primary.Attributes["zone_id"] + "/" + rs.Primary.Attributes["record_id"], nil
}

func testAccDNSRecordImportStep() resource.TestStep {
	return resource.TestStep{
		ResourceName:                         "gigahost_dns_record.test",
		ImportState:                          true,
		ImportStateVerify:                    true,
		ImportStateVerifyIdentifierAttribute: "record_id",
		ImportStateIdFunc:                    testAccDNSRecordImportID,
	}
}

func TestAccDNSRecordResource_A(t *testing.T) {
	zoneName := testAccZoneName()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDNSRecordDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDNSRecordConfig(zoneName, "  record_name  = \"a\"\n  record_type  = \"A\"\n  record_value = \"203.0.113.10\""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("gigahost_dns_record.test", "record_type", "A"),
					resource.TestCheckResourceAttr("gigahost_dns_record.test", "record_value", "203.0.113.10"),
					resource.TestCheckResourceAttr("gigahost_dns_record.test", "record_ttl", "3600"),
					resource.TestCheckResourceAttrSet("gigahost_dns_record.test", "record_id"),
				),
			},
			{
				Config: testAccDNSRecordConfig(zoneName, "  record_name  = \"a\"\n  record_type  = \"A\"\n  record_value = \"203.0.113.20\""),
				Check:  resource.TestCheckResourceAttr("gigahost_dns_record.test", "record_value", "203.0.113.20"),
			},
			testAccDNSRecordImportStep(),
		},
	})
}

func TestAccDNSRecordResource_AAAA(t *testing.T) {
	zoneName := testAccZoneName()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDNSRecordDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDNSRecordConfig(zoneName, "  record_name  = \"v6\"\n  record_type  = \"AAAA\"\n  record_value = \"2001:db8::1\""),
				Check:  resource.TestCheckResourceAttr("gigahost_dns_record.test", "record_value", "2001:db8::1"),
			},
			testAccDNSRecordImportStep(),
			{
				Config: testAccDNSRecordConfig(zoneName, "  record_name  = \"v6\"\n  record_type  = \"AAAA\"\n  record_value = \"2001:0db8:0000:0000:0000:0000:0000:0001\""),
				Check:  resource.TestCheckResourceAttr("gigahost_dns_record.test", "record_value", "2001:0db8:0000:0000:0000:0000:0000:0001"),
			},
		},
	})
}

func TestAccDNSRecordResource_CNAME(t *testing.T) {
	zoneName := testAccZoneName()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDNSRecordDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDNSRecordConfig(zoneName, "  record_name  = \"alias\"\n  record_type  = \"CNAME\"\n  record_value = \"target1.example.com\""),
				Check:  resource.TestCheckResourceAttr("gigahost_dns_record.test", "record_value", "target1.example.com"),
			},
			{
				Config: testAccDNSRecordConfig(zoneName, "  record_name  = \"alias\"\n  record_type  = \"CNAME\"\n  record_value = \"target2.example.com\""),
				Check:  resource.TestCheckResourceAttr("gigahost_dns_record.test", "record_value", "target2.example.com"),
			},
			testAccDNSRecordImportStep(),
		},
	})
}

func TestAccDNSRecordResource_MX(t *testing.T) {
	zoneName := testAccZoneName()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDNSRecordDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDNSRecordConfig(zoneName, "  record_type     = \"MX\"\n  record_value    = \"mail.example.com\"\n  record_priority = 10"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("gigahost_dns_record.test", "record_value", "mail.example.com"),
					resource.TestCheckResourceAttr("gigahost_dns_record.test", "record_priority", "10"),
				),
			},
			{
				Config: testAccDNSRecordConfig(zoneName, "  record_type     = \"MX\"\n  record_value    = \"mail.example.com\"\n  record_priority = 20"),
				Check:  resource.TestCheckResourceAttr("gigahost_dns_record.test", "record_priority", "20"),
			},
			testAccDNSRecordImportStep(),
		},
	})
}

func TestAccDNSRecordResource_TXT(t *testing.T) {
	zoneName := testAccZoneName()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDNSRecordDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDNSRecordConfig(zoneName, "  record_name  = \"txt\"\n  record_type  = \"TXT\"\n  record_value = \"v=spf1 include:example.com -all\""),
				Check:  resource.TestCheckResourceAttr("gigahost_dns_record.test", "record_value", "v=spf1 include:example.com -all"),
			},
			{
				Config: testAccDNSRecordConfig(zoneName, "  record_name  = \"txt\"\n  record_type  = \"TXT\"\n  record_value = \"v=spf1 -all\""),
				Check:  resource.TestCheckResourceAttr("gigahost_dns_record.test", "record_value", "v=spf1 -all"),
			},
			testAccDNSRecordImportStep(),
		},
	})
}

func testAccCheckDNSRecordDestroy(s *terraform.State) error {
	c, err := client.NewClient(&client.Config{
		Address: os.Getenv("GIGAHOST_BASE_URL"),
		Token:   os.Getenv("GIGAHOST_API_TOKEN"),
	})
	if err != nil {
		return err
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gigahost_dns_record" {
			continue
		}
		_, err := c.GetRecord(context.Background(), rs.Primary.Attributes["zone_id"], rs.Primary.Attributes["record_id"])
		if err == nil {
			return fmt.Errorf("dns record %s still exists", rs.Primary.Attributes["record_id"])
		}
		if !errors.Is(err, client.ErrNotFound) {
			return err
		}
	}
	return nil
}
