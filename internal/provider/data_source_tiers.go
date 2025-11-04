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
	_ datasource.DataSource              = &tiersDataSource{}
	_ datasource.DataSourceWithConfigure = &tiersDataSource{}
)

// NewTiersDataSource is a helper function to simplify the provider implementation.
func NewTiersDataSource() datasource.DataSource {
	return &tiersDataSource{}
}

// tiersDataSource is the data source implementation.
type tiersDataSource struct {
	client *vmcloudapi.VMCloudAPIClient
}

// tiersDataSourceModel maps the data source schema data.
type tiersDataSourceModel struct {
	Tiers []tierModel `tfsdk:"tiers"`
}

// tierModel maps tier data.
type tierModel struct {
	ID                            types.Int64   `tfsdk:"id"`
	Type                          types.String  `tfsdk:"type"`
	CloudProvider                 types.String  `tfsdk:"cloud_provider"`
	Name                          types.String  `tfsdk:"name"`
	ComputeCostPerHour            types.Float64 `tfsdk:"compute_cost_per_hour"`
	IngestionRate                 types.Int64   `tfsdk:"ingestion_rate"`
	ActiveTimeSeries              types.Int64   `tfsdk:"active_time_series"`
	NewSeriesOver24h              types.Int64   `tfsdk:"new_series_over_24h"`
	DatapointsReadRate            types.Int64   `tfsdk:"datapoints_read_rate"`
	SeriesReadPerQuery            types.Int64   `tfsdk:"series_read_per_query"`
	AccessTokenConcurrentRequests types.Int64   `tfsdk:"access_token_concurrent_requests"`
}

// Metadata returns the data source type name.
func (d *tiersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tiers"
}

// Schema defines the schema for the data source.
func (d *tiersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches the list of available tiers for VictoriaMetrics Cloud deployments.",
		Attributes: map[string]schema.Attribute{
			"tiers": schema.ListNestedAttribute{
				Description: "List of available tiers.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							Description: "Unique identifier of the tier.",
							Computed:    true,
						},
						"type": schema.StringAttribute{
							Description: "Type of deployment (single_node or cluster).",
							Computed:    true,
						},
						"cloud_provider": schema.StringAttribute{
							Description: "Cloud provider for this tier.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "Name of the tier.",
							Computed:    true,
						},
						"compute_cost_per_hour": schema.Float64Attribute{
							Description: "Compute cost per hour in USD.",
							Computed:    true,
						},
						"ingestion_rate": schema.Int64Attribute{
							Description: "Maximum ingestion rate (samples per second).",
							Computed:    true,
						},
						"active_time_series": schema.Int64Attribute{
							Description: "Maximum number of active time series.",
							Computed:    true,
						},
						"new_series_over_24h": schema.Int64Attribute{
							Description: "Maximum number of new series over 24 hours.",
							Computed:    true,
						},
						"datapoints_read_rate": schema.Int64Attribute{
							Description: "Maximum datapoints read rate.",
							Computed:    true,
						},
						"series_read_per_query": schema.Int64Attribute{
							Description: "Maximum series read per query.",
							Computed:    true,
						},
						"access_token_concurrent_requests": schema.Int64Attribute{
							Description: "Maximum concurrent requests per access token.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *tiersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *tiersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state tiersDataSourceModel

	tiers, err := d.client.ListTiers(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Tiers",
			err.Error(),
		)
		return
	}

	// Map response to state
	for _, tier := range tiers {
		tierState := tierModel{
			ID:                            types.Int64Value(int64(tier.ID)),
			Type:                          types.StringValue(tier.Type.String()),
			CloudProvider:                 types.StringValue(tier.CloudProvider.String()),
			Name:                          types.StringValue(tier.Name),
			ComputeCostPerHour:            types.Float64Value(tier.ComputeCostPerHour),
			IngestionRate:                 types.Int64Value(int64(tier.IngestionRate)),
			ActiveTimeSeries:              types.Int64Value(int64(tier.ActiveTimeSeries)),
			NewSeriesOver24h:              types.Int64Value(int64(tier.NewSeriesOver24h)),
			DatapointsReadRate:            types.Int64Value(int64(tier.DatapointsReadRate)),
			SeriesReadPerQuery:            types.Int64Value(int64(tier.SeriesReadPerQuery)),
			AccessTokenConcurrentRequests: types.Int64Value(int64(tier.AccessTokenConcurrentRequests)),
		}
		state.Tiers = append(state.Tiers, tierState)
	}

	// Set state
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
