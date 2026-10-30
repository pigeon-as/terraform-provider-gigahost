package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pigeon-as/terraform-provider-gigahost/internal/client"
)

var (
	_ resource.Resource                = &dnsZoneDNSSECResource{}
	_ resource.ResourceWithConfigure   = &dnsZoneDNSSECResource{}
	_ resource.ResourceWithImportState = &dnsZoneDNSSECResource{}
)

func NewDNSZoneDNSSECResource() resource.Resource {
	return &dnsZoneDNSSECResource{}
}

type dnsZoneDNSSECResource struct {
	client *client.Client
}

type dnsZoneDNSSECResourceModel struct {
	ZoneID    types.String `tfsdk:"zone_id"`
	DSRecords types.String `tfsdk:"ds_records"`
}

func (r *dnsZoneDNSSECResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_zone_dnssec"
}

func (r *dnsZoneDNSSECResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Enables DNSSEC for a domain registered with Gigahost. " +
			"For zones on Gigahost nameservers the zone is signed and the DS records are submitted to the registry automatically. " +
			"For externally hosted domains only the DNSSEC flag is set; submit the DS records with `gigahost_dns_zone_ds_records`. " +
			"Destroying the resource disables DNSSEC.",
		Attributes: map[string]schema.Attribute{
			"zone_id": schema.StringAttribute{
				Required:            true,
				Description:         "Id of the DNS zone (registered domain).",
				MarkdownDescription: "Id of the DNS zone (registered domain).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"ds_records": schema.StringAttribute{
				Computed:            true,
				Description:         "DS records as reported by the Gigahost API.",
				MarkdownDescription: "DS records as reported by the Gigahost API (text, or configuration instructions for externally hosted domains).",
			},
		},
	}
}

func (r *dnsZoneDNSSECResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *dnsZoneDNSSECResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan dnsZoneDNSSECResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zoneID := plan.ZoneID.ValueString()
	if err := r.client.SetDNSSEC(ctx, zoneID, true); err != nil {
		resp.Diagnostics.AddError("Unable to Enable Gigahost DNSSEC", err.Error())
		return
	}

	ds, err := r.client.GetDSRecords(ctx, zoneID)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Gigahost DS Records", err.Error())
		return
	}
	plan.DSRecords = types.StringValue(ds)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dnsZoneDNSSECResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state dnsZoneDNSSECResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ds, err := r.client.GetDSRecords(ctx, state.ZoneID.ValueString())
	if err != nil {
		if errors.Is(err, client.ErrNotFound) || errors.Is(err, client.ErrDNSSECNotEnabled) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to Read Gigahost DS Records", err.Error())
		return
	}
	state.DSRecords = types.StringValue(ds)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dnsZoneDNSSECResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// zone_id forces replacement and ds_records is computed, so there is
	// nothing to update; keep the planned state.
	var plan dnsZoneDNSSECResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dnsZoneDNSSECResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state dnsZoneDNSSECResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.SetDNSSEC(ctx, state.ZoneID.ValueString(), false); err != nil && !errors.Is(err, client.ErrNotFound) {
		resp.Diagnostics.AddError("Unable to Disable Gigahost DNSSEC", err.Error())
	}
}

func (r *dnsZoneDNSSECResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID == "" {
		resp.Diagnostics.AddError("Invalid Import Id", "Expected the zone id as import id.")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("zone_id"), req.ID)...)
}
