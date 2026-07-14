// Package api implements an HTTP API server for the GGID Operator.
// It exposes CRUD endpoints for GGIDInstance and GGIDTenant custom resources,
// plus an environment detection endpoint that returns smart defaults.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	iamv1alpha1 "github.com/ggid/ggid/deploy/operator/api/v1alpha1"
)

// APIServer exposes HTTP endpoints for managing GGID CRs.
type APIServer struct {
	client     client.Client
	scheme     *runtime.Scheme
	gatewayURL string
	httpServer *http.Server
}

// NewAPIServer creates a new API server.
func NewAPIServer(k8sClient client.Client, scheme *runtime.Scheme, gatewayURL string) *APIServer {
	return &APIServer{
		client:     k8sClient,
		scheme:     scheme,
		gatewayURL: gatewayURL,
	}
}

// Start begins serving HTTP on the given address.
func (s *APIServer) Start(addr string) error {
	mux := http.NewServeMux()
	s.registerRoutes(mux)

	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	fmt.Printf("operator API server listening on %s\n", addr)
	return s.httpServer.ListenAndServe()
}

// Stop gracefully shuts down the API server.
func (s *APIServer) Stop(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}
	return s.httpServer.Shutdown(ctx)
}

func (s *APIServer) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.HandleFunc("/api/v1/provisioning/environment", s.handleEnvironment)
	mux.HandleFunc("/api/v1/provisioning/instances", s.handleInstances)
	mux.HandleFunc("/api/v1/provisioning/instances/", s.handleInstance)
	mux.HandleFunc("/api/v1/provisioning/tenants", s.handleTenants)
	mux.HandleFunc("/api/v1/provisioning/tenants/", s.handleTenant)
}

func (s *APIServer) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ===== Environment Detection =====

// EnvironmentInfo describes the detected K8s environment and smart defaults.
type EnvironmentInfo struct {
	// Kubernetes version
	KubernetesVersion string `json:"kubernetesVersion"`

	// OperatorNamespace is the namespace where the operator runs.
	OperatorNamespace string `json:"operatorNamespace"`

	// AvailableNamespaces is a list of namespaces the operator can see.
	AvailableNamespaces []string `json:"availableNamespaces"`

	// GatewayURL is the configured GGID gateway URL.
	GatewayURL string `json:"gatewayURL"`

	// DatabaseDrivers lists supported database drivers.
	DatabaseDrivers []string `json:"databaseDrivers"`

	// DefaultDatabase is the recommended default database config.
	DefaultDatabase DefaultDBConfig `json:"defaultDatabase"`

	// DefaultReplicas is the recommended replica count.
	DefaultReplicas int32 `json:"defaultReplicas"`

	// DefaultPlan is the recommended tenant plan.
	DefaultPlan string `json:"defaultPlan"`

	// DefaultSSLMode is the recommended PostgreSQL SSL mode.
	DefaultSSLMode string `json:"defaultSslMode"`

	// ExistingInstances is the count of existing GGIDInstance CRs.
	ExistingInstances int `json:"existingInstances"`

	// ExistingTenants is the count of existing GGIDTenant CRs.
	ExistingTenants int `json:"existingTenants"`

	// IdPProviders lists supported external IdP types.
	IdPProviders []string `json:"idpProviders"`
}

// DefaultDBConfig holds recommended database defaults.
type DefaultDBConfig struct {
	Driver string `json:"driver"`
	Host   string `json:"host"`
	Port   int32  `json:"port"`
	Name   string `json:"name"`
}

func (s *APIServer) handleEnvironment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// List namespaces
	nsList := &metav1.PartialObjectMetadataList{}
	nsList.SetGroupVersionKind(metav1.SchemeGroupVersion.WithKind("Namespace"))
	namespaces := []string{}
	if err := s.client.List(ctx, nsList); err == nil {
		for _, ns := range nsList.Items {
			// Filter out system namespaces
			if !strings.HasPrefix(ns.Name, "kube-") && ns.Name != "default" && ns.Name != "ggid-system" {
				namespaces = append(namespaces, ns.Name)
			}
		}
	}
	// Always include "default" and "ggid-system" as options
	namespaces = append([]string{"default", "ggid-system"}, namespaces...)

	// Count existing CRs
	instanceList := &iamv1alpha1.GGIDInstanceList{}
	existingInstances := 0
	if err := s.client.List(ctx, instanceList); err == nil {
		existingInstances = len(instanceList.Items)
	}

	tenantList := &iamv1alpha1.GGIDTenantList{}
	existingTenants := 0
	if err := s.client.List(ctx, tenantList); err == nil {
		existingTenants = len(tenantList.Items)
	}

	// Determine default DB host
	defaultDBHost := "ggid-postgresql"
	if s.gatewayURL != "" {
		// If gateway URL is set, infer we're in K8s and use the service name
		defaultDBHost = "ggid-postgresql.ggid-system.svc.cluster.local"
	}

	info := EnvironmentInfo{
		KubernetesVersion:   "1.28+", // controller-runtime client doesn't expose discovery API
		OperatorNamespace:   "ggid-system",
		AvailableNamespaces: namespaces,
		GatewayURL:          s.gatewayURL,
		DatabaseDrivers:     []string{"postgres", "mysql", "sqlite"},
		DefaultDatabase: DefaultDBConfig{
			Driver: "postgres",
			Host:   defaultDBHost,
			Port:   5432,
			Name:   "ggid",
		},
		DefaultReplicas:  2,
		DefaultPlan:      "starter",
		DefaultSSLMode:   "require",
		ExistingInstances: existingInstances,
		ExistingTenants:   existingTenants,
		IdPProviders:      []string{"saml", "oidc", "ldap"},
	}

	writeJSON(w, http.StatusOK, info)
}

// ===== GGIDInstance CRUD =====

// CreateInstanceRequest is the payload for creating a dedicated instance.
type CreateInstanceRequest struct {
	Name             string      `json:"name"`
	OrganizationName string      `json:"organizationName"`
	Namespace        string      `json:"namespace,omitempty"`
	Replicas         int32       `json:"replicas,omitempty"`
	Database         DBConfig    `json:"database"`
	AdminEmail       string      `json:"adminEmail"`
	IdPConfig        *IdPConfig  `json:"idpConfig,omitempty"`
	HelmChartRef     string      `json:"helmChartRef,omitempty"`
}

// DBConfig is the database configuration for an instance.
type DBConfig struct {
	Driver  string `json:"driver"`
	Host    string `json:"host"`
	Port    int32  `json:"port,omitempty"`
	Name    string `json:"name,omitempty"`
	SSLMode string `json:"sslMode,omitempty"`
}

// IdPConfig is the external IdP configuration.
type IdPConfig struct {
	Provider    string `json:"provider"`
	EntityID    string `json:"entityId"`
	SSOURL      string `json:"ssoUrl"`
	Certificate string `json:"certificate,omitempty"`
}

// InstanceInfo is the response for a single instance.
type InstanceInfo struct {
	Name             string    `json:"name"`
	OrganizationName string    `json:"organizationName"`
	Namespace        string    `json:"namespace"`
	Replicas         int32     `json:"replicas"`
	Database         DBConfig  `json:"database"`
	AdminEmail       string    `json:"adminEmail"`
	Phase            string    `json:"phase"`
	TenantID         string    `json:"tenantId"`
	HelmRelease      string    `json:"helmRelease"`
	CreatedAt        string    `json:"createdAt,omitempty"`
}

func (s *APIServer) handleInstances(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listInstances(w, r)
	case http.MethodPost:
		s.createInstance(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (s *APIServer) handleInstance(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/api/v1/provisioning/instances/")
	if name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "instance name required"})
		return
	}

	switch r.Method {
	case http.MethodDelete:
		s.deleteInstance(w, r, name)
	case http.MethodGet:
		s.getInstance(w, r, name)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (s *APIServer) listInstances(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	list := &iamv1alpha1.GGIDInstanceList{}
	if err := s.client.List(ctx, list); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list instances: " + err.Error()})
		return
	}

	items := make([]InstanceInfo, 0, len(list.Items))
	for _, inst := range list.Items {
		items = append(items, instanceToInfo(&inst))
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"instances": items,
		"total":     len(items),
	})
}

func (s *APIServer) getInstance(w http.ResponseWriter, r *http.Request, name string) {
	ctx := r.Context()
	inst := &iamv1alpha1.GGIDInstance{}
	if err := s.client.Get(ctx, types.NamespacedName{Name: name}, inst); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "instance not found: " + err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, instanceToInfo(inst))
}

func (s *APIServer) createInstance(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateInstanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body: " + err.Error()})
		return
	}

	if req.Name == "" || req.OrganizationName == "" || req.AdminEmail == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name, organizationName, and adminEmail are required"})
		return
	}

	// Apply defaults
	if req.Replicas == 0 {
		req.Replicas = 2
	}
	if req.Database.Driver == "" {
		req.Database.Driver = "postgres"
	}
	if req.Database.Port == 0 {
		req.Database.Port = 5432
	}
	if req.Database.Name == "" {
		req.Database.Name = "ggid"
	}
	if req.Database.SSLMode == "" {
		req.Database.SSLMode = "require"
	}

	namespace := req.Namespace
	if namespace == "" {
		namespace = "ggid-" + req.Name
	}

	instance := &iamv1alpha1.GGIDInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:       req.Name,
			Labels:     map[string]string{"app.kubernetes.io/managed-by": "ggid-operator"},
		},
		Spec: iamv1alpha1.GGIDInstanceSpec{
			Tier:             "dedicated",
			OrganizationName: req.OrganizationName,
			Namespace:        namespace,
			Replicas:         req.Replicas,
			Database: iamv1alpha1.DatabaseConfig{
				Driver:  req.Database.Driver,
				Host:    req.Database.Host,
				Port:    req.Database.Port,
				Name:    req.Database.Name,
				SSLMode: req.Database.SSLMode,
			},
			AdminEmail: req.AdminEmail,
		},
	}

	if req.IdPConfig != nil {
		instance.Spec.IdPConfig = &iamv1alpha1.IdPConfig{
			Provider:    req.IdPConfig.Provider,
			EntityID:    req.IdPConfig.EntityID,
			SSOURL:      req.IdPConfig.SSOURL,
			Certificate: req.IdPConfig.Certificate,
		}
	}

	if req.HelmChartRef != "" {
		instance.Spec.HelmChartRef = req.HelmChartRef
	}

	if err := s.client.Create(ctx, instance); err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "failed to create instance: " + err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, instanceToInfo(instance))
}

func (s *APIServer) deleteInstance(w http.ResponseWriter, r *http.Request, name string) {
	ctx := r.Context()
	inst := &iamv1alpha1.GGIDInstance{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}
	if err := s.client.Delete(ctx, inst); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "failed to delete instance: " + err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "name": name})
}

func instanceToInfo(inst *iamv1alpha1.GGIDInstance) InstanceInfo {
	return InstanceInfo{
		Name:             inst.Name,
		OrganizationName: inst.Spec.OrganizationName,
		Namespace:        inst.Spec.Namespace,
		Replicas:         inst.Spec.Replicas,
		Database: DBConfig{
			Driver:  inst.Spec.Database.Driver,
			Host:    inst.Spec.Database.Host,
			Port:    inst.Spec.Database.Port,
			Name:    inst.Spec.Database.Name,
			SSLMode: inst.Spec.Database.SSLMode,
		},
		AdminEmail:  inst.Spec.AdminEmail,
		Phase:       inst.Status.Phase,
		TenantID:    inst.Status.TenantID,
		HelmRelease: inst.Status.HelmRelease,
		CreatedAt:   inst.CreationTimestamp.Format("2006-01-02T15:04:05Z"),
	}
}

// ===== GGIDTenant CRUD =====

// CreateTenantRequest is the payload for creating a shared tenant.
type CreateTenantRequest struct {
	Name           string     `json:"name"`
	Slug           string     `json:"slug"`
	Plan           string     `json:"plan,omitempty"`
	MaxUsers       int32      `json:"maxUsers,omitempty"`
	AdminEmail     string     `json:"adminEmail"`
	AdminPassword  string     `json:"adminPassword,omitempty"`
	GGIDInstanceRef string    `json:"ggidInstanceRef,omitempty"`
	IdPConfig      *IdPConfig `json:"idpConfig,omitempty"`
}

// TenantInfo is the response for a single tenant.
type TenantInfo struct {
	Name            string `json:"name"`
	Slug            string `json:"slug"`
	Plan            string `json:"plan"`
	MaxUsers        int32  `json:"maxUsers"`
	AdminEmail      string `json:"adminEmail"`
	GGIDInstanceRef string `json:"ggidInstanceRef,omitempty"`
	Phase           string `json:"phase"`
	TenantID        string `json:"tenantId"`
	GatewayURL      string `json:"gatewayUrl"`
	CreatedAt       string `json:"createdAt,omitempty"`
}

func (s *APIServer) handleTenants(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listTenants(w, r)
	case http.MethodPost:
		s.createTenant(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (s *APIServer) handleTenant(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/api/v1/provisioning/tenants/")
	if name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "tenant name required"})
		return
	}

	switch r.Method {
	case http.MethodDelete:
		s.deleteTenant(w, r, name)
	case http.MethodGet:
		s.getTenant(w, r, name)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (s *APIServer) listTenants(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	list := &iamv1alpha1.GGIDTenantList{}
	if err := s.client.List(ctx, list); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list tenants: " + err.Error()})
		return
	}

	items := make([]TenantInfo, 0, len(list.Items))
	for _, t := range list.Items {
		items = append(items, tenantToInfo(&t))
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"tenants": items,
		"total":   len(items),
	})
}

func (s *APIServer) getTenant(w http.ResponseWriter, r *http.Request, name string) {
	ctx := r.Context()
	t := &iamv1alpha1.GGIDTenant{}
	if err := s.client.Get(ctx, types.NamespacedName{Name: name}, t); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "tenant not found: " + err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, tenantToInfo(t))
}

func (s *APIServer) createTenant(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body: " + err.Error()})
		return
	}

	if req.Name == "" || req.Slug == "" || req.AdminEmail == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name, slug, and adminEmail are required"})
		return
	}

	// Apply defaults
	if req.Plan == "" {
		req.Plan = "starter"
	}
	if req.MaxUsers == 0 {
		req.MaxUsers = 1000
	}

	tenant := &iamv1alpha1.GGIDTenant{
		ObjectMeta: metav1.ObjectMeta{
			Name:   req.Name,
			Labels: map[string]string{"app.kubernetes.io/managed-by": "ggid-operator"},
		},
		Spec: iamv1alpha1.GGIDTenantSpec{
			Tier:           "shared",
			TenantName:     req.Name,
			Slug:           req.Slug,
			Plan:           req.Plan,
			MaxUsers:       req.MaxUsers,
			AdminEmail:     req.AdminEmail,
			AdminPassword:  req.AdminPassword,
			GGIDInstanceRef: req.GGIDInstanceRef,
		},
	}

	if req.IdPConfig != nil {
		tenant.Spec.IdPConfig = &iamv1alpha1.IdPConfig{
			Provider:    req.IdPConfig.Provider,
			EntityID:    req.IdPConfig.EntityID,
			SSOURL:      req.IdPConfig.SSOURL,
			Certificate: req.IdPConfig.Certificate,
		}
	}

	if err := s.client.Create(ctx, tenant); err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "failed to create tenant: " + err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, tenantToInfo(tenant))
}

func (s *APIServer) deleteTenant(w http.ResponseWriter, r *http.Request, name string) {
	ctx := r.Context()
	tenant := &iamv1alpha1.GGIDTenant{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}
	if err := s.client.Delete(ctx, tenant); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "failed to delete tenant: " + err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "name": name})
}

func tenantToInfo(t *iamv1alpha1.GGIDTenant) TenantInfo {
	return TenantInfo{
		Name:            t.Spec.TenantName,
		Slug:            t.Spec.Slug,
		Plan:            t.Spec.Plan,
		MaxUsers:        t.Spec.MaxUsers,
		AdminEmail:      t.Spec.AdminEmail,
		GGIDInstanceRef: t.Spec.GGIDInstanceRef,
		Phase:           t.Status.Phase,
		TenantID:        t.Status.TenantID,
		GatewayURL:      t.Status.GatewayURL,
		CreatedAt:       t.CreationTimestamp.Format("2006-01-02T15:04:05Z"),
	}
}

// ===== Helpers =====

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
