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

# Get list of available cloud providers
data "victoriametricscloud_cloud_providers" "available" {}

# Get list of available regions
data "victoriametricscloud_regions" "available" {}

# Get list of available tiers
data "victoriametricscloud_tiers" "available" {}

# Get list of all deployments
data "victoriametricscloud_deployments" "all" {}

# Get details of a specific deployment (if you have one)
# data "victoriametricscloud_deployment" "specific" {
#   id = "your-deployment-id"
# }

output "cloud_providers" {
  description = "Available cloud providers"
  value       = data.victoriametricscloud_cloud_providers.available.cloud_providers
}

output "regions" {
  description = "Available regions"
  value       = data.victoriametricscloud_regions.available.regions
}

output "tiers" {
  description = "Available tiers"
  value       = data.victoriametricscloud_tiers.available.tiers
}

output "deployments" {
  description = "All deployments"
  value       = data.victoriametricscloud_deployments.all.deployments
}
