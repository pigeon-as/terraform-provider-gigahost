package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pigeon-as/terraform-provider-gigahost/internal/client"
)

var (
	_ datasource.DataSource                     = &osDataSource{}
	_ datasource.DataSourceWithConfigure        = &osDataSource{}
	_ datasource.DataSourceWithConfigValidators = &osDataSource{}
)

func NewOSDataSource() datasource.DataSource {
	return &osDataSource{}
}

type osDataSource struct {
	client *client.Client
}

type osModel struct {
	Distro    types.String `tfsdk:"distro"`
	Version   types.String `tfsdk:"version"`
	OsId      types.Int64  `tfsdk:"os_id"`
	OsName    types.String `tfsdk:"os_name"`
	DistId    types.Int64  `tfsdk:"dist_id"`
	OsRelease types.String `tfsdk:"os_release"`
	OsDist    types.String `tfsdk:"os_dist"`
	OsArch    types.String `tfsdk:"os_arch"`
	OsMinram  types.Int64  `tfsdk:"os_minram"`

	OsCustomPartition types.Bool `tfsdk:"os_custom_partition"`
	OsSingleDiskOnly  types.Bool `tfsdk:"os_single_disk_only"`
	OsSupportRaid     types.Bool `tfsdk:"os_support_raid"`
	OsDedicatedOnly   types.Bool `tfsdk:"os_dedicated_only"`
}

func (d *osDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_os"
}

func (d *osDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up a single deployable OS image by distribution and version, returning the os_id used by the gigahost_server resource.",
		Attributes: map[string]schema.Attribute{
			"distro": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Filter by distribution name (e.g. \"Ubuntu\") or slug (e.g. \"ubuntu\").",
				MarkdownDescription: "Filter by distribution name (e.g. \"Ubuntu\") or slug (e.g. \"ubuntu\").",
			},
			"version": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Filter by version — matches part of the OS name (e.g. \"24.04\") or the release codename (e.g. \"noble\").",
				MarkdownDescription: "Filter by version — matches part of the OS name (e.g. \"24.04\") or the release codename (e.g. \"noble\").",
			},
			"os_id": schema.Int64Attribute{
				Computed:            true,
				Description:         "OS image id, used as os_id when deploying a gigahost_server.",
				MarkdownDescription: "OS image id, used as os_id when deploying a gigahost_server.",
			},
			"os_name": schema.StringAttribute{
				Computed:            true,
				Description:         "Full OS name, e.g. \"Ubuntu 24.04 LTS\".",
				MarkdownDescription: "Full OS name, e.g. \"Ubuntu 24.04 LTS\".",
			},
			"dist_id": schema.Int64Attribute{
				Computed:            true,
				Description:         "Distribution id.",
				MarkdownDescription: "Distribution id.",
			},
			"os_release": schema.StringAttribute{
				Computed:            true,
				Description:         "Distribution family, e.g. \"ubuntu\".",
				MarkdownDescription: "Distribution family, e.g. \"ubuntu\".",
			},
			"os_dist": schema.StringAttribute{
				Computed:            true,
				Description:         "Release codename, e.g. \"noble\".",
				MarkdownDescription: "Release codename, e.g. \"noble\".",
			},
			"os_arch": schema.StringAttribute{
				Computed:            true,
				Description:         "Architecture, e.g. \"amd64\".",
				MarkdownDescription: "Architecture, e.g. \"amd64\".",
			},
			"os_minram": schema.Int64Attribute{
				Computed:            true,
				Description:         "Minimum memory required, in GB.",
				MarkdownDescription: "Minimum memory required, in GB.",
			},
			"os_custom_partition": schema.BoolAttribute{
				Computed:            true,
				Description:         "Whether the image supports custom partitioning.",
				MarkdownDescription: "Whether the image supports custom partitioning.",
			},
			"os_single_disk_only": schema.BoolAttribute{
				Computed:            true,
				Description:         "Whether the image must be installed on a single disk.",
				MarkdownDescription: "Whether the image must be installed on a single disk.",
			},
			"os_support_raid": schema.BoolAttribute{
				Computed:            true,
				Description:         "Whether the image supports RAID.",
				MarkdownDescription: "Whether the image supports RAID.",
			},
			"os_dedicated_only": schema.BoolAttribute{
				Computed:            true,
				Description:         "Whether the image is only deployable on dedicated servers.",
				MarkdownDescription: "Whether the image is only deployable on dedicated servers.",
			},
		},
	}
}

func (d *osDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *osDataSource) ConfigValidators(_ context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.AtLeastOneOf(path.MatchRoot("distro"), path.MatchRoot("version")),
	}
}

func (d *osDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config osModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	catalog, err := d.client.GetOSCatalog(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Gigahost OS Catalog", err.Error())
		return
	}

	distro := config.Distro.ValueString()
	version := config.Version.ValueString()
	var matches []client.OSCatalogEntry
	for _, e := range catalog {
		if osMatches(e, distro, version) {
			matches = append(matches, e)
		}
	}

	if len(matches) == 0 {
		resp.Diagnostics.AddError(
			"OS Not Found",
			"No OS image matches the given filters. Use a distro (e.g. \"Ubuntu\") and/or version (e.g. \"24.04\").",
		)
		return
	}
	if len(matches) > 1 {
		names := make([]string, len(matches))
		for i, m := range matches {
			names[i] = m.OS.OsName
		}
		resp.Diagnostics.AddError(
			"Ambiguous OS",
			fmt.Sprintf("%d OS images match the given filters (%s); narrow distro or version to exactly one.", len(matches), strings.Join(names, ", ")),
		)
		return
	}

	m := matches[0]

	osID, err := strconv.ParseInt(m.OS.OsID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Unexpected Catalog Value", fmt.Sprintf("Could not parse os_id %q as an integer: %s", m.OS.OsID, err))
		return
	}
	distID, err := strconv.ParseInt(m.OS.DistID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Unexpected Catalog Value", fmt.Sprintf("Could not parse dist_id %q as an integer: %s", m.OS.DistID, err))
		return
	}

	osMinRAM := types.Int64Null()
	if m.OS.OsMinRAM != "" {
		if v, err := strconv.ParseInt(m.OS.OsMinRAM, 10, 64); err == nil {
			osMinRAM = types.Int64Value(v)
		}
	}

	state := osModel{
		Distro:    config.Distro,
		Version:   config.Version,
		OsId:      types.Int64Value(osID),
		OsName:    types.StringValue(m.OS.OsName),
		DistId:    types.Int64Value(distID),
		OsRelease: types.StringValue(m.OS.OsRelease),
		OsDist:    types.StringValue(m.OS.OsDist),
		OsArch:    types.StringValue(m.OS.OsArch),
		OsMinram:  osMinRAM,

		OsCustomPartition: types.BoolValue(bool(m.OS.OsCustomPartition)),
		OsSingleDiskOnly:  types.BoolValue(bool(m.OS.OsSingleDiskOnly)),
		OsSupportRaid:     types.BoolValue(bool(m.OS.OsSupportRAID)),
		OsDedicatedOnly:   types.BoolValue(bool(m.OS.OsDedicatedOnly)),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
