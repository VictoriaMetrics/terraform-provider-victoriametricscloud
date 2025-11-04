package provider

import (
	"context"
	"fmt"
	"time"

	vmcloudapi "github.com/VictoriaMetrics/victoriametrics-cloud-api-go/v1"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &deploymentsDataSource{}
	_ datasource.DataSourceWithConfigure = &deploymentsDataSource{}
)

// NewDeploymentsDataSource is a helper function to simplify the provider implementation.
func NewDeploymentsDataSource() datasource.DataSource {
	return &deploymentsDataSource{}
}

// deploymentsDataSource is the data source implementation.
type deploymentsDataSource struct {
	client *vmcloudapi.VMCloudAPIClient
}

// deploymentsDataSourceModel maps the data source schema data.
type deploymentsDataSourceModel struct {
	Deployments []deploymentSummaryModel `tfsdk:"deployments"`
}

// deploymentSummaryModel maps deployment summary data.
type deploymentSummaryModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Type          types.String `tfsdk:"type"`
	Tier          types.Int64  `tfsdk:"tier"`
	Version       types.String `tfsdk:"version"`
	CloudProvider types.String `tfsdk:"cloud_provider"`
	Region        types.String `tfsdk:"region"`
	Status        types.String `tfsdk:"status"`
	CreatedAt     types.String `tfsdk:"created_at"`
}

// Metadata returns the data source type name.
func (d *deploymentsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_deployments"
}

// Schema defines the schema for the data source.
func (d *deploymentsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches the list of VictoriaMetrics Cloud deployments.",
		Attributes: map[string]schema.Attribute{
			"deployments": schema.ListNestedAttribute{
				Description: "List of deployments.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "Unique identifier of the deployment.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "Human-readable name of the deployment.",
							Computed:    true,
						},
						"type": schema.StringAttribute{
							Description: "Type of the deployment.",
							Computed:    true,
						},
						"tier": schema.Int64Attribute{
							Description: "Tier identifier.",
							Computed:    true,
						},
						"version": schema.StringAttribute{
							Description: "Version of VictoriaMetrics.",
							Computed:    true,
						},
						"cloud_provider": schema.StringAttribute{
							Description: "Cloud provider.",
							Computed:    true,
						},
						"region": schema.StringAttribute{
							Description: "Region of the deployment.",
							Computed:    true,
						},
						"status": schema.StringAttribute{
							Description: "Current status.",
							Computed:    true,
						},
						"created_at": schema.StringAttribute{
							Description: "Timestamp of creation.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *deploymentsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *deploymentsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state deploymentsDataSourceModel
	state.Deployments = []deploymentSummaryModel{}

	deployments, err := d.client.ListDeployments(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Deployments",
			err.Error(),
		)
		return
	}

	// Map response to state
	for _, deployment := range deployments {
		deploymentState := deploymentSummaryModel{
			ID:            types.StringValue(deployment.ID),
			Name:          types.StringValue(deployment.Name),
			Type:          types.StringValue(deployment.Type.String()),
			Tier:          types.Int64Value(int64(deployment.Tier)),
			Version:       types.StringValue(deployment.Version),
			CloudProvider: types.StringValue(deployment.CloudProvider.String()),
			Region:        types.StringValue(deployment.Region),
			Status:        types.StringValue(deployment.Status.String()),
			CreatedAt:     types.StringValue(deployment.CreatedAt.Format(time.RFC3339)),
		}
		state.Deployments = append(state.Deployments, deploymentState)
	}

	// Set state
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
