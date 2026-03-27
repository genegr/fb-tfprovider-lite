package resources

import (
	"context"
	"fmt"

	"github.com/PureStorage-OpenConnect/terraform-provider-purefb/internal/fbclient"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &s3AccountResource{}
	_ resource.ResourceWithImportState = &s3AccountResource{}
)

type s3AccountResource struct {
	client *fbclient.Client
}

type s3AccountResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Quota            types.String `tfsdk:"quota"`
	HardLimitEnabled types.Bool   `tfsdk:"hard_limit_enabled"`
	QuotaLimit       types.Int64  `tfsdk:"quota_limit"`
	ObjectCount      types.Int64  `tfsdk:"object_count"`
	Created          types.Int64  `tfsdk:"created"`
}

func NewS3AccountResource() resource.Resource {
	return &s3AccountResource{}
}

func (r *s3AccountResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_s3_account"
}

func (r *s3AccountResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a FlashBlade S3 object store account.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "FlashBlade internal ID.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "Account name. Cannot be changed after creation.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"quota": schema.StringAttribute{
				Description: "Quota limit in human-readable format (e.g. '100G', '1T', '500M'). Empty string or omitted for unlimited.",
				Optional:    true,
			},
			"hard_limit_enabled": schema.BoolAttribute{
				Description: "Whether the quota is enforced as a hard limit.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"quota_limit": schema.Int64Attribute{
				Description: "Effective quota limit in bytes (read-only, from API).",
				Computed:    true,
			},
			"object_count": schema.Int64Attribute{
				Description: "Number of objects in the account.",
				Computed:    true,
			},
			"created": schema.Int64Attribute{
				Description: "Creation timestamp.",
				Computed:    true,
			},
		},
	}
}

func (r *s3AccountResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*fbclient.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", fmt.Sprintf("Expected *fbclient.Client, got %T", req.ProviderData))
		return
	}
	r.client = client
}

func (r *s3AccountResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan s3AccountResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := plan.Name.ValueString()

	// Step 1: create the account
	_, err := r.client.CreateObjectStoreAccount(name)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create account", err.Error())
		return
	}

	// Step 2: set quota/hard_limit if specified
	if !plan.Quota.IsNull() && !plan.Quota.IsUnknown() && plan.Quota.ValueString() != "" {
		quotaBytes, err := fbclient.HumanToBytes(plan.Quota.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Invalid quota", err.Error())
			return
		}
		patch := map[string]interface{}{
			"quota_limit":        fmt.Sprintf("%d", quotaBytes),
			"hard_limit_enabled": plan.HardLimitEnabled.ValueBool(),
		}
		_, err = r.client.UpdateObjectStoreAccount(name, patch)
		if err != nil {
			resp.Diagnostics.AddError("Failed to set account quota", err.Error())
			return
		}
	}

	// Read back to populate computed fields
	r.readIntoState(name, &plan, resp)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *s3AccountResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state s3AccountResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.readIntoState(state.Name.ValueString(), &state, resp)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *s3AccountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan s3AccountResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := plan.Name.ValueString()
	patch := map[string]interface{}{}

	// Handle quota
	if !plan.Quota.IsNull() && !plan.Quota.IsUnknown() && plan.Quota.ValueString() != "" {
		quotaBytes, err := fbclient.HumanToBytes(plan.Quota.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Invalid quota", err.Error())
			return
		}
		patch["quota_limit"] = fmt.Sprintf("%d", quotaBytes)
	} else {
		// Clear quota
		patch["quota_limit"] = ""
		patch["hard_limit_enabled"] = false
	}

	if !plan.HardLimitEnabled.IsNull() && !plan.HardLimitEnabled.IsUnknown() {
		patch["hard_limit_enabled"] = plan.HardLimitEnabled.ValueBool()
	}

	if len(patch) > 0 {
		_, err := r.client.UpdateObjectStoreAccount(name, patch)
		if err != nil {
			resp.Diagnostics.AddError("Failed to update account", err.Error())
			return
		}
	}

	r.readIntoState(name, &plan, resp)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *s3AccountResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state s3AccountResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteObjectStoreAccount(state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete account", err.Error())
	}
}

func (r *s3AccountResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}

// readIntoState reads the account from the API and populates the model.
// If the account is not found, it removes the resource from state.
func (r *s3AccountResource) readIntoState(name string, model *s3AccountResourceModel, resp interface{}) {
	acct, err := r.client.GetObjectStoreAccount(name)

	switch v := resp.(type) {
	case *resource.ReadResponse:
		if err != nil {
			v.Diagnostics.AddError("Failed to read account", err.Error())
			return
		}
		if acct == nil {
			v.State.RemoveResource(context.Background())
			return
		}
	case *resource.CreateResponse:
		if err != nil {
			v.Diagnostics.AddError("Failed to read account after create", err.Error())
			return
		}
		if acct == nil {
			v.Diagnostics.AddError("Account not found after create", fmt.Sprintf("Account %q was not found after creation", name))
			return
		}
	case *resource.UpdateResponse:
		if err != nil {
			v.Diagnostics.AddError("Failed to read account after update", err.Error())
			return
		}
		if acct == nil {
			v.Diagnostics.AddError("Account not found after update", fmt.Sprintf("Account %q was not found after update", name))
			return
		}
	}

	model.ID = types.StringValue(acct.ID)
	model.Name = types.StringValue(acct.Name)
	model.ObjectCount = types.Int64Value(acct.ObjectCount)
	model.Created = types.Int64Value(acct.Created)

	if acct.QuotaLimit != nil && *acct.QuotaLimit > 0 {
		model.QuotaLimit = types.Int64Value(*acct.QuotaLimit)
		// Preserve user's quota string if set; otherwise compute from bytes
		if model.Quota.IsNull() || model.Quota.IsUnknown() || model.Quota.ValueString() == "" {
			model.Quota = types.StringValue(fbclient.BytesToHuman(*acct.QuotaLimit))
		}
	} else {
		model.QuotaLimit = types.Int64Value(0)
		if model.Quota.IsNull() || model.Quota.IsUnknown() {
			model.Quota = types.StringNull()
		}
	}

	if acct.HardLimitEnabled != nil {
		model.HardLimitEnabled = types.BoolValue(*acct.HardLimitEnabled)
	} else {
		model.HardLimitEnabled = types.BoolValue(false)
	}
}
