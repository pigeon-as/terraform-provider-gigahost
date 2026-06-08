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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pigeon-as/terraform-provider-gigahost/internal/client"
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

type dnsRedirectResourceModel struct {
	Enabled   types.Bool   `tfsdk:"enabled"`
	Source    types.String `tfsdk:"source"`
	TargetUrl types.String `tfsdk:"target_url"`
	ZoneId    types.String `tfsdk:"zone_id"`
}

func (r *dnsRedirectResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_redirect"
}

func (r *dnsRedirectResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an HTTP redirect for a Gigahost DNS zone.",
		Attributes: map[string]schema.Attribute{
			"enabled": schema.BoolAttribute{
				Computed:            true,
				Description:         "Whether the redirect is active.",
				MarkdownDescription: "Whether the redirect is active.",
			},
			"source": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Subdomain the redirect applies to; defaults to \"@\" (the zone root).",
				MarkdownDescription: "Subdomain the redirect applies to; defaults to \"@\" (the zone root).",
				Default:             stringdefault.StaticString("@"),
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"target_url": schema.StringAttribute{
				Required:            true,
				Description:         "URL that requests are redirected to.",
				MarkdownDescription: "URL that requests are redirected to.",
			},
			"zone_id": schema.StringAttribute{
				Required:            true,
				Description:         "Id of the DNS zone that contains the redirect.",
				MarkdownDescription: "Id of the DNS zone that contains the redirect.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
		},
	}
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
	var plan dnsRedirectResourceModel
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
	var state dnsRedirectResourceModel
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
	var plan dnsRedirectResourceModel
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
	var state dnsRedirectResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteRedirect(ctx, state.ZoneId.ValueString(), state.Source.ValueString()); err != nil && !errors.Is(err, client.ErrNotFound) {
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
