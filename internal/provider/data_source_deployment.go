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
	_ datasource.DataSource              = &deploymentDataSource{}
	_ datasource.DataSourceWithConfigure = &deploymentDataSource{}
)

// NewDeploymentDataSource is a helper function to simplify the provider implementation.
func NewDeploymentDataSource() datasource.DataSource {
	return &deploymentDataSource{}
}

// deploymentDataSource is the data source implementation.
type deploymentDataSource struct {
	client *vmcloudapi.VMCloudAPIClient
}

// deploymentDataSourceModel maps the data source schema data.
type deploymentDataSourceModel struct {
	ID                types.String  `tfsdk:"id"`
	Name              types.String  `tfsdk:"name"`
	Type              types.String  `tfsdk:"type"`
	CloudProvider     types.String  `tfsdk:"cloud_provider"`
	Region            types.String  `tfsdk:"region"`
	Tier              types.Int64   `tfsdk:"tier"`
	Retention         types.Int64   `tfsdk:"retention"`
	RetentionUnit     types.String  `tfsdk:"retention_unit"`
	Deduplication     types.Int64   `tfsdk:"deduplication"`
	DeduplicationUnit types.String  `tfsdk:"deduplication_unit"`
	MaintenanceWindow types.String  `tfsdk:"maintenance_window"`
	Version           types.String  `tfsdk:"version"`
	Status            types.String  `tfsdk:"status"`
	CreatedAt         types.String  `tfsdk:"created_at"`
	AccessEndpoint    types.String  `tfsdk:"access_endpoint"`
	StorageSizeGb     types.Int64   `tfsdk:"storage_size_gb"`
	ComputeCost       types.Float64 `tfsdk:"compute_cost"`
	StorageCost       types.Float64 `tfsdk:"storage_cost"`
	TotalCost         types.Float64 `tfsdk:"total_cost"`
}

// Metadata returns the data source type name.
func (d *deploymentDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_deployment"
}

// Schema defines the schema for the data source.
func (d *deploymentDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches details of a specific VictoriaMetrics Cloud deployment.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier of the deployment.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "Human-readable name of the deployment.",
				Computed:    true,
			},
			"type": schema.StringAttribute{
				Description: "Type of the deployment.",
				Computed:    true,
			},
			"cloud_provider": schema.StringAttribute{
				Description: "Cloud provider for the deployment.",
				Computed:    true,
			},
			"region": schema.StringAttribute{
				Description: "Region of the deployment.",
				Computed:    true,
			},
			"tier": schema.Int64Attribute{
				Description: "Tier identifier for the deployment.",
				Computed:    true,
			},
			"retention": schema.Int64Attribute{
				Description: "Retention period for metrics.",
				Computed:    true,
			},
			"retention_unit": schema.StringAttribute{
				Description: "Retention period unit.",
				Computed:    true,
			},
			"deduplication": schema.Int64Attribute{
				Description: "Deduplication window.",
				Computed:    true,
			},
			"deduplication_unit": schema.StringAttribute{
				Description: "Deduplication window unit.",
				Computed:    true,
			},
			"maintenance_window": schema.StringAttribute{
				Description: "Maintenance window for the deployment.",
				Computed:    true,
			},
			"version": schema.StringAttribute{
				Description: "Version of VictoriaMetrics.",
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "Current status of the deployment.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "Timestamp of deployment creation.",
				Computed:    true,
			},
			"access_endpoint": schema.StringAttribute{
				Description: "API endpoint URL for the deployment.",
				Computed:    true,
			},
			"storage_size_gb": schema.Int64Attribute{
				Description: "Storage size in GB.",
				Computed:    true,
			},
			"compute_cost": schema.Float64Attribute{
				Description: "Monthly compute cost in USD.",
				Computed:    true,
			},
			"storage_cost": schema.Float64Attribute{
				Description: "Monthly storage cost in USD.",
				Computed:    true,
			},
			"total_cost": schema.Float64Attribute{
				Description: "Total monthly cost in USD.",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *deploymentDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *deploymentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state deploymentDataSourceModel

	// Get deployment ID from config
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	deployment, err := d.client.GetDeploymentDetails(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Deployment",
			err.Error(),
		)
		return
	}

	// Map response to state
	state.Name = types.StringValue(deployment.Name)
	state.Type = types.StringValue(deployment.Type.String())
	state.CloudProvider = types.StringValue(deployment.CloudProvider.String())
	state.Region = types.StringValue(deployment.Region)
	state.Tier = types.Int64Value(int64(deployment.Tier))
	state.Retention = types.Int64Value(int64(deployment.RetentionValue))
	state.RetentionUnit = types.StringValue(string(deployment.RetentionUnit))
	state.Deduplication = types.Int64Value(int64(deployment.DeduplicationValue))
	state.DeduplicationUnit = types.StringValue(string(deployment.DeduplicationUnit))
	state.MaintenanceWindow = types.StringValue(string(deployment.MaintenanceWindow))
	state.Version = types.StringValue(deployment.Version)
	state.Status = types.StringValue(deployment.Status.String())
	state.CreatedAt = types.StringValue(deployment.CreatedAt.Format(time.RFC3339))
	state.AccessEndpoint = types.StringValue(deployment.AccessEndpoint)
	state.StorageSizeGb = types.Int64Value(int64(deployment.StorageSizeGb))
	state.ComputeCost = types.Float64Value(deployment.Price.ComputeCost)
	state.StorageCost = types.Float64Value(deployment.Price.StorageCost)
	state.TotalCost = types.Float64Value(deployment.Price.TotalCost)

	// Set state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
