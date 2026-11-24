package provider

import (
	"context"
	"fmt"

	vmcloudapi "github.com/VictoriaMetrics/victoriametrics-cloud-api-go/v1"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &deploymentNotifierResource{}
	_ resource.ResourceWithConfigure   = &deploymentNotifierResource{}
	_ resource.ResourceWithImportState = &deploymentNotifierResource{}
)

// NewDeploymentNotifierResource is a helper function to simplify the provider implementation.
func NewDeploymentNotifierResource() resource.Resource {
	return &deploymentNotifierResource{}
}

// deploymentNotifierResource is the resource implementation.
type deploymentNotifierResource struct {
	client *vmcloudapi.VMCloudAPIClient
}

// deploymentNotifierResourceModel maps the resource schema data.
type deploymentNotifierResourceModel struct {
	ID           types.String `tfsdk:"id"`
	DeploymentID types.String `tfsdk:"deployment_id"`
	Notifier     types.Object `tfsdk:"notifier"`
	NotifierName types.String `tfsdk:"notifier_name"`
}

// notifierRefModel is the shape of the notifier argument: the id and type
// attributes of a victoriametricscloud_custom_alertmanager or
// victoriametricscloud_cloud_alertmanager resource. Terraform drops any other
// attribute of the referenced resource when converting it to this type, so
// users can pass the whole resource directly.
type notifierRefModel struct {
	ID   types.String `tfsdk:"id"`
	Type types.String `tfsdk:"type"`
}

var notifierRefAttrTypes = map[string]attr.Type{
	"id":   types.StringType,
	"type": types.StringType,
}

// Metadata returns the resource type name.
func (r *deploymentNotifierResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_deployment_notifier"
}

// Schema defines the schema for the resource.
func (r *deploymentNotifierResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Attaches a notifier (custom or cloud Alertmanager) to a VictoriaMetrics Cloud deployment. One deployment has at most one notifier attached.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier of this attachment. Equal to deployment_id.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"deployment_id": schema.StringAttribute{
				Description: "ID of the deployment the notifier is attached to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"notifier": schema.SingleNestedAttribute{
				Description: "The notifier to attach: the full output of a victoriametricscloud_custom_alertmanager or victoriametricscloud_cloud_alertmanager resource. Only its id and type attributes are used; the resource's other attributes are ignored.",
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Description: "ID of the notifier.",
						Required:    true,
					},
					"type": schema.StringAttribute{
						Description: "Type of the notifier: 'custom-alertmanager' or 'cloud-alertmanager'.",
						Required:    true,
					},
				},
			},
			"notifier_name": schema.StringAttribute{
				Description: "Name of the attached notifier, as reported by the API.",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *deploymentNotifierResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (m *deploymentNotifierResourceModel) setFromAPI(ctx context.Context, dn vmcloudapi.DeploymentNotifier) diag.Diagnostics {
	notifier, diags := types.ObjectValueFrom(ctx, notifierRefAttrTypes, notifierRefModel{
		ID:   types.StringValue(dn.NotifierID),
		Type: types.StringValue(string(dn.NotifierType)),
	})
	m.Notifier = notifier
	m.NotifierName = types.StringValue(dn.NotifierName)
	return diags
}

// notifierRefFromObject reads the id and type attributes out of the notifier
// argument, which Terraform accepts as the full output of either alertmanager
// resource and structurally narrows down to notifierRefAttrTypes.
func notifierRefFromObject(ctx context.Context, obj types.Object) (notifierRefModel, diag.Diagnostics) {
	var ref notifierRefModel
	diags := obj.As(ctx, &ref, basetypes.ObjectAsOptions{})
	return ref, diags
}

// Create creates the resource and sets the initial Terraform state.
func (r *deploymentNotifierResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan deploymentNotifierResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ref, diags := notifierRefFromObject(ctx, plan.Notifier)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	dn, err := r.client.SetDeploymentNotifier(ctx, plan.DeploymentID.ValueString(), vmcloudapi.DeploymentNotifierRequest{
		NotifierID:   ref.ID.ValueString(),
		NotifierType: vmcloudapi.NotifierType(ref.Type.ValueString()),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error attaching notifier to deployment",
			"Could not attach notifier, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = plan.DeploymentID
	resp.Diagnostics.Append(plan.setFromAPI(ctx, dn)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "attached notifier to deployment", map[string]any{"deployment_id": plan.DeploymentID.ValueString(), "notifier_id": dn.NotifierID})

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data.
func (r *deploymentNotifierResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state deploymentNotifierResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	dn, err := r.client.GetDeploymentNotifier(ctx, state.DeploymentID.ValueString())
	if isNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading deployment notifier",
			"Could not read notifier for deployment "+state.DeploymentID.ValueString()+": "+err.Error(),
		)
		return
	}
	if dn.NotifierID == "" {
		// Detached outside Terraform: drop it from state so the next plan reattaches it.
		resp.State.RemoveResource(ctx)
		return
	}

	state.ID = state.DeploymentID
	resp.Diagnostics.Append(state.setFromAPI(ctx, dn)...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *deploymentNotifierResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan deploymentNotifierResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ref, diags := notifierRefFromObject(ctx, plan.Notifier)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	dn, err := r.client.SetDeploymentNotifier(ctx, plan.DeploymentID.ValueString(), vmcloudapi.DeploymentNotifierRequest{
		NotifierID:   ref.ID.ValueString(),
		NotifierType: vmcloudapi.NotifierType(ref.Type.ValueString()),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating deployment notifier",
			"Could not update notifier attachment, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = plan.DeploymentID
	resp.Diagnostics.Append(plan.setFromAPI(ctx, dn)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "updated deployment notifier", map[string]any{"deployment_id": plan.DeploymentID.ValueString(), "notifier_id": dn.NotifierID})

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *deploymentNotifierResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state deploymentNotifierResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteDeploymentNotifier(ctx, state.DeploymentID.ValueString())
	if err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError(
			"Error detaching notifier from deployment",
			"Could not detach notifier, unexpected error: "+err.Error(),
		)
		return
	}

	tflog.Trace(ctx, "detached notifier from deployment", map[string]any{"deployment_id": state.DeploymentID.ValueString()})
}

// ImportState imports the resource state.
func (r *deploymentNotifierResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("deployment_id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
