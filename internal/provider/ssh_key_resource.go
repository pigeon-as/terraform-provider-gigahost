package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/joakimhellum/terraform-provider-gigahost/internal/client"
	"github.com/joakimhellum/terraform-provider-gigahost/internal/resource_ssh_key"
)

var (
	_ resource.Resource                = &sshKeyResource{}
	_ resource.ResourceWithConfigure   = &sshKeyResource{}
	_ resource.ResourceWithImportState = &sshKeyResource{}
)

func NewSSHKeyResource() resource.Resource {
	return &sshKeyResource{}
}

type sshKeyResource struct {
	client *client.Client
}

func (r *sshKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ssh_key"
}

func (r *sshKeyResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	s := resource_ssh_key.SshKeyResourceSchema(ctx)
	s.MarkdownDescription = "Manages an SSH key on the Gigahost account."

	keyName := s.Attributes["key_name"].(schema.StringAttribute)
	keyName.PlanModifiers = []planmodifier.String{stringplanmodifier.RequiresReplace()}
	s.Attributes["key_name"] = keyName

	keyData := s.Attributes["key_data"].(schema.StringAttribute)
	keyData.PlanModifiers = []planmodifier.String{stringplanmodifier.RequiresReplace()}
	s.Attributes["key_data"] = keyData

	keyID := s.Attributes["key_id"].(schema.StringAttribute)
	keyID.PlanModifiers = []planmodifier.String{stringplanmodifier.UseStateForUnknown()}
	s.Attributes["key_id"] = keyID

	keyAdded := s.Attributes["key_added"].(schema.StringAttribute)
	keyAdded.PlanModifiers = []planmodifier.String{stringplanmodifier.UseStateForUnknown()}
	s.Attributes["key_added"] = keyAdded

	resp.Schema = s
}

func (r *sshKeyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = c
}

func (r *sshKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan resource_ssh_key.SshKeyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	key, err := r.client.CreateSSHKey(ctx, plan.KeyName.ValueString(), plan.KeyData.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to Create Gigahost SSH Key", err.Error())
		return
	}

	state := plan
	state.KeyId = types.StringValue(key.KeyID)
	state.KeyAdded = types.StringValue(key.KeyAdded)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *sshKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state resource_ssh_key.SshKeyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	key, err := r.client.GetSSHKey(ctx, state.KeyId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Gigahost SSH Key", err.Error())
		return
	}
	if key == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.KeyName = types.StringValue(key.KeyName)
	state.KeyData = types.StringValue(key.KeyData)
	state.KeyAdded = types.StringValue(key.KeyAdded)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *sshKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan resource_ssh_key.SshKeyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *sshKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state resource_ssh_key.SshKeyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteSSHKey(ctx, state.KeyId.ValueString()); err != nil {
		resp.Diagnostics.AddError("Unable to Delete Gigahost SSH Key", err.Error())
	}
}

func (r *sshKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("key_id"), req, resp)
}
