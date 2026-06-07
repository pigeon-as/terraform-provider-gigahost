data "gigahost_account" "current" {}

output "account_name" {
  value = data.gigahost_account.current.cust_name
}
