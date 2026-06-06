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
	"github.com/pigeon-as/terraform-provider-gigahost/internal/resource_dns_zone"
)

var (
	_ resource.Resource                = &dnsZoneResource{}
	_ resource.ResourceWithConfigure   = &dnsZoneResource{}
	_ resource.ResourceWithImportState = &dnsZoneResource{}
)

func NewDNSZoneResource() resource.Resource {
	return &dnsZoneResource{}
}

type dnsZoneResource struct {
	client *client.Client
}

func (r *dnsZoneResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_zone"
}

func (r *dnsZoneResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	s := resource_dns_zone.DnsZoneResourceSchema(ctx)
	s.MarkdownDescription = "Manages a DNS zone on the Gigahost account."

	zoneName, ok := s.Attributes["zone_name"].(schema.StringAttribute)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Schema Type", `Generated attribute "zone_name" is not a string attribute. This is a bug in the provider, please report it.`)
		return
	}
	zoneName.PlanModifiers = []planmodifier.String{stringplanmodifier.RequiresReplace()}
	s.Attributes["zone_name"] = zoneName

	zoneType, ok := s.Attributes["zone_type"].(schema.StringAttribute)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Schema Type", `Generated attribute "zone_type" is not a string attribute. This is a bug in the provider, please report it.`)
		return
	}
	zoneType.PlanModifiers = []planmodifier.String{stringplanmodifier.RequiresReplace()}
	s.Attributes["zone_type"] = zoneType

	zoneID, ok := s.Attributes["zone_id"].(schema.StringAttribute)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Schema Type", `Generated attribute "zone_id" is not a string attribute. This is a bug in the provider, please report it.`)
		return
	}
	zoneID.PlanModifiers = []planmodifier.String{stringplanmodifier.UseStateForUnknown()}
	s.Attributes["zone_id"] = zoneID

	resp.Schema = s
}

func (r *dnsZoneResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *dnsZoneResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan resource_dns_zone.DnsZoneModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zone, err := r.client.CreateZone(ctx, plan.ZoneName.ValueString(), plan.ZoneType.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to Create Gigahost DNS Zone", err.Error())
		return
	}

	state := dnsZoneToModel(zone)
	state.ZoneName = plan.ZoneName
	state.ZoneType = plan.ZoneType

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dnsZoneResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state resource_dns_zone.DnsZoneModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zone, err := r.client.GetZone(ctx, state.ZoneId.ValueString())
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to Read Gigahost DNS Zone", err.Error())
		return
	}

	name, zoneType := state.ZoneName, state.ZoneType
	state = dnsZoneToModel(zone)
	if !name.IsNull() {
		state.ZoneName = name
	}
	if !zoneType.IsNull() {
		state.ZoneType = zoneType
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dnsZoneResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"The gigahost_dns_zone resource cannot be updated in place; every attribute requires replacement. This is a bug in the provider, please report it.",
	)
}

func (r *dnsZoneResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state resource_dns_zone.DnsZoneModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteZone(ctx, state.ZoneId.ValueString()); err != nil {
		resp.Diagnostics.AddError("Unable to Delete Gigahost DNS Zone", err.Error())
	}
}

func (r *dnsZoneResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("zone_id"), req, resp)
}

func dnsZoneToModel(z *client.DnsZone) resource_dns_zone.DnsZoneModel {
	return resource_dns_zone.DnsZoneModel{
		ZoneId:        types.StringValue(z.ZoneID),
		ZoneName:      types.StringValue(z.ZoneName),
		ZoneType:      types.StringValue(z.ZoneType),
		ZoneActive:    types.BoolValue(bool(z.ZoneActive)),
		ZoneProtected: types.BoolValue(bool(z.ZoneProtected)),
		ExternalDns:   types.BoolValue(bool(z.ExternalDNS)),
	}
}
