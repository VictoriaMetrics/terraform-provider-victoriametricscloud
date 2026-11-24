# Attach the hosted Alertmanager notifier to the single-node deployment.
# notifier accepts the full output of either victoriametricscloud_custom_alertmanager
# or victoriametricscloud_cloud_alertmanager; only their id and type attributes are used.
resource "victoriametricscloud_deployment_notifier" "single_demo" {
  deployment_id = victoriametricscloud_deployment.single_demo.id
  notifier      = victoriametricscloud_cloud_alertmanager.hosted.notifier
}
