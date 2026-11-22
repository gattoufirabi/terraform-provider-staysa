package provider

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/gattoufirabi/terraform-provider-staysa/internal/client"
)

type serviceResource struct{ base }

func newServiceResource() resource.Resource { return &serviceResource{} }

type autoscalingModel struct {
	Min        types.Int64 `tfsdk:"min"`
	Max        types.Int64 `tfsdk:"max"`
	CPUPercent types.Int64 `tfsdk:"cpu_percent"`
}

type serviceModel struct {
	Project      types.String      `tfsdk:"project"`
	ID           types.String      `tfsdk:"id"`
	Image        types.String      `tfsdk:"image"`
	Port         types.Int64       `tfsdk:"port"`
	HealthPath   types.String      `tfsdk:"health_path"`
	CPUMillis    types.Int64       `tfsdk:"cpu_millis"`
	MemoryMB     types.Int64       `tfsdk:"memory_mb"`
	VolumePath   types.String      `tfsdk:"volume_path"`
	Instances    types.Int64       `tfsdk:"instances"`
	Worker       types.Bool        `tfsdk:"worker"`
	Env          types.Map         `tfsdk:"env"`
	PublicPorts  types.Set         `tfsdk:"public_ports"`
	Autoscaling  *autoscalingModel `tfsdk:"autoscaling"`
	MetricsPath  types.String      `tfsdk:"metrics_path"`
	MetricsPort  types.Int64       `tfsdk:"metrics_port"`
	EdgeCache    types.Bool        `tfsdk:"edge_cache"`
	Public       types.Bool        `tfsdk:"public"`
	Deploy       types.Bool        `tfsdk:"deploy"`
	Wait         types.Bool        `tfsdk:"wait_for_live"`
	URL          types.String      `tfsdk:"url"`
	InternalHost types.String      `tfsdk:"internal_host"`
	Status       types.String      `tfsdk:"status"`
	Revision     types.Int64       `tfsdk:"revision"`
}

func (r *serviceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service"
}

func (r *serviceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	keep := []planmodifier.Int64{int64planmodifier.UseStateForUnknown()}
	keepS := []planmodifier.String{stringplanmodifier.UseStateForUnknown()}
	replace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		Description: "An application (web service or worker) running a container image. Each change is deployed (blue/green) unless deploy is false.",
		Attributes: map[string]schema.Attribute{
			"project": schema.StringAttribute{Required: true, PlanModifiers: replace, Description: "Project ID."},
			"id":      schema.StringAttribute{Required: true, PlanModifiers: replace, Description: "Application ID: 2 to 20 lowercase letters, digits and dashes."},
			"image":   schema.StringAttribute{Required: true, Description: "Container image, such as nginx:1.27-alpine or an image of the platform's registry."},
			"port": schema.Int64Attribute{Optional: true, Computed: true, PlanModifiers: keep,
				Description: "Port the container listens on (web services)."},
			"health_path": schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: keepS,
				Description: "HTTP path checked before traffic switches to a new deployment."},
			"cpu_millis": schema.Int64Attribute{Optional: true, Computed: true, PlanModifiers: keep,
				Description: "CPU per instance, in millicores (1000 = one core)."},
			"memory_mb": schema.Int64Attribute{Optional: true, Computed: true, PlanModifiers: keep,
				Description: "Memory per instance, in MB."},
			"volume_path": schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: keepS,
				Description: "Folder kept across deployments (persistent volume); empty for none. An application with a volume runs one instance."},
			"instances": schema.Int64Attribute{Optional: true, Computed: true, PlanModifiers: keep,
				Description: "Number of instances (load balanced). Ignored while autoscaling is set: the autoscaler decides."},
			"worker": schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false),
				Description: "A background worker: no port, no address."},
			"env": schema.MapAttribute{Optional: true, Computed: true, ElementType: types.StringType,
				Default:     mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})),
				Description: "Environment variables (plain values; use staysa_service_secret for secrets)."},
			"public_ports": schema.SetAttribute{Optional: true, Computed: true, ElementType: types.StringType,
				Default:     setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
				Description: "Raw TCP or UDP ports open to the internet, such as \"25565/tcp\"."},
			"autoscaling": schema.SingleNestedAttribute{Optional: true,
				Description: "Scale between min and max instances around a CPU target.",
				Attributes: map[string]schema.Attribute{
					"min":         schema.Int64Attribute{Required: true, Description: "Fewest instances (1 to 10)."},
					"max":         schema.Int64Attribute{Required: true, Description: "Most instances (up to 10)."},
					"cpu_percent": schema.Int64Attribute{Optional: true, Computed: true, Default: int64default.StaticInt64(70), Description: "CPU target, 10 to 95 (default 70)."},
				}},
			"metrics_path": schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: keepS,
				Description: "Prometheus endpoint the platform scrapes, such as /metrics; empty for none."},
			"metrics_port": schema.Int64Attribute{Optional: true, Computed: true, PlanModifiers: keep,
				Description: "Port of the metrics endpoint; 0 for the application's port."},
			"edge_cache": schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false),
				Description: "The proxy caches responses that allow it."},
			"public": schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true),
				Description: "Has a public address; false keeps it on the private network only."},
			"deploy": schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true),
				Description: "Deploy each change at once (default). False only saves settings for the next deployment."},
			"wait_for_live": schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true),
				Description: "Wait until a deployment is live, and fail if it fails (default true)."},
			"url":           schema.StringAttribute{Computed: true, PlanModifiers: keepS, Description: "Public address."},
			"internal_host": schema.StringAttribute{Computed: true, PlanModifiers: keepS, Description: "Name on the project's private network."},
			"status":        schema.StringAttribute{Computed: true, Description: "Status of the latest deployment."},
			"revision":      schema.Int64Attribute{Computed: true, Description: "Latest deployment revision."},
		},
	}
}

func (r *serviceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.configure(req.ProviderData, &resp.Diagnostics)
}

func known[T interface {
	IsNull() bool
	IsUnknown() bool
}](v T) bool {
	return !v.IsNull() && !v.IsUnknown()
}

// body turns the plan into the API's fields; unknown values are left out (the platform
// keeps or chooses them).
func (m *serviceModel) body(ctx context.Context, diags *diag.Diagnostics) map[string]any {
	b := map[string]any{"image": m.Image.ValueString(), "deploy": m.Deploy.ValueBool()}
	for k, v := range map[string]types.Int64{"port": m.Port, "cpu_millis": m.CPUMillis, "memory_mb": m.MemoryMB, "metrics_port": m.MetricsPort} {
		if known(v) {
			b[k] = v.ValueInt64()
		}
	}
	if known(m.Instances) && m.Autoscaling == nil {
		b["instances"] = m.Instances.ValueInt64()
	}
	for k, v := range map[string]types.String{"health_path": m.HealthPath, "volume_path": m.VolumePath, "metrics_path": m.MetricsPath} {
		if known(v) {
			b[k] = v.ValueString()
		}
	}
	for k, v := range map[string]types.Bool{"worker": m.Worker, "edge_cache": m.EdgeCache, "public": m.Public} {
		if known(v) {
			b[k] = v.ValueBool()
		}
	}
	if known(m.Env) {
		env := map[string]string{}
		diags.Append(m.Env.ElementsAs(ctx, &env, false)...)
		b["env"] = env
	}
	if known(m.PublicPorts) {
		ports := []string{}
		diags.Append(m.PublicPorts.ElementsAs(ctx, &ports, false)...)
		b["public_ports"] = ports
	}
	if a := m.Autoscaling; a != nil {
		as := map[string]any{"min": a.Min.ValueInt64(), "max": a.Max.ValueInt64()}
		if known(a.CPUPercent) {
			as["cpu_percent"] = a.CPUPercent.ValueInt64()
		}
		b["autoscaling"] = as
	} else {
		b["autoscaling"] = nil
	}
	return b
}

// fill copies the platform's answer into the model.
func (m *serviceModel) fill(s *client.Service, diags *diag.Diagnostics) {
	m.Project, m.ID = types.StringValue(s.Project), types.StringValue(s.ID)
	m.Image = types.StringValue(s.Image)
	m.Port, m.CPUMillis, m.MemoryMB = types.Int64Value(int64(s.Port)), types.Int64Value(int64(s.CPUMillis)), types.Int64Value(int64(s.MemoryMB))
	m.HealthPath, m.VolumePath, m.MetricsPath = types.StringValue(s.HealthPath), types.StringValue(s.VolumePath), types.StringValue(s.MetricsPath)
	m.MetricsPort = types.Int64Value(int64(s.MetricsPort))
	m.Worker, m.EdgeCache, m.Public = types.BoolValue(s.Worker), types.BoolValue(s.EdgeCache), types.BoolValue(s.Public)
	if s.Autoscaling != nil {
		m.Autoscaling = &autoscalingModel{types.Int64Value(int64(s.Autoscaling.Min)), types.Int64Value(int64(s.Autoscaling.Max)), types.Int64Value(int64(s.Autoscaling.CPUPercent))}
		if !known(m.Instances) { // the autoscaler moves instances: keep what was planned
			m.Instances = types.Int64Value(int64(s.Instances))
		}
	} else {
		m.Autoscaling = nil
		m.Instances = types.Int64Value(int64(s.Instances))
	}
	env := map[string]attr.Value{}
	for k, v := range s.Env {
		env[k] = types.StringValue(v)
	}
	var d diag.Diagnostics
	m.Env, d = types.MapValue(types.StringType, env)
	diags.Append(d...)
	sort.Strings(s.PublicPorts)
	ports := []attr.Value{}
	for _, p := range s.PublicPorts {
		ports = append(ports, types.StringValue(p))
	}
	m.PublicPorts, d = types.SetValue(types.StringType, ports)
	diags.Append(d...)
	m.URL, m.InternalHost = types.StringValue(s.URL), types.StringValue(s.InternalHost)
	m.Status, m.Revision = types.StringValue(s.Status), types.Int64Value(int64(s.Revision))
	if m.Deploy.IsNull() || m.Deploy.IsUnknown() {
		m.Deploy = types.BoolValue(true)
	}
	if m.Wait.IsNull() || m.Wait.IsUnknown() {
		m.Wait = types.BoolValue(true)
	}
}

func servicePath(project, id string) string {
	return "/projects/" + client.P(project) + "/services/" + client.P(id)
}

// waitLive polls a deployment until it is live (or fails, or the context ends).
func (r *serviceResource) waitLive(ctx context.Context, project, id string, rev int) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()
	for {
		var d client.Deployment
		if err := r.c.Do(ctx, "GET", servicePath(project, id)+fmt.Sprintf("/deployments/%d", rev), nil, &d); err != nil {
			return err
		}
		switch d.Status {
		case "live":
			return nil
		case "failed":
			return fmt.Errorf("deployment %d of %s/%s failed: %s", rev, project, id, d.Message)
		case "superseded": // a newer deployment (the autoscaler, someone else) carries on: follow it
			var s client.Service
			if err := r.c.Do(ctx, "GET", servicePath(project, id), nil, &s); err != nil {
				return err
			}
			if s.Revision <= rev {
				return fmt.Errorf("deployment %d of %s/%s was superseded", rev, project, id)
			}
			rev = s.Revision
			continue
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("deployment %d of %s/%s not live after 15 minutes (status %s)", rev, project, id, d.Status)
		case <-time.After(3 * time.Second):
		}
	}
}

func (r *serviceResource) save(ctx context.Context, method, p string, m *serviceModel, diags *diag.Diagnostics, prevRev int64) {
	body := m.body(ctx, diags)
	if diags.HasError() {
		return
	}
	if method == "POST" {
		body["id"] = m.ID.ValueString()
	}
	var s client.Service
	if err := r.c.Do(ctx, method, p, body, &s); err != nil {
		apiDiag(diags, "saving the application", err)
		return
	}
	if m.Deploy.ValueBool() && m.Wait.ValueBool() && int64(s.Revision) > prevRev {
		if err := r.waitLive(ctx, s.Project, s.ID, s.Revision); err != nil {
			diags.AddError("Staysa Cloud: deployment failed", err.Error())
		}
		if err := r.c.Do(ctx, "GET", servicePath(s.Project, s.ID), nil, &s); err != nil {
			apiDiag(diags, "reading the application", err)
			return
		}
	}
	m.fill(&s, diags)
}

func (r *serviceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var m serviceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.save(ctx, "POST", "/projects/"+client.P(m.Project.ValueString())+"/services", &m, &resp.Diagnostics, 0)
	if m.Revision.IsUnknown() { // the creation itself failed
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *serviceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var m serviceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var s client.Service
	err := r.c.Do(ctx, "GET", servicePath(m.Project.ValueString(), m.ID.ValueString()), nil, &s)
	if notFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		apiDiag(&resp.Diagnostics, "reading the application", err)
		return
	}
	m.fill(&s, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *serviceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var m, st serviceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &st)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.save(ctx, "PATCH", servicePath(m.Project.ValueString(), m.ID.ValueString()), &m, &resp.Diagnostics, st.Revision.ValueInt64())
	if m.Revision.IsUnknown() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *serviceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var m serviceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.c.Do(ctx, "DELETE", servicePath(m.Project.ValueString(), m.ID.ValueString()), nil, nil); err != nil && !notFound(err) {
		apiDiag(&resp.Diagnostics, "deleting the application", err)
	}
}

func (r *serviceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := splitID(req.ID, 2, &resp.Diagnostics, "project/application")
	if parts == nil {
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("deploy"), true)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("wait_for_live"), true)...)
}

// ---------- data source ----------

type serviceDataSource struct{ base }

func newServiceDataSource() datasource.DataSource { return &serviceDataSource{} }

type serviceDataModel struct {
	Project      types.String `tfsdk:"project"`
	ID           types.String `tfsdk:"id"`
	Image        types.String `tfsdk:"image"`
	URL          types.String `tfsdk:"url"`
	InternalHost types.String `tfsdk:"internal_host"`
	Port         types.Int64  `tfsdk:"port"`
	Instances    types.Int64  `tfsdk:"instances"`
	Status       types.String `tfsdk:"status"`
}

func (d *serviceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service"
}

func (d *serviceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dschema.Schema{
		Description: "An existing application.",
		Attributes: map[string]dschema.Attribute{
			"project":       dschema.StringAttribute{Required: true},
			"id":            dschema.StringAttribute{Required: true},
			"image":         dschema.StringAttribute{Computed: true},
			"url":           dschema.StringAttribute{Computed: true},
			"internal_host": dschema.StringAttribute{Computed: true},
			"port":          dschema.Int64Attribute{Computed: true},
			"instances":     dschema.Int64Attribute{Computed: true},
			"status":        dschema.StringAttribute{Computed: true},
		},
	}
}

func (d *serviceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.configure(req.ProviderData, &resp.Diagnostics)
}

func (d *serviceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var m serviceDataModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var s client.Service
	if err := d.c.Do(ctx, "GET", servicePath(m.Project.ValueString(), m.ID.ValueString()), nil, &s); err != nil {
		apiDiag(&resp.Diagnostics, "reading the application", err)
		return
	}
	m.Image, m.URL, m.InternalHost = types.StringValue(s.Image), types.StringValue(s.URL), types.StringValue(s.InternalHost)
	m.Port, m.Instances, m.Status = types.Int64Value(int64(s.Port)), types.Int64Value(int64(s.Instances)), types.StringValue(s.Status)
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}
