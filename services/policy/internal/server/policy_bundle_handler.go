package httpserver

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type PolicyBundle struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	PolicyIDs []string  `json:"policy_ids"`
	Version   string    `json:"version"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *HTTPServer) handleBundles(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req struct {
			Name      string   `json:"name"`
			PolicyIDs []string `json:"policy_ids"`
			Version   string   `json:"version"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		b := &PolicyBundle{
			ID: uuid.New().String(), Name: req.Name, PolicyIDs: req.PolicyIDs,
			Version: req.Version, CreatedAt: time.Now().UTC(),
		}
		if s.policyMap != nil {
			s.policyMap.Store(r.Context(), "policy_bundles", b.ID, map[string]any{
				"name": b.Name, "policy_ids": b.PolicyIDs, "version": b.Version,
			})
		}
		writeJSON(w, http.StatusCreated, b)
	case http.MethodGet:
		var result []map[string]any
		if s.policyMap != nil {
			rows, _ := s.policyMap.List(r.Context(), "policy_bundles")
			result = rows
		}
		if result == nil { result = []map[string]any{} }
		writeJSON(w, http.StatusOK, map[string]any{"bundles": result, "count": len(result)})
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
