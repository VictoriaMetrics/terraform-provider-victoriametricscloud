package provider

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"

	vmcloudapi "github.com/VictoriaMetrics/victoriametrics-cloud-api-go/v1"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// singleNodeConfig is a single_node deployment.
func singleNodeConfig(api *fakeAPI, tier int, dedup int, dedupUnit, flags string) string {
	return providerConfig(api) + fmt.Sprintf(`
resource "victoriametricscloud_deployment" "test" {
  name               = "test"
  type               = "single_node"
  cloud_provider     = "aws"
  region             = "eu-west-1"
  tier               = %d
  storage_size       = 20
  storage_size_unit  = "GB"
  retention          = 30
  retention_unit     = "d"
  deduplication      = %d
  deduplication_unit = %q
  maintenance_window = "Sat-Sun 3-4am"
  single_flags       = %s
}
`, tier, dedup, dedupUnit, flags)
}

func TestDeploymentSingleNode(t *testing.T) {
	api := newFakeAPI(t, true)
	const name = "victoriametricscloud_deployment.test"
	var id string
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create with flags: they are applied by an update once the deployment runs.
				Config: singleNodeConfig(api, 14, 30, "s", `["-search.maxUniqueTimeseries=50000"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					captureAttr(name, "id", &id),
					resource.TestCheckResourceAttr(name, "status", "RUNNING"),
					resource.TestCheckResourceAttr(name, "metrics.deduplication", "30"),
					resource.TestCheckResourceAttr(name, "metrics.deduplication_unit", "s"),
					resource.TestCheckResourceAttr(name, "deduplication", "30"),
					resource.TestCheckResourceAttr(name, "deduplication_unit", "s"),
					resource.TestCheckResourceAttr(name, "single_flags.#", "1"),
					resource.TestCheckResourceAttr(name, "single.flags.#", "1"),
					resource.TestCheckResourceAttr(name, "single.flags.0", "-search.maxUniqueTimeseries=50000"),
					resource.TestCheckNoResourceAttr(name, "cluster.%"),
					checkAPIFlags(api, &id, vmcloudapi.DeploymentFlags{SingleFlags: vmcloudapi.FlagList{"-search.maxUniqueTimeseries=50000"}}),
				),
			},
			{
				Config: singleNodeConfig(api, 15, 500, "ms", `["-search.maxUniqueTimeseries=50000", "-search.maxQueryDuration=60s"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "tier", "15"),
					resource.TestCheckResourceAttr(name, "metrics.deduplication", "500"),
					resource.TestCheckResourceAttr(name, "metrics.deduplication_unit", "ms"),
					resource.TestCheckResourceAttr(name, "single.flags.#", "2"),
					checkAPIFlags(api, &id, vmcloudapi.DeploymentFlags{SingleFlags: vmcloudapi.FlagList{"-search.maxUniqueTimeseries=50000", "-search.maxQueryDuration=60s"}}),
				),
			},
			{
				// Only flags change.
				Config: singleNodeConfig(api, 15, 500, "ms", `[]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "single.flags.#", "0"),
					checkAPIFlags(api, &id, vmcloudapi.DeploymentFlags{}),
				),
			},
			{
				ResourceName:      name,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestDeploymentLogsAndTraces(t *testing.T) {
	api := newFakeAPI(t, true)
	config := func(retention int) string {
		cfg := providerConfig(api)
		for _, typ := range []string{"vlogs_single", "vtraces_single"} {
			cfg += fmt.Sprintf(`
resource "victoriametricscloud_deployment" %[1]q {
  name               = %[1]q
  type               = %[1]q
  cloud_provider     = "aws"
  region             = "eu-west-1"
  tier               = 1
  storage_size       = 20
  storage_size_unit  = "GB"
  retention          = %[2]d
  retention_unit     = "d"
  maintenance_window = "Mon-Fri 4-5am"
}
`, typ, retention)
		}
		return cfg
	}
	checks := func(retention string) resource.TestCheckFunc {
		var fs []resource.TestCheckFunc
		for _, name := range []string{"victoriametricscloud_deployment.vlogs_single", "victoriametricscloud_deployment.vtraces_single"} {
			fs = append(fs,
				resource.TestCheckResourceAttr(name, "retention", retention),
				resource.TestCheckNoResourceAttr(name, "deduplication"),
				resource.TestCheckNoResourceAttr(name, "metrics.%"),
				resource.TestCheckResourceAttr(name, "single.%", "1"),
				resource.TestCheckNoResourceAttr(name, "single.flags.#"),
				resource.TestCheckNoResourceAttr(name, "cluster.%"),
			)
		}
		return resource.ComposeAggregateTestCheckFunc(fs...)
	}
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories,
		Steps: []resource.TestStep{
			{Config: config(7), Check: checks("7")},
			{Config: config(14), Check: checks("14")},
			{
				ResourceName:      "victoriametricscloud_deployment.vlogs_single",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestDeploymentCluster(t *testing.T) {
	api := newFakeAPI(t, true)
	const name = "victoriametricscloud_deployment.test"
	var id string
	config := func(storageFlags string) string {
		return providerConfig(api) + fmt.Sprintf(`
resource "victoriametricscloud_deployment" "test" {
  name               = "test"
  type               = "cluster"
  cloud_provider     = "aws"
  region             = "us-east-1"
  tier               = 38
  storage_size       = 40
  storage_size_unit  = "GB"
  retention          = 30
  retention_unit     = "d"
  maintenance_window = "Sat-Sun 3-4am"
  deduplication      = 60
  deduplication_unit = "s"
  select_flags       = ["-search.maxQueryDuration=60s"]
  storage_flags      = %s
  insert_flags       = ["-maxLabelsPerTimeseries=40"]
}
`, storageFlags)
	}
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config("null"),
				Check: resource.ComposeAggregateTestCheckFunc(
					captureAttr(name, "id", &id),
					resource.TestCheckNoResourceAttr(name, "single.%"),
					resource.TestCheckResourceAttr(name, "cluster.select_flags.0", "-search.maxQueryDuration=60s"),
					resource.TestCheckNoResourceAttr(name, "cluster.storage_flags.#"),
					resource.TestCheckResourceAttr(name, "cluster.insert_flags.0", "-maxLabelsPerTimeseries=40"),
					checkAPIFlags(api, &id, vmcloudapi.DeploymentFlags{
						SelectFlags: vmcloudapi.FlagList{"-search.maxQueryDuration=60s"},
						InsertFlags: vmcloudapi.FlagList{"-maxLabelsPerTimeseries=40"},
					}),
				),
			},
			{
				Config: config(`["-search.maxUniqueTimeseries=100000"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "cluster.storage_flags.0", "-search.maxUniqueTimeseries=100000"),
					checkAPIFlags(api, &id, vmcloudapi.DeploymentFlags{
						SelectFlags:  vmcloudapi.FlagList{"-search.maxQueryDuration=60s"},
						StorageFlags: vmcloudapi.FlagList{"-search.maxUniqueTimeseries=100000"},
						InsertFlags:  vmcloudapi.FlagList{"-maxLabelsPerTimeseries=40"},
					}),
				),
			},
			{
				ResourceName:      name,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestDeploymentFlagsNotReported covers the API as it behaves today: it neither
// returns flags nor compares them on update.
func TestDeploymentFlagsNotReported(t *testing.T) {
	api := newFakeAPI(t, false)
	const name = "victoriametricscloud_deployment.test"
	var id string
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: singleNodeConfig(api, 14, 30, "s", `["-a=1"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					captureAttr(name, "id", &id),
					resource.TestCheckResourceAttr(name, "single.flags.0", "-a=1"),
					// Known API limitation: the flags-only update that follows the create
					// is refused as "no changes", so the API does not store the flags.
					checkAPIFlags(api, &id, vmcloudapi.DeploymentFlags{}),
				),
			},
			{
				// The provider keeps flags the API does not report.
				RefreshState: true,
				Check:        resource.TestCheckResourceAttr(name, "single.flags.0", "-a=1"),
			},
			{
				// Imported state has no flags: the API does not return them.
				ResourceName:            name,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"single_flags", "single.flags"},
			},
			{
				// Only flags change; the API refuses the update as "no changes" because
				// it does not compare flags, and the provider treats that as success.
				Config: singleNodeConfig(api, 14, 30, "s", `["-a=1", "-b=2"]`),
				Check:  resource.TestCheckResourceAttr(name, "single.flags.1", "-b=2"),
			},
		},
	})
}

func TestDeploymentDeletedOutsideTerraform(t *testing.T) {
	api := newFakeAPI(t, true)
	var id string
	config := singleNodeConfig(api, 14, 30, "s", "null")
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check:  captureAttr("victoriametricscloud_deployment.test", "id", &id),
			},
			{
				PreConfig:          func() { api.removeDeployment(id) },
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestDeploymentValidation(t *testing.T) {
	api := newFakeAPI(t, true)
	config := func(typ, extra string) string {
		return providerConfig(api) + fmt.Sprintf(`
resource "victoriametricscloud_deployment" "test" {
  name               = "test"
  type               = %q
  cloud_provider     = "aws"
  region             = "eu-west-1"
  tier               = 1
  storage_size       = 20
  storage_size_unit  = "GB"
  retention          = 30
  retention_unit     = "d"
  maintenance_window = "Sat-Sun 3-4am"
  %s
}
`, typ, extra)
	}
	dedup := "deduplication = 30\n  deduplication_unit = \"s\""
	cases := []struct {
		name, typ, extra, err string
	}{
		{"deduplication without unit", "single_node", "deduplication = 30", `must be set together`},
		{"missing deduplication", "single_node", "", `"single_node" deployments require "deduplication"`},
		{"missing deduplication for cluster", "cluster", "", `"cluster" deployments require "deduplication"`},
		{"deduplication for logs", "vlogs_single", dedup, `"vlogs_single" deployments have no deduplication window`},
		{"deduplication for traces", "vtraces_single", dedup, `"vtraces_single" deployments have no deduplication window`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: testProviderFactories,
				Steps: []resource.TestStep{{
					Config:      config(tc.typ, tc.extra),
					PlanOnly:    true,
					ExpectError: regexp.MustCompile(regexp.QuoteMeta(tc.err)),
				}},
			})
		})
	}
}

// Changing an attribute the update API cannot change replaces the deployment.
func TestDeploymentRegionChangeReplaces(t *testing.T) {
	api := newFakeAPI(t, true)
	const name = "victoriametricscloud_deployment.test"
	config := func(region string) string {
		return strings.Replace(singleNodeConfig(api, 14, 30, "s", "null"), `"eu-west-1"`, strconv.Quote(region), 1)
	}
	var firstID, secondID string
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config("eu-west-1"),
				Check:  captureAttr(name, "id", &firstID),
			},
			{
				Config: config("us-east-1"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(name, plancheck.ResourceActionReplace)},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					captureAttr(name, "id", &secondID),
					resource.TestCheckResourceAttr(name, "region", "us-east-1"),
					func(*terraform.State) error {
						if secondID == firstID {
							return fmt.Errorf("deployment was not replaced: id is still %q", firstID)
						}
						return nil
					},
				),
			},
		},
	})
}

// A deployment that does not become running in time is still created, without
// being tainted; its flags stay pending and the next apply sends them.
func TestDeploymentFlagsPendingWhenNotRunning(t *testing.T) {
	shortenWaits(t)
	api := newFakeAPI(t, true)
	api.provisioningReads = 1 << 30 // never running on its own
	const name = "victoriametricscloud_deployment.test"
	config := singleNodeConfig(api, 14, 30, "s", `["-a=1"]`)
	var id string
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create succeeds with a warning; the refreshed plan shows the pending flags.
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					captureAttr(name, "id", &id),
					checkAPIFlags(api, &id, vmcloudapi.DeploymentFlags{}),
				),
				ExpectNonEmptyPlan: true,
			},
			{
				// Once the deployment runs, the next apply updates it in place.
				PreConfig: func() { api.setStatus(id, vmcloudapi.DeploymentStatusRunning) },
				Config:    config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(name, plancheck.ResourceActionUpdate)},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPtr(name, "id", &id),
					checkAPIFlags(api, &id, vmcloudapi.DeploymentFlags{SingleFlags: vmcloudapi.FlagList{"-a=1"}}),
				),
			},
		},
	})
}

// updateDeployment reports the API's "no changes" answer, which Create and Update
// turn into a warning, instead of an error or a silent success.
func TestUpdateDeploymentNoChanges(t *testing.T) {
	for _, tc := range []struct {
		reportFlags   bool
		wantNoChanges bool
	}{
		{reportFlags: false, wantNoChanges: true}, // the API today: flags are not compared
		{reportFlags: true, wantNoChanges: false}, // the API as intended
	} {
		t.Run(fmt.Sprintf("reportFlags=%v", tc.reportFlags), func(t *testing.T) {
			api := newFakeAPI(t, tc.reportFlags)
			client, err := vmcloudapi.New("test", vmcloudapi.WithBaseURL(api.server.URL))
			if err != nil {
				t.Fatal(err)
			}
			ctx := t.Context()
			dedup, unit := uint32(30), vmcloudapi.DurationUnit("s")
			created, err := client.CreateDeployment(ctx, vmcloudapi.DeploymentCreationRequest{
				Name: "test", Type: vmcloudapi.DeploymentTypeSingleNode, Provider: "aws", Region: "eu-west-1",
				Tier: 14, StorageSize: 20, StorageSizeUnit: "GB", Retention: 30, RetentionUnit: "d",
				Deduplication: &dedup, DeduplicationUnit: &unit, MaintenanceWindow: "Sat-Sun 3-4am",
			})
			if err != nil {
				t.Fatal(err)
			}
			api.setStatus(created.ID, vmcloudapi.DeploymentStatusRunning)

			r := &deploymentResource{client: client}
			_, noChanges, err := r.updateDeployment(ctx, created.ID, vmcloudapi.DeploymentUpdateRequest{
				Name: "test", Tier: 14, StorageSize: 20, StorageSizeUnit: "GB", Retention: 30, RetentionUnit: "d",
				Deduplication: &dedup, DeduplicationUnit: &unit, MaintenanceWindow: "Sat-Sun 3-4am",
				Flags: vmcloudapi.DeploymentFlags{SingleFlags: vmcloudapi.FlagList{"-a=1"}},
			})
			if err != nil {
				t.Fatalf("updateDeployment: %s", err)
			}
			if noChanges != tc.wantNoChanges {
				t.Fatalf("noChanges = %v, want %v", noChanges, tc.wantNoChanges)
			}
		})
	}
}
