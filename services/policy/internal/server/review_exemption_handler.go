package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
)

type ReviewExemption struct {
	ID        string    `json:"id"`
	Role      string    `json:"role"`
	Reason    string    `json:"reason"`
	ExemptedBy string   `json:"exempted_by"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

var (
	exemptMu sync.RWMutex
	exemptions = make(map[string]*ReviewExemption)
)

// POST/GET/DELETE /api/v1/policies/access-review-exemptions
func (s *HTTPServer) handleReviewExemptions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req struct{ Role, Reason, ExemptedBy string; ExpiresAt string }
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeJSONError(w, http.StatusBadRequest, "invalid JSON"); return }
		if req.Role == "" { writeJSONError(w, http.StatusBadRequest, "role required"); return }
		exp, _ := time.Parse(time.RFC3339, req.ExpiresAt)
		if exp.IsZero() { exp = time.Now().UTC().Add(90 * 24 * time.Hour) }
		e := &ReviewExemption{ID: "exm-" + time.Now().Format("20060102") + "-" + req.Role, Role: req.Role, Reason: req.Reason, ExemptedBy: req.ExemptedBy, ExpiresAt: exp, CreatedAt: time.Now().UTC()}
		exemptMu.Lock(); exemptions[e.ID] = e; exemptMu.Unlock()
		writeJSON(w, http.StatusCreated, e)
	case http.MethodGet:
		exemptMu.RLock(); result := []*ReviewExemption{}
		for _, e := range exemptions { result = append(result, e) }
		exemptMu.RUnlock()
		writeJSON(w, http.StatusOK, map[string]any{"exemptions": result, "count": len(result)})
	case http.MethodDelete:
		id := strings.TrimPrefix(r.URL.Path, "/api/v1/policies/access-review-exemptions/")
		if id == "" || id == r.URL.Path { writeJSONError(w, http.StatusBadRequest, "id required"); return }
		exemptMu.Lock(); delete(exemptions, id); exemptMu.Unlock()
		writeJSON(w, http.StatusOK, map[string]any{"status": "deleted", "id": id})
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
