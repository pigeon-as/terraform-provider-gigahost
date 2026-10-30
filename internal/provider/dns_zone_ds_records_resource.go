package provider

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pigeon-as/terraform-provider-gigahost/internal/client"
)

var (
	_ resource.Resource              = &dnsZoneDSRecordsResource{}
	_ resource.ResourceWithConfigure = &dnsZoneDSRecordsResource{}
)

func NewDNSZoneDSRecordsResource() resource.Resource {
	return &dnsZoneDSRecordsResource{}
}

type dnsZoneDSRecordsResource struct {
	client *client.Client
}

type dnsZoneDSRecordsResourceModel struct {
	ZoneID    types.String `tfsdk:"zone_id"`
	DSRecords types.Set    `tfsdk:"ds_records"`
}

type dsRecordModel struct {
	KeyTag     types.Int64  `tfsdk:"key_tag"`
	Algorithm  types.Int64  `tfsdk:"algorithm"`
	DigestType types.Int64  `tfsdk:"digest_type"`
	Digest     types.String `tfsdk:"digest"`
	PublicKey  types.String `tfsdk:"public_key"`
	KeyType    types.Int64  `tfsdk:"key_type"`
}

func (r *dnsZoneDSRecordsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_zone_ds_records"
}

func (r *dnsZoneDSRecordsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Submits DS records to the registry for a domain registered with Gigahost but served by external nameservers. " +
			"The configured set replaces all DS records at the registry. The API has no endpoint to read external DS records back, so drift is not detected. " +
			"Destroying the resource disables DNSSEC for the domain, which removes the DS records.",
		Attributes: map[string]schema.Attribute{
			"zone_id": schema.StringAttribute{
				Required:            true,
				Description:         "Id of the DNS zone (registered domain).",
				MarkdownDescription: "Id of the DNS zone (registered domain).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"ds_records": schema.SetNestedAttribute{
				Required:            true,
				Description:         "DS records to submit to the registry. At least one is required.",
				MarkdownDescription: "DS records to submit to the registry. At least one is required.",
				Validators:          []validator.Set{setvalidator.SizeAtLeast(1)},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"key_tag": schema.Int64Attribute{
							Required:            true,
							Description:         "Key tag (0-65535).",
							MarkdownDescription: "Key tag (0-65535).",
							Validators:          []validator.Int64{int64validator.Between(0, 65535)},
						},
						"algorithm": schema.Int64Attribute{
							Required:            true,
							Description:         "DNSSEC algorithm number. Valid values are 5, 7, 8, 10, 13, 14, 15 and 16.",
							MarkdownDescription: "DNSSEC algorithm number. Valid values are `5`, `7`, `8`, `10`, `13`, `14`, `15` and `16`.",
							Validators:          []validator.Int64{int64validator.OneOf(5, 7, 8, 10, 13, 14, 15, 16)},
						},
						"digest_type": schema.Int64Attribute{
							Required:            true,
							Description:         "Digest type: 1 (SHA-1), 2 (SHA-256) or 4 (SHA-384).",
							MarkdownDescription: "Digest type: `1` (SHA-1), `2` (SHA-256) or `4` (SHA-384). International TLDs accept only `1` and `2`.",
							Validators:          []validator.Int64{int64validator.OneOf(1, 2, 4)},
						},
						"digest": schema.StringAttribute{
							Required:            true,
							Description:         "Hexadecimal digest. Its length must match the digest type.",
							MarkdownDescription: "Hexadecimal digest. Its length must match the digest type.",
							Validators:          []validator.String{stringvalidator.RegexMatches(hexRegexp, "must be a hexadecimal string")},
						},
						"public_key": schema.StringAttribute{
							Optional:            true,
							Description:         "Base64 DNSKEY public key.",
							MarkdownDescription: "Base64 DNSKEY public key. Required by some registries (for example `.be`, `.ch`, `.cz`, `.de`, `.ee`, `.eu`, `.li` and `.nl`).",
						},
						"key_type": schema.Int64Attribute{
							Optional:            true,
							Computed:            true,
							Default:             int64default.StaticInt64(257),
							Description:         "DNSKEY flags: 257 (KSK) or 256 (ZSK). Defaults to 257.",
							MarkdownDescription: "DNSKEY flags: `257` (KSK) or `256` (ZSK). Defaults to `257`.",
							Validators:          []validator.Int64{int64validator.OneOf(256, 257)},
						},
					},
				},
			},
		},
	}
}

func (r *dnsZoneDSRecordsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *dnsZoneDSRecordsResource) submit(ctx context.Context, plan *dnsZoneDSRecordsResourceModel) error {
	var models []dsRecordModel
	if diags := plan.DSRecords.ElementsAs(ctx, &models, false); diags.HasError() {
		return errors.New("reading ds_records from plan")
	}
	records := make([]client.DSRecord, 0, len(models))
	for _, m := range models {
		rec := client.DSRecord{
			KeyTag:     m.KeyTag.ValueInt64(),
			Algorithm:  m.Algorithm.ValueInt64(),
			DigestType: m.DigestType.ValueInt64(),
			Digest:     m.Digest.ValueString(),
			PublicKey:  m.PublicKey.ValueString(),
		}
		if !m.KeyType.IsNull() && !m.KeyType.IsUnknown() {
			kt := m.KeyType.ValueInt64()
			rec.KeyType = &kt
		}
		records = append(records, rec)
	}
	return r.client.SubmitExternalDSRecords(ctx, plan.ZoneID.ValueString(), records)
}

func (r *dnsZoneDSRecordsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan dnsZoneDSRecordsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.submit(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Unable to Submit Gigahost DS Records", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dnsZoneDSRecordsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state dnsZoneDSRecordsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The API cannot return external DS records in structured form, so only
	// the existence of the zone is verified.
	if _, err := r.client.GetZone(ctx, state.ZoneID.ValueString()); err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to Read Gigahost DNS Zone", err.Error())
	}
}

func (r *dnsZoneDSRecordsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan dnsZoneDSRecordsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Submitting a set replaces all existing DS records.
	if err := r.submit(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Unable to Submit Gigahost DS Records", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dnsZoneDSRecordsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state dnsZoneDSRecordsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// There is no DS record delete endpoint; disabling DNSSEC removes them.
	if err := r.client.SetDNSSEC(ctx, state.ZoneID.ValueString(), false); err != nil && !errors.Is(err, client.ErrNotFound) {
		resp.Diagnostics.AddError("Unable to Remove Gigahost DS Records", err.Error())
	}
}

var hexRegexp = regexp.MustCompile(`^[0-9a-fA-F]+$`)
