package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/gattoufirabi/terraform-provider-staysa/internal/client"
)

// Where an application's builds come from and run: its git source and build settings
// (staysa_service_source), and the organization's own servers that run builds, like
// self-hosted CI agents (staysa_server_builds).

func emptySet() types.Set { return types.SetValueMust(types.StringType, []attr.Value{}) }

func toSet(list []string) types.Set {
	v := make([]attr.Value, 0, len(list))
	for _, s := range list {
		v = append(v, types.StringValue(s))
	}
	return types.SetValueMust(types.StringType, v)
}

func fromSet(ctx context.Context, s types.Set, diags *diag.Diagnostics) []string {
	out := []string{}
	if s.IsNull() || s.IsUnknown() {
		return out
	}
	diags.Append(s.ElementsAs(ctx, &out, false)...)
	return out
}

// ==================== staysa_service_source ====================

type serviceSourceResource struct{ base }

func newServiceSourceResource() resource.Resource { return &serviceSourceResource{} }

type serviceSourceModel struct {
	Project     types.String `tfsdk:"project"`
	Service     types.String `tfsdk:"service"`
	Repo        types.String `tfsdk:"repo"`
	Branch      types.String `tfsdk:"branch"`
	Dir         types.String `tfsdk:"dir"`
	Token       types.String `tfsdk:"token"`
	PRPreviews  types.Bool   `tfsdk:"pr_previews"`
	BuildOn     types.String `tfsdk:"build_on"`
	BuildLabels types.Set    `tfsdk:"build_labels"`
	HasToken    types.Bool   `tfsdk:"has_token"`
}

type sourceAPI struct {
	Repo        string   `json:"repo"`
	Branch      string   `json:"branch"`
	Dir         string   `json:"dir"`
	PRPreviews  bool     `json:"pr_previews"`
	BuildOn     string   `json:"build_on"`
	BuildLabels []string `json:"build_labels"`
	HasToken    bool     `json:"has_token"`
	Token       *string  `json:"token,omitempty"`
}

func (r *serviceSourceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_source"
}

func (r *serviceSourceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The git repository an application is built from, and where its builds run: on the platform's builder, or on the organization's own build servers (see staysa_server_builds). Destroying it removes the repository from the application.",
		Attributes: map[string]schema.Attribute{
			"project": schema.StringAttribute{Required: true, PlanModifiers: replaceS},
			"service": schema.StringAttribute{Required: true, PlanModifiers: replaceS, Description: "Application ID."},
			"repo":    schema.StringAttribute{Required: true, Description: "HTTPS address of the git repository. Builds on your own servers may also use a private address, such as an on-premises GitLab."},
			"branch": schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("main"),
				Description: "Branch built on each push (default main)."},
			"dir": schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString(""),
				Description: "Folder inside the repository to build, such as services/web; empty for the root."},
			"token": schema.StringAttribute{Optional: true, Sensitive: true,
				Description: "Read token of a private repository. The platform never shows it again, so changes made outside Terraform are not detected; removing it from the configuration removes it from the platform."},
			"pr_previews": schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false),
				Description: "Deploy a preview of each pull request."},
			"build_on": schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("platform"),
				Description: "Where builds run: platform (the platform's builder, billed as build minutes) or servers (the organization's own servers that run builds)."},
			"build_labels": schema.SetAttribute{Optional: true, Computed: true, ElementType: types.StringType, Default: setdefault.StaticValue(emptySet()),
				Description: "With build_on = servers: a build runs only on a server that has every one of these labels. A server's operating system and architecture (linux, amd64, arm64) count as labels."},
			"has_token": schema.BoolAttribute{Computed: true, Description: "Whether the platform holds a read token."},
		},
	}
}

func (r *serviceSourceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.configure(req.ProviderData, &resp.Diagnostics)
}

func (r *serviceSourceResource) put(ctx context.Context, m *serviceSourceModel, prevToken types.String, diags *diag.Diagnostics) {
	body := sourceAPI{Repo: m.Repo.ValueString(), Branch: m.Branch.ValueString(), Dir: m.Dir.ValueString(),
		PRPreviews: m.PRPreviews.ValueBool(), BuildOn: m.BuildOn.ValueString(), BuildLabels: fromSet(ctx, m.BuildLabels, diags)}
	if diags.HasError() {
		return
	}
	// Send the token only when it changed: "" removes it.
	if tok := m.Token.ValueString(); tok != prevToken.ValueString() || (prevToken.IsNull() && !m.Token.IsNull()) {
		body.Token = &tok
	}
	var out sourceAPI
	if err := r.c.Do(ctx, "PUT", servicePath(m.Project.ValueString(), m.Service.ValueString())+"/source", body, &out); err != nil {
		apiDiag(diags, "saving the source", err)
		return
	}
	m.fill(out)
}

func (m *serviceSourceModel) fill(out sourceAPI) {
	m.Repo, m.Branch, m.Dir = types.StringValue(out.Repo), types.StringValue(out.Branch), types.StringValue(out.Dir)
	m.PRPreviews, m.BuildOn, m.HasToken = types.BoolValue(out.PRPreviews), types.StringValue(out.BuildOn), types.BoolValue(out.HasToken)
	m.BuildLabels = toSet(out.BuildLabels)
}

func (r *serviceSourceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var m serviceSourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.put(ctx, &m, types.StringNull(), &resp.Diagnostics)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
	}
}

func (r *serviceSourceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var m serviceSourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var out sourceAPI
	err := r.c.Do(ctx, "GET", servicePath(m.Project.ValueString(), m.Service.ValueString())+"/source", nil, &out)
	if notFound(err) || (err == nil && out.Repo == "") {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		apiDiag(&resp.Diagnostics, "reading the source", err)
		return
	}
	m.fill(out)
	if !out.HasToken && !m.Token.IsNull() && m.Token.ValueString() != "" {
		m.Token = types.StringNull() // removed outside Terraform: the next plan sets it again
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *serviceSourceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var m, prev serviceSourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &prev)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.put(ctx, &m, prev.Token, &resp.Diagnostics)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
	}
}

func (r *serviceSourceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var m serviceSourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	empty := ""
	err := r.c.Do(ctx, "PUT", servicePath(m.Project.ValueString(), m.Service.ValueString())+"/source", sourceAPI{Token: &empty, BuildLabels: []string{}}, nil)
	if err != nil && !notFound(err) {
		apiDiag(&resp.Diagnostics, "removing the source", err)
	}
}

// ImportState takes "project/service".
func (r *serviceSourceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	p, s, ok := strings.Cut(req.ID, "/")
	if !ok || p == "" || s == "" {
		resp.Diagnostics.AddError("Invalid import ID", "Use project/service, for example shop/web.")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project"), p)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("service"), s)...)
}

// ==================== staysa_server_builds ====================

type serverBuildsResource struct{ base }

func newServerBuildsResource() resource.Resource { return &serverBuildsResource{} }

type serverBuildsModel struct {
	Server     types.String `tfsdk:"server"`
	Labels     types.Set    `tfsdk:"labels"`
	MaxBuilds  types.Int64  `tfsdk:"max_builds"`
	Clean      types.Bool   `tfsdk:"clean"`
	Name       types.String `tfsdk:"name"`
	AutoLabels types.Set    `tfsdk:"auto_labels"`
}

type serverAPI struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Builds     bool     `json:"builds"`
	Labels     []string `json:"labels"`
	AutoLabels []string `json:"auto_labels"`
	MaxBuilds  int64    `json:"max_builds"`
	Clean      bool     `json:"clean_builds"`
}

func (r *serverBuildsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server_builds"
}

func (r *serverBuildsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Makes one of the organization's own servers run builds, like a self-hosted CI agent: it takes the builds of applications whose staysa_service_source has build_on = servers and whose build_labels it has. Destroying it stops the server taking builds. Needs an organization-wide token with the admin role.",
		Attributes: map[string]schema.Attribute{
			"server": schema.StringAttribute{Required: true, PlanModifiers: replaceS, Description: "Server ID, as shown on the Your servers page."},
			"labels": schema.SetAttribute{Optional: true, Computed: true, ElementType: types.StringType, Default: setdefault.StaticValue(emptySet()),
				Description: "Labels matched against an application's build_labels (up to 10; lowercase letters, digits, dots, dashes, underscores)."},
			"max_builds": schema.Int64Attribute{Optional: true, Computed: true, Default: int64default.StaticInt64(1),
				Description: "Builds the server runs at once, 1 to 8 (default 1)."},
			"clean": schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false),
				Description: "Start every build from an empty cache and keep nothing after it, like a throwaway agent (slower builds)."},
			"name":        schema.StringAttribute{Computed: true, PlanModifiers: keepS, Description: "Server name."},
			"auto_labels": schema.SetAttribute{Computed: true, ElementType: types.StringType, Description: "Labels the server has automatically: its operating system and architecture."},
		},
	}
}

func (r *serverBuildsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.configure(req.ProviderData, &resp.Diagnostics)
}

func (m *serverBuildsModel) fill(out serverAPI) {
	m.Labels, m.MaxBuilds, m.Clean = toSet(out.Labels), types.Int64Value(out.MaxBuilds), types.BoolValue(out.Clean)
	m.Name, m.AutoLabels = types.StringValue(out.Name), toSet(out.AutoLabels)
}

func (r *serverBuildsResource) put(ctx context.Context, m *serverBuildsModel, enabled bool, diags *diag.Diagnostics) {
	body := map[string]any{"enabled": enabled, "labels": fromSet(ctx, m.Labels, diags), "max_builds": m.MaxBuilds.ValueInt64(), "clean": m.Clean.ValueBool()}
	if diags.HasError() {
		return
	}
	var out serverAPI
	if err := r.c.Do(ctx, "PUT", "/servers/"+client.P(m.Server.ValueString())+"/builds", body, &out); err != nil {
		apiDiag(diags, "saving the server's build settings", err)
		return
	}
	m.fill(out)
}

func (r *serverBuildsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var m serverBuildsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.put(ctx, &m, true, &resp.Diagnostics)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
	}
}

func (r *serverBuildsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var m serverBuildsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var out serverAPI
	err := r.c.Do(ctx, "GET", "/servers/"+client.P(m.Server.ValueString()), nil, &out)
	if notFound(err) || (err == nil && !out.Builds) {
		resp.State.RemoveResource(ctx) // server removed, or builds turned off outside Terraform
		return
	}
	if err != nil {
		apiDiag(&resp.Diagnostics, "reading the server", err)
		return
	}
	m.fill(out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *serverBuildsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var m serverBuildsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.put(ctx, &m, true, &resp.Diagnostics)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
	}
}

func (r *serverBuildsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var m serverBuildsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := r.c.Do(ctx, "PUT", "/servers/"+client.P(m.Server.ValueString())+"/builds",
		map[string]any{"enabled": false, "labels": []string{}, "max_builds": 1, "clean": false}, nil)
	if err != nil && !notFound(err) {
		apiDiag(&resp.Diagnostics, "turning builds off on the server", err)
	}
}

func (r *serverBuildsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("server"), req, resp)
}
