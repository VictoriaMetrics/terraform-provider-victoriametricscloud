package provider

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	vmcloudapi "github.com/VictoriaMetrics/victoriametrics-cloud-api-go/v1"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func init() {
	// Fake deployments become running after a few reads; don't wait between them.
	deploymentPollInterval = time.Millisecond
}

// shortenWaits makes the provider give up waiting for a deployment quickly.
func shortenWaits(t *testing.T) {
	t.Helper()
	previous := deploymentWaitTimeout
	deploymentWaitTimeout = 50 * time.Millisecond
	t.Cleanup(func() { deploymentWaitTimeout = previous })
}

// testProviderFactories serves the provider in-process for terraform-plugin-testing.
var testProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"victoriametricscloud": providerserver.NewProtocol6WithError(New("test")()),
}

// providerConfig points the provider at api.
func providerConfig(api *fakeAPI) string {
	return fmt.Sprintf(`
provider "victoriametricscloud" {
  api_key  = "test"
  base_url = %q
}
`, api.server.URL)
}

// captureAttr stores attribute key of resource name into *dst.
func captureAttr(name, key string, dst *string) func(*terraform.State) error {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("resource %s not found in state", name)
		}
		*dst = rs.Primary.Attributes[key]
		return nil
	}
}

// checkAPIFlags checks the flags the fake API stores for the deployment in *id.
func checkAPIFlags(api *fakeAPI, id *string, want vmcloudapi.DeploymentFlags) func(*terraform.State) error {
	return func(*terraform.State) error {
		got, ok := api.deploymentFlags(*id)
		if !ok {
			return fmt.Errorf("deployment %q not found in the API", *id)
		}
		if !reflect.DeepEqual(normalizeFlags(got), normalizeFlags(want)) {
			return fmt.Errorf("API flags = %+v, want %+v", got, want)
		}
		return nil
	}
}
