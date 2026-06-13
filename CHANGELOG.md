## 0.5.0 (June 13, 2026)

BREAKING CHANGES:

* `gigahost_server` - every attribute is renamed to its exact Gigahost API field, removing provider-invented names so the resource mirrors the API (and the `gigahost_server` data source). Inputs: `os_distro`/`os_version` → `os_name` and `os_dist` (the catalog name or release codename, e.g. `"Ubuntu 24.04 LTS"` or `"noble"`; provide exactly one), each matched exactly; `region` → `region_name`; `name` → `srv_name`. Computed: `server_id` → `srv_id`, `ipv4` → `srv_primary_ip`, `root_password` → `password`, and `cores`/`ram`/`location`/`type`/`vps_type`/`running`/`installing`/`suspended` → `srv_cores`/`srv_ram`/`srv_location`/`srv_type`/`srv_vps_type`/`srv_status`/`srv_status_install`/`srv_suspended`. The top-level `ipv6` attribute is removed — read IPv6 from `ips` (the entry whose `ip_v4v6` is `"ipv6"`).
* `gigahost_os` data source - the `distro` and `version` filters are replaced by `os_name` and `os_dist` (provide exactly one), each matched exactly against the catalog.

ENHANCEMENTS:

* `gigahost_server` - `terraform plan` no longer makes catalog or OS API calls; product, region and OS names are resolved during apply, and changing the OS forces replacement directly. Plans now work without network access to the API.
* `gigahost_server` - the deploy wait now polls only the deploy status endpoint, following the standard provider pattern: a deploy that never reaches a final status (including one the status API stops reporting) surfaces as a create timeout, which the `timeouts.create` setting controls.
* `gigahost_server` - an absent primary IP is stored as null.

BUG FIXES:

* `gigahost_server` - removing `srv_name` from the configuration no longer renames the server to an empty string.
* `gigahost_server` - the root password reported while a server installs is kept when later deploy-status polls no longer carry it.
* `gigahost_server` - the client rejects empty or non-numeric server ids, which previously turned a malformed id lookup into a read of the whole server list that could answer with another server's details.
* `gigahost_server` - post-deploy details are read directly by id, closing a window where a transient server-list gap left them silently empty.

## 0.4.0 (June 12, 2026)

FEATURES:

* `gigahost_server` data source - expose single-server details from `GET /servers/{id}`: `srv_date_created`, `srv_bw`, `srv_bw_type`, and `hdds`.

ENHANCEMENTS:

* `gigahost_server` data source - id lookups use the single-server endpoint directly, and name lookups fetch the full details after resolving the id.
* `gigahost_server` - refresh reads the server directly by id: a 404 definitively means the server is gone, a key lacking the permission surfaces the API's 403 message, and a 401 (bad token) keeps the state untouched.
* `gigahost_server` - deploy waits read the server directly once its id is known, a disappeared server fails the create after about a minute instead of five, and failure messages include the observed server id.

## 0.3.2 (June 11, 2026)

DEPRECATIONS:

* `gigahost_server` and `gigahost_servers` data sources - `srv_hostname` is deprecated: the API does not populate it (the requested deploy hostname is recorded in `srv_name`).

BUG FIXES:

* `gigahost_server` - a transient gap in the server list no longer removes a live server from state: absence is confirmed across repeated reads before the resource is treated as deleted.
* `gigahost_server` - destroying a server that died during provisioning no longer fails forever: the API refuses cancellation of nonexistent servers with a 400, so a refused cancellation is followed by an absence check, and a confirmed-gone server is cleared from state with a warning naming the order.

ENHANCEMENTS:

* `gigahost_server` - document that a requested `hostname` is recorded as the server name (`srv_name`, replaced by `name` when both are set) and is not separately readable; the server data sources' `srv_name`/`srv_hostname` descriptions now reflect this.

## 0.3.1 (June 11, 2026)

BUG FIXES:

* `gigahost_server` - a deploy that fails or times out after the order is placed no longer orphans the billed server: the resource is saved to state as tainted, `terraform destroy` cancels it, and refresh adopts a server that only appears later by its deployment order.
* `gigahost_server` - deploy waits now follow the deploy status view's real lifecycle (orders are only listed while their server exists, and there is no failure status): an order missing from the status is tolerated, the server list is polled as the durable completion source after a short grace, a finished install there completes the create, a server that disappears from both views fails fast instead of waiting out the timeout, and any observed server id is kept so a failed create can still be destroyed.
* `gigahost_server` - `ipv6` no longer flips to null on refresh: the address reported at deploy time is kept when the server list does not expose it, the server list takes precedence when it does, and an absent address is stored as null instead of an empty string.
* `gigahost_server` - a failure to read the server's details right after deploy now fails the create with the server kept in state as tainted, instead of being silently ignored.

ENHANCEMENTS:

* `gigahost_server` - document import, including the `ssh_keys` limitation (the API does not return deployed keys, so declaring them for an imported server forces replacement).

## 0.3.0 (June 10, 2026)

BREAKING CHANGES:

* `gigahost_server` - `ssh_keys` ids are now strings (e.g. `["123"]`, matching `gigahost_ssh_key.key_id`) rather than numbers (`[123]`).

ENHANCEMENTS:

* `gigahost_server_catalog` - add per-product `type` (vm, dedicated, or auction).

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
