package provider

import (
	"context"
	"fmt"
	"strconv"

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
	OsName    types.String `tfsdk:"os_name"`
	OsDist    types.String `tfsdk:"os_dist"`
	OsID      types.Int64  `tfsdk:"os_id"`
	DistID    types.Int64  `tfsdk:"dist_id"`
	OsRelease types.String `tfsdk:"os_release"`
	OsArch    types.String `tfsdk:"os_arch"`
	OsMinRAM  types.Int64  `tfsdk:"os_minram"`

	OsCustomPartition types.Bool `tfsdk:"os_custom_partition"`
	OsSingleDiskOnly  types.Bool `tfsdk:"os_single_disk_only"`
	OsSupportRAID     types.Bool `tfsdk:"os_support_raid"`
	OsDedicatedOnly   types.Bool `tfsdk:"os_dedicated_only"`
}

func (d *osDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_os"
}

func (d *osDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up a deployable OS image by name or release codename, returning the os_id used by the gigahost_server resource.",
		Attributes: map[string]schema.Attribute{
			"os_name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Look up the image by its full name, e.g. \"Ubuntu 24.04 LTS\".",
				MarkdownDescription: "Look up the image by its full name, e.g. \"Ubuntu 24.04 LTS\".",
			},
			"os_dist": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Look up the image by its release codename, e.g. \"noble\".",
				MarkdownDescription: "Look up the image by its release codename, e.g. \"noble\".",
			},
			"os_id": schema.Int64Attribute{
				Computed:            true,
				Description:         "OS image id, used as os_id when deploying a gigahost_server.",
				MarkdownDescription: "OS image id, used as os_id when deploying a gigahost_server.",
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
		datasourcevalidator.ExactlyOneOf(path.MatchRoot("os_name"), path.MatchRoot("os_dist")),
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

	lookup := config.OsName.ValueString()
	if config.OsName.IsNull() {
		lookup = config.OsDist.ValueString()
	}
	entry, err := findOS(catalog, lookup)
	if err != nil {
		resp.Diagnostics.AddError("OS Not Found", err.Error())
		return
	}
	m := entry.OS

	osID, err := strconv.ParseInt(m.OSID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Unexpected Catalog Value", fmt.Sprintf("Could not parse os_id %q as an integer: %s", m.OSID, err))
		return
	}
	distID, err := strconv.ParseInt(m.DistID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Unexpected Catalog Value", fmt.Sprintf("Could not parse dist_id %q as an integer: %s", m.DistID, err))
		return
	}

	osMinRAM := types.Int64Null()
	if m.OSMinRAM != "" {
		if v, err := strconv.ParseInt(m.OSMinRAM, 10, 64); err == nil {
			osMinRAM = types.Int64Value(v)
		}
	}

	state := osModel{
		OsName:    types.StringValue(m.OSName),
		OsDist:    types.StringValue(m.OSDist),
		OsID:      types.Int64Value(osID),
		DistID:    types.Int64Value(distID),
		OsRelease: types.StringValue(m.OSRelease),
		OsArch:    types.StringValue(m.OSArch),
		OsMinRAM:  osMinRAM,

		OsCustomPartition: types.BoolValue(bool(m.OSCustomPartition)),
		OsSingleDiskOnly:  types.BoolValue(bool(m.OSSingleDiskOnly)),
		OsSupportRAID:     types.BoolValue(bool(m.OSSupportRAID)),
		OsDedicatedOnly:   types.BoolValue(bool(m.OSDedicatedOnly)),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
