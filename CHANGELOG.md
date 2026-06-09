## 0.2.1 (June 9, 2026)

ENHANCEMENTS:

* `gigahost_server` - expose `ips`, `os`, `cores`, `ram`, and additional server details as read-only attributes.
* `gigahost_server` - support a configurable `create` timeout via a `timeouts` block.
* `gigahost_server` data source - add `order` and `datacenter` (parity with the `gigahost_servers` data source).

BUG FIXES:

* `gigahost_server` - `terraform validate` no longer reports a spurious OS/rescue configuration error when `os_distro`, `os_version`, or `rescue` are set from variables.
* `gigahost_server` - reordering `ssh_keys` no longer forces the server to be replaced (it is now a set).

## 0.2.0 (June 8, 2026)

FEATURES:

* **New Resource:** `gigahost_server`
* **New Data Source:** `gigahost_dns_zone`
* **New Data Source:** `gigahost_os`
* **New Data Source:** `gigahost_server_catalog`
* **New Data Source:** `gigahost_server`
* **New Data Source:** `gigahost_servers`

ENHANCEMENTS:

* client: surface the API `meta.error` field in error messages.

## 0.1.0 (June 6, 2026)

FEATURES:

* **New Resource:** `gigahost_dns_record`
* **New Resource:** `gigahost_dns_redirect`
* **New Resource:** `gigahost_dns_zone`
* **New Resource:** `gigahost_ssh_key`
* **New Data Source:** `gigahost_account`
* **New Data Source:** `gigahost_dns_records`
* **New Data Source:** `gigahost_dns_zones`
* **New Data Source:** `gigahost_ssh_keys`
