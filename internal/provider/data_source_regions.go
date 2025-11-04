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
	_ datasource.DataSource              = &regionsDataSource{}
	_ datasource.DataSourceWithConfigure = &regionsDataSource{}
)

// NewRegionsDataSource is a helper function to simplify the provider implementation.
func NewRegionsDataSource() datasource.DataSource {
	return &regionsDataSource{}
}

// regionsDataSource is the data source implementation.
type regionsDataSource struct {
	client *vmcloudapi.VMCloudAPIClient
}

// regionsDataSourceModel maps the data source schema data.
type regionsDataSourceModel struct {
	Regions []regionModel `tfsdk:"regions"`
}

// regionModel maps region data.
type regionModel struct {
	Name          types.String `tfsdk:"name"`
	CloudProvider types.String `tfsdk:"cloud_provider"`
}

// Metadata returns the data source type name.
func (d *regionsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_regions"
}

// Schema defines the schema for the data source.
func (d *regionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches the list of available regions for VictoriaMetrics Cloud deployments.",
		Attributes: map[string]schema.Attribute{
			"regions": schema.ListNestedAttribute{
				Description: "List of available regions.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "Name of the region.",
							Computed:    true,
						},
						"cloud_provider": schema.StringAttribute{
							Description: "Cloud provider of the region.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *regionsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *regionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state regionsDataSourceModel

	regions, err := d.client.ListRegions(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Regions",
			err.Error(),
		)
		return
	}

	// Map response to state
	for _, region := range regions {
		regionState := regionModel{
			Name:          types.StringValue(region.Name),
			CloudProvider: types.StringValue(region.CloudProvider.String()),
		}
		state.Regions = append(state.Regions, regionState)
	}

	// Set state
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
