package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pigeon-as/terraform-provider-gigahost/internal/client"
	"github.com/pigeon-as/terraform-provider-gigahost/internal/resource_dns_redirect"
)

var (
	_ resource.Resource                = &dnsRedirectResource{}
	_ resource.ResourceWithConfigure   = &dnsRedirectResource{}
	_ resource.ResourceWithImportState = &dnsRedirectResource{}
)

func NewDNSRedirectResource() resource.Resource {
	return &dnsRedirectResource{}
}

type dnsRedirectResource struct {
	client *client.Client
}

func (r *dnsRedirectResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_redirect"
}

func (r *dnsRedirectResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	s := resource_dns_redirect.DnsRedirectResourceSchema(ctx)
	s.MarkdownDescription = "Manages an HTTP redirect for a Gigahost DNS zone."

	zoneID, ok := s.Attributes["zone_id"].(schema.StringAttribute)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Schema Type", `Generated attribute "zone_id" is not a string attribute. This is a bug in the provider, please report it.`)
		return
	}
	zoneID.Required = true
	zoneID.Optional = false
	zoneID.Computed = false
	zoneID.Description = "Id of the DNS zone that contains the redirect."
	zoneID.MarkdownDescription = zoneID.Description
	zoneID.PlanModifiers = []planmodifier.String{stringplanmodifier.RequiresReplace()}
	s.Attributes["zone_id"] = zoneID

	source, ok := s.Attributes["source"].(schema.StringAttribute)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Schema Type", `Generated attribute "source" is not a string attribute. This is a bug in the provider, please report it.`)
		return
	}
	source.PlanModifiers = []planmodifier.String{stringplanmodifier.RequiresReplace()}
	s.Attributes["source"] = source

	resp.Schema = s
}

func (r *dnsRedirectResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *dnsRedirectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan resource_dns_redirect.DnsRedirectModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	redirect, err := r.client.CreateRedirect(ctx, plan.ZoneId.ValueString(), plan.Source.ValueString(), plan.TargetUrl.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to Create Gigahost DNS Redirect", err.Error())
		return
	}

	state := plan
	state.Enabled = types.BoolValue(bool(redirect.Enabled))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dnsRedirectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state resource_dns_redirect.DnsRedirectModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	redirect, err := r.client.GetRedirect(ctx, state.ZoneId.ValueString(), state.Source.ValueString())
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to Read Gigahost DNS Redirect", err.Error())
		return
	}

	state.TargetUrl = types.StringValue(redirect.TargetURL)
	state.Enabled = types.BoolValue(bool(redirect.Enabled))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dnsRedirectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan resource_dns_redirect.DnsRedirectModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	redirect, err := r.client.UpdateRedirect(ctx, plan.ZoneId.ValueString(), plan.Source.ValueString(), plan.TargetUrl.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to Update Gigahost DNS Redirect", err.Error())
		return
	}

	state := plan
	state.Enabled = types.BoolValue(bool(redirect.Enabled))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dnsRedirectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state resource_dns_redirect.DnsRedirectModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteRedirect(ctx, state.ZoneId.ValueString(), state.Source.ValueString()); err != nil {
		resp.Diagnostics.AddError("Unable to Delete Gigahost DNS Redirect", err.Error())
	}
}

func (r *dnsRedirectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	zoneID, source, ok := strings.Cut(req.ID, "/")
	if !ok || zoneID == "" || source == "" {
		resp.Diagnostics.AddError(
			"Invalid Import Id",
			fmt.Sprintf("Expected import id in the format \"zone_id/source\", got: %q.", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("zone_id"), zoneID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("source"), source)...)
}
