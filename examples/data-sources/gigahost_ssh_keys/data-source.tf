# List the SSH keys registered on the Gigahost account.
data "gigahost_ssh_keys" "all" {}

output "ssh_keys" {
  value = data.gigahost_ssh_keys.all.ssh_keys
}
