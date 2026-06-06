# List the DNS records in a DNS zone.
data "gigahost_dns_records" "example" {
  zone_id = "2826"
}
