resource "gigahost_dns_zone_ds_records" "example" {
  zone_id = gigahost_dns_zone.example.zone_id

  ds_records = [
    {
      key_tag     = 12345
      algorithm   = 13
      digest_type = 2
      digest      = "1234567890ABCDEF1234567890ABCDEF1234567890ABCDEF1234567890ABCDEF"
    },
  ]

  depends_on = [gigahost_dns_zone_dnssec.example]
}
