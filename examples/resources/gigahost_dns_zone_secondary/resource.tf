resource "gigahost_dns_zone_secondary" "example" {
  zone_name = "example.com"
  masters   = ["192.0.2.1", "192.0.2.2"]
}
