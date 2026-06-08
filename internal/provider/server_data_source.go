package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pigeon-as/terraform-provider-gigahost/internal/client"
)

var (
	_ datasource.DataSource                     = &serverDataSource{}
	_ datasource.DataSourceWithConfigure        = &serverDataSource{}
	_ datasource.DataSourceWithConfigValidators = &serverDataSource{}
)

func NewServerDataSource() datasource.DataSource {
	return &serverDataSource{}
}

type serverDataSource struct {
	client *client.Client
}

type serverDataSourceModel struct {
	SrvId            types.String    `tfsdk:"srv_id"`
	SrvName          types.String    `tfsdk:"srv_name"`
	SrvHostname      types.String    `tfsdk:"srv_hostname"`
	SrvStatus        types.Bool      `tfsdk:"srv_status"`
	SrvStatusRescue  types.Bool      `tfsdk:"srv_status_rescue"`
	SrvStatusInstall types.Bool      `tfsdk:"srv_status_install"`
	SrvSuspended     types.Bool      `tfsdk:"srv_suspended"`
	SrvLocation      types.String    `tfsdk:"srv_location"`
	SrvType          types.String    `tfsdk:"srv_type"`
	SrvVpsType       types.String    `tfsdk:"srv_vps_type"`
	SrvPrimaryIp     types.String    `tfsdk:"srv_primary_ip"`
	SrvCores         types.Int64     `tfsdk:"srv_cores"`
	SrvRam           types.Int64     `tfsdk:"srv_ram"`
	ProductId        types.String    `tfsdk:"product_id"`
	OsId             types.String    `tfsdk:"os_id"`
	Os               *serverOSModel  `tfsdk:"os"`
	Ips              []serverIPModel `tfsdk:"ips"`
}

type serverOSModel struct {
	OsId      types.Int64  `tfsdk:"os_id"`
	OsName    types.String `tfsdk:"os_name"`
	OsRelease types.String `tfsdk:"os_release"`
}

type serverIPModel struct {
	IpId      types.Int64  `tfsdk:"ip_id"`
	IpAddress types.String `tfsdk:"ip_address"`
	IpV4v6    types.String `tfsdk:"ip_v4v6"`
	IpReverse types.String `tfsdk:"ip_reverse"`
	IpType    types.String `tfsdk:"ip_type"`
	IpNetmask types.String `tfsdk:"ip_netmask"`
	IpGateway types.String `tfsdk:"ip_gateway"`
}

func (d *serverDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server"
}

func (d *serverDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up a single server by id or name.",
		Attributes: map[string]schema.Attribute{
			"srv_id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Look up the server by id.",
				MarkdownDescription: "Look up the server by id.",
			},
			"srv_name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Look up the server by name.",
				MarkdownDescription: "Look up the server by name.",
			},
			"srv_hostname": schema.StringAttribute{
				Computed:            true,
				Description:         "Server hostname.",
				MarkdownDescription: "Server hostname.",
			},
			"srv_status": schema.BoolAttribute{
				Computed:            true,
				Description:         "Whether the server is running.",
				MarkdownDescription: "Whether the server is running.",
			},
			"srv_status_rescue": schema.BoolAttribute{
				Computed:            true,
				Description:         "Whether the server is in rescue mode.",
				MarkdownDescription: "Whether the server is in rescue mode.",
			},
			"srv_status_install": schema.BoolAttribute{
				Computed:            true,
				Description:         "Whether the server is installing.",
				MarkdownDescription: "Whether the server is installing.",
			},
			"srv_suspended": schema.BoolAttribute{
				Computed:            true,
				Description:         "Whether the server is suspended.",
				MarkdownDescription: "Whether the server is suspended.",
			},
			"srv_location": schema.StringAttribute{
				Computed:            true,
				Description:         "Datacenter location code.",
				MarkdownDescription: "Datacenter location code.",
			},
			"srv_type": schema.StringAttribute{
				Computed:            true,
				Description:         "Server type (vps or dedicated).",
				MarkdownDescription: "Server type (vps or dedicated).",
			},
			"srv_vps_type": schema.StringAttribute{
				Computed:            true,
				Description:         "Virtualization type (e.g. kvm).",
				MarkdownDescription: "Virtualization type (e.g. kvm).",
			},
			"srv_primary_ip": schema.StringAttribute{
				Computed:            true,
				Description:         "Primary IPv4 address.",
				MarkdownDescription: "Primary IPv4 address.",
			},
			"srv_cores": schema.Int64Attribute{
				Computed:            true,
				Description:         "Number of CPU cores.",
				MarkdownDescription: "Number of CPU cores.",
			},
			"srv_ram": schema.Int64Attribute{
				Computed:            true,
				Description:         "Memory, in GB.",
				MarkdownDescription: "Memory, in GB.",
			},
			"product_id": schema.StringAttribute{
				Computed:            true,
				Description:         "Product id.",
				MarkdownDescription: "Product id.",
			},
			"os_id": schema.StringAttribute{
				Computed:            true,
				Description:         "Installed OS image (version) id.",
				MarkdownDescription: "Installed OS image (version) id.",
			},
			"os": schema.SingleNestedAttribute{
				Computed:            true,
				Description:         "Installed operating system.",
				MarkdownDescription: "Installed operating system.",
				Attributes: map[string]schema.Attribute{
					"os_id": schema.Int64Attribute{
						Computed:            true,
						Description:         "OS image (version) id.",
						MarkdownDescription: "OS image (version) id.",
					},
					"os_name": schema.StringAttribute{
						Computed:            true,
						Description:         "OS image name.",
						MarkdownDescription: "OS image name.",
					},
					"os_release": schema.StringAttribute{
						Computed:            true,
						Description:         "OS release/version.",
						MarkdownDescription: "OS release/version.",
					},
				},
			},
			"ips": schema.ListNestedAttribute{
				Computed:            true,
				Description:         "IP addresses assigned to the server.",
				MarkdownDescription: "IP addresses assigned to the server.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"ip_id": schema.Int64Attribute{
							Computed:            true,
							Description:         "IP address id.",
							MarkdownDescription: "IP address id.",
						},
						"ip_address": schema.StringAttribute{
							Computed:            true,
							Description:         "The IP address.",
							MarkdownDescription: "The IP address.",
						},
						"ip_v4v6": schema.StringAttribute{
							Computed:            true,
							Description:         "Address family (ipv4 or ipv6).",
							MarkdownDescription: "Address family (ipv4 or ipv6).",
						},
						"ip_reverse": schema.StringAttribute{
							Computed:            true,
							Description:         "Reverse DNS (PTR) for the address.",
							MarkdownDescription: "Reverse DNS (PTR) for the address.",
						},
						"ip_type": schema.StringAttribute{
							Computed:            true,
							Description:         "Address type (primary or extra).",
							MarkdownDescription: "Address type (primary or extra).",
						},
						"ip_netmask": schema.StringAttribute{
							Computed:            true,
							Description:         "Netmask.",
							MarkdownDescription: "Netmask.",
						},
						"ip_gateway": schema.StringAttribute{
							Computed:            true,
							Description:         "Gateway.",
							MarkdownDescription: "Gateway.",
						},
					},
				},
			},
		},
	}
}

func (d *serverDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *serverDataSource) ConfigValidators(_ context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(path.MatchRoot("srv_id"), path.MatchRoot("srv_name")),
	}
}

func (d *serverDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config serverDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	servers, err := d.client.ListServers(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Gigahost Servers", err.Error())
		return
	}

	var matches []client.Server
	for _, s := range servers {
		if !config.SrvId.IsNull() && !equalID(s.SrvID, config.SrvId.ValueString()) {
			continue
		}
		if !config.SrvName.IsNull() && !strings.EqualFold(s.SrvName, config.SrvName.ValueString()) {
			continue
		}
		matches = append(matches, s)
	}

	if len(matches) == 0 {
		resp.Diagnostics.AddError("Server Not Found", "No server matches the given srv_id or srv_name on the account.")
		return
	}
	if len(matches) > 1 {
		ids := make([]string, len(matches))
		for i, m := range matches {
			ids[i] = m.SrvID
		}
		resp.Diagnostics.AddError(
			"Ambiguous Server",
			fmt.Sprintf("%d servers match (ids %s); use srv_id to select one.", len(matches), strings.Join(ids, ", ")),
		)
		return
	}

	s := matches[0]
	ips := make([]serverIPModel, 0, len(s.IPs))
	for _, ip := range s.IPs {
		ips = append(ips, serverIPModel{
			IpId:      types.Int64Value(int64(ip.IPID)),
			IpAddress: types.StringValue(ip.IPAddress),
			IpV4v6:    types.StringValue(ip.IPv4v6),
			IpReverse: types.StringValue(ip.IPReverse),
			IpType:    types.StringValue(ip.IPType),
			IpNetmask: types.StringValue(ip.IPNetmask),
			IpGateway: types.StringValue(ip.IPGateway),
		})
	}

	state := serverDataSourceModel{
		SrvId:            types.StringValue(s.SrvID),
		SrvName:          types.StringValue(s.SrvName),
		SrvHostname:      types.StringValue(s.SrvHostname),
		SrvStatus:        types.BoolValue(bool(s.SrvStatus)),
		SrvStatusRescue:  types.BoolValue(bool(s.SrvStatusRescue)),
		SrvStatusInstall: types.BoolValue(bool(s.SrvStatusInstall)),
		SrvSuspended:     types.BoolValue(bool(s.SrvSuspended)),
		SrvLocation:      types.StringValue(s.SrvLocation),
		SrvType:          types.StringValue(s.SrvType),
		SrvVpsType:       types.StringValue(s.SrvVpsType),
		SrvPrimaryIp:     types.StringValue(s.SrvPrimaryIP),
		SrvCores:         types.Int64Value(int64(s.SrvCores)),
		SrvRam:           types.Int64Value(int64(s.SrvRAM)),
		ProductId:        types.StringValue(s.ProductID),
		OsId:             types.StringValue(s.OsID),
		Os: &serverOSModel{
			OsId:      types.Int64Value(int64(s.OS.OsID)),
			OsName:    types.StringValue(s.OS.OsName),
			OsRelease: types.StringValue(s.OS.OsRelease),
		},
		Ips: ips,
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
