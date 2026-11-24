# Register an externally hosted Alertmanager as a custom notifier
resource "victoriametricscloud_custom_alertmanager" "external" {
  name      = "External Alertmanager"
  url       = "https://alertmanager.example.com"
  auth_type = "basic"
  username  = "demo"
  password  = "demo-password"
}
