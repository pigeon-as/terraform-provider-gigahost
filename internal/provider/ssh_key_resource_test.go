package provider

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"golang.org/x/crypto/ssh"

	"github.com/pigeon-as/terraform-provider-gigahost/internal/client"
)

func testAccSSHPublicKey(t *testing.T) string {
	t.Helper()

	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sshPub)))
}

func TestAccSSHKeyResource_basic(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc")
	publicKey := testAccSSHPublicKey(t)
	rotatedKey := testAccSSHPublicKey(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckSSHKeyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccSSHKeyResourceConfig(name, publicKey),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("gigahost_ssh_key.test", "key_name", name),
					resource.TestCheckResourceAttr("gigahost_ssh_key.test", "key_data", publicKey),
					resource.TestCheckResourceAttrSet("gigahost_ssh_key.test", "key_id"),
					resource.TestCheckResourceAttrSet("gigahost_ssh_key.test", "key_added"),
				),
			},
			{
				Config: testAccSSHKeyResourceConfig(name, rotatedKey),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("gigahost_ssh_key.test", plancheck.ResourceActionReplace),
					},
				},
				Check: resource.TestCheckResourceAttr("gigahost_ssh_key.test", "key_data", rotatedKey),
			},
			{
				ResourceName:                         "gigahost_ssh_key.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "key_id",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["gigahost_ssh_key.test"]
					if !ok {
						return "", fmt.Errorf("resource not found in state")
					}
					return rs.Primary.Attributes["key_id"], nil
				},
			},
		},
	})
}

func testAccSSHKeyResourceConfig(name, publicKey string) string {
	return fmt.Sprintf(`
resource "gigahost_ssh_key" "test" {
  key_name = %q
  key_data = %q
}
`, name, publicKey)
}

func testAccCheckSSHKeyDestroy(s *terraform.State) error {
	c, err := client.NewClient(&client.Config{
		Address: os.Getenv("GIGAHOST_BASE_URL"),
		Token:   os.Getenv("GIGAHOST_API_TOKEN"),
	})
	if err != nil {
		return err
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gigahost_ssh_key" {
			continue
		}
		_, err := c.GetSSHKey(context.Background(), rs.Primary.Attributes["key_id"])
		if err == nil {
			return fmt.Errorf("ssh key %s still exists", rs.Primary.Attributes["key_id"])
		}
		if !errors.Is(err, client.ErrNotFound) {
			return err
		}
	}
	return nil
}
