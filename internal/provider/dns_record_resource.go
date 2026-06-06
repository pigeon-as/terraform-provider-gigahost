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
	"github.com/pigeon-as/terraform-provider-gigahost/internal/resource_dns_record"
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

func (r *dnsRecordResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_record"
}

func (r *dnsRecordResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	s := resource_dns_record.DnsRecordResourceSchema(ctx)
	s.MarkdownDescription = "Manages a DNS record within a Gigahost DNS zone."

	zoneID, ok := s.Attributes["zone_id"].(schema.StringAttribute)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Schema Type", `Generated attribute "zone_id" is not a string attribute. This is a bug in the provider, please report it.`)
		return
	}
	zoneID.Required = true
	zoneID.Optional = false
	zoneID.Computed = false
	zoneID.Description = "Id of the DNS zone that contains the record."
	zoneID.MarkdownDescription = zoneID.Description
	zoneID.PlanModifiers = []planmodifier.String{stringplanmodifier.RequiresReplace()}
	s.Attributes["zone_id"] = zoneID

	recordPriority, ok := s.Attributes["record_priority"].(schema.Int64Attribute)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Schema Type", `Generated attribute "record_priority" is not an int64 attribute. This is a bug in the provider, please report it.`)
		return
	}
	recordPriority.Computed = false
	s.Attributes["record_priority"] = recordPriority

	resp.Schema = s
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
	var plan resource_dns_record.DnsRecordModel
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
	var state resource_dns_record.DnsRecordModel
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
	var plan, state resource_dns_record.DnsRecordModel
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
	var state resource_dns_record.DnsRecordModel
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
	if err != nil {
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

func recordRequest(m resource_dns_record.DnsRecordModel) client.RecordRequest {
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
