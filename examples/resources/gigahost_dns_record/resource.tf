# Manage a DNS record within a zone.
resource "gigahost_dns_zone" "example" {
  zone_name = "example.com"
}

resource "gigahost_dns_record" "example" {
  zone_id      = gigahost_dns_zone.example.zone_id
  record_name  = "www"
  record_type  = "A"
  record_value = "203.0.113.10"
}
