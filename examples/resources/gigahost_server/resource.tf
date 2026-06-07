resource "gigahost_server" "example" {
  name         = "web-01"
  product_name = "KVM Value VPS 4GB"
  region       = "Sandefjord"
  os_distro    = "Ubuntu"
  os_version   = "24.04"
}
