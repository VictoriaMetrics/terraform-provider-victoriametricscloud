package provider

import (
	"context"
	"fmt"
	"time"

	vmcloudapi "github.com/VictoriaMetrics/victoriametrics-cloud-api-go/v1"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &deploymentResource{}
	_ resource.ResourceWithConfigure   = &deploymentResource{}
	_ resource.ResourceWithImportState = &deploymentResource{}
)

// NewDeploymentResource is a helper function to simplify the provider implementation.
func NewDeploymentResource() resource.Resource {
	return &deploymentResource{}
}

// deploymentResource is the resource implementation.
type deploymentResource struct {
	client *vmcloudapi.VMCloudAPIClient
}

// deploymentResourceModel maps the resource schema data.
type deploymentResourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	Type              types.String `tfsdk:"type"`
	CloudProvider     types.String `tfsdk:"cloud_provider"`
	Region            types.String `tfsdk:"region"`
	Tier              types.Int64  `tfsdk:"tier"`
	StorageSize       types.Int64  `tfsdk:"storage_size"`
	StorageSizeUnit   types.String `tfsdk:"storage_size_unit"`
	Retention         types.Int64  `tfsdk:"retention"`
	RetentionUnit     types.String `tfsdk:"retention_unit"`
	Deduplication     types.Int64  `tfsdk:"deduplication"`
	DeduplicationUnit types.String `tfsdk:"deduplication_unit"`
	MaintenanceWindow types.String `tfsdk:"maintenance_window"`
	SingleFlags       types.List   `tfsdk:"single_flags"`
	SelectFlags       types.List   `tfsdk:"select_flags"`
	StorageFlags      types.List   `tfsdk:"storage_flags"`
	InsertFlags       types.List   `tfsdk:"insert_flags"`
	Version           types.String `tfsdk:"version"`
	Status            types.String `tfsdk:"status"`
	CreatedAt         types.String `tfsdk:"created_at"`
	AccessEndpoint    types.String `tfsdk:"access_endpoint"`
}

// Metadata returns the resource type name.
func (r *deploymentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_deployment"
}

// Schema defines the schema for the resource.
func (r *deploymentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a VictoriaMetrics Cloud deployment.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier of the deployment.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Human-readable name of the deployment.",
				Required:    true,
			},
			"type": schema.StringAttribute{
				Description: "Type of the deployment. Valid values: 'single_node', 'cluster'.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cloud_provider": schema.StringAttribute{
				Description: "Cloud provider for the deployment. Valid values: 'aws'.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"region": schema.StringAttribute{
				Description: "Region of the deployment in the cloud provider.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"tier": schema.Int64Attribute{
				Description: "Tier identifier for the deployment.",
				Required:    true,
			},
			"storage_size": schema.Int64Attribute{
				Description: "Storage size in units specified in storage_size_unit.",
				Required:    true,
			},
			"storage_size_unit": schema.StringAttribute{
				Description: "Storage size unit. Valid values: 'GB', 'TB'.",
				Required:    true,
			},
			"retention": schema.Int64Attribute{
				Description: "Retention period for metrics.",
				Required:    true,
			},
			"retention_unit": schema.StringAttribute{
				Description: "Retention period unit. Valid values: 'd' (days), 'm' (months).",
				Required:    true,
			},
			"deduplication": schema.Int64Attribute{
				Description: "Deduplication window for the deployment.",
				Required:    true,
			},
			"deduplication_unit": schema.StringAttribute{
				Description: "Deduplication window unit. Valid values: 'ms' (milliseconds), 's' (seconds).",
				Required:    true,
			},
			"maintenance_window": schema.StringAttribute{
				Description: "Maintenance window for the deployment. Valid values: 'Sat-Sun 3-4am', 'Mon-Fri 4-5am'.",
				Required:    true,
			},
			"single_flags": schema.ListAttribute{
				Description: "Custom command-line flags for the vmsingle component.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"select_flags": schema.ListAttribute{
				Description: "Custom command-line flags for the vmselect component.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"storage_flags": schema.ListAttribute{
				Description: "Custom command-line flags for the vmstorage component.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"insert_flags": schema.ListAttribute{
				Description: "Custom command-line flags for the vminsert component.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"version": schema.StringAttribute{
				Description: "Version of VictoriaMetrics used in the deployment.",
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "Current status of the deployment.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "Timestamp of deployment creation.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"access_endpoint": schema.StringAttribute{
				Description: "API endpoint URL for the deployment.",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *deploymentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*vmcloudapi.VMCloudAPIClient)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *vmcloudapi.VMCloudAPIClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

// Create creates the resource and sets the initial Terraform state.
func (r *deploymentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan deploymentResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create the deployment
	createRequest := vmcloudapi.DeploymentCreationRequest{
		Name:              plan.Name.ValueString(),
		Type:              vmcloudapi.DeploymentType(plan.Type.ValueString()),
		Provider:          vmcloudapi.DeploymentCloudProvider(plan.CloudProvider.ValueString()),
		Region:            plan.Region.ValueString(),
		Tier:              uint32(plan.Tier.ValueInt64()),
		StorageSize:       uint64(plan.StorageSize.ValueInt64()),
		StorageSizeUnit:   vmcloudapi.StorageUnit(plan.StorageSizeUnit.ValueString()),
		Retention:         uint32(plan.Retention.ValueInt64()),
		RetentionUnit:     vmcloudapi.DurationUnit(plan.RetentionUnit.ValueString()),
		Deduplication:     uint32(plan.Deduplication.ValueInt64()),
		DeduplicationUnit: vmcloudapi.DurationUnit(plan.DeduplicationUnit.ValueString()),
		MaintenanceWindow: vmcloudapi.MaintenanceWindow(plan.MaintenanceWindow.ValueString()),
	}

	deployment, err := r.client.CreateDeployment(ctx, createRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating deployment",
			"Could not create deployment, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response to state
	plan.ID = types.StringValue(deployment.ID)
	plan.Version = types.StringValue(deployment.Version)
	plan.Status = types.StringValue(deployment.Status.String())
	plan.CreatedAt = types.StringValue(deployment.CreatedAt.Format(time.RFC3339))
	plan.AccessEndpoint = types.StringValue(deployment.AccessEndpoint)

	tflog.Trace(ctx, "created deployment", map[string]any{"id": deployment.ID})

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data.
func (r *deploymentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state deploymentResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get refreshed deployment value from API
	deployment, err := r.client.GetDeploymentDetails(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Deployment",
			"Could not read deployment ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Update state with refreshed values
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

	// Calculate storage_size from storage_size_gb
	if !state.StorageSizeUnit.IsNull() {
		switch state.StorageSizeUnit.ValueString() {
		case "GB":
			state.StorageSize = types.Int64Value(int64(deployment.StorageSizeGb))
		case "TB":
			state.StorageSize = types.Int64Value(int64(deployment.StorageSizeGb / 1024))
		}
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *deploymentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan deploymentResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Prepare flags
	flags := vmcloudapi.DeploymentFlags{
		SingleFlags:  []string{},
		SelectFlags:  []string{},
		StorageFlags: []string{},
		InsertFlags:  []string{},
	}

	if !plan.SingleFlags.IsNull() {
		diags = plan.SingleFlags.ElementsAs(ctx, &flags.SingleFlags, false)
		resp.Diagnostics.Append(diags...)
	}
	if !plan.SelectFlags.IsNull() {
		diags = plan.SelectFlags.ElementsAs(ctx, &flags.SelectFlags, false)
		resp.Diagnostics.Append(diags...)
	}
	if !plan.StorageFlags.IsNull() {
		diags = plan.StorageFlags.ElementsAs(ctx, &flags.StorageFlags, false)
		resp.Diagnostics.Append(diags...)
	}
	if !plan.InsertFlags.IsNull() {
		diags = plan.InsertFlags.ElementsAs(ctx, &flags.InsertFlags, false)
		resp.Diagnostics.Append(diags...)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Update the deployment
	updateRequest := vmcloudapi.DeploymentUpdateRequest{
		Name:              plan.Name.ValueString(),
		Tier:              uint32(plan.Tier.ValueInt64()),
		StorageSize:       uint64(plan.StorageSize.ValueInt64()),
		StorageSizeUnit:   vmcloudapi.StorageUnit(plan.StorageSizeUnit.ValueString()),
		Retention:         uint32(plan.Retention.ValueInt64()),
		RetentionUnit:     vmcloudapi.DurationUnit(plan.RetentionUnit.ValueString()),
		Deduplication:     uint32(plan.Deduplication.ValueInt64()),
		DeduplicationUnit: vmcloudapi.DurationUnit(plan.DeduplicationUnit.ValueString()),
		MaintenanceWindow: vmcloudapi.MaintenanceWindow(plan.MaintenanceWindow.ValueString()),
		Flags:             flags,
	}

	deployment, err := r.client.UpdateDeployment(ctx, plan.ID.ValueString(), updateRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating deployment",
			"Could not update deployment, unexpected error: "+err.Error(),
		)
		return
	}

	// Update state with values from API
	plan.Version = types.StringValue(deployment.Version)
	plan.Status = types.StringValue(deployment.Status.String())
	plan.AccessEndpoint = types.StringValue(deployment.AccessEndpoint)

	tflog.Trace(ctx, "updated deployment", map[string]any{"id": deployment.ID})

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *deploymentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state deploymentResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the deployment
	err := r.client.DeleteDeployment(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting deployment",
			"Could not delete deployment, unexpected error: "+err.Error(),
		)
		return
	}

	tflog.Trace(ctx, "deleted deployment", map[string]any{"id": state.ID.ValueString()})
}

// ImportState imports the resource state.
func (r *deploymentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Retrieve import ID and save to id attribute
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
