# A dedicated (bare metal) server running Ubuntu.
resource "gigahost_server" "example" {
  product_name = "Intro - Intel Core i3 4GB"
  region_name  = "Sandefjord"
  os_name      = "Ubuntu 24.04 LTS"
  srv_name     = "db-01"
}
