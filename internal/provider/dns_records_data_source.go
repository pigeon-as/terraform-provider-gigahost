package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/joakimhellum/terraform-provider-gigahost/internal/client"
	"github.com/joakimhellum/terraform-provider-gigahost/internal/datasource_dns_records"
)

var (
	_ datasource.DataSource              = &dnsRecordsDataSource{}
	_ datasource.DataSourceWithConfigure = &dnsRecordsDataSource{}
)

func NewDNSRecordsDataSource() datasource.DataSource {
	return &dnsRecordsDataSource{}
}

type dnsRecordsDataSource struct {
	client *client.Client
}

type dnsRecordsDataSourceModel struct {
	ZoneID  types.String     `tfsdk:"zone_id"`
	Records []dnsRecordModel `tfsdk:"records"`
}

type dnsRecordModel struct {
	RecordID       types.String `tfsdk:"record_id"`
	RecordName     types.String `tfsdk:"record_name"`
	RecordPriority types.Int64  `tfsdk:"record_priority"`
	RecordTTL      types.Int64  `tfsdk:"record_ttl"`
	RecordType     types.String `tfsdk:"record_type"`
	RecordValue    types.String `tfsdk:"record_value"`
}

func (d *dnsRecordsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_records"
}

func (d *dnsRecordsDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	s := datasource_dns_records.DnsRecordsDataSourceSchema(ctx)
	s.MarkdownDescription = "Lists the DNS records in a Gigahost DNS zone."

	zoneID := s.Attributes["zone_id"].(schema.StringAttribute)
	zoneID.Description = "Id of the DNS zone to list records for."
	zoneID.MarkdownDescription = zoneID.Description
	s.Attributes["zone_id"] = zoneID

	resp.Schema = s
}

func (d *dnsRecordsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = c
}

func (d *dnsRecordsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config dnsRecordsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	records, err := d.client.ListRecords(ctx, config.ZoneID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Gigahost DNS Records", err.Error())
		return
	}

	state := dnsRecordsDataSourceModel{
		ZoneID:  config.ZoneID,
		Records: make([]dnsRecordModel, 0, len(records)),
	}
	for _, rec := range records {
		model := dnsRecordModel{
			RecordID:       types.StringValue(rec.RecordID),
			RecordName:     types.StringValue(rec.RecordName),
			RecordType:     types.StringValue(rec.RecordType),
			RecordValue:    types.StringValue(rec.RecordValue),
			RecordTTL:      types.Int64Value(rec.RecordTTL),
			RecordPriority: types.Int64Null(),
		}
		if rec.RecordPriority != nil {
			model.RecordPriority = types.Int64Value(*rec.RecordPriority)
		}
		state.Records = append(state.Records, model)
	}

	tflog.Trace(ctx, "read gigahost dns records", map[string]any{"zone_id": config.ZoneID.ValueString(), "count": len(records)})
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
