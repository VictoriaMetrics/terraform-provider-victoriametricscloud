package provider

import (
	"context"
	"fmt"
	"strings"
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
	_ resource.Resource                = &accessTokenResource{}
	_ resource.ResourceWithConfigure   = &accessTokenResource{}
	_ resource.ResourceWithImportState = &accessTokenResource{}
)

// NewAccessTokenResource is a helper function to simplify the provider implementation.
func NewAccessTokenResource() resource.Resource {
	return &accessTokenResource{}
}

// accessTokenResource is the resource implementation.
type accessTokenResource struct {
	client *vmcloudapi.VMCloudAPIClient
}

// accessTokenResourceModel maps the resource schema data.
type accessTokenResourceModel struct {
	ID           types.String `tfsdk:"id"`
	DeploymentID types.String `tfsdk:"deployment_id"`
	Type         types.String `tfsdk:"type"`
	Description  types.String `tfsdk:"description"`
	TenantID     types.String `tfsdk:"tenant_id"`
	Secret       types.String `tfsdk:"secret"`
	CreatedBy    types.String `tfsdk:"created_by"`
	CreatedAt    types.String `tfsdk:"created_at"`
	LastUsedAt   types.String `tfsdk:"last_used_at"`
}

// Metadata returns the resource type name.
func (r *accessTokenResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_access_token"
}

// Schema defines the schema for the resource.
func (r *accessTokenResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an access token for a VictoriaMetrics Cloud deployment.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier of the access token.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"deployment_id": schema.StringAttribute{
				Description: "ID of the deployment this token belongs to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"type": schema.StringAttribute{
				Description: "Access mode of the token. Valid values: 'r' (read-only), 'w' (write-only), 'rw' (read-write).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"description": schema.StringAttribute{
				Description: "Human-readable description of the access token.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"tenant_id": schema.StringAttribute{
				Description: "Optional tenant ID for cluster deployments (format: accountID or accountID:projectID).",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"secret": schema.StringAttribute{
				Description: "Secret value of the access token. Only available after creation.",
				Computed:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_by": schema.StringAttribute{
				Description: "Email of the user who created the token.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "Timestamp of token creation.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"last_used_at": schema.StringAttribute{
				Description: "Timestamp of last token usage (within the last 7 days).",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *accessTokenResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *accessTokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan accessTokenResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create the access token
	createRequest := vmcloudapi.AccessTokenCreateRequest{
		Type:        vmcloudapi.AccessMode(plan.Type.ValueString()),
		Description: plan.Description.ValueString(),
	}

	if !plan.TenantID.IsNull() {
		createRequest.TenantID = plan.TenantID.ValueString()
	}

	token, err := r.client.CreateDeploymentAccessToken(ctx, plan.DeploymentID.ValueString(), createRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating access token",
			"Could not create access token, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response to state
	plan.ID = types.StringValue(token.ID)
	plan.Secret = types.StringValue(token.Secret)
	plan.CreatedBy = types.StringValue(token.CreatedBy)
	plan.CreatedAt = types.StringValue(token.CreatedAt.Format(time.RFC3339))
	if token.LastUsedAt != nil {
		plan.LastUsedAt = types.StringValue(token.LastUsedAt.Format(time.RFC3339))
	}

	tflog.Trace(ctx, "created access token", map[string]any{"id": token.ID, "deployment_id": plan.DeploymentID.ValueString()})

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data.
func (r *accessTokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state accessTokenResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get refreshed token value from API
	// Note: We use the reveal endpoint to get the full secret
	token, err := r.client.RevealDeploymentAccessToken(ctx, state.DeploymentID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Access Token",
			"Could not read access token ID "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Update state with refreshed values
	state.Type = types.StringValue(token.Type.String())
	state.Description = types.StringValue(token.Description)
	state.Secret = types.StringValue(token.Secret)
	state.CreatedBy = types.StringValue(token.CreatedBy)
	state.CreatedAt = types.StringValue(token.CreatedAt.Format(time.RFC3339))
	if token.LastUsedAt != nil {
		state.LastUsedAt = types.StringValue(token.LastUsedAt.Format(time.RFC3339))
	} else {
		state.LastUsedAt = types.StringNull()
	}
	if token.TenantID != "" {
		state.TenantID = types.StringValue(token.TenantID)
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

// Update is not supported for access tokens (requires replacement).
func (r *accessTokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Access tokens cannot be updated, they require replacement
	resp.Diagnostics.AddError(
		"Update not supported",
		"Access tokens cannot be updated. Any changes require replacement.",
	)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *accessTokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state accessTokenResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the access token
	err := r.client.DeleteDeploymentAccessToken(ctx, state.DeploymentID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting access token",
			"Could not delete access token, unexpected error: "+err.Error(),
		)
		return
	}

	tflog.Trace(ctx, "deleted access token", map[string]any{"id": state.ID.ValueString(), "deployment_id": state.DeploymentID.ValueString()})
}

// ImportState imports the resource state.
func (r *accessTokenResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Expected format: deployment_id/token_id
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import identifier with format: deployment_id/token_id. Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("deployment_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}
