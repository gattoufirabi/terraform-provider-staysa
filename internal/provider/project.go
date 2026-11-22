package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/gattoufirabi/terraform-provider-staysa/internal/client"
)

type projectResource struct{ base }

func newProjectResource() resource.Resource { return &projectResource{} }

type projectModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	ForceDelete types.Bool   `tfsdk:"force_delete"`
}

func (r *projectResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (r *projectResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A project: a group of applications and databases sharing a private network. Needs an admin token.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:      true,
				Description:   "Project ID: 2 to 20 lowercase letters, digits and dashes. Changing it replaces the project.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Optional: true, Computed: true,
				Description: "Display name (defaults to the ID).",
			},
			"force_delete": schema.BoolAttribute{
				Optional: true, Computed: true, Default: booldefault.StaticBool(false),
				Description: "Delete the project even if it still holds applications or databases not managed here. Default false.",
			},
		},
	}
}

func (r *projectResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.configure(req.ProviderData, &resp.Diagnostics)
}

func (r *projectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var m projectModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body := map[string]any{"id": m.ID.ValueString()}
	if !m.Name.IsNull() && !m.Name.IsUnknown() {
		body["name"] = m.Name.ValueString()
	}
	var p client.Project
	if err := r.c.Do(ctx, "POST", "/projects", body, &p); err != nil {
		apiDiag(&resp.Diagnostics, "creating the project", err)
		return
	}
	m.Name = types.StringValue(p.Name)
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *projectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var m projectModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var p client.Project
	err := r.c.Do(ctx, "GET", "/projects/"+client.P(m.ID.ValueString()), nil, &p)
	if notFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		apiDiag(&resp.Diagnostics, "reading the project", err)
		return
	}
	m.Name = types.StringValue(p.Name)
	if m.ForceDelete.IsNull() {
		m.ForceDelete = types.BoolValue(false)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *projectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var m projectModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !m.Name.IsUnknown() && !m.Name.IsNull() {
		var p client.Project
		if err := r.c.Do(ctx, "PATCH", "/projects/"+client.P(m.ID.ValueString()), map[string]any{"name": m.Name.ValueString()}, &p); err != nil {
			apiDiag(&resp.Diagnostics, "renaming the project", err)
			return
		}
		m.Name = types.StringValue(p.Name)
	} else {
		var st projectModel
		resp.Diagnostics.Append(req.State.Get(ctx, &st)...)
		m.Name = st.Name
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *projectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var m projectModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	q := ""
	if m.ForceDelete.ValueBool() {
		q = "?force=true"
	}
	if err := r.c.Do(ctx, "DELETE", "/projects/"+client.P(m.ID.ValueString())+q, nil, nil); err != nil && !notFound(err) {
		apiDiag(&resp.Diagnostics, "deleting the project", err)
	}
}

func (r *projectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("force_delete"), false)...)
}

// ---------- data source ----------

type projectDataSource struct{ base }

func newProjectDataSource() datasource.DataSource { return &projectDataSource{} }

func (d *projectDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (d *projectDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dschema.Schema{
		Description: "An existing project.",
		Attributes: map[string]dschema.Attribute{
			"id":   dschema.StringAttribute{Required: true, Description: "Project ID."},
			"name": dschema.StringAttribute{Computed: true, Description: "Display name."},
		},
	}
}

func (d *projectDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.configure(req.ProviderData, &resp.Diagnostics)
}

func (d *projectDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var m struct {
		ID   types.String `tfsdk:"id"`
		Name types.String `tfsdk:"name"`
	}
	resp.Diagnostics.Append(req.Config.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var p client.Project
	if err := d.c.Do(ctx, "GET", "/projects/"+client.P(m.ID.ValueString()), nil, &p); err != nil {
		apiDiag(&resp.Diagnostics, "reading the project", err)
		return
	}
	m.Name = types.StringValue(p.Name)
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}
