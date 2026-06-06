# List the DNS zones on the Gigahost account.
data "gigahost_dns_zones" "all" {}

output "dns_zones" {
  value = data.gigahost_dns_zones.all.zones
}
