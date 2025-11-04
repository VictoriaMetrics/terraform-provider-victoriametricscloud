# Create a single-node deployment
resource "victoriametricscloud_deployment" "single_demo" {
  name               = "Single demo"
  type               = "single_node"
  cloud_provider     = "aws"
  region             = "eu-west-1"
  tier               = 21
  storage_size       = 20
  storage_size_unit  = "GB"
  retention          = 30
  retention_unit     = "d"
  deduplication      = 30
  deduplication_unit = "s"
  maintenance_window = "Sat-Sun 3-4am"

  # Custom flags for cluster components
  single_flags = [
    "-search.maxQueryDuration=360s",
  ]
}

output "deployment_id" {
  description = "ID of the deployment"
  value       = victoriametricscloud_deployment.single_demo.id
}

output "deployment_endpoint" {
  description = "API endpoint URL"
  value       = victoriametricscloud_deployment.single_demo.access_endpoint
}
