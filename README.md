# Terraform Provider for VictoriaMetrics Cloud

Manage [VictoriaMetrics Cloud](https://victoriametrics.cloud/) deployments, access tokens, rule files, and notifiers with Terraform. 
This provider wraps the [VictoriaMetrics Cloud API](https://docs.victoriametrics.com/victoriametrics-cloud/api/) and exposes both resources and data sources so you can manage infrastructure as code alongside the rest of your stack.

## Getting Started

- Add the provider to your configuration:

```hcl
terraform {
  required_providers {
    victoriametricscloud = {
      source  = "VictoriaMetrics/victoriametricscloud"
      version = "0.1.0"
    }
  }
}

provider "victoriametricscloud" {
  api_key = var.api_key
}
```

- Save your VictoriaMetrics Cloud API key in Terraform locals, variables, or use the `VMCLOUD_API_KEY` environment variable.
- Initialize and apply as usual: `terraform init && terraform apply`.

## Authentication & Configuration

- `api_key` – required unless `VMCLOUD_API_KEY` is set. Marked sensitive inside Terraform state.

## Supported Resources
<!-- resources:start -->
| Resource | Purpose |
|---|---|
| `victoriametricscloud_access_token` | Manages an access token for a VictoriaMetrics Cloud deployment. |
| `victoriametricscloud_cloud_alertmanager` | Manages a VictoriaMetrics-hosted Alertmanager notifier in VictoriaMetrics Cloud. |
| `victoriametricscloud_custom_alertmanager` | Manages a custom (externally hosted) Alertmanager notifier in VictoriaMetrics Cloud. |
| `victoriametricscloud_deployment_notifier` | Attaches a notifier (custom or cloud Alertmanager) to a VictoriaMetrics Cloud deployment. One deployment has at most one notifier attached. |
| `victoriametricscloud_deployment` | Manages a VictoriaMetrics Cloud deployment. |
| `victoriametricscloud_rule_file` | Manages an alerting or recording rules file for a VictoriaMetrics Cloud deployment. |
<!-- resources:end -->

## Supported Data Sources
<!-- datasources:start -->
| Data Source | Purpose |
|---|---|
| `victoriametricscloud_cloud_providers` | Fetches the list of available cloud providers for VictoriaMetrics Cloud deployments. |
| `victoriametricscloud_deployment` | Fetches details of a specific VictoriaMetrics Cloud deployment. |
| `victoriametricscloud_deployments` | Fetches the list of VictoriaMetrics Cloud deployments. |
| `victoriametricscloud_regions` | Fetches the list of available regions for VictoriaMetrics Cloud deployments. |
| `victoriametricscloud_tiers` | Fetches the list of available tiers for VictoriaMetrics Cloud deployments. |
<!-- datasources:end -->

## Examples

- **Provider bootstrap** – minimal provider configuration: [`examples/provider`](examples/provider)
- **Resources** – end-to-end configuration covering deployments, tokens, rule files, and notifiers: [`examples/resources`](examples/resources)
- **Data sources** – discover providers, regions, tiers, and deployments: [`examples/datasources`](examples/datasources)

Import the examples into your own workspace or run them directly after setting `TF_VAR_api_key` or exporting `VMCLOUD_API_KEY`.

## Documentation
Registry-ready documentation is generated into the `docs/` directory via [`tfplugindocs`](https://github.com/hashicorp/terraform-plugin-docs). 
Whenever you update schemas, run `make docs` to refresh resource and data source pages; it also regenerates the tables above from those pages, so this README can't go stale.

## License

Licensed under the [Apache 2.0 License](LICENSE).
