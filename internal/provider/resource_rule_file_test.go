package provider

import (
	"fmt"
	"regexp"
	"testing"

	vmcloudapi "github.com/VictoriaMetrics/victoriametrics-cloud-api-go/v1"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestRuleFile(t *testing.T) {
	api := newFakeAPI(t, true)
	const name = "victoriametricscloud_rule_file.test"
	config := func(interval string) string {
		return deploymentWithConfig(api, fmt.Sprintf(`
resource "victoriametricscloud_rule_file" "test" {
  deployment_id = victoriametricscloud_deployment.test.id
  file_name     = "rules.yaml"
  content       = <<-EOT
    groups:
      - name: "recording-rules"
        interval: %q
        rules:
          - record: "request_duration_seconds:5m"
            expr: "avg(rate(request_duration_seconds[5m])) by (job)"
  EOT
}
`, interval))
	}
	var deploymentID string
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config("30s"),
				Check: resource.ComposeAggregateTestCheckFunc(
					captureAttr("victoriametricscloud_deployment.test", "id", &deploymentID),
					resource.TestCheckResourceAttrPair(name, "deployment_id", "victoriametricscloud_deployment.test", "id"),
					resource.TestMatchResourceAttr(name, "content", regexp.MustCompile(`interval: "30s"`)),
				),
			},
			{
				Config: config("1m"),
				Check:  resource.TestMatchResourceAttr(name, "content", regexp.MustCompile(`interval: "1m"`)),
			},
			{
				ResourceName:      name,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				PreConfig:          func() { api.removeRuleFile(deploymentID, "rules.yaml") },
				Config:             config("1m"),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// Writing a rule file of a failed deployment stops retrying instead of waiting for
// the lock to clear.
func TestRuleFileOnFailedDeployment(t *testing.T) {
	api := newFakeAPI(t, true)
	var deploymentID string
	ruleFile := `
resource "victoriametricscloud_rule_file" "test" {
  deployment_id = victoriametricscloud_deployment.test.id
  file_name     = "rules.yaml"
  content       = "groups: []"
}
`
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: deploymentWithConfig(api, ""),
				Check:  captureAttr("victoriametricscloud_deployment.test", "id", &deploymentID),
			},
			{
				PreConfig:   func() { api.setStatus(deploymentID, vmcloudapi.DeploymentStatusError) },
				Config:      deploymentWithConfig(api, ruleFile),
				ExpectError: regexp.MustCompile(`deployment is in status ERROR`),
			},
		},
	})
}

// Writing a rule file of a deployment that stays locked gives up after
// deploymentWaitTimeout instead of hanging the apply.
func TestRuleFileGivesUpWhenLocked(t *testing.T) {
	shortenWaits(t)
	api := newFakeAPI(t, true)
	api.provisioningReads = 1 << 30 // never running on its own
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories,
		Steps: []resource.TestStep{{
			Config: deploymentWithConfig(api, `
resource "victoriametricscloud_rule_file" "test" {
  deployment_id = victoriametricscloud_deployment.test.id
  file_name     = "rules.yaml"
  content       = "groups: []"
}
`),
			ExpectError: regexp.MustCompile(`still locked after`),
		}},
	})
}
