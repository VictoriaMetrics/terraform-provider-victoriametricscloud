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
	_ resource.Resource                = &customAlertmanagerResource{}
	_ resource.ResourceWithConfigure   = &customAlertmanagerResource{}
	_ resource.ResourceWithImportState = &customAlertmanagerResource{}
)

// NewCustomAlertmanagerResource is a helper function to simplify the provider implementation.
func NewCustomAlertmanagerResource() resource.Resource {
	return &customAlertmanagerResource{}
}

// customAlertmanagerResource is the resource implementation.
type customAlertmanagerResource struct {
	client *vmcloudapi.VMCloudAPIClient
}

// customAlertmanagerResourceModel maps the resource schema data.
type customAlertmanagerResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Notifier    types.Object `tfsdk:"notifier"`
	Name        types.String `tfsdk:"name"`
	URL         types.String `tfsdk:"url"`
	AuthType    types.String `tfsdk:"auth_type"`
	Username    types.String `tfsdk:"username"`
	Password    types.String `tfsdk:"password"`
	BearerToken types.String `tfsdk:"bearer_token"`
	Status      types.String `tfsdk:"status"`
	Message     types.String `tfsdk:"message"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

// Metadata returns the resource type name.
func (r *customAlertmanagerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_alertmanager"
}

// Schema defines the schema for the resource.
func (r *customAlertmanagerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a custom (externally hosted) Alertmanager notifier in VictoriaMetrics Cloud.",
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
			"url": schema.StringAttribute{
				Description: "URL of the externally hosted Alertmanager. Must use https://.",
				Required:    true,
			},
			"auth_type": schema.StringAttribute{
				Description: "Authentication method used to reach the Alertmanager. Valid values: 'basic', 'bearer'.",
				Required:    true,
			},
			"username": schema.StringAttribute{
				Description: "Username for basic authentication. Required when auth_type is 'basic'.",
				Optional:    true,
			},
			"password": schema.StringAttribute{
				Description: "Password for basic authentication. Required when auth_type is 'basic'.",
				Optional:    true,
				Sensitive:   true,
			},
			"bearer_token": schema.StringAttribute{
				Description: "Bearer token for bearer authentication. Required when auth_type is 'bearer'.",
				Optional:    true,
				Sensitive:   true,
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
func (r *customAlertmanagerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func customAlertmanagerRequestFrom(plan customAlertmanagerResourceModel) vmcloudapi.CustomAlertmanagerRequest {
	return vmcloudapi.CustomAlertmanagerRequest{
		Name:        plan.Name.ValueString(),
		URL:         plan.URL.ValueString(),
		AuthType:    vmcloudapi.NotifierAuthType(plan.AuthType.ValueString()),
		Username:    plan.Username.ValueString(),
		Password:    plan.Password.ValueString(),
		BearerToken: plan.BearerToken.ValueString(),
	}
}

func (m *customAlertmanagerResourceModel) setComputedFromAPI(ctx context.Context, am vmcloudapi.CustomAlertmanager) diag.Diagnostics {
	m.ID = types.StringValue(am.ID)
	m.Status = types.StringValue(string(am.Status))
	m.Message = types.StringValue(am.Message)
	m.CreatedAt = types.StringValue(am.CreatedAt.Format(time.RFC3339))
	m.UpdatedAt = types.StringValue(am.UpdatedAt.Format(time.RFC3339))

	notifier, diags := types.ObjectValueFrom(ctx, notifierRefAttrTypes, notifierRefModel{
		ID:   m.ID,
		Type: types.StringValue(string(vmcloudapi.NotifierTypeCustomAlertmanager)),
	})
	m.Notifier = notifier
	return diags
}

// Create creates the resource and sets the initial Terraform state.
func (r *customAlertmanagerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan customAlertmanagerResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	am, err := r.client.CreateCustomAlertmanager(ctx, customAlertmanagerRequestFrom(plan))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating custom Alertmanager notifier",
			"Could not create notifier, unexpected error: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(plan.setComputedFromAPI(ctx, am)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "created custom alertmanager notifier", map[string]any{"id": am.ID})

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data.
func (r *customAlertmanagerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state customAlertmanagerResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	am, err := r.client.GetCustomAlertmanager(ctx, state.ID.ValueString())
	if isNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading custom Alertmanager notifier",
			"Could not read notifier ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	state.Name = types.StringValue(am.Name)
	state.URL = types.StringValue(am.URL)
	state.AuthType = types.StringValue(string(am.AuthType))
	if am.Username != "" {
		state.Username = types.StringValue(am.Username)
	}
	if am.Password != "" {
		state.Password = types.StringValue(am.Password)
	}
	if am.BearerToken != "" {
		state.BearerToken = types.StringValue(am.BearerToken)
	}
	resp.Diagnostics.Append(state.setComputedFromAPI(ctx, am)...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *customAlertmanagerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan customAlertmanagerResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	am, err := r.client.UpdateCustomAlertmanager(ctx, plan.ID.ValueString(), customAlertmanagerRequestFrom(plan))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating custom Alertmanager notifier",
			"Could not update notifier, unexpected error: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(plan.setComputedFromAPI(ctx, am)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "updated custom alertmanager notifier", map[string]any{"id": am.ID})

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *customAlertmanagerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state customAlertmanagerResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteCustomAlertmanager(ctx, state.ID.ValueString())
	if err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError(
			"Error deleting custom Alertmanager notifier",
			"Could not delete notifier, unexpected error: "+err.Error(),
		)
		return
	}

	tflog.Trace(ctx, "deleted custom alertmanager notifier", map[string]any{"id": state.ID.ValueString()})
}

// ImportState imports the resource state.
func (r *customAlertmanagerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
