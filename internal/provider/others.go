package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/gattoufirabi/terraform-provider-staysa/internal/client"
)

var replaceS = []planmodifier.String{stringplanmodifier.RequiresReplace()}
var keepS = []planmodifier.String{stringplanmodifier.UseStateForUnknown()}

// ==================== staysa_service_secret ====================

type serviceSecretResource struct{ base }

func newServiceSecretResource() resource.Resource { return &serviceSecretResource{} }

type serviceSecretModel struct {
	Project  types.String `tfsdk:"project"`
	Service  types.String `tfsdk:"service"`
	Name     types.String `tfsdk:"name"`
	Value    types.String `tfsdk:"value"`
	Redeploy types.Bool   `tfsdk:"redeploy"`
}

func (r *serviceSecretResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_secret"
}

func (r *serviceSecretResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "An encrypted secret given to an application as an environment variable. The platform never shows the value again, so changes made outside Terraform are not detected.",
		Attributes: map[string]schema.Attribute{
			"project": schema.StringAttribute{Required: true, PlanModifiers: replaceS},
			"service": schema.StringAttribute{Required: true, PlanModifiers: replaceS, Description: "Application ID."},
			"name":    schema.StringAttribute{Required: true, PlanModifiers: replaceS, Description: "Variable name: uppercase letters, digits and underscores."},
			"value":   schema.StringAttribute{Required: true, Sensitive: true, Description: "Value (up to 8192 characters)."},
			"redeploy": schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true),
				Description: "Redeploy the application after a change so it gets the new value (default true)."},
		},
	}
}

func (r *serviceSecretResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.configure(req.ProviderData, &resp.Diagnostics)
}

func (r *serviceSecretResource) put(ctx context.Context, m *serviceSecretModel) error {
	sp := servicePath(m.Project.ValueString(), m.Service.ValueString())
	if err := r.c.Do(ctx, "PUT", sp+"/secrets/"+client.P(m.Name.ValueString()), map[string]string{"value": m.Value.ValueString()}, nil); err != nil {
		return err
	}
	if m.Redeploy.ValueBool() {
		var s client.Service
		if err := r.c.Do(ctx, "GET", sp, nil, &s); err != nil {
			return err
		}
		if s.Image != "" && s.Revision > 0 { // only an application that was deployed before
			return r.c.Do(ctx, "POST", sp+"/deploy", nil, nil)
		}
	}
	return nil
}

func (r *serviceSecretResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var m serviceSecretModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.put(ctx, &m); err != nil {
		apiDiag(&resp.Diagnostics, "saving the secret", err)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *serviceSecretResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var m serviceSecretModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var list []client.Secret
	err := r.c.Do(ctx, "GET", servicePath(m.Project.ValueString(), m.Service.ValueString())+"/secrets", nil, &list)
	if notFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		apiDiag(&resp.Diagnostics, "reading the secrets", err)
		return
	}
	for _, s := range list {
		if s.Name == m.Name.ValueString() {
			if m.Redeploy.IsNull() {
				m.Redeploy = types.BoolValue(true)
			}
			resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
			return
		}
	}
	resp.State.RemoveResource(ctx)
}

func (r *serviceSecretResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var m serviceSecretModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.put(ctx, &m); err != nil {
		apiDiag(&resp.Diagnostics, "saving the secret", err)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *serviceSecretResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var m serviceSecretModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	p := servicePath(m.Project.ValueString(), m.Service.ValueString()) + "/secrets/" + client.P(m.Name.ValueString())
	if err := r.c.Do(ctx, "DELETE", p, nil, nil); err != nil && !notFound(err) {
		apiDiag(&resp.Diagnostics, "deleting the secret", err)
	}
}

// ==================== staysa_project_variable ====================

type projectVariableResource struct{ base }

func newProjectVariableResource() resource.Resource { return &projectVariableResource{} }

type projectVariableModel struct {
	Project types.String `tfsdk:"project"`
	Name    types.String `tfsdk:"name"`
	Value   types.String `tfsdk:"value"`
	Secret  types.Bool   `tfsdk:"secret"`
}

func (r *projectVariableResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_variable"
}

func (r *projectVariableResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A variable shared by every application of a project (applied at their next deployment).",
		Attributes: map[string]schema.Attribute{
			"project": schema.StringAttribute{Required: true, PlanModifiers: replaceS},
			"name":    schema.StringAttribute{Required: true, PlanModifiers: replaceS, Description: "Uppercase letters, digits and underscores."},
			"value":   schema.StringAttribute{Required: true, Sensitive: true},
			"secret": schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true),
				Description: "Encrypted and never shown again (default true). Plain variables can be read back, so outside changes are detected."},
		},
	}
}

func (r *projectVariableResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.configure(req.ProviderData, &resp.Diagnostics)
}

func (r *projectVariableResource) put(ctx context.Context, m *projectVariableModel) error {
	return r.c.Do(ctx, "PUT", "/projects/"+client.P(m.Project.ValueString())+"/variables/"+client.P(m.Name.ValueString()),
		map[string]any{"value": m.Value.ValueString(), "secret": m.Secret.ValueBool()}, nil)
}

func (r *projectVariableResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var m projectVariableModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.put(ctx, &m); err != nil {
		apiDiag(&resp.Diagnostics, "saving the variable", err)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *projectVariableResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var m projectVariableModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var list []client.ProjectVariable
	err := r.c.Do(ctx, "GET", "/projects/"+client.P(m.Project.ValueString())+"/variables", nil, &list)
	if notFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		apiDiag(&resp.Diagnostics, "reading the variables", err)
		return
	}
	for _, v := range list {
		if v.Name == m.Name.ValueString() {
			m.Secret = types.BoolValue(v.Secret)
			if !v.Secret {
				m.Value = types.StringValue(v.Value)
			}
			resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
			return
		}
	}
	resp.State.RemoveResource(ctx)
}

func (r *projectVariableResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var m projectVariableModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.put(ctx, &m); err != nil {
		apiDiag(&resp.Diagnostics, "saving the variable", err)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *projectVariableResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var m projectVariableModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	p := "/projects/" + client.P(m.Project.ValueString()) + "/variables/" + client.P(m.Name.ValueString())
	if err := r.c.Do(ctx, "DELETE", p, nil, nil); err != nil && !notFound(err) {
		apiDiag(&resp.Diagnostics, "deleting the variable", err)
	}
}

func (r *projectVariableResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := splitID(req.ID, 2, &resp.Diagnostics, "project/NAME")
	if parts == nil {
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), parts[1])...)
}

// ==================== staysa_domain ====================

type domainResource struct{ base }

func newDomainResource() resource.Resource { return &domainResource{} }

type domainModel struct {
	Project    types.String `tfsdk:"project"`
	Service    types.String `tfsdk:"service"`
	Hostname   types.String `tfsdk:"hostname"`
	Verified   types.Bool   `tfsdk:"verified"`
	DNSRecords types.List   `tfsdk:"dns_records"`
}

var dnsRecordType = types.ObjectType{AttrTypes: map[string]attr.Type{"type": types.StringType, "name": types.StringType, "value": types.StringType}}

func (r *domainResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain"
}

func (r *domainResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A custom domain for a web application. In a zone hosted on the platform its DNS records are written and it is verified at once; elsewhere, create dns_records at your DNS provider, then verify it in the console.",
		Attributes: map[string]schema.Attribute{
			"project":  schema.StringAttribute{Required: true, PlanModifiers: replaceS},
			"service":  schema.StringAttribute{Required: true, PlanModifiers: replaceS, Description: "Application ID."},
			"hostname": schema.StringAttribute{Required: true, PlanModifiers: replaceS, Description: "Domain name, such as shop.example.com."},
			"verified": schema.BoolAttribute{Computed: true, Description: "Ownership verified; HTTPS is then served."},
			"dns_records": schema.ListAttribute{Computed: true, ElementType: dnsRecordType,
				PlanModifiers: []planmodifier.List{listplanmodifier.UseStateForUnknown()},
				Description:   "Records to create at your DNS provider: the CNAME to the application and the TXT ownership check."},
		},
	}
}

func (r *domainResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.configure(req.ProviderData, &resp.Diagnostics)
}

func (m *domainModel) fill(d *client.Domain) {
	m.Hostname, m.Verified = types.StringValue(d.Hostname), types.BoolValue(d.Verified)
	recs := []attr.Value{}
	for _, x := range d.DNSRecords {
		recs = append(recs, types.ObjectValueMust(dnsRecordType.AttrTypes, map[string]attr.Value{
			"type": types.StringValue(x.Type), "name": types.StringValue(x.Name), "value": types.StringValue(x.Value)}))
	}
	m.DNSRecords = types.ListValueMust(dnsRecordType, recs)
}

func (m *domainModel) path() string {
	return servicePath(m.Project.ValueString(), m.Service.ValueString()) + "/domains"
}

func (r *domainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var m domainModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var d client.Domain
	if err := r.c.Do(ctx, "POST", m.path(), map[string]string{"hostname": strings.ToLower(m.Hostname.ValueString())}, &d); err != nil {
		apiDiag(&resp.Diagnostics, "adding the domain", err)
		return
	}
	m.fill(&d)
	m.Hostname = types.StringValue(m.Hostname.ValueString()) // as written
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *domainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var m domainModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var d client.Domain
	written := m.Hostname.ValueString()
	err := r.c.Do(ctx, "GET", m.path()+"/"+client.P(strings.ToLower(written)), nil, &d)
	if notFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		apiDiag(&resp.Diagnostics, "reading the domain", err)
		return
	}
	m.fill(&d)
	m.Hostname = types.StringValue(written)
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *domainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Unexpected update", "every attribute of staysa_domain replaces it")
}

func (r *domainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var m domainModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.c.Do(ctx, "DELETE", m.path()+"/"+client.P(strings.ToLower(m.Hostname.ValueString())), nil, nil); err != nil && !notFound(err) {
		apiDiag(&resp.Diagnostics, "removing the domain", err)
	}
}

func (r *domainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := splitID(req.ID, 3, &resp.Diagnostics, "project/application/hostname")
	if parts == nil {
		return
	}
	for i, k := range []string{"project", "service", "hostname"} {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(k), parts[i])...)
	}
}

// ==================== staysa_dns_zone ====================

type dnsZoneResource struct{ base }

func newDNSZoneResource() resource.Resource { return &dnsZoneResource{} }

type dnsZoneModel struct {
	Domain      types.String `tfsdk:"domain"`
	Status      types.String `tfsdk:"status"`
	Nameservers types.List   `tfsdk:"nameservers"`
}

func (r *dnsZoneResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_zone"
}

func (r *dnsZoneResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A DNS zone hosted on the platform. Set the nameservers at your registrar; the zone is served once the delegation is seen. Deleting it needs an admin token.",
		Attributes: map[string]schema.Attribute{
			"domain": schema.StringAttribute{Required: true, PlanModifiers: replaceS, Description: "Domain, such as example.com."},
			"status": schema.StringAttribute{Computed: true, Description: "pending until the registrar delegates the domain, then active."},
			"nameservers": schema.ListAttribute{Computed: true, ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{listplanmodifier.UseStateForUnknown()},
				Description:   "Name servers to set at your registrar."},
		},
	}
}

func (r *dnsZoneResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.configure(req.ProviderData, &resp.Diagnostics)
}

// read fills the model from the zone list; false if the zone is gone.
func (r *dnsZoneResource) read(ctx context.Context, m *dnsZoneModel) (bool, error) {
	var zl client.ZoneList
	if err := r.c.Do(ctx, "GET", "/dns/zones", nil, &zl); err != nil {
		return false, err
	}
	ns := []attr.Value{}
	for _, n := range zl.Nameservers {
		ns = append(ns, types.StringValue(n))
	}
	m.Nameservers = types.ListValueMust(types.StringType, ns)
	for _, z := range zl.Zones {
		if strings.EqualFold(z.Domain, strings.TrimSuffix(m.Domain.ValueString(), ".")) {
			m.Status = types.StringValue(z.Status)
			return true, nil
		}
	}
	return false, nil
}

func (r *dnsZoneResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var m dnsZoneModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.c.Do(ctx, "POST", "/dns/zones", map[string]string{"domain": m.Domain.ValueString()}, nil); err != nil {
		apiDiag(&resp.Diagnostics, "adding the zone", err)
		return
	}
	if found, err := r.read(ctx, &m); err != nil || !found {
		if err == nil {
			err = client.ErrNotFound
		}
		apiDiag(&resp.Diagnostics, "reading the new zone", err)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *dnsZoneResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var m dnsZoneModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	found, err := r.read(ctx, &m)
	if err != nil {
		apiDiag(&resp.Diagnostics, "reading the zone", err)
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *dnsZoneResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Unexpected update", "changing the domain replaces the zone")
}

func (r *dnsZoneResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var m dnsZoneModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.c.Do(ctx, "DELETE", "/dns/zones/"+client.P(m.Domain.ValueString()), nil, nil); err != nil && !notFound(err) {
		apiDiag(&resp.Diagnostics, "deleting the zone", err)
	}
}

func (r *dnsZoneResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("domain"), req, resp)
}

// ==================== staysa_dns_record ====================

type dnsRecordResource struct{ base }

func newDNSRecordResource() resource.Resource { return &dnsRecordResource{} }

type dnsRecordModel struct {
	ID       types.String `tfsdk:"id"`
	Zone     types.String `tfsdk:"zone"`
	Name     types.String `tfsdk:"name"`
	Type     types.String `tfsdk:"type"`
	Value    types.String `tfsdk:"value"`
	Priority types.Int64  `tfsdk:"priority"`
	TTL      types.Int64  `tfsdk:"ttl"`
}

func (r *dnsRecordResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_record"
}

func (r *dnsRecordResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	replaceI := []planmodifier.Int64{int64planmodifier.RequiresReplace(), int64planmodifier.UseStateForUnknown()}
	resp.Schema = schema.Schema{
		Description: "A record in a hosted DNS zone. Any change replaces the record.",
		Attributes: map[string]schema.Attribute{
			"id":       schema.StringAttribute{Computed: true, PlanModifiers: keepS},
			"zone":     schema.StringAttribute{Required: true, PlanModifiers: replaceS, Description: "Zone domain."},
			"name":     schema.StringAttribute{Required: true, PlanModifiers: replaceS, Description: "Name inside the zone (www), or @ for the zone itself."},
			"type":     schema.StringAttribute{Required: true, PlanModifiers: replaceS, Description: "A, AAAA, CNAME, MX, TXT, CAA, SRV or NS."},
			"value":    schema.StringAttribute{Required: true, PlanModifiers: replaceS},
			"priority": schema.Int64Attribute{Optional: true, Computed: true, PlanModifiers: replaceI, Description: "MX and SRV priority."},
			"ttl":      schema.Int64Attribute{Optional: true, Computed: true, PlanModifiers: replaceI, Description: "Time to live in seconds."},
		},
	}
}

func (r *dnsRecordResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.configure(req.ProviderData, &resp.Diagnostics)
}

func (m *dnsRecordModel) fill(x *client.ZoneRecord) {
	m.ID = types.StringValue(x.ID)
	m.Priority, m.TTL = types.Int64Value(int64(x.Priority)), types.Int64Value(int64(x.TTL))
	// the platform stores names and values in a normal form; keep what was written when it means the same
	if !strings.EqualFold(strings.TrimSuffix(m.Name.ValueString(), "."), x.Name) {
		m.Name = types.StringValue(x.Name)
	}
	if !strings.EqualFold(strings.Trim(m.Value.ValueString(), `". `), strings.Trim(x.Value, `". `)) {
		m.Value = types.StringValue(x.Value)
	}
	if !strings.EqualFold(m.Type.ValueString(), x.Type) {
		m.Type = types.StringValue(x.Type)
	}
}

func (r *dnsRecordResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var m dnsRecordModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body := map[string]any{"name": m.Name.ValueString(), "type": strings.ToUpper(m.Type.ValueString()), "value": m.Value.ValueString()}
	if known(m.Priority) {
		body["priority"] = m.Priority.ValueInt64()
	}
	if known(m.TTL) {
		body["ttl"] = m.TTL.ValueInt64()
	}
	var x client.ZoneRecord
	if err := r.c.Do(ctx, "POST", "/dns/zones/"+client.P(m.Zone.ValueString())+"/records", body, &x); err != nil {
		apiDiag(&resp.Diagnostics, "adding the record", err)
		return
	}
	m.fill(&x)
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *dnsRecordResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var m dnsRecordModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var zr client.ZoneRecords
	err := r.c.Do(ctx, "GET", "/dns/zones/"+client.P(m.Zone.ValueString())+"/records", nil, &zr)
	if notFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		apiDiag(&resp.Diagnostics, "reading the records", err)
		return
	}
	for i := range zr.Records {
		if zr.Records[i].ID == m.ID.ValueString() {
			if m.Name.IsNull() { // after an import
				m.Name, m.Type, m.Value = types.StringValue(zr.Records[i].Name), types.StringValue(zr.Records[i].Type), types.StringValue(zr.Records[i].Value)
			}
			m.fill(&zr.Records[i])
			resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
			return
		}
	}
	resp.State.RemoveResource(ctx)
}

func (r *dnsRecordResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Unexpected update", "every attribute of staysa_dns_record replaces it")
}

func (r *dnsRecordResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var m dnsRecordModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	p := "/dns/zones/" + client.P(m.Zone.ValueString()) + "/records/" + client.P(m.ID.ValueString())
	if err := r.c.Do(ctx, "DELETE", p, nil, nil); err != nil && !notFound(err) {
		apiDiag(&resp.Diagnostics, "deleting the record", err)
	}
}

func (r *dnsRecordResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := splitID(req.ID, 2, &resp.Diagnostics, "zone/record-id")
	if parts == nil {
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("zone"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

// ==================== staysa_bucket ====================

type bucketResource struct{ base }

func newBucketResource() resource.Resource { return &bucketResource{} }

type bucketModel struct {
	Name     types.String `tfsdk:"name"`
	QuotaMB  types.Int64  `tfsdk:"quota_mb"`
	S3Name   types.String `tfsdk:"s3_name"`
	Endpoint types.String `tfsdk:"endpoint"`
	Region   types.String `tfsdk:"region"`
}

func (r *bucketResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bucket"
}

func (r *bucketResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "An S3-compatible storage bucket of the organization. Deleting it needs an admin token and deletes its files.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{Required: true, PlanModifiers: replaceS, Description: "3 to 40 lowercase letters, digits and dashes."},
			"quota_mb": schema.Int64Attribute{Optional: true, Computed: true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
				Description:   "Space reserved, in MB (default 1024 or what the plan has left). Change it in the console."},
			"s3_name":  schema.StringAttribute{Computed: true, PlanModifiers: keepS, Description: "Bucket name in S3 requests."},
			"endpoint": schema.StringAttribute{Computed: true, PlanModifiers: keepS, Description: "S3 endpoint."},
			"region":   schema.StringAttribute{Computed: true, PlanModifiers: keepS, Description: "S3 region."},
		},
	}
}

func (r *bucketResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.configure(req.ProviderData, &resp.Diagnostics)
}

func (m *bucketModel) fill(b *client.Bucket) {
	m.Name, m.QuotaMB = types.StringValue(b.Name), types.Int64Value(int64(b.QuotaMB))
	m.S3Name, m.Endpoint, m.Region = types.StringValue(b.S3Name), types.StringValue(b.Endpoint), types.StringValue(b.Region)
}

func (r *bucketResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var m bucketModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body := map[string]any{"name": m.Name.ValueString()}
	if known(m.QuotaMB) {
		body["quota_mb"] = m.QuotaMB.ValueInt64()
	}
	var b client.Bucket
	if err := r.c.Do(ctx, "POST", "/buckets", body, &b); err != nil {
		apiDiag(&resp.Diagnostics, "creating the bucket", err)
		return
	}
	m.fill(&b)
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func (r *bucketResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var m bucketModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var list []client.Bucket
	if err := r.c.Do(ctx, "GET", "/buckets", nil, &list); err != nil {
		apiDiag(&resp.Diagnostics, "reading the buckets", err)
		return
	}
	for i := range list {
		if list[i].Name == m.Name.ValueString() {
			m.fill(&list[i])
			resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
			return
		}
	}
	resp.State.RemoveResource(ctx)
}

func (r *bucketResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Quota changes are made in the console",
		"The API does not change a bucket's quota yet. Change it in the console, then set quota_mb to the same value.")
}

func (r *bucketResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var m bucketModel
	resp.Diagnostics.Append(req.State.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.c.Do(ctx, "DELETE", "/buckets/"+client.P(m.Name.ValueString()), nil, nil); err != nil && !notFound(err) {
		apiDiag(&resp.Diagnostics, "deleting the bucket", err)
	}
}

func (r *bucketResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}
