# Terraform Provider for VictoriaMetrics Cloud

Manage VictoriaMetrics Cloud deployments, access tokens, and rule files with Terraform. 
This provider wraps the [VictoriaMetrics Cloud API](https://docs.victoriametrics.com/victoriametrics-cloud/api/) and exposes both resources and data sources so you can manage infrastructure as code alongside the rest of your stack.

## Getting Started

- Add the provider to your configuration:

```hcl
terraform {
  required_providers {
    victoriametricscloud = {
      source  = "VictoriaMetrics/victoriametricscloud"
      version = "0.0.1"
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
| Resource                            | Purpose                                                                                                                                             |
|-------------------------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------|
| `victoriametricscloud_deployment`   | Provisions single-node or cluster VictoriaMetrics deployments, including retention, deduplication, maintenance windows, and custom component flags. |
| `victoriametricscloud_access_token` | Manages scoped access tokens (`r`, `w`, or `rw`) for a deployment, optionally targeting a cluster tenant.                                           |
| `victoriametricscloud_rule_file`    | Uploads and manages alerting/recording rule files associated with a deployment.                                                                     |

## Supported Data Sources
| Data Source                            | Purpose                                                                        |
|----------------------------------------|--------------------------------------------------------------------------------|
| `victoriametricscloud_cloud_providers` | Lists available cloud providers and their metadata.                            |
| `victoriametricscloud_regions`         | Lists deployment regions per cloud provider.                                   |
| `victoriametricscloud_tiers`           | Lists available deployment tiers with capacity and pricing information.        |
| `victoriametricscloud_deployments`     | Returns summaries of all deployments visible to the API key.                   |
| `victoriametricscloud_deployment`      | Retrieves detailed information (including costs) for a specific deployment ID. |

## Examples

- **Provider bootstrap** – minimal provider configuration: [`examples/provider`](examples/provider)
- **Resources** – end-to-end configuration covering deployments, tokens, and rule files: [`examples/resources`](examples/resources)
- **Data sources** – discover providers, regions, tiers, and deployments: [`examples/datasources`](examples/datasources)

Import the examples into your own workspace or run them directly after setting `TF_VAR_api_key` or exporting `VMCLOUD_API_KEY`.

## Documentation
Registry-ready documentation is generated into the `docs/` directory via [`tfplugindocs`](https://github.com/hashicorp/terraform-plugin-docs). 
Whenever you update schemas, run `make docs` to refresh resource and data source pages.

## License

Licensed under the [Apache 2.0 License](LICENSE).
