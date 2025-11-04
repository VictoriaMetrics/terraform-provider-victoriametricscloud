package provider

import (
	"context"
	"fmt"
	"strings"

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
	_ resource.Resource                = &ruleFileResource{}
	_ resource.ResourceWithConfigure   = &ruleFileResource{}
	_ resource.ResourceWithImportState = &ruleFileResource{}
)

// NewRuleFileResource is a helper function to simplify the provider implementation.
func NewRuleFileResource() resource.Resource {
	return &ruleFileResource{}
}

// ruleFileResource is the resource implementation.
type ruleFileResource struct {
	client *vmcloudapi.VMCloudAPIClient
}

// ruleFileResourceModel maps the resource schema data.
type ruleFileResourceModel struct {
	ID           types.String `tfsdk:"id"`
	DeploymentID types.String `tfsdk:"deployment_id"`
	FileName     types.String `tfsdk:"file_name"`
	Content      types.String `tfsdk:"content"`
}

// Metadata returns the resource type name.
func (r *ruleFileResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rule_file"
}

// Schema defines the schema for the resource.
func (r *ruleFileResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an alerting or recording rules file for a VictoriaMetrics Cloud deployment.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Composite identifier in format 'deployment_id/file_name'.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"deployment_id": schema.StringAttribute{
				Description: "ID of the deployment this rule file belongs to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"file_name": schema.StringAttribute{
				Description: "Name of the rule file (e.g., 'alerting-rules.yaml').",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"content": schema.StringAttribute{
				Description: "YAML content of the alerting or recording rules file.",
				Required:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *ruleFileResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *ruleFileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ruleFileResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create or update the rule file
	err := r.client.CreateDeploymentRuleFileContent(
		ctx,
		plan.DeploymentID.ValueString(),
		plan.FileName.ValueString(),
		plan.Content.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating rule file",
			"Could not create rule file, unexpected error: "+err.Error(),
		)
		return
	}

	// Set the composite ID
	plan.ID = types.StringValue(fmt.Sprintf("%s/%s", plan.DeploymentID.ValueString(), plan.FileName.ValueString()))

	tflog.Trace(ctx, "created rule file", map[string]any{
		"deployment_id": plan.DeploymentID.ValueString(),
		"file_name":     plan.FileName.ValueString(),
	})

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data.
func (r *ruleFileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ruleFileResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get refreshed rule file content from API
	content, err := r.client.GetDeploymentRuleFileContent(
		ctx,
		state.DeploymentID.ValueString(),
		state.FileName.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Rule File",
			"Could not read rule file "+state.FileName.ValueString()+": "+err.Error(),
		)
		return
	}

	// Update state with refreshed values
	state.Content = types.StringValue(content)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *ruleFileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ruleFileResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update the rule file content
	err := r.client.UpdateDeploymentRuleFileContent(
		ctx,
		plan.DeploymentID.ValueString(),
		plan.FileName.ValueString(),
		plan.Content.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating rule file",
			"Could not update rule file, unexpected error: "+err.Error(),
		)
		return
	}

	tflog.Trace(ctx, "updated rule file", map[string]any{
		"deployment_id": plan.DeploymentID.ValueString(),
		"file_name":     plan.FileName.ValueString(),
	})

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *ruleFileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ruleFileResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the rule file
	err := r.client.DeleteDeploymentRuleFile(
		ctx,
		state.DeploymentID.ValueString(),
		state.FileName.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting rule file",
			"Could not delete rule file, unexpected error: "+err.Error(),
		)
		return
	}

	tflog.Trace(ctx, "deleted rule file", map[string]any{
		"deployment_id": state.DeploymentID.ValueString(),
		"file_name":     state.FileName.ValueString(),
	})
}

// ImportState imports the resource state.
func (r *ruleFileResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Expected format: deployment_id/file_name
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import identifier with format: deployment_id/file_name. Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("deployment_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("file_name"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
