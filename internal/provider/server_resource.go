package provider

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/pigeon-as/terraform-provider-gigahost/internal/client"
)

const (
	serverDeployTimeout      = 30 * time.Minute
	serverDeployPollInterval = 5 * time.Second
)

var (
	_ resource.Resource                     = &serverResource{}
	_ resource.ResourceWithConfigure        = &serverResource{}
	_ resource.ResourceWithConfigValidators = &serverResource{}
	_ resource.ResourceWithModifyPlan       = &serverResource{}
	_ resource.ResourceWithImportState      = &serverResource{}
	_ resource.ResourceWithValidateConfig   = &serverResource{}
)

func NewServerResource() resource.Resource {
	return &serverResource{}
}

type serverResource struct {
	client *client.Client
}

type serverResourceModel struct {
	Name         types.String   `tfsdk:"name"`
	ProductName  types.String   `tfsdk:"product_name"`
	Region       types.String   `tfsdk:"region"`
	OsDistro     types.String   `tfsdk:"os_distro"`
	OsVersion    types.String   `tfsdk:"os_version"`
	Rescue       types.Bool     `tfsdk:"rescue"`
	Hostname     types.String   `tfsdk:"hostname"`
	SshKeys      types.Set      `tfsdk:"ssh_keys"`
	Backups      types.Bool     `tfsdk:"backups"`
	ProductId    types.Int64    `tfsdk:"product_id"`
	PriceId      types.Int64    `tfsdk:"price_id"`
	RegionId     types.Int64    `tfsdk:"region_id"`
	OsId         types.Int64    `tfsdk:"os_id"`
	ServerId     types.String   `tfsdk:"server_id"`
	OrderId      types.Int64    `tfsdk:"order_id"`
	Ipv4         types.String   `tfsdk:"ipv4"`
	Ipv6         types.String   `tfsdk:"ipv6"`
	RootPassword types.String   `tfsdk:"root_password"`
	OrderNumber  types.Int64    `tfsdk:"order_number"`
	RateHourly   types.Float64  `tfsdk:"rate_hourly"`
	MonthlyCap   types.Int64    `tfsdk:"monthly_cap"`
	Currency     types.String   `tfsdk:"currency"`
	Cores        types.Int64    `tfsdk:"cores"`
	Ram          types.Int64    `tfsdk:"ram"`
	Location     types.String   `tfsdk:"location"`
	Type         types.String   `tfsdk:"type"`
	VpsType      types.String   `tfsdk:"vps_type"`
	Running      types.Bool     `tfsdk:"running"`
	Installing   types.Bool     `tfsdk:"installing"`
	Suspended    types.Bool     `tfsdk:"suspended"`
	Os           types.Object   `tfsdk:"os"`
	Ips          types.List     `tfsdk:"ips"`
	Timeouts     timeouts.Value `tfsdk:"timeouts"`
}

func (r *serverResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server"
}

func (r *serverResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.RequiredTogether(
			path.MatchRoot("os_distro"),
			path.MatchRoot("os_version"),
		),
	}
}

func (r *serverResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data serverResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Unknown inputs (e.g. terraform validate, or values from other resources)
	// can't be evaluated; the rule is checked once they are known.
	if data.OsDistro.IsUnknown() || data.Rescue.IsUnknown() {
		return
	}
	hasOS := !data.OsDistro.IsNull()
	hasRescue := !data.Rescue.IsNull() && data.Rescue.ValueBool()
	if hasOS == hasRescue {
		resp.Diagnostics.AddAttributeError(
			path.Root("rescue"),
			"Invalid OS or Rescue Configuration",
			"Provide os_distro and os_version to install an OS, or set rescue = true (exactly one).",
		)
	}
}

func (r *serverResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Deploys and manages an hourly Gigahost cloud server.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Optional:            true,
				Description:         "Descriptive name for the server.",
				MarkdownDescription: "Descriptive name for the server.",
			},
			"product_name": schema.StringAttribute{
				Required:            true,
				Description:         "Product name from the catalog, e.g. \"KVM Value VPS 4GB\".",
				MarkdownDescription: "Product name from the catalog, e.g. \"KVM Value VPS 4GB\".",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"region": schema.StringAttribute{
				Required:            true,
				Description:         "Region name to deploy in, e.g. \"Sandefjord\".",
				MarkdownDescription: "Region name to deploy in, e.g. \"Sandefjord\".",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"os_distro": schema.StringAttribute{
				Optional:            true,
				Description:         "OS distribution to install, e.g. \"Ubuntu\". Provide os_distro + os_version, or rescue.",
				MarkdownDescription: "OS distribution to install, e.g. \"Ubuntu\". Provide os_distro + os_version, or rescue.",
			},
			"os_version": schema.StringAttribute{
				Optional:            true,
				Description:         "OS version to install, e.g. \"24.04\" (matches the OS name or release codename).",
				MarkdownDescription: "OS version to install, e.g. \"24.04\" (matches the OS name or release codename).",
			},
			"rescue": schema.BoolAttribute{
				Optional:            true,
				Description:         "Boot the server into rescue mode instead of installing an OS.",
				MarkdownDescription: "Boot the server into rescue mode instead of installing an OS.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
			},
			"hostname": schema.StringAttribute{
				Optional:            true,
				Description:         "Requested hostname.",
				MarkdownDescription: "Requested hostname.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"ssh_keys": schema.SetAttribute{
				ElementType:         types.Int64Type,
				Optional:            true,
				Description:         "Ids of SSH keys to authorize on the server.",
				MarkdownDescription: "Ids of SSH keys to authorize on the server.",
				PlanModifiers:       []planmodifier.Set{setplanmodifier.RequiresReplace()},
			},
			"backups": schema.BoolAttribute{
				Optional:            true,
				Description:         "Whether to enable daily backups (adds 25% to the price).",
				MarkdownDescription: "Whether to enable daily backups (adds 25% to the price).",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
			},
			"product_id": schema.Int64Attribute{
				Computed:            true,
				Description:         "Resolved product id (from product_name).",
				MarkdownDescription: "Resolved product id (from product_name).",
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"price_id": schema.Int64Attribute{
				Computed:            true,
				Description:         "Resolved price id (from product_name).",
				MarkdownDescription: "Resolved price id (from product_name).",
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"region_id": schema.Int64Attribute{
				Computed:            true,
				Description:         "Resolved region id (from region).",
				MarkdownDescription: "Resolved region id (from region).",
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"os_id": schema.Int64Attribute{
				Computed:            true,
				Description:         "Resolved OS image id (from os_distro + os_version).",
				MarkdownDescription: "Resolved OS image id (from os_distro + os_version).",
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"server_id": schema.StringAttribute{
				Computed:            true,
				Description:         "Server id (srv_id).",
				MarkdownDescription: "Server id (srv_id).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"order_id": schema.Int64Attribute{
				Computed:            true,
				Description:         "Id of the deployment order.",
				MarkdownDescription: "Id of the deployment order.",
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"ipv4": schema.StringAttribute{
				Computed:            true,
				Description:         "Primary IPv4 address.",
				MarkdownDescription: "Primary IPv4 address.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"ipv6": schema.StringAttribute{
				Computed:            true,
				Description:         "Primary IPv6 address.",
				MarkdownDescription: "Primary IPv6 address.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"root_password": schema.StringAttribute{
				Computed:            true,
				Sensitive:           true,
				Description:         "Initial root password (only set when the server is deployed without an SSH key). Stored in Terraform state in plaintext.",
				MarkdownDescription: "Initial root password (only set when the server is deployed without an SSH key). Stored in Terraform state in plaintext.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"order_number": schema.Int64Attribute{
				Computed:            true,
				Description:         "Human-facing order number for the deployment.",
				MarkdownDescription: "Human-facing order number for the deployment.",
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"rate_hourly": schema.Float64Attribute{
				Computed:            true,
				Description:         "Hourly rate for the server.",
				MarkdownDescription: "Hourly rate for the server.",
				PlanModifiers:       []planmodifier.Float64{float64planmodifier.UseStateForUnknown()},
			},
			"monthly_cap": schema.Int64Attribute{
				Computed:            true,
				Description:         "Monthly price cap (the most charged per month).",
				MarkdownDescription: "Monthly price cap (the most charged per month).",
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"currency": schema.StringAttribute{
				Computed:            true,
				Description:         "Currency of the pricing.",
				MarkdownDescription: "Currency of the pricing.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"cores": schema.Int64Attribute{
				Computed:            true,
				Description:         "Number of CPU cores.",
				MarkdownDescription: "Number of CPU cores.",
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"ram": schema.Int64Attribute{
				Computed:            true,
				Description:         "Memory, in GB.",
				MarkdownDescription: "Memory, in GB.",
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"location": schema.StringAttribute{
				Computed:            true,
				Description:         "Datacenter location code.",
				MarkdownDescription: "Datacenter location code.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"type": schema.StringAttribute{
				Computed:            true,
				Description:         "Server type (vps or dedicated).",
				MarkdownDescription: "Server type (vps or dedicated).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"vps_type": schema.StringAttribute{
				Computed:            true,
				Description:         "Virtualization type (e.g. kvm).",
				MarkdownDescription: "Virtualization type (e.g. kvm).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"running": schema.BoolAttribute{
				Computed:            true,
				Description:         "Whether the server is running.",
				MarkdownDescription: "Whether the server is running.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"installing": schema.BoolAttribute{
				Computed:            true,
				Description:         "Whether the server is installing.",
				MarkdownDescription: "Whether the server is installing.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"suspended": schema.BoolAttribute{
				Computed:            true,
				Description:         "Whether the server is suspended.",
				MarkdownDescription: "Whether the server is suspended.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"os": schema.SingleNestedAttribute{
				Computed:            true,
				Description:         "Installed operating system.",
				MarkdownDescription: "Installed operating system.",
				PlanModifiers:       []planmodifier.Object{objectplanmodifier.UseStateForUnknown()},
				Attributes: map[string]schema.Attribute{
					"os_id":      schema.Int64Attribute{Computed: true, Description: "OS image (version) id.", MarkdownDescription: "OS image (version) id."},
					"os_name":    schema.StringAttribute{Computed: true, Description: "OS image name.", MarkdownDescription: "OS image name."},
					"os_release": schema.StringAttribute{Computed: true, Description: "OS release/version.", MarkdownDescription: "OS release/version."},
				},
			},
			"ips": schema.ListNestedAttribute{
				Computed:            true,
				Description:         "IP addresses assigned to the server.",
				MarkdownDescription: "IP addresses assigned to the server.",
				PlanModifiers:       []planmodifier.List{listplanmodifier.UseStateForUnknown()},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"ip_id":      schema.Int64Attribute{Computed: true, Description: "IP address id.", MarkdownDescription: "IP address id."},
						"ip_address": schema.StringAttribute{Computed: true, Description: "The IP address.", MarkdownDescription: "The IP address."},
						"ip_v4v6":    schema.StringAttribute{Computed: true, Description: "Address family (ipv4 or ipv6).", MarkdownDescription: "Address family (ipv4 or ipv6)."},
						"ip_reverse": schema.StringAttribute{Computed: true, Description: "Reverse DNS (PTR) for the address.", MarkdownDescription: "Reverse DNS (PTR) for the address."},
						"ip_type":    schema.StringAttribute{Computed: true, Description: "Address type (primary or extra).", MarkdownDescription: "Address type (primary or extra)."},
						"ip_netmask": schema.StringAttribute{Computed: true, Description: "Netmask.", MarkdownDescription: "Netmask."},
						"ip_gateway": schema.StringAttribute{Computed: true, Description: "Gateway.", MarkdownDescription: "Gateway."},
					},
				},
			},
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{Create: true}),
		},
	}
}

func (r *serverResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = c
}

func (r *serverResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() || r.client == nil {
		return
	}

	var plan serverResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	needProduct := plan.ProductId.IsUnknown() && !plan.ProductName.IsUnknown()
	needRegion := plan.RegionId.IsUnknown() && !plan.Region.IsUnknown()
	needOS := !plan.OsDistro.IsNull() && !plan.OsDistro.IsUnknown() && !plan.OsVersion.IsNull() && !plan.OsVersion.IsUnknown()
	if needOS && !req.State.Raw.IsNull() {
		var stateDistro, stateVersion types.String
		resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("os_distro"), &stateDistro)...)
		resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("os_version"), &stateVersion)...)
		if plan.OsDistro.Equal(stateDistro) && plan.OsVersion.Equal(stateVersion) {
			needOS = false
		}
	}

	if needProduct || needRegion {
		catalog, err := r.client.GetDeployCatalog(ctx)
		if err != nil {
			resp.Diagnostics.AddError("Unable to Read Gigahost Server Catalog", err.Error())
			return
		}
		if needProduct {
			productID, priceID, err := resolveProduct(catalog, plan.ProductName.ValueString())
			if err != nil {
				resp.Diagnostics.AddAttributeError(path.Root("product_name"), "Invalid Server Product", err.Error())
			} else {
				resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("product_id"), productID)...)
				resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("price_id"), priceID)...)
			}
		}
		if needRegion {
			regionID, err := resolveRegion(catalog, plan.Region.ValueString())
			if err != nil {
				resp.Diagnostics.AddAttributeError(path.Root("region"), "Invalid Region", err.Error())
			} else {
				resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("region_id"), regionID)...)
			}
		}

		var pid, rid types.Int64
		resp.Diagnostics.Append(resp.Plan.GetAttribute(ctx, path.Root("product_id"), &pid)...)
		resp.Diagnostics.Append(resp.Plan.GetAttribute(ctx, path.Root("region_id"), &rid)...)
		if !pid.IsUnknown() && !pid.IsNull() && !rid.IsUnknown() && !rid.IsNull() &&
			!productOffersRegion(catalog, pid.ValueInt64(), rid.ValueInt64()) {
			resp.Diagnostics.AddAttributeError(path.Root("region"), "Incompatible Product and Region",
				fmt.Sprintf("Product %q is not available in region %q.", plan.ProductName.ValueString(), plan.Region.ValueString()))
		}
	}

	if needOS {
		osCatalog, err := r.client.GetOSCatalog(ctx)
		if err != nil {
			resp.Diagnostics.AddError("Unable to Read Gigahost OS Catalog", err.Error())
			return
		}
		osID, err := resolveOS(osCatalog, plan.OsDistro.ValueString(), plan.OsVersion.ValueString())
		if err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("os_distro"), "Invalid OS", err.Error())
		} else {
			resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("os_id"), osID)...)
			if !req.State.Raw.IsNull() {
				var stateOSID types.Int64
				resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("os_id"), &stateOSID)...)
				if !stateOSID.IsNull() && stateOSID.ValueInt64() != osID {
					resp.RequiresReplace = append(resp.RequiresReplace, path.Root("os_id"))
				}
			}
		}
	}
}

func (r *serverResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan serverResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout, diags := plan.Timeouts.Create(ctx, serverDeployTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	productID := plan.ProductId.ValueInt64()
	priceID := plan.PriceId.ValueInt64()
	regionID := plan.RegionId.ValueInt64()
	if plan.ProductId.IsUnknown() || plan.RegionId.IsUnknown() {
		catalog, err := r.client.GetDeployCatalog(ctx)
		if err != nil {
			resp.Diagnostics.AddError("Unable to Read Gigahost Server Catalog", err.Error())
			return
		}
		if plan.ProductId.IsUnknown() {
			productID, priceID, err = resolveProduct(catalog, plan.ProductName.ValueString())
			if err != nil {
				resp.Diagnostics.AddError("Invalid Server Product", err.Error())
				return
			}
		}
		if plan.RegionId.IsUnknown() {
			regionID, err = resolveRegion(catalog, plan.Region.ValueString())
			if err != nil {
				resp.Diagnostics.AddError("Invalid Region", err.Error())
				return
			}
		}
	}

	osID := types.Int64Null()
	var osIDPtr *int64
	if !plan.OsDistro.IsNull() && plan.OsDistro.ValueString() != "" {
		if !plan.OsId.IsUnknown() && !plan.OsId.IsNull() {
			osID = plan.OsId
		} else {
			osCatalog, err := r.client.GetOSCatalog(ctx)
			if err != nil {
				resp.Diagnostics.AddError("Unable to Read Gigahost OS Catalog", err.Error())
				return
			}
			id, err := resolveOS(osCatalog, plan.OsDistro.ValueString(), plan.OsVersion.ValueString())
			if err != nil {
				resp.Diagnostics.AddError("Invalid OS", err.Error())
				return
			}
			osID = types.Int64Value(id)
		}
		v := osID.ValueInt64()
		osIDPtr = &v
	}

	var sshKeys []int64
	if !plan.SshKeys.IsNull() && !plan.SshKeys.IsUnknown() {
		resp.Diagnostics.Append(plan.SshKeys.ElementsAs(ctx, &sshKeys, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	in := client.DeployInput{
		ProductID: productID,
		PriceID:   priceID,
		RegionID:  regionID,
		OSID:      osIDPtr,
		Rescue:    plan.Rescue.ValueBool(),
		Hostname:  plan.Hostname.ValueString(),
		SSHKeys:   sshKeys,
		Backups:   plan.Backups.ValueBool(),
	}

	result, err := r.client.Deploy(ctx, in)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Deploy Gigahost Server", err.Error())
		return
	}
	if len(result.OrderIDs) == 0 {
		resp.Diagnostics.AddError("Unable to Deploy Gigahost Server", "the deploy API did not return an order id.")
		return
	}
	orderID := result.OrderIDs[0]

	server, err := r.waitForServer(ctx, orderID)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Deploy Gigahost Server", err.Error())
		return
	}
	serverID := strconv.FormatInt(int64(server.SrvID), 10)

	state := plan
	state.ProductId = types.Int64Value(productID)
	state.PriceId = types.Int64Value(priceID)
	state.RegionId = types.Int64Value(regionID)
	state.OsId = osID
	state.ServerId = types.StringValue(serverID)
	state.OrderId = types.Int64Value(orderID)
	state.RootPassword = types.StringNull()
	if server.Password != "" {
		state.RootPassword = types.StringValue(server.Password)
	}
	state.OrderNumber = types.Int64Null()
	if len(result.OrderNumbers) > 0 {
		state.OrderNumber = types.Int64Value(result.OrderNumbers[0])
	}
	state.RateHourly = types.Float64Value(result.RateHourly)
	state.MonthlyCap = types.Int64Value(result.MonthlyCap)
	state.Currency = types.StringValue(result.Currency)

	var full *client.Server
	if servers, err := r.client.ListServers(ctx); err == nil {
		for i := range servers {
			if equalID(servers[i].SrvID, serverID) {
				full = &servers[i]
				break
			}
		}
	}
	applyServerState(&state, full)
	state.Ipv4 = types.StringValue(server.IP)
	state.Ipv6 = types.StringValue(server.IPv6)

	if !plan.Name.IsNull() && plan.Name.ValueString() != "" {
		if err := r.client.UpdateServerName(ctx, serverID, plan.Name.ValueString()); err != nil {
			state.Name = types.StringNull()
			resp.Diagnostics.AddError("Unable to Set Gigahost Server Name", err.Error())
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *serverResource) waitForServer(ctx context.Context, orderID int64) (*client.DeployStatusServer, error) {
	ticker := time.NewTicker(serverDeployPollInterval)
	defer ticker.Stop()

	const maxPollErrors = 4
	pollErrors := 0

	for {
		status, err := r.client.GetDeployStatus(ctx, []int64{orderID})
		if err != nil {
			pollErrors++
			if pollErrors > maxPollErrors {
				return nil, fmt.Errorf("polling deploy status for order %d failed %d times in a row: %w", orderID, pollErrors, err)
			}
		} else {
			pollErrors = 0
			for i := range status.Servers {
				if int64(status.Servers[i].OrderID) != orderID {
					continue
				}
				switch status.Servers[i].Status {
				case "ready", "rescue", "iso":
					return &status.Servers[i], nil
				case "error", "failed", "cancelled", "suspended":
					return nil, fmt.Errorf("server (order %d) failed to deploy: status %q", orderID, status.Servers[i].Status)
				default:
					tflog.Debug(ctx, "waiting for Gigahost server to deploy", map[string]any{
						"order_id": orderID,
						"status":   status.Servers[i].Status,
					})
				}
			}
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timed out waiting for server (order %d) to be ready: %w", orderID, ctx.Err())
		case <-ticker.C:
		}
	}
}

var serverOSAttrTypes = map[string]attr.Type{
	"os_id":      types.Int64Type,
	"os_name":    types.StringType,
	"os_release": types.StringType,
}

var serverIPAttrTypes = map[string]attr.Type{
	"ip_id":      types.Int64Type,
	"ip_address": types.StringType,
	"ip_v4v6":    types.StringType,
	"ip_reverse": types.StringType,
	"ip_type":    types.StringType,
	"ip_netmask": types.StringType,
	"ip_gateway": types.StringType,
}

func applyServerState(state *serverResourceModel, s *client.Server) {
	if s == nil {
		state.Cores = types.Int64Null()
		state.Ram = types.Int64Null()
		state.Location = types.StringNull()
		state.Type = types.StringNull()
		state.VpsType = types.StringNull()
		state.Running = types.BoolNull()
		state.Installing = types.BoolNull()
		state.Suspended = types.BoolNull()
		state.Os = types.ObjectNull(serverOSAttrTypes)
		state.Ips = types.ListNull(types.ObjectType{AttrTypes: serverIPAttrTypes})
		return
	}

	state.Ipv4 = types.StringValue(s.SrvPrimaryIP)
	state.Ipv6 = types.StringNull()
	for _, ip := range s.IPs {
		if strings.EqualFold(ip.IPv4v6, "ipv6") {
			state.Ipv6 = types.StringValue(ip.IPAddress)
			break
		}
	}
	state.Cores = types.Int64Value(int64(s.SrvCores))
	state.Ram = types.Int64Value(int64(s.SrvRAM))
	state.Location = types.StringValue(s.SrvLocation)
	state.Type = types.StringValue(s.SrvType)
	state.VpsType = types.StringValue(s.SrvVpsType)
	state.Running = types.BoolValue(bool(s.SrvStatus))
	state.Installing = types.BoolValue(bool(s.SrvStatusInstall))
	state.Suspended = types.BoolValue(bool(s.SrvSuspended))
	state.Os = types.ObjectValueMust(serverOSAttrTypes, map[string]attr.Value{
		"os_id":      types.Int64Value(int64(s.OS.OsID)),
		"os_name":    types.StringValue(s.OS.OsName),
		"os_release": types.StringValue(s.OS.OsRelease),
	})
	ipElems := make([]attr.Value, 0, len(s.IPs))
	for _, ip := range s.IPs {
		ipElems = append(ipElems, types.ObjectValueMust(serverIPAttrTypes, map[string]attr.Value{
			"ip_id":      types.Int64Value(int64(ip.IPID)),
			"ip_address": types.StringValue(ip.IPAddress),
			"ip_v4v6":    types.StringValue(ip.IPv4v6),
			"ip_reverse": types.StringValue(ip.IPReverse),
			"ip_type":    types.StringValue(ip.IPType),
			"ip_netmask": types.StringValue(ip.IPNetmask),
			"ip_gateway": types.StringValue(ip.IPGateway),
		}))
	}
	state.Ips = types.ListValueMust(types.ObjectType{AttrTypes: serverIPAttrTypes}, ipElems)
}

func (r *serverResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state serverResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	servers, err := r.client.ListServers(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Gigahost Server", err.Error())
		return
	}

	var found *client.Server
	for i := range servers {
		if equalID(servers[i].SrvID, state.ServerId.ValueString()) {
			found = &servers[i]
			break
		}
	}
	if found == nil || strings.EqualFold(found.Order.OrderStatus, "cancelled") {
		resp.State.RemoveResource(ctx)
		return
	}

	if !state.Name.IsNull() {
		state.Name = types.StringValue(found.SrvName)
	}
	applyServerState(&state, found)

	if state.ProductName.IsNull() && found.Order.ProductName != "" {
		state.ProductName = types.StringValue(found.Order.ProductName)
	}
	if state.ProductId.IsNull() {
		if id, err := strconv.ParseInt(found.Order.ProductID, 10, 64); err == nil && id != 0 {
			state.ProductId = types.Int64Value(id)
		}
	}
	if state.Region.IsNull() && found.Datacenter.RegionName != "" {
		state.Region = types.StringValue(found.Datacenter.RegionName)
	}
	if state.RegionId.IsNull() {
		if id, err := strconv.ParseInt(found.Datacenter.RegionID, 10, 64); err == nil && id != 0 {
			state.RegionId = types.Int64Value(id)
		}
	}
	if state.OsId.IsNull() {
		if id := int64(found.OS.OsID); id != 0 {
			state.OsId = types.Int64Value(id)
		}
	}
	if state.Rescue.IsNull() && bool(found.SrvStatusRescue) {
		state.Rescue = types.BoolValue(true)
	}
	if state.Backups.IsNull() && bool(found.SrvFeatureBackups) {
		state.Backups = types.BoolValue(true)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *serverResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state serverResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !plan.Name.Equal(state.Name) {
		if err := r.client.UpdateServerName(ctx, state.ServerId.ValueString(), plan.Name.ValueString()); err != nil {
			resp.Diagnostics.AddError("Unable to Update Gigahost Server Name", err.Error())
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *serverResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state serverResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.CancelServer(ctx, state.ServerId.ValueString()); err != nil && !errors.Is(err, client.ErrNotFound) {
		resp.Diagnostics.AddError("Unable to Cancel Gigahost Server", err.Error())
	}
}

func (r *serverResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("server_id"), req, resp)
}
