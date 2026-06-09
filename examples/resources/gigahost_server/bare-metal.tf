# A dedicated (bare metal) server running Ubuntu.
resource "gigahost_server" "example" {
  product_name = "Intro - Intel Core i3 4GB"
  region       = "Sandefjord"
  os_distro    = "Ubuntu"
  os_version   = "24.04"
  name         = "db-01"
}
