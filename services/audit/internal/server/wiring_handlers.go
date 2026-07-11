package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/ggid/ggid/services/audit/internal/service"
	"github.com/google/uuid"
)

// GET /api/v1/audit/risk-score?user_id=X&device=...&ip=...&country=...
func (s *HTTPServer) handleRiskScore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		writeJSONError(w, http.StatusBadRequest, "user_id required")
		return
	}
	engine := service.NewRiskEngine()
	score := engine.Evaluate(
		r.URL.Query().Get("user_id"),
		r.URL.Query().Get("device"),
		r.URL.Query().Get("ip"),
		r.URL.Query().Get("country"),
	)
	writeJSON(w, http.StatusOK, score)
}

// POST /api/v1/audit/access-reviews — create a pending access review
// GET /api/v1/audit/access-reviews — list pending reviews for a manager
func (s *HTTPServer) handleAccessReviews(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req struct {
			ManagerID string   `json:"manager_id"`
			UserID    string   `json:"user_id"`
			Roles     []string `json:"roles"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		review := service.CreateAccessReview(
			parseUUID(req.ManagerID), parseUUID(req.UserID), parseUUID(""), req.Roles,
		)
		writeJSON(w, http.StatusCreated, review)

	case http.MethodGet:
		managerID := r.URL.Query().Get("manager_id")
		pending := service.ListPendingAccessReviews(parseUUID(managerID))
		writeJSON(w, http.StatusOK, map[string]any{"reviews": pending})

	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// POST /api/v1/audit/access-reviews/pending — submit approve/revoke decision
func (s *HTTPServer) handlePendingReviews(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		ReviewID string `json:"review_id"`
		Decision string `json:"decision"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	result, err := service.SubmitAccessReview(parseUUID(req.ReviewID), req.Decision)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func parseUUID(s string) uuid.UUID {
	id, _ := uuid.Parse(s)
	return id
}
