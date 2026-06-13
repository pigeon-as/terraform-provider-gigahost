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

const serverDeployTimeout = 30 * time.Minute

var serverDeployPollInterval = 5 * time.Second

var (
	_ resource.Resource                     = &serverResource{}
	_ resource.ResourceWithConfigure        = &serverResource{}
	_ resource.ResourceWithConfigValidators = &serverResource{}
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
	SrvName          types.String   `tfsdk:"srv_name"`
	ProductName      types.String   `tfsdk:"product_name"`
	RegionName       types.String   `tfsdk:"region_name"`
	OSName           types.String   `tfsdk:"os_name"`
	OSDist           types.String   `tfsdk:"os_dist"`
	Rescue           types.Bool     `tfsdk:"rescue"`
	Hostname         types.String   `tfsdk:"hostname"`
	SshKeys          types.Set      `tfsdk:"ssh_keys"`
	Backups          types.Bool     `tfsdk:"backups"`
	ProductId        types.Int64    `tfsdk:"product_id"`
	PriceId          types.Int64    `tfsdk:"price_id"`
	RegionId         types.Int64    `tfsdk:"region_id"`
	OsId             types.Int64    `tfsdk:"os_id"`
	SrvID            types.String   `tfsdk:"srv_id"`
	OrderId          types.Int64    `tfsdk:"order_id"`
	SrvPrimaryIP     types.String   `tfsdk:"srv_primary_ip"`
	Password         types.String   `tfsdk:"password"`
	OrderNumber      types.Int64    `tfsdk:"order_number"`
	RateHourly       types.Float64  `tfsdk:"rate_hourly"`
	MonthlyCap       types.Int64    `tfsdk:"monthly_cap"`
	Currency         types.String   `tfsdk:"currency"`
	SrvCores         types.Int64    `tfsdk:"srv_cores"`
	SrvRAM           types.Int64    `tfsdk:"srv_ram"`
	SrvLocation      types.String   `tfsdk:"srv_location"`
	SrvType          types.String   `tfsdk:"srv_type"`
	SrvVpsType       types.String   `tfsdk:"srv_vps_type"`
	SrvStatus        types.Bool     `tfsdk:"srv_status"`
	SrvStatusInstall types.Bool     `tfsdk:"srv_status_install"`
	SrvSuspended     types.Bool     `tfsdk:"srv_suspended"`
	Os               types.Object   `tfsdk:"os"`
	Ips              types.List     `tfsdk:"ips"`
	Timeouts         timeouts.Value `tfsdk:"timeouts"`
}

func (r *serverResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server"
}

func (r *serverResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.Conflicting(
			path.MatchRoot("os_name"),
			path.MatchRoot("os_dist"),
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
	if data.OSName.IsUnknown() || data.OSDist.IsUnknown() || data.Rescue.IsUnknown() {
		return
	}
	hasOS := !data.OSName.IsNull() || !data.OSDist.IsNull()
	hasRescue := !data.Rescue.IsNull() && data.Rescue.ValueBool()
	if hasOS == hasRescue {
		resp.Diagnostics.AddAttributeError(
			path.Root("rescue"),
			"Invalid OS or Rescue Configuration",
			"Provide os_name or os_dist to install an OS, or set rescue = true (exactly one).",
		)
	}
}

func (r *serverResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Deploys and manages an hourly-billed Gigahost cloud server — a KVM virtual machine or a dedicated (bare metal) server, depending on the chosen product.",
		Attributes: map[string]schema.Attribute{
			"srv_name": schema.StringAttribute{
				Optional:            true,
				Description:         "Descriptive name for the server. When unset, the server keeps its initial name.",
				MarkdownDescription: "Descriptive name for the server. When unset, the server keeps its initial name.",
			},
			"product_name": schema.StringAttribute{
				Required:            true,
				Description:         "Product name from the catalog, e.g. \"KVM Value VPS 4GB\".",
				MarkdownDescription: "Product name from the catalog, e.g. \"KVM Value VPS 4GB\".",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"region_name": schema.StringAttribute{
				Required:            true,
				Description:         "Region name to deploy in, e.g. \"Sandefjord\".",
				MarkdownDescription: "Region name to deploy in, e.g. \"Sandefjord\".",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"os_name": schema.StringAttribute{
				Optional:            true,
				Description:         "OS image to install, by catalog name, e.g. \"Ubuntu 24.04 LTS\". Provide os_name or os_dist (to install an OS), or rescue.",
				MarkdownDescription: "OS image to install, by catalog name (`os_name`), e.g. \"Ubuntu 24.04 LTS\". Provide `os_name` or `os_dist` (to install an OS), or `rescue`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"os_dist": schema.StringAttribute{
				Optional:            true,
				Description:         "OS image to install, by release codename, e.g. \"noble\". Provide os_name or os_dist (to install an OS), or rescue.",
				MarkdownDescription: "OS image to install, by release codename (`os_dist`), e.g. \"noble\". Provide `os_name` or `os_dist` (to install an OS), or `rescue`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"rescue": schema.BoolAttribute{
				Optional:            true,
				Description:         "Boot the server into rescue mode instead of installing an OS.",
				MarkdownDescription: "Boot the server into rescue mode instead of installing an OS.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
			},
			"hostname": schema.StringAttribute{
				Optional:            true,
				Description:         "Requested hostname. Stored by the API as the server's initial name (srv_name); unset after import.",
				MarkdownDescription: "Requested hostname. Stored by the API as the server's initial name (`srv_name`); unset after import.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"ssh_keys": schema.SetAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				Description:         "Ids of SSH keys to authorize on the server. Changing this replaces the server. The API does not return deployed keys, so this is unset after import; on imported servers, omit ssh_keys or use lifecycle ignore_changes to avoid replacement.",
				MarkdownDescription: "Ids of SSH keys to authorize on the server. Changing this replaces the server. The API does not return deployed keys, so this is unset after `terraform import`; on imported servers, omit `ssh_keys` or use `lifecycle { ignore_changes = [ssh_keys] }` to avoid replacement.",
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
				Description:         "Resolved OS image id (from os_name or os_dist).",
				MarkdownDescription: "Resolved OS image id (from os_name or os_dist).",
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"srv_id": schema.StringAttribute{
				Computed:            true,
				Description:         "Server id.",
				MarkdownDescription: "Server id.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"order_id": schema.Int64Attribute{
				Computed:            true,
				Description:         "Id of the deployment order.",
				MarkdownDescription: "Id of the deployment order.",
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"srv_primary_ip": schema.StringAttribute{
				Computed:            true,
				Description:         "Primary IP address. Additional addresses, including IPv6, are in ips.",
				MarkdownDescription: "Primary IP address. Additional addresses, including IPv6, are in `ips`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"password": schema.StringAttribute{
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
			"srv_cores": schema.Int64Attribute{
				Computed:            true,
				Description:         "Number of CPU cores.",
				MarkdownDescription: "Number of CPU cores.",
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"srv_ram": schema.Int64Attribute{
				Computed:            true,
				Description:         "Memory, in GB.",
				MarkdownDescription: "Memory, in GB.",
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"srv_location": schema.StringAttribute{
				Computed:            true,
				Description:         "Datacenter location code.",
				MarkdownDescription: "Datacenter location code.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"srv_type": schema.StringAttribute{
				Computed:            true,
				Description:         "Server type (vps or dedicated).",
				MarkdownDescription: "Server type (vps or dedicated).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"srv_vps_type": schema.StringAttribute{
				Computed:            true,
				Description:         "Virtualization type (e.g. kvm).",
				MarkdownDescription: "Virtualization type (e.g. kvm).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"srv_status": schema.BoolAttribute{
				Computed:            true,
				Description:         "Whether the server is running.",
				MarkdownDescription: "Whether the server is running.",
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

	catalog, err := r.client.GetDeployCatalog(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Gigahost Server Catalog", err.Error())
		return
	}
	productID, priceID, err := resolveProduct(catalog, plan.ProductName.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("product_name"), "Invalid Server Product", err.Error())
		return
	}
	regionID, err := resolveRegion(catalog, plan.RegionName.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("region_name"), "Invalid Region", err.Error())
		return
	}
	if !productOffersRegion(catalog, productID, regionID) {
		resp.Diagnostics.AddAttributeError(path.Root("region_name"), "Incompatible Product and Region",
			fmt.Sprintf("Product %q is not available in region %q.", plan.ProductName.ValueString(), plan.RegionName.ValueString()))
		return
	}

	osID := types.Int64Null()
	var osIDPtr *int64
	osImage, osPath := plan.OSName.ValueString(), path.Root("os_name")
	if plan.OSName.IsNull() {
		osImage, osPath = plan.OSDist.ValueString(), path.Root("os_dist")
	}
	if osImage != "" {
		osCatalog, err := r.client.GetOSCatalog(ctx)
		if err != nil {
			resp.Diagnostics.AddError("Unable to Read Gigahost OS Catalog", err.Error())
			return
		}
		id, err := resolveOS(osCatalog, osImage)
		if err != nil {
			resp.Diagnostics.AddAttributeError(osPath, "Invalid OS", err.Error())
			return
		}
		osID = types.Int64Value(id)
		v := id
		osIDPtr = &v
	}

	var sshKeys []int64
	if !plan.SshKeys.IsNull() && !plan.SshKeys.IsUnknown() {
		var keyIDs []string
		resp.Diagnostics.Append(plan.SshKeys.ElementsAs(ctx, &keyIDs, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		for _, k := range keyIDs {
			id, err := strconv.ParseInt(k, 10, 64)
			if err != nil {
				resp.Diagnostics.AddAttributeError(path.Root("ssh_keys"), "Invalid SSH Key Id", fmt.Sprintf("%q is not a valid SSH key id: %s", k, err))
				return
			}
			sshKeys = append(sshKeys, id)
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

	// The order is placed (and billed) from here on, so every return must
	// persist state: a failed wait leaves a tainted resource, not an orphan.
	state := plan
	state.ProductId = types.Int64Value(productID)
	state.PriceId = types.Int64Value(priceID)
	state.RegionId = types.Int64Value(regionID)
	state.OsId = osID
	state.SrvID = types.StringNull()
	state.OrderId = types.Int64Value(orderID)
	state.Password = types.StringNull()
	state.OrderNumber = types.Int64Null()
	if len(result.OrderNumbers) > 0 {
		state.OrderNumber = types.Int64Value(result.OrderNumbers[0])
	}
	state.RateHourly = types.Float64Value(result.RateHourly)
	state.MonthlyCap = types.Int64Value(result.MonthlyCap)
	state.Currency = types.StringValue(result.Currency)
	applyServerState(&state, nil)

	server, err := r.waitForServer(ctx, orderID)
	if err != nil {
		if server != nil && server.SrvID != "" {
			state.SrvID = types.StringValue(server.SrvID)
		}
		var hint string
		if state.SrvID.IsNull() {
			hint = fmt.Sprintf("No server id was observed for %s, so terraform destroy cannot cancel it; check the Gigahost control panel and cancel it manually if needed.", orderRef(&state))
		} else {
			hint = fmt.Sprintf("The server was saved to Terraform state and marked tainted; terraform destroy will cancel %s, or clear the resource if the server no longer exists.", orderRef(&state))
		}
		resp.Diagnostics.AddError("Unable to Deploy Gigahost Server", fmt.Sprintf("%s\n\n%s", err, hint))
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}
	serverID := server.SrvID
	state.SrvID = types.StringValue(serverID)
	if server.Password != "" {
		state.Password = types.StringValue(server.Password)
	}

	var full *client.Server
	detail, err := r.client.GetServer(ctx, serverID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Gigahost Server Details",
			fmt.Sprintf("The server deployed, but reading its details failed: %s\n\nThe server was saved to Terraform state and marked tainted; terraform untaint will keep it, and the next refresh fills in the missing details.", err),
		)
	} else {
		full = &detail.Server
	}
	applyServerState(&state, full)
	if state.SrvPrimaryIP.IsNull() && server.IP != "" {
		state.SrvPrimaryIP = types.StringValue(server.IP)
	}
	if resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}

	if !plan.SrvName.IsNull() && plan.SrvName.ValueString() != "" {
		if err := r.client.UpdateServerName(ctx, serverID, plan.SrvName.ValueString()); err != nil {
			state.SrvName = types.StringNull()
			resp.Diagnostics.AddError("Unable to Set Gigahost Server Name", err.Error())
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// deployedServer carries what a deploy revealed about the server, sourced
// from deploy status or, when that loses track of the order, the server list.
type deployedServer struct {
	SrvID    string
	IP       string
	IPv6     string
	Password string
}

func deployedFromStatus(s *client.DeployStatusServer) *deployedServer {
	out := &deployedServer{IP: s.IP, IPv6: s.IPv6, Password: s.Password}
	if id := int64(s.SrvID); id != 0 {
		out.SrvID = strconv.FormatInt(id, 10)
	}
	return out
}

// mergeDeployed carries forward fields an earlier poll revealed that a later
// one omits — chiefly the root password, which deploy status only reports
// while the server is installing and drops once it is ready. The primary IPv4
// (IP) is not carried forward: it is assigned up front and reported on every
// poll, so it never needs rescuing.
func mergeDeployed(prev, next *deployedServer) *deployedServer {
	if prev == nil {
		return next
	}
	if next.SrvID == "" {
		next.SrvID = prev.SrvID
	}
	if next.Password == "" {
		next.Password = prev.Password
	}
	if next.IPv6 == "" {
		next.IPv6 = prev.IPv6
	}
	return next
}

// waitForServer polls the deploy status until the order's server reaches a
// final status. The last server seen is returned even on failure, so callers
// can persist its id. An order that never reaches a final status — including
// one the deploy status API stops reporting — surfaces as a timeout.
func (r *serverResource) waitForServer(ctx context.Context, orderID int64) (*deployedServer, error) {
	ticker := time.NewTicker(serverDeployPollInterval)
	defer ticker.Stop()

	const maxPollErrors = 4
	pollErrors := 0
	var last *deployedServer

	for {
		status, err := r.client.GetDeployStatus(ctx, []int64{orderID})
		if err != nil {
			pollErrors++
			if pollErrors > maxPollErrors {
				return last, fmt.Errorf("polling deploy status for order %d failed %d times in a row: %w", orderID, pollErrors, err)
			}
		} else {
			pollErrors = 0
			for i := range status.Servers {
				if int64(status.Servers[i].OrderID) != orderID {
					continue
				}
				last = mergeDeployed(last, deployedFromStatus(&status.Servers[i]))
				switch status.Servers[i].Status {
				case "ready", "rescue", "iso":
					// Terminal, but unusable until a server id is known.
					if last.SrvID != "" {
						return last, nil
					}
				case "error", "failed", "cancelled", "suspended":
					return last, fmt.Errorf("server (order %d) failed to deploy: status %q", orderID, status.Servers[i].Status)
				default:
					tflog.Debug(ctx, "waiting for Gigahost server to deploy", map[string]any{
						"order_id": orderID,
						"status":   status.Servers[i].Status,
					})
				}
				break
			}
		}

		select {
		case <-ctx.Done():
			return last, fmt.Errorf("timed out waiting for server (order %d) to be ready: %w", orderID, ctx.Err())
		case <-ticker.C:
		}
	}
}

// orderRef names the deployment order for error messages, preferring the
// human-facing order number.
func orderRef(state *serverResourceModel) string {
	if !state.OrderNumber.IsNull() {
		return fmt.Sprintf("order number %d", state.OrderNumber.ValueInt64())
	}
	if !state.OrderId.IsNull() {
		return fmt.Sprintf("order id %d", state.OrderId.ValueInt64())
	}
	return "the order"
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

// applyServerState writes the API-derived fields of a server onto the model
// in place. When s is nil (the server is not yet known) it nulls those same
// computed fields, so the model is always left fully known.
func applyServerState(state *serverResourceModel, s *client.Server) {
	if s == nil {
		state.SrvPrimaryIP = types.StringNull()
		state.SrvCores = types.Int64Null()
		state.SrvRAM = types.Int64Null()
		state.SrvLocation = types.StringNull()
		state.SrvType = types.StringNull()
		state.SrvVpsType = types.StringNull()
		state.SrvStatus = types.BoolNull()
		state.SrvStatusInstall = types.BoolNull()
		state.SrvSuspended = types.BoolNull()
		state.Os = types.ObjectNull(serverOSAttrTypes)
		state.Ips = types.ListNull(types.ObjectType{AttrTypes: serverIPAttrTypes})
		return
	}

	state.SrvPrimaryIP = types.StringNull()
	if s.SrvPrimaryIP != "" {
		state.SrvPrimaryIP = types.StringValue(s.SrvPrimaryIP)
	}
	state.SrvCores = types.Int64Value(int64(s.SrvCores))
	state.SrvRAM = types.Int64Value(int64(s.SrvRAM))
	state.SrvLocation = types.StringValue(s.SrvLocation)
	state.SrvType = types.StringValue(s.SrvType)
	state.SrvVpsType = types.StringValue(s.SrvVpsType)
	state.SrvStatus = types.BoolValue(bool(s.SrvStatus))
	state.SrvStatusInstall = types.BoolValue(bool(s.SrvStatusInstall))
	state.SrvSuspended = types.BoolValue(bool(s.SrvSuspended))
	state.Os = types.ObjectValueMust(serverOSAttrTypes, map[string]attr.Value{
		"os_id":      types.Int64Value(int64(s.OS.OSID)),
		"os_name":    types.StringValue(s.OS.OSName),
		"os_release": types.StringValue(s.OS.OSRelease),
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

	var found *client.Server
	if state.SrvID.IsNull() {
		servers, err := r.client.ListServers(ctx)
		if err != nil {
			resp.Diagnostics.AddError("Unable to Read Gigahost Server", err.Error())
			return
		}
		// A partially created server has no id in state yet; adopt it by its
		// deployment order once it appears, and never treat it as deleted.
		if !state.OrderId.IsNull() {
			orderID := strconv.FormatInt(state.OrderId.ValueInt64(), 10)
			for i := range servers {
				if equalID(servers[i].Order.OrderID, orderID) {
					found = &servers[i]
					state.SrvID = types.StringValue(servers[i].SrvID)
					break
				}
			}
		}
		if found == nil {
			resp.Diagnostics.AddWarning(
				"Gigahost Server Not Yet Identified",
				fmt.Sprintf("The server has no id in state and no server for %s has appeared in the server list yet. The resource is kept in state; its id will be adopted once the server appears.", orderRef(&state)),
			)
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			return
		}
	} else {
		detail, err := r.client.GetServer(ctx, state.SrvID.ValueString())
		switch {
		case err == nil:
			found = &detail.Server
		case errors.Is(err, client.ErrNotFound):
		default:
			resp.Diagnostics.AddError("Unable to Read Gigahost Server", err.Error())
			return
		}
	}
	if found == nil || strings.EqualFold(found.Order.OrderStatus, "cancelled") {
		resp.State.RemoveResource(ctx)
		return
	}

	if !state.SrvName.IsNull() {
		state.SrvName = types.StringValue(found.SrvName)
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
	if state.RegionName.IsNull() && found.Datacenter.RegionName != "" {
		state.RegionName = types.StringValue(found.Datacenter.RegionName)
	}
	if state.RegionId.IsNull() {
		if id, err := strconv.ParseInt(found.Datacenter.RegionID, 10, 64); err == nil && id != 0 {
			state.RegionId = types.Int64Value(id)
		}
	}
	if state.OsId.IsNull() {
		if id := int64(found.OS.OSID); id != 0 {
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

	if !plan.SrvName.IsNull() && !plan.SrvName.Equal(state.SrvName) {
		if err := r.client.UpdateServerName(ctx, state.SrvID.ValueString(), plan.SrvName.ValueString()); err != nil {
			resp.Diagnostics.AddError("Unable to Update Gigahost Server Name", err.Error())
			return
		}
	}

	// Apply the in-place changes onto prior state, so computed attributes
	// keep their stored values and never inherit unknowns from the plan.
	state.SrvName = plan.SrvName
	state.OSName = plan.OSName
	state.OSDist = plan.OSDist
	state.OsId = plan.OsId
	state.Timeouts = plan.Timeouts
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *serverResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state serverResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// A partially created server (the deploy wait failed before a server id
	// was seen) cannot be cancelled through the API.
	if state.SrvID.IsNull() || state.SrvID.ValueString() == "" {
		resp.Diagnostics.AddError(
			"Unable to Cancel Gigahost Server",
			fmt.Sprintf("The server has no id in state. Cancel %s in the Gigahost control panel if it is still active, then remove the resource with terraform state rm.", orderRef(&state)),
		)
		return
	}

	if err := r.client.CancelServer(ctx, state.SrvID.ValueString()); err != nil && !errors.Is(err, client.ErrNotFound) {
		// Cancel answers 400 rather than 404 for nonexistent or already-cancelled servers.
		detail, getErr := r.client.GetServer(ctx, state.SrvID.ValueString())
		if errors.Is(getErr, client.ErrNotFound) ||
			(getErr == nil && strings.EqualFold(detail.Order.OrderStatus, "cancelled")) {
			resp.Diagnostics.AddWarning(
				"Gigahost Server Already Gone",
				fmt.Sprintf("The server is already cancelled or gone, so the cancellation was refused (%s). Verify in the Gigahost control panel that %s is not active.", err, orderRef(&state)),
			)
			return
		}
		resp.Diagnostics.AddError(
			"Unable to Cancel Gigahost Server",
			fmt.Sprintf("%s\n\nRetry the destroy, or cancel %s in the Gigahost control panel.", err, orderRef(&state)),
		)
	}
}

func (r *serverResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("srv_id"), req, resp)
}
