package provider

import (
	"context"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/gattoufirabi/terraform-provider-staysa/internal/client"
)

type databaseResource struct{ base }

func newDatabaseResource() resource.Resource { return &databaseResource{} }

type databaseModel struct {
	Project       types.String `tfsdk:"project"`
	ID            types.String `tfsdk:"id"`
	Engine        types.String `tfsdk:"engine"`
	Version       types.String `tfsdk:"version"`
	CPUMillis     types.Int64  `tfsdk:"cpu_millis"`
	MemoryMB      types.Int64  `tfsdk:"memory_mb"`
	BackupEnabled types.Bool   `tfsdk:"backup_enabled"`
	BackupHour    types.Int64  `tfsdk:"backup_hour"`
	WaitRunning   types.Bool   `tfsdk:"wait_for_running"`
	Status        types.String `tfsdk:"status"`
	Host          types.String `tfsdk:"host"`
	Port          types.Int64  `tfsdk:"port"`
	Database      types.String `tfsdk:"database"`
	Username      types.String `tfsdk:"username"`
	Password      types.String `tfsdk:"password"`
	URL           types.String `tfsdk:"url"`
	SecretName    types.String `tfsdk:"secret_name"`
}

func (r *databaseResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database"
}

func (r *databaseResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	replace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	keepS := []planmodifier.String{stringplanmodifier.UseStateForUnknown()}
	keep := []planmodifier.Int64{int64planmodifier.UseStateForUnknown()}
	resp.Schema = schema.Schema{
		Description: "A managed database (PostgreSQL, MySQL, MariaDB, MongoDB, Redis or Kafka) on a project's private network. Deleting it needs an admin token and deletes its data.",
		Attributes: map[string]schema.Attribute{
			"project": schema.StringAttribute{Required: true, PlanModifiers: replace, Description: "Project ID."},
			"id":      schema.StringAttribute{Required: true, PlanModifiers: replace, Description: "Database ID: 2 to 20 lowercase letters, digits and dashes."},
			"engine": schema.StringAttribute{Required: true, PlanModifiers: replace,
				Description: "postgres, mysql, mariadb, mongodb, redis or kafka, optionally with a version (postgres:16). Changing it replaces the database."},
			"version": schema.StringAttribute{Computed: true, PlanModifiers: keepS, Description: "Engine version."},
			"cpu_millis": schema.Int64Attribute{Optional: true, Computed: true, PlanModifiers: keep,
				Description: "CPU in millicores (50 to 4000)."},
			"memory_mb": schema.Int64Attribute{Optional: true, Computed: true, PlanModifiers: keep,
				Description: "Memory in MB (128 to 8192; each engine has a minimum)."},
			"backup_enabled": schema.BoolAttribute{Optional: true, Computed: true,
				Description: "Daily backups (on by default; Kafka has none)."},
			"backup_hour": schema.Int64Attribute{Optional: true, Computed: true, PlanModifiers: keep,
				Description: "Hour of the daily backup, 0 to 23 UTC."},
			"wait_for_running": schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true),
				Description: "Wait until the database runs after creation (default true)."},
			"status":      schema.StringAttribute{Computed: true, Description: "starting, running, failed..."},
			"host":        schema.StringAttribute{Computed: true, PlanModifiers: keepS, Description: "Host name on the project's private network."},
			"port":        schema.Int64Attribute{Computed: true, PlanModifiers: keep, Description: "Port."},
			"database":    schema.StringAttribute{Computed: true, PlanModifiers: keepS, Description: "Database name."},
			"username":    schema.StringAttribute{Computed: true, PlanModifiers: keepS, Description: "User name."},
			"password":    schema.StringAttribute{Computed: true, Sensitive: true, PlanModifiers: keepS, Description: "Password (needs a developer or admin token)."},
			"url":         schema.StringAttribute{Computed: true, Sensitive: true, PlanModifiers: keepS, Description: "Connection URL, to give an application as a secret."},
			"secret_name": schema.StringAttribute{Computed: true, PlanModifiers: keepS, Description: "Usual secret name for the URL (DATABASE_URL, REDIS_URL...)."},
		},
	}
}

func (r *databaseResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.configure(req.ProviderData, &resp.Diagnostics)
}

func dbPath(project, id string) string {
	return "/projects/" + client.P(project) + "/databases/" + client.P(id)
}

func (m *databaseModel) fill(d *client.Database) {
	m.Project, m.ID = types.StringValue(d.Project), types.StringValue(d.ID)
	// keep the engine as written ("postgres" or "postgres:17") when it names this one
	if e := m.Engine.ValueString(); e != d.Engine && e != d.Engine+":"+d.Version {
		m.Engine = types.StringValue(d.Engine)
	}
	m.Version, m.Status = types.StringValue(d.Version), types.StringValue(d.Status)
	m.CPUMillis, m.MemoryMB = types.Int64Value(int64(d.CPUMillis)), types.Int64Value(int64(d.MemoryMB))
	m.BackupEnabled, m.BackupHour = types.BoolValue(d.BackupEnabled), types.Int64Value(int64(d.BackupHour))
	m.Host, m.Port = types.StringValue(d.Host), types.Int64Value(int64(d.Port))
	m.Database, m.Username, m.SecretName = types.StringValue(d.Database), types.StringValue(d.Username), types.StringValue(d.SecretName)
	m.Password, m.URL = strOrNull(d.Password), strOrNull(d.URL)
	if m.WaitRunning.IsNull() || m.WaitRunning.IsUnknown() {
		m.WaitRunning = types.BoolValue(true)
	}
}

func (m *databaseModel) settings() map[string]any {
	b := map[string]any{}
	if known(m.CPUMillis) {
		b["cpu_millis"] = m.CPUMillis.ValueInt64()
	}
	if known(m.MemoryMB) {
		b["memory_mb"] = m.MemoryMB.ValueInt64()
	}
	if known(m.BackupEnabled) {
		b["backup_enabled"] = m.BackupEnabled.ValueBool()
	}
	if known(m.BackupHour) {
		b["backup_hour"] = m.BackupHour.ValueInt64()
	}
	return b
}

func (r *databaseResource) waitRunning(ctx context.Context, project, id string, diags *diag.Diagnostics) *client.Database {
	deadline := time.Now().Add(10 * time.Minute)
	for {
		var d client.Database
		if err := r.c.Do(ctx, "GET", dbPath(project, id), nil, &d); err != nil {
			apiDiag(diags, "reading the database", err)
			return nil
		}
		if d.Status == "running" || d.Status == "failed" || time.Now().After(deadline) {
			if d.Status != "running" {
				diags.AddError("Staysa Cloud: database not running", "database "+project+"/"+id+" is "+d.Status)
			}
			return &d
		}
		select {
		case <-ctx.Done():
			diags.AddError("Staysa Cloud: interrupted", ctx.Err().Error())
			return &d
		case <-time.After(3 * time.Second):
		}
	}
}

func (r *databaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var m databaseModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body := m.settings()
	body["id"], body["engine"] = m.ID.ValueString(), strings.TrimSpace(m.Engine.ValueString())
	var d client.Database
	if err := r.c.Do(ctx, "POST", "/projects/"+client.P(m.Project.ValueString())+"/databases", body, &d); err != nil {
		apiDiag(&resp.Diagnostics, "creating the database", err)
		return
	}
	if m.WaitRunning.ValueBool() {
		if w := r.waitRunning(ctx, d.Project, d.ID, &resp.Diagnostics); w != nil {
			d = *w
		}
	}
	m.fill(&d)
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *databaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var m databaseModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var d client.Database
	err := r.c.Do(ctx, "GET", dbPath(m.Project.ValueString(), m.ID.ValueString()), nil, &d)
	if notFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		apiDiag(&resp.Diagnostics, "reading the database", err)
		return
	}
	m.fill(&d)
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *databaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var m databaseModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var d client.Database
	if err := r.c.Do(ctx, "PATCH", dbPath(m.Project.ValueString(), m.ID.ValueString()), m.settings(), &d); err != nil {
		apiDiag(&resp.Diagnostics, "updating the database", err)
		return
	}
	m.fill(&d)
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *databaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var m databaseModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.c.Do(ctx, "DELETE", dbPath(m.Project.ValueString(), m.ID.ValueString()), nil, nil); err != nil && !notFound(err) {
		apiDiag(&resp.Diagnostics, "deleting the database", err)
	}
}

func (r *databaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := splitID(req.ID, 2, &resp.Diagnostics, "project/database")
	if parts == nil {
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("wait_for_running"), true)...)
}

// ---------- data source ----------

type databaseDataSource struct{ base }

func newDatabaseDataSource() datasource.DataSource { return &databaseDataSource{} }

type databaseDataModel struct {
	Project  types.String `tfsdk:"project"`
	ID       types.String `tfsdk:"id"`
	Engine   types.String `tfsdk:"engine"`
	Version  types.String `tfsdk:"version"`
	Status   types.String `tfsdk:"status"`
	Host     types.String `tfsdk:"host"`
	Port     types.Int64  `tfsdk:"port"`
	Database types.String `tfsdk:"database"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
	URL      types.String `tfsdk:"url"`
}

func (d *databaseDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database"
}

func (d *databaseDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dschema.Schema{
		Description: "An existing managed database and how to connect to it.",
		Attributes: map[string]dschema.Attribute{
			"project":  dschema.StringAttribute{Required: true},
			"id":       dschema.StringAttribute{Required: true},
			"engine":   dschema.StringAttribute{Computed: true},
			"version":  dschema.StringAttribute{Computed: true},
			"status":   dschema.StringAttribute{Computed: true},
			"host":     dschema.StringAttribute{Computed: true},
			"port":     dschema.Int64Attribute{Computed: true},
			"database": dschema.StringAttribute{Computed: true},
			"username": dschema.StringAttribute{Computed: true},
			"password": dschema.StringAttribute{Computed: true, Sensitive: true},
			"url":      dschema.StringAttribute{Computed: true, Sensitive: true},
		},
	}
}

func (d *databaseDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.configure(req.ProviderData, &resp.Diagnostics)
}

func (d *databaseDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var m databaseDataModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var db client.Database
	if err := d.c.Do(ctx, "GET", dbPath(m.Project.ValueString(), m.ID.ValueString()), nil, &db); err != nil {
		apiDiag(&resp.Diagnostics, "reading the database", err)
		return
	}
	m.Engine, m.Version, m.Status = types.StringValue(db.Engine), types.StringValue(db.Version), types.StringValue(db.Status)
	m.Host, m.Port, m.Database, m.Username = types.StringValue(db.Host), types.Int64Value(int64(db.Port)), types.StringValue(db.Database), types.StringValue(db.Username)
	m.Password, m.URL = strOrNull(db.Password), strOrNull(db.URL)
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}
