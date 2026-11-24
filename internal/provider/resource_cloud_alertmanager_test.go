package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestCloudAlertmanager(t *testing.T) {
	api := newFakeAPI(t, true)
	const name = "victoriametricscloud_cloud_alertmanager.test"
	var id string
	config := func(receiver string) string {
		return providerConfig(api) + `
resource "victoriametricscloud_cloud_alertmanager" "test" {
  name = "hosted"
  config = <<-EOT
    route:
      receiver: ` + receiver + `
    receivers:
      - name: ` + receiver + `
  EOT
}
`
	}
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config("default"),
				Check: resource.ComposeAggregateTestCheckFunc(
					captureAttr(name, "id", &id),
					resource.TestCheckResourceAttr(name, "name", "hosted"),
					resource.TestCheckResourceAttr(name, "status", "unknown"),
					resource.TestCheckResourceAttr(name, "notifier.type", "cloud-alertmanager"),
					resource.TestCheckResourceAttrPair(name, "notifier.id", name, "id"),
				),
			},
			{
				// A changed config updates in place, keeping the same id.
				Config: config("other"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPtr(name, "id", &id),
				),
			},
			{
				ResourceName:      name,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				PreConfig:          func() { api.removeCloudAlertmanager(id) },
				Config:             config("other"),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
