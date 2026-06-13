package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pigeon-as/terraform-provider-gigahost/internal/client"
	"github.com/pigeon-as/terraform-provider-gigahost/internal/datasource_servers"
)

var (
	_ datasource.DataSource              = &serversDataSource{}
	_ datasource.DataSourceWithConfigure = &serversDataSource{}
)

func NewServersDataSource() datasource.DataSource {
	return &serversDataSource{}
}

type serversDataSource struct {
	client *client.Client
}

func (d *serversDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_servers"
}

func (d *serversDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	s := datasource_servers.ServersDataSourceSchema(ctx)
	s.MarkdownDescription = "Lists the servers on the Gigahost account."
	resp.Schema = s
}

func (d *serversDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *serversDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	servers, err := d.client.ListServers(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Gigahost Servers", err.Error())
		return
	}

	elements := make([]datasource_servers.ServersValue, 0, len(servers))
	for _, s := range servers {
		os, osDiags := types.ObjectValue(
			datasource_servers.OsValue{}.AttributeTypes(ctx),
			map[string]attr.Value{
				"os_id":      types.Int64Value(int64(s.OS.OSID)),
				"os_name":    types.StringValue(s.OS.OSName),
				"os_release": types.StringValue(s.OS.OSRelease),
			},
		)
		resp.Diagnostics.Append(osDiags...)
		if resp.Diagnostics.HasError() {
			return
		}

		order, orderDiags := types.ObjectValue(
			datasource_servers.OrderValue{}.AttributeTypes(ctx),
			map[string]attr.Value{
				"order_id":     types.StringValue(s.Order.OrderID),
				"order_number": types.StringValue(s.Order.OrderNumber),
				"order_status": types.StringValue(s.Order.OrderStatus),
				"product_id":   types.StringValue(s.Order.ProductID),
				"product_name": types.StringValue(s.Order.ProductName),
			},
		)
		resp.Diagnostics.Append(orderDiags...)
		if resp.Diagnostics.HasError() {
			return
		}

		datacenter, dcDiags := types.ObjectValue(
			datasource_servers.DatacenterValue{}.AttributeTypes(ctx),
			map[string]attr.Value{
				"region_id":   types.StringValue(s.Datacenter.RegionID),
				"region_name": types.StringValue(s.Datacenter.RegionName),
			},
		)
		resp.Diagnostics.Append(dcDiags...)
		if resp.Diagnostics.HasError() {
			return
		}

		ipElems := make([]datasource_servers.IpsValue, 0, len(s.IPs))
		for _, ip := range s.IPs {
			ipElems = append(ipElems, datasource_servers.NewIpsValueMust(
				datasource_servers.IpsValue{}.AttributeTypes(ctx),
				map[string]attr.Value{
					"ip_id":      types.Int64Value(int64(ip.IPID)),
					"ip_address": types.StringValue(ip.IPAddress),
					"ip_v4v6":    types.StringValue(ip.IPv4v6),
					"ip_reverse": types.StringValue(ip.IPReverse),
					"ip_type":    types.StringValue(ip.IPType),
					"ip_netmask": types.StringValue(ip.IPNetmask),
					"ip_gateway": types.StringValue(ip.IPGateway),
				},
			))
		}
		ips, diags := types.ListValueFrom(ctx, datasource_servers.IpsValue{}.Type(ctx), ipElems)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		elements = append(elements, datasource_servers.NewServersValueMust(
			datasource_servers.ServersValue{}.AttributeTypes(ctx),
			map[string]attr.Value{
				"srv_id":              types.StringValue(s.SrvID),
				"srv_name":            types.StringValue(s.SrvName),
				"srv_hostname":        types.StringValue(s.SrvHostname),
				"srv_status":          types.BoolValue(bool(s.SrvStatus)),
				"srv_status_rescue":   types.BoolValue(bool(s.SrvStatusRescue)),
				"srv_status_install":  types.BoolValue(bool(s.SrvStatusInstall)),
				"srv_suspended":       types.BoolValue(bool(s.SrvSuspended)),
				"srv_location":        types.StringValue(s.SrvLocation),
				"srv_type":            types.StringValue(s.SrvType),
				"srv_vps_type":        types.StringValue(s.SrvVpsType),
				"srv_primary_ip":      types.StringValue(s.SrvPrimaryIP),
				"srv_cores":           types.Int64Value(int64(s.SrvCores)),
				"srv_ram":             types.Int64Value(int64(s.SrvRAM)),
				"product_id":          types.StringValue(s.ProductID),
				"os_id":               types.StringValue(s.OSID),
				"srv_feature_backups": types.BoolValue(bool(s.SrvFeatureBackups)),
				"os":                  os,
				"ips":                 ips,
				"order":               order,
				"datacenter":          datacenter,
			},
		))
	}

	list, diags := types.ListValueFrom(ctx, datasource_servers.ServersValue{}.Type(ctx), elements)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := datasource_servers.ServersModel{Servers: list}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
