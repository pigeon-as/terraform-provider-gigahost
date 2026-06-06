# Manage a DNS zone (DNS hosting for a domain).
resource "gigahost_dns_zone" "example" {
  zone_name = "example.com"
}
