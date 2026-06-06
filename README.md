# Terraform Provider for Gigahost

![Gigahost](https://gigahost.no/en/img/header/gigahost_logo_website.png)

A [Terraform](https://www.terraform.io) provider for [Gigahost](https://gigahost.no)
built on the [Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework)
and the [Gigahost API](https://gigahost.no/en/api-dokumentasjon).

> **Status:** early development.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://go.dev/doc/install) >= 1.25 (to build the provider)

## Using the provider

Configure the provider with a Gigahost API token (created under **Account → API
keys**). Prefer the `GIGAHOST_API_TOKEN` environment variable over hardcoding the
token in configuration.

```terraform
terraform {
  required_providers {
    gigahost = {
      source = "pigeon-as/gigahost"
    }
  }
}

provider "gigahost" {
  # api_token = "flux_live_..."   # or set GIGAHOST_API_TOKEN
}

data "gigahost_account" "current" {}

output "account_name" {
  value = data.gigahost_account.current.cust_name
}
```

See the [`examples/`](./examples) directory and the generated [`docs/`](./docs)
for full reference documentation.

## Developing the provider

Common tasks are wired into the [`GNUmakefile`](./GNUmakefile):

```shell
make build      # go build ./...
make install    # go install ./...
make test       # unit tests
make testacc    # acceptance tests (TF_ACC=1; needs GIGAHOST_API_TOKEN)
make generate   # regenerate code (codegen) and docs
make lint       # golangci-lint
make fmt        # gofmt
```

Acceptance tests run against the real Gigahost API and require `GIGAHOST_API_TOKEN`.

## License

[MPL-2.0](./LICENSE)
