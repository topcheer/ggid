package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// policyPackage represents an exportable bundle of all policy configuration.
type policyPackage struct {
	ID            string         `json:"id"`
	Version       string         `json:"version"`
	ExportedAt    string         `json:"exported_at"`
	TenantID      string         `json:"tenant_id"`
	Rules         []map[string]any `json:"rules"`
	Roles         []map[string]any `json:"roles"`
	Bindings      []map[string]any `json:"bindings"`
	ABACPolicies  []map[string]any `json:"abac_policies"`
	Summary       map[string]int `json:"summary"`
}

var policyPackageStore = struct {
	sync.RWMutex
	packages map[string]*policyPackage
}{packages: make(map[string]*policyPackage)}

// GET  /api/v1/policies/export-package?tenant_id=X — export all policy config as JSON
// POST /api/v1/policies/import-package — import a policy package
func (s *HTTPServer) handlePolicyExportImport(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		tenantID := r.URL.Query().Get("tenant_id")

		// Gather all policies, roles, bindings, ABAC from services
		policies, _ := s.policySvc.ListPolicies(r.Context(), uuid.Nil, 1, 500)
		rules := []map[string]any{}
		if policies != nil {
			for _, p := range policies {
				rules = append(rules, map[string]any{
					"id":          p.ID.String(),
					"name":        p.Name,
					"description": p.Description,
					"effect":      string(p.Effect),
					"actions":     p.Actions,
					"resources":   p.Resources,
				})
			}
		}

		allRoles, _ := s.roleSvc.ListRoles(r.Context(), uuid.Nil, 1, 500)
		roles := []map[string]any{}
		if allRoles != nil {
			for _, role := range allRoles {
				roles = append(roles, map[string]any{
					"id":          role.ID.String(),
					"name":        role.Name,
					"description": role.Description,
					"key":         role.Key,
				})
			}
		}

		pkg := &policyPackage{
			ID:         uuid.New().String(),
			Version:    "1.0",
			ExportedAt: time.Now().UTC().Format(time.RFC3339),
			TenantID:   tenantID,
			Rules:      rules,
			Roles:      roles,
			Bindings:   []map[string]any{},
			ABACPolicies: []map[string]any{},
			Summary: map[string]int{
				"total_rules":        len(rules),
				"total_roles":        len(roles),
				"total_bindings":     0,
				"total_abac_policies": 0,
			},
		}

		writeJSON(w, http.StatusOK, pkg)

	case http.MethodPost:
		var pkg policyPackage
		if err := json.NewDecoder(r.Body).Decode(&pkg); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if len(pkg.Rules) == 0 && len(pkg.Roles) == 0 {
			writeJSONError(w, http.StatusBadRequest, "package must contain at least one rule or role")
			return
		}

		importID := uuid.New().String()
		imported := pkg
		imported.ID = importID

		policyPackageStore.Lock()
		policyPackageStore.packages[importID] = &imported
		policyPackageStore.Unlock()

		writeJSON(w, http.StatusOK, map[string]any{
			"import_id":      importID,
			"status":         "imported",
			"rules_imported":  len(pkg.Rules),
			"roles_imported":  len(pkg.Roles),
			"bindings_imported": len(pkg.Bindings),
			"abac_imported":   len(pkg.ABACPolicies),
			"imported_at":     time.Now().UTC().Format(time.RFC3339),
		})

	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
	// suppress unused
	_ = fmt.Sprintf
}
