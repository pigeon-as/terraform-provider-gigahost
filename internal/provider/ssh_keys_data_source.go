package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/joakimhellum/terraform-provider-gigahost/internal/client"
	"github.com/joakimhellum/terraform-provider-gigahost/internal/datasource_ssh_keys"
)

var (
	_ datasource.DataSource              = &sshKeysDataSource{}
	_ datasource.DataSourceWithConfigure = &sshKeysDataSource{}
)

func NewSSHKeysDataSource() datasource.DataSource {
	return &sshKeysDataSource{}
}

type sshKeysDataSource struct {
	client *client.Client
}

type sshKeysDataSourceModel struct {
	SSHKeys []sshKeyModel `tfsdk:"ssh_keys"`
}

type sshKeyModel struct {
	KeyID    types.String `tfsdk:"key_id"`
	KeyName  types.String `tfsdk:"key_name"`
	KeyAdded types.String `tfsdk:"key_added"`
	KeyData  types.String `tfsdk:"key_data"`
}

func (d *sshKeysDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ssh_keys"
}

func (d *sshKeysDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasource_ssh_keys.SshKeysDataSourceSchema(ctx)
	resp.Schema.MarkdownDescription = "Lists the SSH keys registered on the Gigahost account."
}

func (d *sshKeysDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *sshKeysDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	keys, err := d.client.ListSSHKeys(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Gigahost SSH Keys", err.Error())
		return
	}

	var state sshKeysDataSourceModel
	for _, k := range keys {
		state.SSHKeys = append(state.SSHKeys, sshKeyModel{
			KeyID:    types.StringValue(k.KeyID),
			KeyName:  types.StringValue(k.KeyName),
			KeyAdded: types.StringValue(k.KeyAdded),
			KeyData:  types.StringValue(k.KeyData),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
