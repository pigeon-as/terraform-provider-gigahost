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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pigeon-as/terraform-provider-gigahost/internal/client"
)

var (
	_ resource.Resource                = &dnsZoneNameserversResource{}
	_ resource.ResourceWithConfigure   = &dnsZoneNameserversResource{}
	_ resource.ResourceWithImportState = &dnsZoneNameserversResource{}
)

func NewDNSZoneNameserversResource() resource.Resource {
	return &dnsZoneNameserversResource{}
}

type dnsZoneNameserversResource struct {
	client *client.Client
}

type dnsZoneNameserversResourceModel struct {
	ZoneID      types.String `tfsdk:"zone_id"`
	Nameservers types.Set    `tfsdk:"nameservers"`
}

func (r *dnsZoneNameserversResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_zone_nameservers"
}

func (r *dnsZoneNameserversResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the registry nameservers of a domain registered with Gigahost. " +
			"The new nameservers must answer authoritatively for the domain, otherwise Gigahost reverts the change. " +
			"Destroying this resource only removes it from Terraform state; the nameservers are left unchanged, because a registered domain always needs at least two.",
		Attributes: map[string]schema.Attribute{
			"zone_id": schema.StringAttribute{
				Required:            true,
				Description:         "Id of the DNS zone (registered domain).",
				MarkdownDescription: "Id of the DNS zone (registered domain).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"nameservers": schema.SetAttribute{
				Required:            true,
				ElementType:         types.StringType,
				Description:         "Registry nameservers for the domain. At least two are required.",
				MarkdownDescription: "Registry nameservers for the domain. At least two are required. Using `ns1.gigahost.no` marks the domain as hosted on Gigahost nameservers; any other set marks it as externally hosted.",
				Validators:          []validator.Set{setvalidator.SizeAtLeast(2)},
			},
		},
	}
}

func (r *dnsZoneNameserversResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *dnsZoneNameserversResource) apply(ctx context.Context, plan *dnsZoneNameserversResourceModel) error {
	var nameservers []string
	if diags := plan.Nameservers.ElementsAs(ctx, &nameservers, false); diags.HasError() {
		return fmt.Errorf("reading nameservers from plan")
	}
	return r.client.UpdateNameservers(ctx, plan.ZoneID.ValueString(), nameservers)
}

func (r *dnsZoneNameserversResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan dnsZoneNameserversResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.apply(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Unable to Update Gigahost Nameservers", err.Error())
		return
	}
	// The API has no nameserver read endpoint and the NS records may lag the
	// registry update, so the planned value is stored as-is.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dnsZoneNameserversResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state dnsZoneNameserversResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zoneID := state.ZoneID.ValueString()
	if _, err := r.client.GetZone(ctx, zoneID); err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to Read Gigahost DNS Zone", err.Error())
		return
	}

	// Gigahost keeps the apex NS records in sync with the registry
	// nameservers, so they are used to detect drift.
	observed, err := r.client.ListApexNameservers(ctx, zoneID)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Gigahost Nameservers", err.Error())
		return
	}
	if len(observed) == 0 {
		// No apex NS records to compare against; keep the stored value.
		return
	}

	var current []string
	resp.Diagnostics.Append(state.Nameservers.ElementsAs(ctx, &current, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if sameNameservers(current, observed) {
		return
	}

	set, diags := types.SetValueFrom(ctx, types.StringType, observed)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Nameservers = set
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dnsZoneNameserversResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan dnsZoneNameserversResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.apply(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Unable to Update Gigahost Nameservers", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dnsZoneNameserversResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// Intentionally empty: registry nameservers cannot be unset.
}

func (r *dnsZoneNameserversResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID == "" {
		resp.Diagnostics.AddError("Invalid Import Id", "Expected the zone id as import id.")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("zone_id"), req.ID)...)
	// nameservers is populated from the apex NS records by the following read.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("nameservers"), types.SetValueMust(types.StringType, nil))...)
}

func normalizeNameserver(s string) string {
	return strings.ToLower(strings.TrimSuffix(strings.TrimSpace(s), "."))
}

// sameNameservers reports whether a and b hold the same hostnames, ignoring
// order, case and a trailing dot.
func sameNameservers(a, b []string) bool {
	toSet := func(in []string) map[string]struct{} {
		m := make(map[string]struct{}, len(in))
		for _, s := range in {
			m[normalizeNameserver(s)] = struct{}{}
		}
		return m
	}
	ma, mb := toSet(a), toSet(b)
	if len(ma) != len(mb) {
		return false
	}
	for k := range ma {
		if _, ok := mb[k]; !ok {
			return false
		}
	}
	return true
}
