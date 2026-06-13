# A KVM virtual machine running Ubuntu.
resource "gigahost_server" "example" {
  product_name = "KVM Value VPS 4GB"
  region_name  = "Sandefjord"
  os_name      = "Ubuntu 24.04 LTS"
  srv_name     = "web-01"
}
