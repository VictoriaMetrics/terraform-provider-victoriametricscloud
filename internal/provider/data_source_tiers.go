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
	Type  types.String `tfsdk:"type"`
	Tiers []tierModel  `tfsdk:"tiers"`
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
	IngestionRateBytes            types.Int64   `tfsdk:"ingestion_rate_bytes"`
	ActiveLogStreams              types.Int64   `tfsdk:"active_log_streams"`
	NewStreamsOver24h             types.Int64   `tfsdk:"new_streams_over_24h"`
	DataReadRate                  types.Int64   `tfsdk:"data_read_rate"`
	BytesPerQuery                 types.Int64   `tfsdk:"bytes_per_query"`
	AccessTokenConcurrentRequests types.Int64   `tfsdk:"access_token_concurrent_requests"`
	AccessTokenLimit              types.Int64   `tfsdk:"access_token_limit"`
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
			"type": schema.StringAttribute{
				Description: "Restrict the results to a single deployment type. Valid values: 'single_node', 'cluster', 'vlogs_single' (VictoriaLogs), 'vtraces_single' (VictoriaTraces). Tiers of every type are returned when unset.",
				Optional:    true,
			},
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
							Description: "Type of deployment ('single_node', 'cluster', 'vlogs_single', or 'vtraces_single').",
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
							Description: "Maximum ingestion rate (samples per second). Set for 'single_node' and 'cluster' tiers only.",
							Computed:    true,
						},
						"active_time_series": schema.Int64Attribute{
							Description: "Maximum number of active time series. Set for 'single_node' and 'cluster' tiers only.",
							Computed:    true,
						},
						"new_series_over_24h": schema.Int64Attribute{
							Description: "Maximum number of new series over 24 hours. Set for 'single_node' and 'cluster' tiers only.",
							Computed:    true,
						},
						"datapoints_read_rate": schema.Int64Attribute{
							Description: "Maximum datapoints read rate. Set for 'single_node' and 'cluster' tiers only.",
							Computed:    true,
						},
						"series_read_per_query": schema.Int64Attribute{
							Description: "Maximum series read per query. Set for 'single_node' and 'cluster' tiers only.",
							Computed:    true,
						},
						"ingestion_rate_bytes": schema.Int64Attribute{
							Description: "Maximum ingestion rate in bytes per second. Set for 'vlogs_single' and 'vtraces_single' tiers only.",
							Computed:    true,
						},
						"active_log_streams": schema.Int64Attribute{
							Description: "Maximum number of active streams. Set for 'vlogs_single' and 'vtraces_single' tiers only.",
							Computed:    true,
						},
						"new_streams_over_24h": schema.Int64Attribute{
							Description: "Maximum number of new streams over 24 hours. Set for 'vlogs_single' and 'vtraces_single' tiers only.",
							Computed:    true,
						},
						"data_read_rate": schema.Int64Attribute{
							Description: "Maximum read rate in bytes per second. Set for 'vlogs_single' and 'vtraces_single' tiers only.",
							Computed:    true,
						},
						"bytes_per_query": schema.Int64Attribute{
							Description: "Maximum number of bytes scanned per query. Set for 'vlogs_single' and 'vtraces_single' tiers only.",
							Computed:    true,
						},
						"access_token_concurrent_requests": schema.Int64Attribute{
							Description: "Maximum concurrent requests per access token.",
							Computed:    true,
						},
						"access_token_limit": schema.Int64Attribute{
							Description: "Maximum number of access tokens for deployments of this tier.",
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
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var options []vmcloudapi.ListOption
	if !state.Type.IsNull() {
		options = append(options, vmcloudapi.WithDeploymentType(vmcloudapi.DeploymentType(state.Type.ValueString())))
	}

	tiers, err := d.client.ListTiers(ctx, options...)
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
			AccessTokenConcurrentRequests: types.Int64Value(int64(tier.AccessTokenConcurrentRequests)),
			AccessTokenLimit:              types.Int64Value(int64(tier.AccessTokenLimit)),
			IngestionRate:                 types.Int64Null(),
			ActiveTimeSeries:              types.Int64Null(),
			NewSeriesOver24h:              types.Int64Null(),
			DatapointsReadRate:            types.Int64Null(),
			SeriesReadPerQuery:            types.Int64Null(),
			IngestionRateBytes:            types.Int64Null(),
			ActiveLogStreams:              types.Int64Null(),
			NewStreamsOver24h:             types.Int64Null(),
			DataReadRate:                  types.Int64Null(),
			BytesPerQuery:                 types.Int64Null(),
		}

		switch {
		case tier.Metrics != nil:
			tierState.IngestionRate = types.Int64Value(int64(tier.Metrics.IngestionRate))
			tierState.ActiveTimeSeries = types.Int64Value(int64(tier.Metrics.ActiveTimeSeries))
			tierState.NewSeriesOver24h = types.Int64Value(int64(tier.Metrics.NewSeriesOver24h))
			tierState.DatapointsReadRate = types.Int64Value(int64(tier.Metrics.DatapointsReadRate))
			tierState.SeriesReadPerQuery = types.Int64Value(int64(tier.Metrics.SeriesReadPerQuery))
		case tier.Logs != nil:
			tierState.IngestionRateBytes = types.Int64Value(tier.Logs.IngestionRateBytes)
			tierState.ActiveLogStreams = types.Int64Value(int64(tier.Logs.ActiveLogStreams))
			tierState.NewStreamsOver24h = types.Int64Value(int64(tier.Logs.NewStreamsOver24h))
			tierState.DataReadRate = types.Int64Value(tier.Logs.DataReadRate)
			tierState.BytesPerQuery = types.Int64Value(tier.Logs.BytesPerQuery)
		case tier.Traces != nil:
			tierState.IngestionRateBytes = types.Int64Value(tier.Traces.IngestionRateBytes)
			tierState.ActiveLogStreams = types.Int64Value(int64(tier.Traces.ActiveLogStreams))
			tierState.NewStreamsOver24h = types.Int64Value(int64(tier.Traces.NewStreamsOver24h))
			tierState.DataReadRate = types.Int64Value(tier.Traces.DataReadRate)
			tierState.BytesPerQuery = types.Int64Value(tier.Traces.BytesPerQuery)
		}

		state.Tiers = append(state.Tiers, tierState)
	}

	// Set state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
