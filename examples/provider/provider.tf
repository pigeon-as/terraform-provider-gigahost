terraform {
  required_providers {
    gigahost = {
      source = "pigeon-as/gigahost"
    }
  }
}

# Configure the Gigahost provider.
provider "gigahost" {
  # API token for the Gigahost API, created under Account -> API keys.
  # Prefer setting it via the GIGAHOST_API_TOKEN environment variable over
  # hardcoding it in configuration.
  api_token = "flux_live_xxxxxxxxxxxxxxxx"
}
