package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/pigeon-as/terraform-provider-gigahost/internal/client"
	"github.com/pigeon-as/terraform-provider-gigahost/internal/datasource_dns_zones"
)

var (
	_ datasource.DataSource              = &dnsZonesDataSource{}
	_ datasource.DataSourceWithConfigure = &dnsZonesDataSource{}
)

func NewDNSZonesDataSource() datasource.DataSource {
	return &dnsZonesDataSource{}
}

type dnsZonesDataSource struct {
	client *client.Client
}

func (d *dnsZonesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_zones"
}

func (d *dnsZonesDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasource_dns_zones.DnsZonesDataSourceSchema(ctx)
	resp.Schema.MarkdownDescription = "Lists the DNS zones on the Gigahost account."
}

func (d *dnsZonesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *dnsZonesDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	zones, err := d.client.ListDnsZones(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Gigahost DNS Zones", err.Error())
		return
	}

	elements := make([]datasource_dns_zones.ZonesValue, 0, len(zones))
	for _, z := range zones {
		elements = append(elements, datasource_dns_zones.NewZonesValueMust(
			datasource_dns_zones.ZonesValue{}.AttributeTypes(ctx),
			map[string]attr.Value{
				"zone_id":        types.StringValue(z.ZoneID),
				"zone_name":      types.StringValue(z.ZoneName),
				"zone_type":      types.StringValue(z.ZoneType),
				"zone_active":    types.BoolValue(bool(z.ZoneActive)),
				"zone_protected": types.BoolValue(bool(z.ZoneProtected)),
				"external_dns":   types.BoolValue(bool(z.ExternalDNS)),
			},
		))
	}

	list, diags := types.ListValueFrom(ctx, datasource_dns_zones.ZonesValue{}.Type(ctx), elements)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := datasource_dns_zones.DnsZonesModel{Zones: list}
	tflog.Trace(ctx, "read gigahost dns zones", map[string]any{"count": len(zones)})
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
