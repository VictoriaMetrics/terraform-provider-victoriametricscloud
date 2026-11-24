package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestDeploymentNotifier(t *testing.T) {
	api := newFakeAPI(t, true)
	const name = "victoriametricscloud_deployment_notifier.test"
	const secondURL = "https://alertmanager.example.com"
	var deploymentID, secondNotifierID string
	config := func(notifierResource, secondURL string) string {
		return deploymentWithConfig(api, `
resource "victoriametricscloud_cloud_alertmanager" "first" {
  name   = "first"
  config = <<-EOT
    route:
      receiver: default
    receivers:
      - name: default
  EOT
}

resource "victoriametricscloud_custom_alertmanager" "second" {
  name      = "second"
  url       = "`+secondURL+`"
  auth_type = "basic"
  username  = "demo"
  password  = "demo-password"
}

resource "victoriametricscloud_deployment_notifier" "test" {
  deployment_id = victoriametricscloud_deployment.test.id
  notifier      = `+notifierResource+`
}
`)
	}
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config("victoriametricscloud_cloud_alertmanager.first.notifier", secondURL),
				Check: resource.ComposeAggregateTestCheckFunc(
					captureAttr("victoriametricscloud_deployment.test", "id", &deploymentID),
					resource.TestCheckResourceAttrPair(name, "id", "victoriametricscloud_deployment.test", "id"),
					resource.TestCheckResourceAttr(name, "notifier.type", "cloud-alertmanager"),
					resource.TestCheckResourceAttr(name, "notifier_name", "first"),
				),
			},
			{
				// Switching to the other notifier attaches it in place.
				Config: config("victoriametricscloud_custom_alertmanager.second.notifier", secondURL),
				Check: resource.ComposeAggregateTestCheckFunc(
					captureAttr(name, "notifier.id", &secondNotifierID),
					resource.TestCheckResourceAttr(name, "notifier.type", "custom-alertmanager"),
					resource.TestCheckResourceAttr(name, "notifier_name", "second"),
				),
			},
			{
				Config: config("victoriametricscloud_custom_alertmanager.second.notifier", "https://alertmanager2.example.com"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(name, plancheck.ResourceActionNoop),
					},
				},
			},
			{
				ResourceName:      name,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(*terraform.State) (string, error) { return deploymentID, nil },
			},
			{
				PreConfig:          func() { api.removeCustomAlertmanager(secondNotifierID) },
				Config:             config("victoriametricscloud_custom_alertmanager.second.notifier", "https://alertmanager2.example.com"),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
