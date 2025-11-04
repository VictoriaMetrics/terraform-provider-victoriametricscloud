# Create read-write access token for application
resource "victoriametricscloud_access_token" "single_app_token" {
  deployment_id = victoriametricscloud_deployment.single_demo.id
  description   = "Application read-write token"
  type          = "rw"
}

output "app_token" {
  description = "Application access token (sensitive)"
  value       = victoriametricscloud_access_token.single_app_token.secret
  sensitive   = true
}

# Create read-only access token for Grafana
resource "victoriametricscloud_access_token" "single_grafana_token" {
  deployment_id = victoriametricscloud_deployment.single_demo.id
  description   = "Grafana read-only token"
  type          = "r"
}

output "grafana_token" {
  description = "Grafana read-only token (sensitive)"
  value       = victoriametricscloud_access_token.single_grafana_token.secret
  sensitive   = true
}

# Create write-only access token for agent
resource "victoriametricscloud_access_token" "single_agent_token" {
  deployment_id = victoriametricscloud_deployment.single_demo.id
  description   = "Agent write-only token"
  type          = "w"
}

output "agent_token" {
  description = "Agent write-only token (sensitive)"
  value       = victoriametricscloud_access_token.single_agent_token.secret
  sensitive   = true
}
