# A dedicated (bare metal) server with an SSH key, daily backups, and a
# longer create timeout for the slower hardware install.
resource "gigahost_ssh_key" "example" {
  key_name = "deploy"
  key_data = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAID... user@example.com"
}

resource "gigahost_server" "example" {
  product_name = "Intro - Intel Core i3 4GB"
  region       = "Sandefjord"
  os_distro    = "Ubuntu"
  os_version   = "24.04"
  backups      = true
  ssh_keys     = [gigahost_ssh_key.example.key_id]

  timeouts = { create = "60m" }
}
