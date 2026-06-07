resource "gigahost_ssh_key" "example" {
  key_name = "laptop"
  key_data = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIH0Ej3qkY8sJp7m5n9bExampleKeyMaterialOnly user@laptop"
}
