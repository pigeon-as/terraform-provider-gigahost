package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/joakimhellum/terraform-provider-gigahost/internal/client"
	"github.com/joakimhellum/terraform-provider-gigahost/internal/datasource_account"
)

var (
	_ datasource.DataSource              = &accountDataSource{}
	_ datasource.DataSourceWithConfigure = &accountDataSource{}
)

func NewAccountDataSource() datasource.DataSource {
	return &accountDataSource{}
}

type accountDataSource struct {
	client *client.Client
}

type accountDataSourceModel struct {
	CustID           types.String `tfsdk:"cust_id"`
	CustName         types.String `tfsdk:"cust_name"`
	CustAddress      types.String `tfsdk:"cust_address"`
	CustZipcode      types.String `tfsdk:"cust_zipcode"`
	CustCity         types.String `tfsdk:"cust_city"`
	CustCountry      types.String `tfsdk:"cust_country"`
	CustPhone        types.String `tfsdk:"cust_phone"`
	CustEmail        types.String `tfsdk:"cust_email"`
	CustCompanyNo    types.String `tfsdk:"cust_company_no"`
	CustBillingEmail types.String `tfsdk:"cust_billing_email"`
}

func (d *accountDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_account"
}

func (d *accountDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasource_account.AccountDataSourceSchema(ctx)
	resp.Schema.MarkdownDescription = "Retrieves the authenticated Gigahost account profile."
}

func (d *accountDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *accountDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	account, err := d.client.GetAccount(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Gigahost Account", err.Error())
		return
	}

	state := accountDataSourceModel{
		CustID:           types.StringValue(account.CustID),
		CustName:         types.StringValue(account.CustName),
		CustAddress:      types.StringValue(account.CustAddress),
		CustZipcode:      types.StringValue(account.CustZipcode),
		CustCity:         types.StringValue(account.CustCity),
		CustCountry:      types.StringValue(account.CustCountry),
		CustPhone:        types.StringValue(account.CustPhone),
		CustEmail:        types.StringValue(account.CustEmail),
		CustCompanyNo:    types.StringValue(account.CustCompanyNo),
		CustBillingEmail: types.StringValue(account.CustBillingEmail),
	}

	tflog.Trace(ctx, "read gigahost account")
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
