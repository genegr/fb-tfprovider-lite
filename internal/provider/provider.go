package provider

import (
	"context"
	"os"

	"github.com/PureStorage-OpenConnect/terraform-provider-purefb/internal/fbclient"
	"github.com/PureStorage-OpenConnect/terraform-provider-purefb/internal/resources"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = &purefbProvider{}

type purefbProvider struct{}

type purefbProviderModel struct {
	FBURL     types.String `tfsdk:"fb_url"`
	APIToken  types.String `tfsdk:"api_token"`
	VerifySSL types.Bool   `tfsdk:"verify_ssl"`
}

func New() provider.Provider {
	return &purefbProvider{}
}

func (p *purefbProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "purefb"
}

func (p *purefbProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Terraform provider for managing Pure Storage FlashBlade S3 resources.",
		Attributes: map[string]schema.Attribute{
			"fb_url": schema.StringAttribute{
				Description: "FlashBlade management IP or hostname. Can also be set via PUREFB_URL env var.",
				Optional:    true,
			},
			"api_token": schema.StringAttribute{
				Description: "FlashBlade API token. Can also be set via PUREFB_API env var.",
				Optional:    true,
				Sensitive:   true,
			},
			"verify_ssl": schema.BoolAttribute{
				Description: "Whether to verify SSL certificates. Defaults to false (FlashBlade typically uses self-signed certs).",
				Optional:    true,
			},
		},
	}
}

func (p *purefbProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config purefbProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	fbURL := os.Getenv("PUREFB_URL")
	if !config.FBURL.IsNull() {
		fbURL = config.FBURL.ValueString()
	}
	if fbURL == "" {
		resp.Diagnostics.AddError("Missing FlashBlade URL", "Set fb_url in provider config or PUREFB_URL env var.")
		return
	}

	apiToken := os.Getenv("PUREFB_API")
	if !config.APIToken.IsNull() {
		apiToken = config.APIToken.ValueString()
	}
	if apiToken == "" {
		resp.Diagnostics.AddError("Missing API token", "Set api_token in provider config or PUREFB_API env var.")
		return
	}

	verifySSL := false
	if !config.VerifySSL.IsNull() {
		verifySSL = config.VerifySSL.ValueBool()
	}

	client, err := fbclient.NewClient(fbURL, apiToken, verifySSL)
	if err != nil {
		resp.Diagnostics.AddError("Failed to connect to FlashBlade", err.Error())
		return
	}

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *purefbProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		resources.NewS3AccountResource,
		resources.NewBucketResource,
	}
}

func (p *purefbProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}
