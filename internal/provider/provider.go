package provider

import (
	"context"
	"fmt"
	"os"

	vmcloudapi "github.com/VictoriaMetrics/victoriametrics-cloud-api-go/v1"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ provider.Provider = &victoriametricsCloudProvider{}
)

// New is a helper function to simplify provider server and testing implementation.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &victoriametricsCloudProvider{
			version: version,
		}
	}
}

// victoriametricsCloudProvider is the provider implementation.
type victoriametricsCloudProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// victoriametricsCloudProviderModel maps provider schema data to a Go type.
type victoriametricsCloudProviderModel struct {
	APIKey  types.String `tfsdk:"api_key"`
	BaseURL types.String `tfsdk:"base_url"`
}

// Metadata returns the provider type name.
func (p *victoriametricsCloudProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "victoriametricscloud"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
func (p *victoriametricsCloudProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Terraform provider for managing VictoriaMetrics Cloud resources.",
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				Description: "API key for VictoriaMetrics Cloud authentication. Can also be set via VMCLOUD_API_KEY environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
			"base_url": schema.StringAttribute{
				Description: "Base URL for VictoriaMetrics Cloud API. Defaults to https://api.victoriametrics.cloud. Can also be set via VMCLOUD_BASE_URL environment variable.",
				Optional:    true,
			},
		},
	}
}

// Configure prepares a VictoriaMetrics Cloud API client for data sources and resources.
func (p *victoriametricsCloudProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config victoriametricsCloudProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Configuration values are now available.
	// If configuration values are not available, default to environment variables.
	apiKey := os.Getenv("VMCLOUD_API_KEY")
	baseURL := os.Getenv("VMCLOUD_BASE_URL")

	if !config.APIKey.IsNull() {
		apiKey = config.APIKey.ValueString()
	}

	if !config.BaseURL.IsNull() {
		baseURL = config.BaseURL.ValueString()
	}

	// If API key is still not available, return an error
	if apiKey == "" {
		resp.Diagnostics.AddError(
			"Missing API Key Configuration",
			"While configuring the provider, the API key was not found in "+
				"the VMCLOUD_API_KEY environment variable or provider "+
				"configuration block api_key attribute.",
		)
		return
	}

	// Create the API client
	var client *vmcloudapi.VMCloudAPIClient
	var err error
	options := make([]vmcloudapi.VMCloudAPIClientOption, 0)
	if baseURL != "" {
		options = append(options, vmcloudapi.WithBaseURL(baseURL))
	}
	client, err = vmcloudapi.New(apiKey, options...)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create VictoriaMetrics Cloud API Client",
			fmt.Sprintf("An unexpected error occurred when creating the VictoriaMetrics Cloud API client: %s", err.Error()),
		)
		return
	}

	// Make the client available during DataSource and Resource type Configure methods.
	resp.DataSourceData = client
	resp.ResourceData = client
}

// DataSources defines the data sources implemented in the provider.
func (p *victoriametricsCloudProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewCloudProvidersDataSource,
		NewRegionsDataSource,
		NewTiersDataSource,
		NewDeploymentDataSource,
		NewDeploymentsDataSource,
	}
}

// Resources defines the resources implemented in the provider.
func (p *victoriametricsCloudProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewDeploymentResource,
		NewAccessTokenResource,
		NewRuleFileResource,
	}
}
