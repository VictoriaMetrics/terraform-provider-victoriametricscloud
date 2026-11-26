package provider

import (
	"context"
	"fmt"
	"time"

	vmcloudapi "github.com/VictoriaMetrics/victoriametrics-cloud-api-go/v1"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &cloudAlertmanagerResource{}
	_ resource.ResourceWithConfigure   = &cloudAlertmanagerResource{}
	_ resource.ResourceWithImportState = &cloudAlertmanagerResource{}
)

// NewCloudAlertmanagerResource is a helper function to simplify the provider implementation.
func NewCloudAlertmanagerResource() resource.Resource {
	return &cloudAlertmanagerResource{}
}

// cloudAlertmanagerResource is the resource implementation.
type cloudAlertmanagerResource struct {
	client *vmcloudapi.VMCloudAPIClient
}

// cloudAlertmanagerResourceModel maps the resource schema data.
type cloudAlertmanagerResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Notifier  types.Object `tfsdk:"notifier"`
	Name      types.String `tfsdk:"name"`
	Config    types.String `tfsdk:"config"`
	Status    types.String `tfsdk:"status"`
	Message   types.String `tfsdk:"message"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

// Metadata returns the resource type name.
func (r *cloudAlertmanagerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cloud_alertmanager"
}

// Schema defines the schema for the resource.
func (r *cloudAlertmanagerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a VictoriaMetrics-hosted Alertmanager notifier in VictoriaMetrics Cloud.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier of the notifier.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"notifier": schema.SingleNestedAttribute{
				Description: "Reference this as the notifier argument of victoriametricscloud_deployment_notifier to attach this notifier to a deployment.",
				Computed:    true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Description: "ID of the notifier.",
						Computed:    true,
					},
					"type": schema.StringAttribute{
						Description: "Type of the notifier.",
						Computed:    true,
					},
				},
			},
			"name": schema.StringAttribute{
				Description: "Human-readable name of the notifier.",
				Required:    true,
			},
			"config": schema.StringAttribute{
				Description: "Alertmanager configuration in YAML format.",
				Required:    true,
			},
			"status": schema.StringAttribute{
				Description: "Last known health status of the notifier: 'unknown', 'error' or 'active'.",
				Computed:    true,
			},
			"message": schema.StringAttribute{
				Description: "Detail message from the last validation of the notifier.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "Timestamp of notifier creation.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description: "Timestamp of the last notifier update.",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *cloudAlertmanagerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func cloudAlertmanagerRequestFrom(plan cloudAlertmanagerResourceModel) vmcloudapi.CloudAlertmanagerRequest {
	return vmcloudapi.CloudAlertmanagerRequest{
		Name:   plan.Name.ValueString(),
		Config: plan.Config.ValueString(),
	}
}

func (m *cloudAlertmanagerResourceModel) setComputedFromAPI(ctx context.Context, am vmcloudapi.CloudAlertmanager) diag.Diagnostics {
	m.ID = types.StringValue(am.ID)
	m.Status = types.StringValue(string(am.Status))
	m.Message = types.StringValue(am.Message)
	m.CreatedAt = types.StringValue(am.CreatedAt.Format(time.RFC3339))
	m.UpdatedAt = types.StringValue(am.UpdatedAt.Format(time.RFC3339))

	notifier, diags := types.ObjectValueFrom(ctx, notifierRefAttrTypes, notifierRefModel{
		ID:   m.ID,
		Type: types.StringValue(string(vmcloudapi.NotifierTypeCloudAlertmanager)),
	})
	m.Notifier = notifier
	return diags
}

// Create creates the resource and sets the initial Terraform state.
func (r *cloudAlertmanagerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan cloudAlertmanagerResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	am, err := r.client.CreateCloudAlertmanager(ctx, cloudAlertmanagerRequestFrom(plan))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating cloud Alertmanager notifier",
			"Could not create notifier, unexpected error: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(plan.setComputedFromAPI(ctx, am)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "created cloud alertmanager notifier", map[string]any{"id": am.ID})

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data.
func (r *cloudAlertmanagerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state cloudAlertmanagerResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	am, err := r.client.GetCloudAlertmanager(ctx, state.ID.ValueString())
	if isNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading cloud Alertmanager notifier",
			"Could not read notifier ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	state.Name = types.StringValue(am.Name)
	state.Config = types.StringValue(am.Config)
	resp.Diagnostics.Append(state.setComputedFromAPI(ctx, am)...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *cloudAlertmanagerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan cloudAlertmanagerResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	am, err := r.client.UpdateCloudAlertmanager(ctx, plan.ID.ValueString(), cloudAlertmanagerRequestFrom(plan))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating cloud Alertmanager notifier",
			"Could not update notifier, unexpected error: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(plan.setComputedFromAPI(ctx, am)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "updated cloud alertmanager notifier", map[string]any{"id": am.ID})

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *cloudAlertmanagerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state cloudAlertmanagerResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteCloudAlertmanager(ctx, state.ID.ValueString())
	if err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError(
			"Error deleting cloud Alertmanager notifier",
			"Could not delete notifier, unexpected error: "+err.Error(),
		)
		return
	}

	tflog.Trace(ctx, "deleted cloud alertmanager notifier", map[string]any{"id": state.ID.ValueString()})
}

// ImportState imports the resource state.
func (r *cloudAlertmanagerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
