package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/joakimhellum/terraform-provider-gigahost/internal/client"
	"github.com/joakimhellum/terraform-provider-gigahost/internal/datasource_dns_zones"
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

type dnsZonesDataSourceModel struct {
	Zones []dnsZoneModel `tfsdk:"zones"`
}

type dnsZoneModel struct {
	ZoneID           types.String `tfsdk:"zone_id"`
	ZoneName         types.String `tfsdk:"zone_name"`
	ZoneType         types.String `tfsdk:"zone_type"`
	ZoneActive       types.String `tfsdk:"zone_active"`
	ZoneProtected    types.String `tfsdk:"zone_protected"`
	ZoneIsRegistered types.String `tfsdk:"zone_is_registered"`
	DomainRegistrar  types.String `tfsdk:"domain_registrar"`
	DomainStatus     types.String `tfsdk:"domain_status"`
	DomainExpiryDate types.String `tfsdk:"domain_expiry_date"`
	DomainAutoRenew  types.String `tfsdk:"domain_auto_renew"`
	ExternalDNS      types.String `tfsdk:"external_dns"`
	RecordCount      types.String `tfsdk:"record_count"`
	ZoneUpdated      types.String `tfsdk:"zone_updated"`
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

	state := dnsZonesDataSourceModel{Zones: make([]dnsZoneModel, 0, len(zones))}
	for _, z := range zones {
		state.Zones = append(state.Zones, dnsZoneModel{
			ZoneID:           types.StringValue(z.ZoneID),
			ZoneName:         types.StringValue(z.ZoneName),
			ZoneType:         types.StringValue(z.ZoneType),
			ZoneActive:       types.StringValue(z.ZoneActive),
			ZoneProtected:    types.StringValue(z.ZoneProtected),
			ZoneIsRegistered: types.StringValue(z.ZoneIsRegistered),
			DomainRegistrar:  types.StringValue(z.DomainRegistrar),
			DomainStatus:     types.StringValue(z.DomainStatus),
			DomainExpiryDate: types.StringValue(z.DomainExpiryDate),
			DomainAutoRenew:  types.StringValue(z.DomainAutoRenew),
			ExternalDNS:      types.StringValue(z.ExternalDNS),
			RecordCount:      types.StringValue(z.RecordCount),
			ZoneUpdated:      types.StringValue(z.ZoneUpdated),
		})
	}

	tflog.Trace(ctx, "read gigahost dns zones", map[string]any{"count": len(zones)})
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
