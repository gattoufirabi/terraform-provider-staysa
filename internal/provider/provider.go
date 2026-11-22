// Package provider is the Terraform / OpenTofu provider for Staysa Cloud.
package provider

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/gattoufirabi/terraform-provider-staysa/internal/client"
)

type staysaProvider struct{ version string }

func New(version string) func() provider.Provider {
	return func() provider.Provider { return &staysaProvider{version: version} }
}

type providerModel struct {
	Endpoint types.String `tfsdk:"endpoint"`
	Token    types.String `tfsdk:"token"`
}

func (p *staysaProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "staysa"
	resp.Version = p.version
}

func (p *staysaProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage projects, applications, databases, domains, DNS and storage on a Staysa Cloud platform.",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				Optional:    true,
				Description: "Address of the console, such as https://console.example.com. Defaults to the STAYSA_ENDPOINT environment variable.",
			},
			"token": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "API token (pla_...) created under Access > API tokens. Defaults to the STAYSA_TOKEN environment variable.",
			},
		},
	}
}

func (p *staysaProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var cfg providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	endpoint, token := os.Getenv("STAYSA_ENDPOINT"), os.Getenv("STAYSA_TOKEN")
	if !cfg.Endpoint.IsNull() && !cfg.Endpoint.IsUnknown() {
		endpoint = cfg.Endpoint.ValueString()
	}
	if !cfg.Token.IsNull() && !cfg.Token.IsUnknown() {
		token = cfg.Token.ValueString()
	}
	if endpoint == "" {
		resp.Diagnostics.AddAttributeError(path.Root("endpoint"), "Missing endpoint",
			"Set the console address in the provider block or in STAYSA_ENDPOINT.")
	}
	if !strings.HasPrefix(token, "pla_") {
		resp.Diagnostics.AddAttributeError(path.Root("token"), "Missing API token",
			"Set an API token (pla_...) in the provider block or in STAYSA_TOKEN.")
	}
	if resp.Diagnostics.HasError() {
		return
	}
	c := client.New(endpoint, token, p.version)
	resp.ResourceData, resp.DataSourceData = c, c
}

func (p *staysaProvider) Resources(context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		newProjectResource,
		newServiceResource,
		newServiceSecretResource,
		newProjectVariableResource,
		newDatabaseResource,
		newDomainResource,
		newDNSZoneResource,
		newDNSRecordResource,
		newBucketResource,
	}
}

func (p *staysaProvider) DataSources(context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		newProjectDataSource,
		newServiceDataSource,
		newDatabaseDataSource,
	}
}

// ---------- shared helpers ----------

// base holds the API client of a resource or data source.
type base struct{ c *client.Client }

func (b *base) configure(data any, diags *diag.Diagnostics) {
	if data == nil {
		return // not configured yet (validation)
	}
	c, ok := data.(*client.Client)
	if !ok {
		diags.AddError("Provider error", "unexpected provider data")
		return
	}
	b.c = c
}

func notFound(err error) bool { return errors.Is(err, client.ErrNotFound) }

func apiDiag(diags *diag.Diagnostics, action string, err error) {
	diags.AddError("Staysa Cloud: "+action+" failed", err.Error())
}

// splitID splits an import ID such as "shop/web" into n parts.
func splitID(id string, n int, diags *diag.Diagnostics, format string) []string {
	parts := strings.Split(id, "/")
	if len(parts) != n {
		diags.AddError("Invalid import ID", "expected "+format+", got "+id)
		return nil
	}
	for _, p := range parts {
		if p == "" {
			diags.AddError("Invalid import ID", "expected "+format+", got "+id)
			return nil
		}
	}
	return parts
}

func strOrNull(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}
