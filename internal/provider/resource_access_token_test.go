package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// deploymentWithConfig is a single_node deployment followed by extra configuration.
func deploymentWithConfig(api *fakeAPI, extra string) string {
	return singleNodeConfig(api, 14, 30, "s", "null") + extra
}

func TestAccessToken(t *testing.T) {
	api := newFakeAPI(t, true)
	const name = "victoriametricscloud_access_token.test"
	config := func(description string) string {
		return deploymentWithConfig(api, fmt.Sprintf(`
resource "victoriametricscloud_access_token" "test" {
  deployment_id = victoriametricscloud_deployment.test.id
  type          = "rw"
  description   = %q
}
`, description))
	}
	var deploymentID, firstID, secondID string
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config("first"),
				Check: resource.ComposeAggregateTestCheckFunc(
					captureAttr("victoriametricscloud_deployment.test", "id", &deploymentID),
					captureAttr(name, "id", &firstID),
					resource.TestCheckResourceAttr(name, "type", "rw"),
					resource.TestCheckResourceAttrSet(name, "secret"),
				),
			},
			{
				// A changed description replaces the token.
				Config: config("second"),
				Check: resource.ComposeAggregateTestCheckFunc(
					captureAttr(name, "id", &secondID),
					resource.TestCheckResourceAttr(name, "description", "second"),
					func(*terraform.State) error {
						if secondID == firstID {
							return fmt.Errorf("token was not replaced: id is still %q", firstID)
						}
						return nil
					},
				),
			},
			{
				ResourceName:      name,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(*terraform.State) (string, error) {
					return deploymentID + "/" + secondID, nil
				},
			},
			{
				PreConfig:          func() { api.removeToken(deploymentID, secondID) },
				Config:             config("second"),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
