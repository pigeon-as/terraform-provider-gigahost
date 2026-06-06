# Redirect a hostname in a zone to another URL.
resource "gigahost_dns_zone" "example" {
  zone_name = "example.com"
}

resource "gigahost_dns_redirect" "example" {
  zone_id    = gigahost_dns_zone.example.zone_id
  source     = "www"
  target_url = "https://example.com"
}
