package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestCustomAlertmanager(t *testing.T) {
	api := newFakeAPI(t, true)
	const name = "victoriametricscloud_custom_alertmanager.test"
	var id string
	config := func(url, authType string) string {
		return providerConfig(api) + `
resource "victoriametricscloud_custom_alertmanager" "test" {
  name      = "external"
  url       = "` + url + `"
  auth_type = "` + authType + `"
  username  = "demo"
  password  = "demo-password"
}
`
	}
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config("https://alertmanager.example.com", "basic"),
				Check: resource.ComposeAggregateTestCheckFunc(
					captureAttr(name, "id", &id),
					resource.TestCheckResourceAttr(name, "name", "external"),
					resource.TestCheckResourceAttr(name, "auth_type", "basic"),
					resource.TestCheckResourceAttr(name, "status", "unknown"),
					resource.TestCheckResourceAttr(name, "notifier.type", "custom-alertmanager"),
					resource.TestCheckResourceAttrPair(name, "notifier.id", name, "id"),
				),
			},
			{
				// A changed URL updates in place, keeping the same id.
				Config: config("https://alertmanager2.example.com", "basic"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "url", "https://alertmanager2.example.com"),
					resource.TestCheckResourceAttrPtr(name, "id", &id),
				),
			},
			{
				ResourceName:            name,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password"},
			},
			{
				PreConfig:          func() { api.removeCustomAlertmanager(id) },
				Config:             config("https://alertmanager2.example.com", "basic"),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestCustomAlertmanagerBearerAuth(t *testing.T) {
	api := newFakeAPI(t, true)
	const name = "victoriametricscloud_custom_alertmanager.test"
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig(api) + `
resource "victoriametricscloud_custom_alertmanager" "test" {
  name         = "external"
  url          = "https://alertmanager.example.com"
  auth_type    = "bearer"
  bearer_token = "demo-token"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "auth_type", "bearer"),
					resource.TestCheckNoResourceAttr(name, "username"),
					resource.TestCheckNoResourceAttr(name, "password"),
				),
			},
		},
	})
}
