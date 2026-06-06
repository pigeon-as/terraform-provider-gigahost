package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pigeon-as/terraform-provider-gigahost/internal/client"
)

var _ provider.Provider = &GigahostProvider{}

type GigahostProvider struct {
	version string
}

type GigahostProviderModel struct {
	APIToken types.String `tfsdk:"api_token"`
	BaseURL  types.String `tfsdk:"base_url"`
}

func (p *GigahostProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "gigahost"
	resp.Version = p.version
}

func (p *GigahostProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The Gigahost provider manages resources on a [Gigahost](https://gigahost.no) account " +
			"through the [Gigahost API](https://gigahost.no/en/api-dokumentasjon).",
		Attributes: map[string]schema.Attribute{
			"api_token": schema.StringAttribute{
				MarkdownDescription: "Gigahost API token used as a bearer token (for example `flux_live_...`). " +
					"Create one under **Account → API keys**. May also be set with the `GIGAHOST_API_TOKEN` " +
					"environment variable.",
				Optional:  true,
				Sensitive: true,
			},
			"base_url": schema.StringAttribute{
				MarkdownDescription: "Base URL of the Gigahost API. Defaults to `" + client.DefaultAddress + "`. " +
					"May also be set with the `GIGAHOST_BASE_URL` environment variable.",
				Optional: true,
			},
		},
	}
}

func (p *GigahostProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config GigahostProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.APIToken.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_token"),
			"Unknown Gigahost API token",
			"The provider cannot be configured with an unknown value for api_token. "+
				"Set a known value, or use the GIGAHOST_API_TOKEN environment variable.",
		)
	}
	if config.BaseURL.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("base_url"),
			"Unknown Gigahost base URL",
			"The provider cannot be configured with an unknown value for base_url.",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	token := os.Getenv("GIGAHOST_API_TOKEN")
	if !config.APIToken.IsNull() {
		token = config.APIToken.ValueString()
	}

	baseURL := os.Getenv("GIGAHOST_BASE_URL")
	if !config.BaseURL.IsNull() {
		baseURL = config.BaseURL.ValueString()
	}

	if token == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_token"),
			"Missing Gigahost API token",
			"The provider requires a Gigahost API token. Set the api_token attribute or the "+
				"GIGAHOST_API_TOKEN environment variable.",
		)
		return
	}

	c, err := client.NewClient(&client.Config{
		Address:   baseURL,
		Token:     token,
		UserAgent: "terraform-provider-gigahost/" + p.version,
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to Create Gigahost API Client", err.Error())
		return
	}

	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *GigahostProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewDNSZoneResource,
		NewDNSRecordResource,
		NewDNSRedirectResource,
		NewSSHKeyResource,
	}
}

func (p *GigahostProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewAccountDataSource,
		NewSSHKeysDataSource,
		NewDNSZonesDataSource,
		NewDNSRecordsDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &GigahostProvider{
			version: version,
		}
	}
}
