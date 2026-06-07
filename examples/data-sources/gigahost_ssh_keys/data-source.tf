data "gigahost_ssh_keys" "all" {}

output "ssh_keys" {
  value = data.gigahost_ssh_keys.all.ssh_keys
}
