package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pigeon-as/terraform-provider-gigahost/internal/client"
)

var (
	_ resource.Resource                = &dnsZoneSecondaryResource{}
	_ resource.ResourceWithConfigure   = &dnsZoneSecondaryResource{}
	_ resource.ResourceWithImportState = &dnsZoneSecondaryResource{}
)

func NewDNSZoneSecondaryResource() resource.Resource {
	return &dnsZoneSecondaryResource{}
}

type dnsZoneSecondaryResource struct {
	client *client.Client
}

type dnsZoneSecondaryResourceModel struct {
	ZoneID   types.String `tfsdk:"zone_id"`
	ZoneName types.String `tfsdk:"zone_name"`
	Masters  types.Set    `tfsdk:"masters"`
}

func (r *dnsZoneSecondaryResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_zone_secondary"
}

func (r *dnsZoneSecondaryResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a secondary (slave) DNS zone that Gigahost transfers from one or more master servers. " +
			"This uses `POST /dns/zones/secondary`, which is not part of the public API documentation but is used by the Gigahost panel. " +
			"No endpoint for changing the masters of an existing zone is known, so changing `masters` replaces the zone.",
		Attributes: map[string]schema.Attribute{
			"zone_id": schema.StringAttribute{
				Computed:            true,
				Description:         "Zone id.",
				MarkdownDescription: "Zone id.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"zone_name": schema.StringAttribute{
				Required:            true,
				Description:         "Domain name.",
				MarkdownDescription: "Domain name.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"masters": schema.SetAttribute{
				Required:            true,
				ElementType:         types.StringType,
				Description:         "IP addresses of the master servers to transfer the zone from.",
				MarkdownDescription: "IP addresses of the master servers to transfer the zone from. Changing this replaces the zone.",
				Validators:          []validator.Set{setvalidator.SizeAtLeast(1)},
				PlanModifiers:       []planmodifier.Set{setplanmodifier.RequiresReplace()},
			},
		},
	}
}

func (r *dnsZoneSecondaryResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *dnsZoneSecondaryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan dnsZoneSecondaryResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var masters []string
	resp.Diagnostics.Append(plan.Masters.ElementsAs(ctx, &masters, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zone, err := r.client.CreateSecondaryZone(ctx, plan.ZoneName.ValueString(), masters)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Create Gigahost Secondary DNS Zone", err.Error())
		return
	}
	plan.ZoneID = types.StringValue(zone.ZoneID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dnsZoneSecondaryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state dnsZoneSecondaryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zone, err := r.client.GetZone(ctx, state.ZoneID.ValueString())
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to Read Gigahost DNS Zone", err.Error())
		return
	}

	state.ZoneName = types.StringValue(zone.ZoneName)
	// Keep the stored masters if the API reports none (unknown format or empty).
	if masters := client.SplitMasters(zone.ZoneMasters); len(masters) > 0 {
		var current []string
		resp.Diagnostics.Append(state.Masters.ElementsAs(ctx, &current, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if !sameNameservers(current, masters) {
			set, diags := types.SetValueFrom(ctx, types.StringType, masters)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			state.Masters = set
		}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dnsZoneSecondaryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Every configurable attribute forces replacement.
	var plan dnsZoneSecondaryResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dnsZoneSecondaryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state dnsZoneSecondaryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteZone(ctx, state.ZoneID.ValueString()); err != nil && !errors.Is(err, client.ErrNotFound) {
		resp.Diagnostics.AddError("Unable to Delete Gigahost DNS Zone", err.Error())
	}
}

func (r *dnsZoneSecondaryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if strings.TrimSpace(req.ID) == "" {
		resp.Diagnostics.AddError("Invalid Import Id", "Expected the zone id as import id.")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("zone_id"), req.ID)...)
	// masters is populated from the zone by the following read.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("masters"), types.SetValueMust(types.StringType, nil))...)
}
