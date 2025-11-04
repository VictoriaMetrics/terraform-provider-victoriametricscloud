package provider

import (
	"context"
	"fmt"

	vmcloudapi "github.com/VictoriaMetrics/victoriametrics-cloud-api-go/v1"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &cloudProvidersDataSource{}
	_ datasource.DataSourceWithConfigure = &cloudProvidersDataSource{}
)

// NewCloudProvidersDataSource is a helper function to simplify the provider implementation.
func NewCloudProvidersDataSource() datasource.DataSource {
	return &cloudProvidersDataSource{}
}

// cloudProvidersDataSource is the data source implementation.
type cloudProvidersDataSource struct {
	client *vmcloudapi.VMCloudAPIClient
}

// cloudProvidersDataSourceModel maps the data source schema data.
type cloudProvidersDataSourceModel struct {
	CloudProviders []cloudProviderModel `tfsdk:"cloud_providers"`
}

// cloudProviderModel maps cloud provider data.
type cloudProviderModel struct {
	ID  types.String `tfsdk:"id"`
	URL types.String `tfsdk:"url"`
}

// Metadata returns the data source type name.
func (d *cloudProvidersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cloud_providers"
}

// Schema defines the schema for the data source.
func (d *cloudProvidersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches the list of available cloud providers for VictoriaMetrics Cloud deployments.",
		Attributes: map[string]schema.Attribute{
			"cloud_providers": schema.ListNestedAttribute{
				Description: "List of available cloud providers.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "Unique identifier of the cloud provider.",
							Computed:    true,
						},
						"url": schema.StringAttribute{
							Description: "URL to the cloud provider website.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *cloudProvidersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*vmcloudapi.VMCloudAPIClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *vmcloudapi.VMCloudAPIClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

// Read refreshes the Terraform state with the latest data.
func (d *cloudProvidersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state cloudProvidersDataSourceModel

	providers, err := d.client.ListCloudProviders(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Cloud Providers",
			err.Error(),
		)
		return
	}

	// Map response to state
	for _, provider := range providers {
		providerState := cloudProviderModel{
			ID:  types.StringValue(provider.ID.String()),
			URL: types.StringValue(provider.URL),
		}
		state.CloudProviders = append(state.CloudProviders, providerState)
	}

	// Set state
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
