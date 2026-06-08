package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pigeon-as/terraform-provider-gigahost/internal/client"
)

var (
	_ resource.Resource                = &dnsRecordResource{}
	_ resource.ResourceWithConfigure   = &dnsRecordResource{}
	_ resource.ResourceWithImportState = &dnsRecordResource{}
)

func NewDNSRecordResource() resource.Resource {
	return &dnsRecordResource{}
}

type dnsRecordResource struct {
	client *client.Client
}

type dnsRecordResourceModel struct {
	RecordId       types.String `tfsdk:"record_id"`
	RecordName     types.String `tfsdk:"record_name"`
	RecordPriority types.Int64  `tfsdk:"record_priority"`
	RecordTtl      types.Int64  `tfsdk:"record_ttl"`
	RecordType     types.String `tfsdk:"record_type"`
	RecordValue    types.String `tfsdk:"record_value"`
	ZoneId         types.String `tfsdk:"zone_id"`
}

func (r *dnsRecordResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_record"
}

func (r *dnsRecordResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a DNS record within a Gigahost DNS zone.",
		Attributes: map[string]schema.Attribute{
			"record_id": schema.StringAttribute{
				Computed:            true,
				Description:         "Record id.",
				MarkdownDescription: "Record id.",
			},
			"record_name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Record name (host); defaults to \"@\".",
				MarkdownDescription: "Record name (host); defaults to \"@\".",
				Default:             stringdefault.StaticString("@"),
			},
			"record_priority": schema.Int64Attribute{
				Optional:            true,
				Description:         "Priority (used by MX and SRV records).",
				MarkdownDescription: "Priority (used by MX and SRV records).",
			},
			"record_ttl": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Description:         "Time to live, in seconds; defaults to 3600.",
				MarkdownDescription: "Time to live, in seconds; defaults to 3600.",
				Default:             int64default.StaticInt64(3600),
			},
			"record_type": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Record type; defaults to \"A\".",
				MarkdownDescription: "Record type; defaults to \"A\".",
				Default:             stringdefault.StaticString("A"),
			},
			"record_value": schema.StringAttribute{
				Required:            true,
				Description:         "Record value (content).",
				MarkdownDescription: "Record value (content).",
			},
			"zone_id": schema.StringAttribute{
				Required:            true,
				Description:         "Id of the DNS zone that contains the record.",
				MarkdownDescription: "Id of the DNS zone that contains the record.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
		},
	}
}

func (r *dnsRecordResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *dnsRecordResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan dnsRecordResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	record, err := r.client.CreateRecord(ctx, plan.ZoneId.ValueString(), recordRequest(plan))
	if err != nil {
		resp.Diagnostics.AddError("Unable to Create Gigahost DNS Record", err.Error())
		return
	}

	state := plan
	state.RecordId = types.StringValue(record.RecordID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dnsRecordResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state dnsRecordResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	record, err := r.client.GetRecord(ctx, state.ZoneId.ValueString(), state.RecordId.ValueString())
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to Read Gigahost DNS Record", err.Error())
		return
	}

	state.RecordName = types.StringValue(record.RecordName)
	state.RecordType = types.StringValue(record.RecordType)

	value := record.RecordValue
	if !state.RecordValue.IsNull() && client.NormalizeRecordValue(record.RecordType, state.RecordValue.ValueString()) == record.RecordValue {
		value = state.RecordValue.ValueString()
	}
	state.RecordValue = types.StringValue(value)

	state.RecordTtl = types.Int64Value(int64(record.RecordTTL))
	state.RecordPriority = types.Int64Null()
	if record.RecordPriority != nil {
		state.RecordPriority = types.Int64Value(int64(*record.RecordPriority))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dnsRecordResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state dnsRecordResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	record, err := r.client.UpdateRecord(ctx, plan.ZoneId.ValueString(), state.RecordId.ValueString(), recordRequest(plan))
	if err != nil {
		resp.Diagnostics.AddError("Unable to Update Gigahost DNS Record", err.Error())
		return
	}

	newState := plan
	newState.RecordId = types.StringValue(record.RecordID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *dnsRecordResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state dnsRecordResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteRecord(
		ctx,
		state.ZoneId.ValueString(),
		state.RecordId.ValueString(),
		state.RecordName.ValueString(),
		state.RecordType.ValueString(),
		state.RecordValue.ValueString(),
	)
	if err != nil && !errors.Is(err, client.ErrNotFound) {
		resp.Diagnostics.AddError("Unable to Delete Gigahost DNS Record", err.Error())
	}
}

func (r *dnsRecordResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	zoneID, recordID, ok := strings.Cut(req.ID, "/")
	if !ok || zoneID == "" || recordID == "" {
		resp.Diagnostics.AddError(
			"Invalid Import Id",
			fmt.Sprintf("Expected import id in the format \"zone_id/record_id\", got: %q.", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("zone_id"), zoneID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("record_id"), recordID)...)
}

func recordRequest(m dnsRecordResourceModel) client.RecordRequest {
	body := client.RecordRequest{
		RecordName:  m.RecordName.ValueString(),
		RecordType:  m.RecordType.ValueString(),
		RecordValue: m.RecordValue.ValueString(),
		RecordTTL:   m.RecordTtl.ValueInt64(),
	}
	if !m.RecordPriority.IsNull() && !m.RecordPriority.IsUnknown() {
		priority := m.RecordPriority.ValueInt64()
		body.RecordPriority = &priority
	}
	return body
}
