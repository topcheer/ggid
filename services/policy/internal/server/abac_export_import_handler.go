package httpserver

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ABACPolicySet represents an importable/exportable set of ABAC policies.
type ABACPolicySet struct {
	ID        string         `json:"id"`
	TenantID  string         `json:"tenant_id"`
	Name      string         `json:"name"`
	Version   string         `json:"version"`
	Policies  []ABACPolicyItem `json:"policies"`
	ExportedAt string        `json:"exported_at"`
}

type ABACPolicyItem struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Resource    string         `json:"resource"`
	Action      string         `json:"action"`
	Effect      string         `json:"effect"` // allow, deny
	Conditions  map[string]any `json:"conditions"`
	Priority    int            `json:"priority"`
	Enabled     bool           `json:"enabled"`
}

var (
	abacPolicyMu sync.RWMutex
	abacPolicies = make(map[string]*ABACPolicySet)
)

// GET /api/v1/policies/abac/export — export all ABAC policies as JSON.
// POST /api/v1/policies/abac/import — import ABAC policy set from JSON.
func (s *HTTPServer) handleABACExportImport(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		tenantID := r.URL.Query().Get("tenant_id")
		abacPolicyMu.RLock()
		result := []*ABACPolicySet{}
		for _, ps := range abacPolicies {
			if tenantID != "" && ps.TenantID != tenantID {
				continue
			}
			result = append(result, ps)
		}
		abacPolicyMu.RUnlock()
		writeJSON(w, http.StatusOK, map[string]any{
			"policy_sets": result,
			"count":       len(result),
			"format":      "json",
		})

	case http.MethodPost:
		var ps ABACPolicySet
		if err := json.NewDecoder(r.Body).Decode(&ps); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if ps.Name == "" {
			writeJSONError(w, http.StatusBadRequest, "name is required")
			return
		}
		ps.ID = uuid.New().String()
		if ps.Version == "" {
			ps.Version = "1.0"
		}
		ps.ExportedAt = time.Now().UTC().Format(time.RFC3339)
		abacPolicyMu.Lock()
		abacPolicies[ps.ID] = &ps
		abacPolicyMu.Unlock()
		writeJSON(w, http.StatusCreated, map[string]any{
			"status":          "imported",
			"policy_set_id":   ps.ID,
			"policies_count":  len(ps.Policies),
		})

	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
