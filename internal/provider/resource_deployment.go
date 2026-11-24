package provider

import (
	"context"
	"fmt"
	"time"

	vmcloudapi "github.com/VictoriaMetrics/victoriametrics-cloud-api-go/v1"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
	_ resource.Resource                   = &deploymentResource{}
	_ resource.ResourceWithConfigure      = &deploymentResource{}
	_ resource.ResourceWithImportState    = &deploymentResource{}
	_ resource.ResourceWithValidateConfig = &deploymentResource{}
)

// deploymentPollInterval is how often the provider checks whether a deployment
// accepts changes yet. A variable so that tests can shorten it.
var deploymentPollInterval = 10 * time.Second

// deploymentWaitTimeout bounds how long the provider waits for a deployment to be
// running or unlocked. Terraform sets no deadline on provider calls, so without it a
// stuck deployment would hang the apply. A variable so that tests can shorten it.
var deploymentWaitTimeout = 60 * time.Minute

// maxConsecutiveReadErrors is how many failed reads in a row waitUntilRunning
// tolerates before giving up, so that a transient API error does not fail it.
const maxConsecutiveReadErrors = 3

// privateFlagsPending is the private state key that marks a deployment whose flags
// could not be applied after it was created. Read then reports the flags as unset,
// so that the next plan shows them and the next apply sends them again.
const privateFlagsPending = "flags_pending"

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
	Metrics           types.Object `tfsdk:"metrics"`
	Single            types.Object `tfsdk:"single"`
	Cluster           types.Object `tfsdk:"cluster"`
}

// deploymentMetricsModel groups the settings specific to 'single_node' and 'cluster'
// deployments, shared by the deployment resource and data source.
type deploymentMetricsModel struct {
	Deduplication     types.Int64  `tfsdk:"deduplication"`
	DeduplicationUnit types.String `tfsdk:"deduplication_unit"`
}

// singleComponentFlagsModel groups the flags of the single component, used by
// 'single_node', 'vlogs_single' and 'vtraces_single' deployments.
type singleComponentFlagsModel struct {
	Flags types.List `tfsdk:"flags"`
}

// clusterComponentFlagsModel groups the flags of the cluster components, used by
// 'cluster' deployments.
type clusterComponentFlagsModel struct {
	SelectFlags  types.List `tfsdk:"select_flags"`
	StorageFlags types.List `tfsdk:"storage_flags"`
	InsertFlags  types.List `tfsdk:"insert_flags"`
}

var (
	metricsAttrTypes = map[string]attr.Type{
		"deduplication":      types.Int64Type,
		"deduplication_unit": types.StringType,
	}
	singleAttrTypes = map[string]attr.Type{
		"flags": types.ListType{ElemType: types.StringType},
	}
	clusterAttrTypes = map[string]attr.Type{
		"select_flags":  types.ListType{ElemType: types.StringType},
		"storage_flags": types.ListType{ElemType: types.StringType},
		"insert_flags":  types.ListType{ElemType: types.StringType},
	}
)

// deduplicationMetrics groups the deduplication window into a metrics block, nil when
// either value is null.
func deduplicationMetrics(dedup types.Int64, dedupUnit types.String) *deploymentMetricsModel {
	if dedup.IsNull() || dedupUnit.IsNull() {
		return nil
	}
	return &deploymentMetricsModel{Deduplication: dedup, DeduplicationUnit: dedupUnit}
}

// flagsGroupsForType groups the component flags by topology: a 'cluster' deployment
// gets the cluster block, every other type gets the single block.
func flagsGroupsForType(deploymentType string, singleFlags, selectFlags, storageFlags, insertFlags types.List) (*singleComponentFlagsModel, *clusterComponentFlagsModel) {
	if deploymentType == vmcloudapi.DeploymentTypeCluster.String() {
		return nil, &clusterComponentFlagsModel{SelectFlags: selectFlags, StorageFlags: storageFlags, InsertFlags: insertFlags}
	}
	return &singleComponentFlagsModel{Flags: singleFlags}, nil
}

// objectOrNull converts a nested model to a types.Object, null when v is nil.
func objectOrNull[T any](ctx context.Context, attrTypes map[string]attr.Type, v *T) (types.Object, diag.Diagnostics) {
	if v == nil {
		return types.ObjectNull(attrTypes), nil
	}
	return types.ObjectValueFrom(ctx, attrTypes, v)
}

// setDeduplication fills the deduplication attributes and the metrics object of m.
// Deployment types without a deduplication window get nulls.
func setDeduplication(ctx context.Context, m *deploymentResourceModel, dedup types.Int64, dedupUnit types.String) diag.Diagnostics {
	if !vmcloudapi.DeploymentType(m.Type.ValueString()).SupportsDeduplication() {
		dedup, dedupUnit = types.Int64Null(), types.StringNull()
	}
	m.Deduplication, m.DeduplicationUnit = dedup, dedupUnit
	var diags diag.Diagnostics
	m.Metrics, diags = objectOrNull(ctx, metricsAttrTypes, deduplicationMetrics(dedup, dedupUnit))
	return diags
}

// setGroupedFlags fills the single and cluster objects of m from its flag attributes.
func setGroupedFlags(ctx context.Context, m *deploymentResourceModel) diag.Diagnostics {
	var diags, d diag.Diagnostics
	single, cluster := flagsGroupsForType(m.Type.ValueString(), m.SingleFlags, m.SelectFlags, m.StorageFlags, m.InsertFlags)
	m.Single, d = objectOrNull(ctx, singleAttrTypes, single)
	diags.Append(d...)
	m.Cluster, d = objectOrNull(ctx, clusterAttrTypes, cluster)
	diags.Append(d...)
	return diags
}

// deduplicationFromAPI returns the deduplication window the API reports, nulls when
// the deployment has none.
func deduplicationFromAPI(deployment vmcloudapi.DeploymentInfo) (types.Int64, types.String) {
	if value, unit, ok := deployment.Deduplication(); ok {
		return types.Int64Value(int64(value)), types.StringValue(string(unit))
	}
	return types.Int64Null(), types.StringNull()
}

// setComputedAfterApply fills the attributes of the planned deployment m that are
// unknown until apply from the API response.
func setComputedAfterApply(ctx context.Context, m *deploymentResourceModel, deployment vmcloudapi.DeploymentInfo) diag.Diagnostics {
	setComputedFromAPI(m, deployment)
	dedup, dedupUnit := m.Deduplication, m.DeduplicationUnit
	if dedup.IsUnknown() || dedupUnit.IsUnknown() {
		dedup, dedupUnit = deduplicationFromAPI(deployment)
	}
	diags := setDeduplication(ctx, m, dedup, dedupUnit)
	diags.Append(setGroupedFlags(ctx, m)...)
	return diags
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
				Description: "Type of the deployment. Valid values: 'single_node', 'cluster', 'vlogs_single' (VictoriaLogs), 'vtraces_single' (VictoriaTraces). Changing it replaces the deployment.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"cloud_provider": schema.StringAttribute{
				Description: "Cloud provider for the deployment. Valid values: 'aws'. Changing it replaces the deployment.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"region": schema.StringAttribute{
				Description: "Region of the deployment in the cloud provider. Changing it replaces the deployment.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
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
				Description: "Deduplication window for the deployment. Required for 'single_node' and 'cluster' deployments; cannot be set for 'vlogs_single' and 'vtraces_single', which have no deduplication window.",
				Optional:    true,
				Computed:    true,
			},
			"deduplication_unit": schema.StringAttribute{
				Description: "Deduplication window unit. Valid values: 'ms' (milliseconds), 's' (seconds). Required for 'single_node' and 'cluster' deployments; cannot be set for 'vlogs_single' and 'vtraces_single'.",
				Optional:    true,
				Computed:    true,
			},
			"metrics": schema.SingleNestedAttribute{
				Description: "Settings specific to 'single_node' and 'cluster' deployments. Null for other deployment types.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"deduplication": schema.Int64Attribute{
						Description: "Deduplication window for the deployment.",
						Computed:    true,
					},
					"deduplication_unit": schema.StringAttribute{
						Description: "Deduplication window unit.",
						Computed:    true,
					},
				},
			},
			"maintenance_window": schema.StringAttribute{
				Description: "Maintenance window for the deployment. Valid values: 'Sat-Sun 3-4am', 'Mon-Fri 4-5am'.",
				Required:    true,
			},
			"single_flags": schema.ListAttribute{
				Description: "Custom command-line flags for the single component of 'single_node', 'vlogs_single' and 'vtraces_single' deployments. The API accepts flags only after a deployment is created, so creating a deployment with flags waits until it is running.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"select_flags": schema.ListAttribute{
				Description: "Custom command-line flags for the vmselect component of 'cluster' deployments.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"storage_flags": schema.ListAttribute{
				Description: "Custom command-line flags for the vmstorage component of 'cluster' deployments.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"insert_flags": schema.ListAttribute{
				Description: "Custom command-line flags for the vminsert component of 'cluster' deployments.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"single": schema.SingleNestedAttribute{
				Description: "Settings specific to 'single_node', 'vlogs_single' and 'vtraces_single' deployments. Null for 'cluster' deployments.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"flags": schema.ListAttribute{
						Description: "Custom command-line flags for the single component.",
						Computed:    true,
						ElementType: types.StringType,
					},
				},
			},
			"cluster": schema.SingleNestedAttribute{
				Description: "Settings specific to 'cluster' deployments. Null for other deployment types.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"select_flags": schema.ListAttribute{
						Description: "Custom command-line flags for the vmselect component.",
						Computed:    true,
						ElementType: types.StringType,
					},
					"storage_flags": schema.ListAttribute{
						Description: "Custom command-line flags for the vmstorage component.",
						Computed:    true,
						ElementType: types.StringType,
					},
					"insert_flags": schema.ListAttribute{
						Description: "Custom command-line flags for the vminsert component.",
						Computed:    true,
						ElementType: types.StringType,
					},
				},
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

// ValidateConfig checks the deduplication window at plan time, which the API would
// otherwise reject only during apply.
func (r *deploymentResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config deploymentResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Values that are not known until apply may still turn out null, so only known
	// values are checked.
	if config.Deduplication.IsUnknown() || config.DeduplicationUnit.IsUnknown() || config.Type.IsUnknown() || config.Type.IsNull() {
		return
	}
	if config.Deduplication.IsNull() != config.DeduplicationUnit.IsNull() {
		resp.Diagnostics.AddAttributeError(path.Root("deduplication"), "Incomplete deduplication window",
			"\"deduplication\" and \"deduplication_unit\" must be set together.")
		return
	}
	deploymentType := vmcloudapi.DeploymentType(config.Type.ValueString())
	switch {
	case deploymentType.SupportsDeduplication() && config.Deduplication.IsNull():
		resp.Diagnostics.AddAttributeError(path.Root("deduplication"), "Missing deduplication window",
			fmt.Sprintf("%q deployments require \"deduplication\" and \"deduplication_unit\".", deploymentType))
	case !deploymentType.SupportsDeduplication() && !config.Deduplication.IsNull():
		resp.Diagnostics.AddAttributeError(path.Root("deduplication"), "Deduplication not supported",
			fmt.Sprintf("%q deployments have no deduplication window: remove \"deduplication\" and \"deduplication_unit\".", deploymentType))
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *deploymentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan deploymentResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	flags, diags := flagsRequest(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create the deployment
	dedup, dedupUnit := deduplicationPointers(plan.Deduplication, plan.DeduplicationUnit)
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
		Deduplication:     dedup,
		DeduplicationUnit: dedupUnit,
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
	plan.ID = types.StringValue(deployment.ID)
	plan.CreatedAt = types.StringValue(deployment.CreatedAt.Format(time.RFC3339))

	// The create request carries no flags, and the API refuses changes until the
	// deployment is running, so flags are applied by an update once it is.
	if hasFlags(flags) {
		updated, noChanges, err := r.applyFlags(ctx, deployment.ID, updateRequestFrom(plan, dedup, dedupUnit, flags))
		switch {
		case err != nil:
			// The deployment exists, so this is not an error: an error would taint it
			// and make the next apply replace it. The flags are marked pending instead,
			// so that the next apply sends them again.
			resp.Diagnostics.Append(resp.Private.SetKey(ctx, privateFlagsPending, []byte("true"))...)
			resp.Diagnostics.AddWarning(
				"Deployment flags not applied",
				fmt.Sprintf("Deployment %q was created, but its flags could not be applied: %s. The next apply sends them again.", plan.ID.ValueString(), err),
			)
		case noChanges:
			resp.Diagnostics.AddWarning(flagsMayNotBeAppliedWarning(plan.ID.ValueString()))
			deployment = updated
		default:
			deployment = updated
		}
	}

	resp.Diagnostics.Append(setComputedAfterApply(ctx, &plan, deployment)...)
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
	if isNotFound(err) {
		// Deleted outside Terraform: drop it from state so the next plan recreates it.
		resp.State.RemoveResource(ctx)
		return
	}
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
	state.MaintenanceWindow = types.StringValue(string(deployment.MaintenanceWindow))
	state.CreatedAt = types.StringValue(deployment.CreatedAt.Format(time.RFC3339))
	setComputedFromAPI(&state, deployment)
	dedup, dedupUnit := deduplicationFromAPI(deployment)
	resp.Diagnostics.Append(setDeduplication(ctx, &state, dedup, dedupUnit)...)

	// Calculate storage_size from storage_size_gb. After an import the unit is not
	// in state yet, so it defaults to GB.
	if state.StorageSizeUnit.IsNull() {
		state.StorageSizeUnit = types.StringValue("GB")
	}
	switch state.StorageSizeUnit.ValueString() {
	case "GB":
		state.StorageSize = types.Int64Value(int64(deployment.StorageSizeGb))
	case "TB":
		state.StorageSize = types.Int64Value(int64(deployment.StorageSizeGb / 1024))
	}

	pending, diags := req.Private.GetKey(ctx, privateFlagsPending)
	resp.Diagnostics.Append(diags...)
	if len(pending) > 0 {
		clearFlags(&state)
	} else {
		resp.Diagnostics.Append(refreshFlags(ctx, &state, deployment)...)
	}
	resp.Diagnostics.Append(setGroupedFlags(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
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
	flags, diags := flagsRequest(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update the deployment
	dedup, dedupUnit := deduplicationPointers(plan.Deduplication, plan.DeduplicationUnit)
	deployment, noChanges, err := r.updateDeployment(ctx, plan.ID.ValueString(), updateRequestFrom(plan, dedup, dedupUnit, flags))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating deployment",
			"Could not update deployment, unexpected error: "+err.Error(),
		)
		return
	}
	if noChanges {
		resp.Diagnostics.AddWarning(flagsMayNotBeAppliedWarning(plan.ID.ValueString()))
	}
	// The flags were sent, so they are no longer pending.
	resp.Diagnostics.Append(resp.Private.SetKey(ctx, privateFlagsPending, nil)...)

	resp.Diagnostics.Append(setComputedAfterApply(ctx, &plan, deployment)...)
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
	if err != nil && !isNotFound(err) {
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

// applyFlags waits until a new deployment is running and then sends the update that
// sets its flags. noChanges reports that the API refused the update as a no-op.
func (r *deploymentResource) applyFlags(ctx context.Context, deploymentID string, update vmcloudapi.DeploymentUpdateRequest) (deployment vmcloudapi.DeploymentInfo, noChanges bool, err error) {
	if _, err := r.waitUntilRunning(ctx, deploymentID); err != nil {
		return vmcloudapi.DeploymentInfo{}, false, err
	}
	return r.updateDeployment(ctx, deploymentID, update)
}

// waitUntilRunning polls a deployment until it is running. It gives up when the
// deployment fails, is gone, keeps failing to read, or deploymentWaitTimeout passes.
func (r *deploymentResource) waitUntilRunning(ctx context.Context, deploymentID string) (vmcloudapi.DeploymentInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, deploymentWaitTimeout)
	defer cancel()
	ticker := time.NewTicker(deploymentPollInterval)
	defer ticker.Stop()
	readErrors := 0
	for {
		deployment, err := r.client.GetDeploymentDetails(ctx, deploymentID)
		switch {
		case err != nil:
			readErrors++
			if isNotFound(err) || readErrors >= maxConsecutiveReadErrors || ctx.Err() != nil {
				return deployment, fmt.Errorf("failed to read deployment %q: %w", deploymentID, err)
			}
			tflog.Debug(ctx, "failed to read deployment, retrying", map[string]any{"id": deploymentID, "error": err.Error()})
		case deployment.Status == vmcloudapi.DeploymentStatusRunning:
			return deployment, nil
		case deployment.Status == vmcloudapi.DeploymentStatusError, deployment.Status == vmcloudapi.DeploymentStatusStopped:
			return deployment, fmt.Errorf("deployment %q is in status %s", deploymentID, deployment.Status)
		default:
			readErrors = 0
			tflog.Debug(ctx, "waiting for deployment to be running", map[string]any{"id": deploymentID, "status": deployment.Status.String()})
		}
		select {
		case <-ctx.Done():
			return deployment, fmt.Errorf("deployment %q is not running after %s: %w", deploymentID, deploymentWaitTimeout, ctx.Err())
		case <-ticker.C:
		}
	}
}

// updateDeployment sends an update. The API refuses an update that changes nothing
// it compares, which includes any update that changes only flags, because it does not
// compare them. Terraform only sends an update when its plan has changes, so that
// answer is returned as noChanges rather than as an error.
func (r *deploymentResource) updateDeployment(ctx context.Context, deploymentID string, update vmcloudapi.DeploymentUpdateRequest) (deployment vmcloudapi.DeploymentInfo, noChanges bool, err error) {
	deployment, err = r.client.UpdateDeployment(ctx, deploymentID, update)
	if isNoChanges(err) {
		tflog.Debug(ctx, "the API reported no changes for the deployment update", map[string]any{"id": deploymentID})
		deployment, err = r.client.GetDeploymentDetails(ctx, deploymentID)
		return deployment, true, err
	}
	return deployment, false, err
}

// flagsMayNotBeAppliedWarning is the warning for an update that the API refused as a
// no-op, which is how it answers flags-only changes.
func flagsMayNotBeAppliedWarning(deploymentID string) (string, string) {
	return "Deployment flags may not be applied",
		fmt.Sprintf("The API reported no changes for deployment %q. It does not compare or report custom flags, so the configured flags may not have been applied.", deploymentID)
}

// updateRequestFrom builds the update request for the planned deployment m.
func updateRequestFrom(m deploymentResourceModel, dedup *uint32, dedupUnit *vmcloudapi.DurationUnit, flags vmcloudapi.DeploymentFlags) vmcloudapi.DeploymentUpdateRequest {
	return vmcloudapi.DeploymentUpdateRequest{
		Name:              m.Name.ValueString(),
		Tier:              uint32(m.Tier.ValueInt64()),
		StorageSize:       uint64(m.StorageSize.ValueInt64()),
		StorageSizeUnit:   vmcloudapi.StorageUnit(m.StorageSizeUnit.ValueString()),
		Retention:         uint32(m.Retention.ValueInt64()),
		RetentionUnit:     vmcloudapi.DurationUnit(m.RetentionUnit.ValueString()),
		Deduplication:     dedup,
		DeduplicationUnit: dedupUnit,
		MaintenanceWindow: vmcloudapi.MaintenanceWindow(m.MaintenanceWindow.ValueString()),
		Flags:             flags,
	}
}

// setComputedFromAPI copies the attributes that only the API decides into m.
func setComputedFromAPI(m *deploymentResourceModel, deployment vmcloudapi.DeploymentInfo) {
	m.Version = types.StringValue(deployment.Version)
	m.Status = types.StringValue(deployment.Status.String())
	m.AccessEndpoint = types.StringValue(deployment.AccessEndpoint)
}

// flagsRequest converts the flags of m to the API representation, with empty lists
// for unset flags.
func flagsRequest(ctx context.Context, m deploymentResourceModel) (vmcloudapi.DeploymentFlags, diag.Diagnostics) {
	var diags diag.Diagnostics
	toSlice := func(l types.List) vmcloudapi.FlagList {
		out := vmcloudapi.FlagList{}
		if !l.IsNull() && !l.IsUnknown() {
			diags.Append(l.ElementsAs(ctx, &out, false)...)
		}
		return out
	}
	return vmcloudapi.DeploymentFlags{
		SingleFlags:  toSlice(m.SingleFlags),
		SelectFlags:  toSlice(m.SelectFlags),
		StorageFlags: toSlice(m.StorageFlags),
		InsertFlags:  toSlice(m.InsertFlags),
	}, diags
}

// hasFlags reports whether any component has flags.
func hasFlags(f vmcloudapi.DeploymentFlags) bool {
	return len(f.SingleFlags)+len(f.SelectFlags)+len(f.StorageFlags)+len(f.InsertFlags) > 0
}

// refreshFlags updates the flags of m with the flags the API reports for the
// deployment.
func refreshFlags(ctx context.Context, m *deploymentResourceModel, deployment vmcloudapi.DeploymentInfo) diag.Diagnostics {
	var diags, d diag.Diagnostics
	refresh := func(flags *types.List, reported []string) {
		*flags, d = reportedFlags(ctx, *flags, reported)
		diags.Append(d...)
	}
	if deployment.Type == vmcloudapi.DeploymentTypeCluster {
		refresh(&m.SelectFlags, deployment.VMSelectSettings)
		refresh(&m.StorageFlags, deployment.VMStorageSettings)
		refresh(&m.InsertFlags, deployment.VMInsertSettings)
	} else {
		refresh(&m.SingleFlags, deployment.VMSingleSettings)
	}
	return diags
}

// reportedFlags returns the flags the API reports for a component. When the API does
// not report them (nil), the known value is kept. An empty report matches a null or
// empty known value, so an unset attribute does not show a difference.
func reportedFlags(ctx context.Context, known types.List, reported []string) (types.List, diag.Diagnostics) {
	if reported == nil {
		return known, nil
	}
	if len(reported) == 0 && (known.IsNull() || len(known.Elements()) == 0) {
		return known, nil
	}
	return types.ListValueFrom(ctx, types.StringType, reported)
}

// clearFlags reports every flag of m as unset, so that the next plan sends the
// configured flags again.
func clearFlags(m *deploymentResourceModel) {
	null := types.ListNull(types.StringType)
	m.SingleFlags, m.SelectFlags, m.StorageFlags, m.InsertFlags = null, null, null, null
}

func deduplicationPointers(dedup types.Int64, dedupUnit types.String) (*uint32, *vmcloudapi.DurationUnit) {
	// Unset optional+computed attributes are unknown in the plan, not null.
	if dedup.IsNull() || dedup.IsUnknown() || dedupUnit.IsNull() || dedupUnit.IsUnknown() {
		return nil, nil
	}
	value := uint32(dedup.ValueInt64())
	unit := vmcloudapi.DurationUnit(dedupUnit.ValueString())
	return &value, &unit
}
