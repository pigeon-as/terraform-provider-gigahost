resource "gigahost_dns_zone_nameservers" "example" {
  zone_id = gigahost_dns_zone.example.zone_id
  nameservers = [
    "ns1.example.net",
    "ns2.example.net",
  ]
}
