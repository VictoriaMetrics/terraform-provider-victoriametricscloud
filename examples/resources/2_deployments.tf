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
}

output "deployment_id" {
  description = "ID of the deployment"
  value       = victoriametricscloud_deployment.single_demo.id
}

output "deployment_endpoint" {
  description = "API endpoint URL"
  value       = victoriametricscloud_deployment.single_demo.access_endpoint
}

# Create a VictoriaLogs deployment
# VictoriaLogs and VictoriaTraces deployments have no deduplication window, so
# deduplication and deduplication_unit are left unset.
resource "victoriametricscloud_deployment" "vlogs_demo" {
  name               = "VictoriaLogs demo"
  type               = "vlogs_single"
  cloud_provider     = "aws"
  region             = "eu-west-1"
  tier               = 101
  storage_size       = 20
  storage_size_unit  = "GB"
  retention          = 30
  retention_unit     = "d"
  maintenance_window = "Sat-Sun 3-4am"
}

# Create a VictoriaTraces deployment
# Available only for accounts where VictoriaTraces is enabled.
resource "victoriametricscloud_deployment" "vtraces_demo" {
  name               = "VictoriaTraces demo"
  type               = "vtraces_single"
  cloud_provider     = "aws"
  region             = "eu-west-1"
  tier               = 201
  storage_size       = 20
  storage_size_unit  = "GB"
  retention          = 30
  retention_unit     = "d"
  maintenance_window = "Sat-Sun 3-4am"
}
