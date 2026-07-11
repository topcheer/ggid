package httpserver

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

type PolicyBundle struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	PolicyIDs []string  `json:"policy_ids"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
}

var (
	bundleMu sync.RWMutex
	bundles  = make(map[string]*PolicyBundle)
)

// POST/GET /api/v1/policies/bundles
func (s *HTTPServer) handleBundles(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req struct {
			Name      string   `json:"name"`
			PolicyIDs []string `json:"policy_ids"`
			Enabled   bool     `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON"); return
		}
		if req.Name == "" || len(req.PolicyIDs) == 0 {
			writeJSONError(w, http.StatusBadRequest, "name and policy_ids required"); return
		}
		b := &PolicyBundle{ID: "pb-" + uuid.New().String()[:8], Name: req.Name, PolicyIDs: req.PolicyIDs, Enabled: req.Enabled, CreatedAt: time.Now().UTC()}
		bundleMu.Lock(); bundles[b.ID] = b; bundleMu.Unlock()
		writeJSON(w, http.StatusCreated, b)
	case http.MethodGet:
		bundleMu.RLock()
		result := make([]*PolicyBundle, 0, len(bundles))
		for _, b := range bundles { result = append(result, b) }
		bundleMu.RUnlock()
		writeJSON(w, http.StatusOK, map[string]any{"bundles": result, "count": len(result)})
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
