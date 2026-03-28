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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &bucketResource{}
	_ resource.ResourceWithImportState = &bucketResource{}
)

type bucketResource struct {
	client *fbclient.Client
}

type bucketResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	AccountName        types.String `tfsdk:"account_name"`
	Versioning         types.String `tfsdk:"versioning"`
	Quota              types.String `tfsdk:"quota"`
	HardLimitEnabled   types.Bool   `tfsdk:"hard_limit_enabled"`
	EradicateOnDestroy types.Bool   `tfsdk:"eradicate_on_destroy"`
	QuotaLimit         types.Int64  `tfsdk:"quota_limit"`
	BucketType         types.String `tfsdk:"bucket_type"`
	ObjectCount        types.Int64  `tfsdk:"object_count"`
	Destroyed          types.Bool   `tfsdk:"destroyed"`
	Created            types.Int64  `tfsdk:"created"`
}

func NewBucketResource() resource.Resource {
	return &bucketResource{}
}

func (r *bucketResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bucket"
}

func (r *bucketResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a FlashBlade S3 bucket.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "FlashBlade internal ID.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "Bucket name. Cannot be changed after creation.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"account_name": schema.StringAttribute{
				Description: "Parent S3 account name. Cannot be changed after creation.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"versioning": schema.StringAttribute{
				Description: "Bucket versioning state: 'none', 'enabled', or 'suspended'.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("none"),
			},
			"quota": schema.StringAttribute{
				Description: "Quota limit in human-readable format (e.g. '100G', '1T'). Empty or omitted for unlimited.",
				Optional:    true,
			},
			"hard_limit_enabled": schema.BoolAttribute{
				Description: "Whether the quota is enforced as a hard limit.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"eradicate_on_destroy": schema.BoolAttribute{
				Description: "If true, permanently eradicate the bucket on destroy. If false, soft-delete only (recoverable from trash).",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"quota_limit": schema.Int64Attribute{
				Description: "Effective quota limit in bytes (read-only, from API).",
				Computed:    true,
			},
			"bucket_type": schema.StringAttribute{
				Description: "Bucket type (e.g. 'classic', 'multi-site-writable'). Read-only.",
				Computed:    true,
			},
			"object_count": schema.Int64Attribute{
				Description: "Number of objects in the bucket.",
				Computed:    true,
			},
			"destroyed": schema.BoolAttribute{
				Description: "Whether the bucket is in destroyed (trash) state.",
				Computed:    true,
			},
			"created": schema.Int64Attribute{
				Description: "Creation timestamp.",
				Computed:    true,
			},
		},
	}
}

func (r *bucketResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *bucketResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan bucketResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := plan.Name.ValueString()
	accountName := plan.AccountName.ValueString()

	// Build create body
	createBody := fbclient.BucketCreateBody{
		Account: &fbclient.BucketAccount{Name: accountName},
	}

	if !plan.Quota.IsNull() && !plan.Quota.IsUnknown() && plan.Quota.ValueString() != "" {
		quotaBytes, err := fbclient.HumanToBytes(plan.Quota.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Invalid quota", err.Error())
			return
		}
		quotaStr := fmt.Sprintf("%d", quotaBytes)
		createBody.QuotaLimit = &quotaStr
		hardLimit := plan.HardLimitEnabled.ValueBool()
		createBody.HardLimitEnabled = &hardLimit
	}

	// Step 1: create the bucket
	_, err := r.client.CreateBucket(name, createBody)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create bucket", err.Error())
		return
	}

	// Step 2: set versioning if not "none"
	versioning := plan.Versioning.ValueString()
	if versioning != "" && versioning != "none" {
		patch := map[string]interface{}{"versioning": versioning}
		_, err = r.client.UpdateBucket(name, patch)
		if err != nil {
			resp.Diagnostics.AddError("Failed to set bucket versioning", err.Error())
			return
		}
	}

	// Read back
	r.readIntoState(name, &plan, resp)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *bucketResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state bucketResourceModel
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

func (r *bucketResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan bucketResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := plan.Name.ValueString()
	patch := map[string]interface{}{}

	// Versioning
	versioning := plan.Versioning.ValueString()
	if versioning != "" {
		patch["versioning"] = versioning
	}

	// Quota
	if !plan.Quota.IsNull() && !plan.Quota.IsUnknown() && plan.Quota.ValueString() != "" {
		quotaBytes, err := fbclient.HumanToBytes(plan.Quota.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Invalid quota", err.Error())
			return
		}
		patch["quota_limit"] = fmt.Sprintf("%d", quotaBytes)
	} else {
		patch["quota_limit"] = ""
		patch["hard_limit_enabled"] = false
	}

	if !plan.HardLimitEnabled.IsNull() && !plan.HardLimitEnabled.IsUnknown() {
		patch["hard_limit_enabled"] = plan.HardLimitEnabled.ValueBool()
	}

	if len(patch) > 0 {
		_, err := r.client.UpdateBucket(name, patch)
		if err != nil {
			resp.Diagnostics.AddError("Failed to update bucket", err.Error())
			return
		}
	}

	r.readIntoState(name, &plan, resp)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *bucketResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state bucketResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	eradicate := state.EradicateOnDestroy.ValueBool()
	err := r.client.DeleteBucket(state.Name.ValueString(), eradicate)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete bucket", err.Error())
	}
}

func (r *bucketResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}

// readIntoState reads the bucket from the API and populates the model.
func (r *bucketResource) readIntoState(name string, model *bucketResourceModel, resp interface{}) {
	bucket, err := r.client.GetBucket(name)

	switch v := resp.(type) {
	case *resource.ReadResponse:
		if err != nil {
			v.Diagnostics.AddError("Failed to read bucket", err.Error())
			return
		}
		if bucket == nil || bucket.Destroyed {
			v.State.RemoveResource(context.Background())
			return
		}
	case *resource.CreateResponse:
		if err != nil {
			v.Diagnostics.AddError("Failed to read bucket after create", err.Error())
			return
		}
		if bucket == nil {
			v.Diagnostics.AddError("Bucket not found after create", fmt.Sprintf("Bucket %q was not found after creation", name))
			return
		}
	case *resource.UpdateResponse:
		if err != nil {
			v.Diagnostics.AddError("Failed to read bucket after update", err.Error())
			return
		}
		if bucket == nil {
			v.Diagnostics.AddError("Bucket not found after update", fmt.Sprintf("Bucket %q was not found after update", name))
			return
		}
	}

	model.ID = types.StringValue(bucket.ID)
	model.Name = types.StringValue(bucket.Name)
	model.BucketType = types.StringValue(bucket.BucketType)
	model.ObjectCount = types.Int64Value(bucket.ObjectCount)
	model.Created = types.Int64Value(bucket.Created)
	model.Destroyed = types.BoolValue(bucket.Destroyed)

	if bucket.Versioning != "" {
		model.Versioning = types.StringValue(bucket.Versioning)
	}

	if bucket.Account != nil {
		model.AccountName = types.StringValue(bucket.Account.Name)
	}

	if bucket.QuotaLimit != nil && *bucket.QuotaLimit > 0 {
		model.QuotaLimit = types.Int64Value(*bucket.QuotaLimit)
		if model.Quota.IsNull() || model.Quota.IsUnknown() || model.Quota.ValueString() == "" {
			model.Quota = types.StringValue(fbclient.BytesToHuman(*bucket.QuotaLimit))
		}
	} else {
		model.QuotaLimit = types.Int64Value(0)
		if model.Quota.IsNull() || model.Quota.IsUnknown() {
			model.Quota = types.StringNull()
		}
	}

	if bucket.HardLimitEnabled != nil {
		model.HardLimitEnabled = types.BoolValue(*bucket.HardLimitEnabled)
	} else {
		model.HardLimitEnabled = types.BoolValue(false)
	}

	// Preserve eradicate_on_destroy from config (not an API field)
	if model.EradicateOnDestroy.IsNull() || model.EradicateOnDestroy.IsUnknown() {
		model.EradicateOnDestroy = types.BoolValue(false)
	}
}
