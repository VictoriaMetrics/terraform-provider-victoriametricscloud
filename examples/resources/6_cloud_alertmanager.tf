# Create a VictoriaMetrics-hosted Alertmanager notifier
resource "victoriametricscloud_cloud_alertmanager" "hosted" {
  name   = "Hosted Alertmanager"
  config = file("${path.module}/cloud_alertmanager.yaml")
}
