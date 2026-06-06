package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/pigeon-as/terraform-provider-gigahost/internal/client"
	"github.com/pigeon-as/terraform-provider-gigahost/internal/datasource_ssh_keys"
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

	elements := make([]datasource_ssh_keys.SshKeysValue, 0, len(keys))
	for _, k := range keys {
		elements = append(elements, datasource_ssh_keys.NewSshKeysValueMust(
			datasource_ssh_keys.SshKeysValue{}.AttributeTypes(ctx),
			map[string]attr.Value{
				"key_id":    types.StringValue(k.KeyID),
				"key_name":  types.StringValue(k.KeyName),
				"key_added": types.StringValue(k.KeyAdded),
				"key_data":  types.StringValue(k.KeyData),
			},
		))
	}

	list, diags := types.ListValueFrom(ctx, datasource_ssh_keys.SshKeysValue{}.Type(ctx), elements)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := datasource_ssh_keys.SshKeysModel{SshKeys: list}
	tflog.Trace(ctx, "read gigahost ssh keys", map[string]any{"count": len(keys)})
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
