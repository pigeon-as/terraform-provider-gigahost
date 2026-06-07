package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pigeon-as/terraform-provider-gigahost/internal/client"
	"github.com/pigeon-as/terraform-provider-gigahost/internal/datasource_server_catalog"
)

var (
	_ datasource.DataSource              = &serverCatalogDataSource{}
	_ datasource.DataSourceWithConfigure = &serverCatalogDataSource{}
)

func NewServerCatalogDataSource() datasource.DataSource {
	return &serverCatalogDataSource{}
}

type serverCatalogDataSource struct {
	client *client.Client
}

func (d *serverCatalogDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server_catalog"
}

func (d *serverCatalogDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	s := datasource_server_catalog.ServerCatalogDataSourceSchema(ctx)
	s.MarkdownDescription = "Lists the hourly cloud server catalog: tiers, products, and regions."
	resp.Schema = s
}

func (d *serverCatalogDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *serverCatalogDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	catalog, err := d.client.GetDeployCatalog(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Gigahost Server Catalog", err.Error())
		return
	}

	regionElems := make([]datasource_server_catalog.RegionsValue, 0, len(catalog.Regions))
	for _, r := range catalog.Regions {
		regionElems = append(regionElems, datasource_server_catalog.NewRegionsValueMust(
			datasource_server_catalog.RegionsValue{}.AttributeTypes(ctx),
			map[string]attr.Value{
				"region_id":         types.StringValue(r.RegionID),
				"region_name":       types.StringValue(r.RegionName),
				"region_name_short": types.StringValue(r.RegionNameShort),
				"region_country":    types.StringValue(r.RegionCountry),
				"region_active":     types.BoolValue(bool(r.RegionActive)),
			},
		))
	}
	regions, diags := types.ListValueFrom(ctx, datasource_server_catalog.RegionsValue{}.Type(ctx), regionElems)
	resp.Diagnostics.Append(diags...)

	tierElems := make([]datasource_server_catalog.TiersValue, 0, len(catalog.Tiers))
	for _, t := range catalog.Tiers {
		productElems := make([]datasource_server_catalog.ProductsValue, 0, len(t.Products))
		for _, p := range t.Products {
			regionIDs, d := types.ListValueFrom(ctx, types.Int64Type, p.RegionIDs)
			resp.Diagnostics.Append(d...)
			productElems = append(productElems, datasource_server_catalog.NewProductsValueMust(
				datasource_server_catalog.ProductsValue{}.AttributeTypes(ctx),
				map[string]attr.Value{
					"product_id":   types.Int64Value(p.ProductID),
					"product_hash": types.StringValue(p.ProductHash),
					"product_name": types.StringValue(p.ProductName),
					"vm_cores":     types.StringValue(p.VMCores),
					"vm_memory":    types.StringValue(p.VMMemory),
					"vm_storage":   types.StringValue(p.VMStorage),
					"vm_bw":        types.StringValue(p.VMBw),
					"vm_bw_type":   types.StringValue(p.VMBwType),
					"price_id":     types.Int64Value(p.PriceID),
					"rate_hourly":  types.Float64Value(p.RateHourly),
					"rate_monthly": types.Int64Value(p.RateMonthly),
					"region_ids":   regionIDs,
				},
			))
		}
		products, d := types.ListValueFrom(ctx, datasource_server_catalog.ProductsValue{}.Type(ctx), productElems)
		resp.Diagnostics.Append(d...)
		tierElems = append(tierElems, datasource_server_catalog.NewTiersValueMust(
			datasource_server_catalog.TiersValue{}.AttributeTypes(ctx),
			map[string]attr.Value{
				"group_id":   types.Int64Value(t.GroupID),
				"group_name": types.StringValue(t.GroupName),
				"products":   products,
			},
		))
	}
	tiers, diags := types.ListValueFrom(ctx, datasource_server_catalog.TiersValue{}.Type(ctx), tierElems)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := datasource_server_catalog.ServerCatalogModel{
		Currency: types.StringValue(catalog.Currency),
		Regions:  regions,
		Tiers:    tiers,
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
