# Upload recording rules
resource "victoriametricscloud_rule_file" "recording_rules" {
  deployment_id = victoriametricscloud_deployment.single_demo.id
  file_name     = "recording_rules.yaml"
  content       = file("${path.module}/recording_rules.yaml")
}
