package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pigeon-as/terraform-provider-gigahost/internal/client"
)

var (
	_ datasource.DataSource              = &dnsZoneDataSource{}
	_ datasource.DataSourceWithConfigure = &dnsZoneDataSource{}
)

func NewDNSZoneDataSource() datasource.DataSource {
	return &dnsZoneDataSource{}
}

type dnsZoneDataSource struct {
	client *client.Client
}

type dnsZoneDataSourceModel struct {
	ZoneName      types.String `tfsdk:"zone_name"`
	ZoneId        types.String `tfsdk:"zone_id"`
	ZoneType      types.String `tfsdk:"zone_type"`
	ZoneActive    types.Bool   `tfsdk:"zone_active"`
	ZoneProtected types.Bool   `tfsdk:"zone_protected"`
	ExternalDns   types.Bool   `tfsdk:"external_dns"`
}

func (d *dnsZoneDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_zone"
}

func (d *dnsZoneDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up a single DNS zone by domain name.",
		Attributes: map[string]schema.Attribute{
			"zone_name": schema.StringAttribute{
				Required:            true,
				Description:         "Domain name of the zone to look up.",
				MarkdownDescription: "Domain name of the zone to look up.",
			},
			"zone_id": schema.StringAttribute{
				Computed:            true,
				Description:         "Zone id.",
				MarkdownDescription: "Zone id.",
			},
			"zone_type": schema.StringAttribute{
				Computed:            true,
				Description:         "Zone type (NATIVE, MASTER, or SLAVE).",
				MarkdownDescription: "Zone type (NATIVE, MASTER, or SLAVE).",
			},
			"zone_active": schema.BoolAttribute{
				Computed:            true,
				Description:         "Whether the zone is active.",
				MarkdownDescription: "Whether the zone is active.",
			},
			"zone_protected": schema.BoolAttribute{
				Computed:            true,
				Description:         "Whether the zone is protected against deletion.",
				MarkdownDescription: "Whether the zone is protected against deletion.",
			},
			"external_dns": schema.BoolAttribute{
				Computed:            true,
				Description:         "Whether the zone is served by external nameservers.",
				MarkdownDescription: "Whether the zone is served by external nameservers.",
			},
		},
	}
}

func (d *dnsZoneDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *dnsZoneDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config dnsZoneDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zones, err := d.client.ListDnsZones(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Gigahost DNS Zones", err.Error())
		return
	}

	name := config.ZoneName.ValueString()
	var matches []client.DnsZone
	for _, z := range zones {
		if strings.EqualFold(z.ZoneName, name) {
			matches = append(matches, z)
		}
	}

	switch len(matches) {
	case 0:
		resp.Diagnostics.AddError("DNS Zone Not Found", fmt.Sprintf("No DNS zone named %q on the account.", name))
		return
	case 1:
	default:
		resp.Diagnostics.AddError("Ambiguous DNS Zone", fmt.Sprintf("%d DNS zones named %q on the account.", len(matches), name))
		return
	}

	z := matches[0]
	state := dnsZoneDataSourceModel{
		ZoneName:      types.StringValue(z.ZoneName),
		ZoneId:        types.StringValue(z.ZoneID),
		ZoneType:      types.StringValue(z.ZoneType),
		ZoneActive:    types.BoolValue(bool(z.ZoneActive)),
		ZoneProtected: types.BoolValue(bool(z.ZoneProtected)),
		ExternalDns:   types.BoolValue(bool(z.ExternalDNS)),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
