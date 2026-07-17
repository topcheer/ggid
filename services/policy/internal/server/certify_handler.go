package httpserver

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type Certification struct {
	ID         string    `json:"id"`
	ResourceID string    `json:"resource_id"`
	Status     string    `json:"status"`
	Reviewer   string    `json:"reviewer"`
	Decision   string    `json:"decision"`
	Comment    string    `json:"comment"`
	CreatedAt  time.Time `json:"created_at"`
}

func (s *HTTPServer) handleCertify(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req struct {
			ResourceID string `json:"resource_id"`
			Reviewer   string `json:"reviewer"`
			Decision   string `json:"decision"`
			Comment    string `json:"comment"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		cert := &Certification{
			ID: uuid.New().String(), ResourceID: req.ResourceID, Status: "certified",
			Reviewer: req.Reviewer, Decision: req.Decision, Comment: req.Comment,
			CreatedAt: time.Now().UTC(),
		}
		if s.policyMap != nil {
			s.policyMap.Store(r.Context(), "policy_certifications", cert.ID, map[string]any{
				"resource_id": cert.ResourceID, "status": cert.Status,
				"reviewer": cert.Reviewer, "decision": cert.Decision, "comment": cert.Comment,
			})
		}
		writeJSON(w, http.StatusCreated, cert)
	case http.MethodGet:
		var result []map[string]any
		if s.policyMap != nil {
			rows, _ := s.policyMap.List(r.Context(), "policy_certifications")
			result = rows
		}
		if result == nil { result = []map[string]any{} }
		writeJSON(w, http.StatusOK, map[string]any{"certifications": result, "count": len(result)})
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
