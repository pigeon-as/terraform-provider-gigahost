package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/pigeon-as/terraform-provider-gigahost/internal/client"
	"github.com/pigeon-as/terraform-provider-gigahost/internal/datasource_dns_records"
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

func (d *dnsRecordsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_records"
}

func (d *dnsRecordsDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	s := datasource_dns_records.DnsRecordsDataSourceSchema(ctx)
	s.MarkdownDescription = "Lists the DNS records in a Gigahost DNS zone."

	zoneID, ok := s.Attributes["zone_id"].(schema.StringAttribute)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Schema Type",
			"The generated DNS records schema does not have the expected attribute types. This is a bug in the provider, please report it.",
		)
		return
	}
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
	var config datasource_dns_records.DnsRecordsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	records, err := d.client.ListRecords(ctx, config.ZoneId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Gigahost DNS Records", err.Error())
		return
	}

	elements := make([]datasource_dns_records.RecordsValue, 0, len(records))
	for _, rec := range records {
		priority := types.Int64Null()
		if rec.RecordPriority != nil {
			priority = types.Int64Value(int64(*rec.RecordPriority))
		}
		elements = append(elements, datasource_dns_records.NewRecordsValueMust(
			datasource_dns_records.RecordsValue{}.AttributeTypes(ctx),
			map[string]attr.Value{
				"record_id":       types.StringValue(rec.RecordID),
				"record_name":     types.StringValue(rec.RecordName),
				"record_type":     types.StringValue(rec.RecordType),
				"record_value":    types.StringValue(rec.RecordValue),
				"record_ttl":      types.Int64Value(int64(rec.RecordTTL)),
				"record_priority": priority,
			},
		))
	}

	list, diags := types.ListValueFrom(ctx, datasource_dns_records.RecordsValue{}.Type(ctx), elements)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := datasource_dns_records.DnsRecordsModel{ZoneId: config.ZoneId, Records: list}
	tflog.Trace(ctx, "read gigahost dns records", map[string]any{"zone_id": config.ZoneId.ValueString(), "count": len(records)})
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
