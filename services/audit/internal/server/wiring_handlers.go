package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

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

// accessReviewTenantID extracts the tenant ID from X-Tenant-ID header.
func accessReviewTenantID(r *http.Request) uuid.UUID {
	id, _ := uuid.Parse(r.Header.Get("X-Tenant-ID"))
	return id
}

// POST /api/v1/audit/access-reviews — create a pending access review
// GET  /api/v1/audit/access-reviews — list reviews (filtered by status, optionally manager_id)
func (s *HTTPServer) handleAccessReviews(w http.ResponseWriter, r *http.Request) {
	tenantID := accessReviewTenantID(r)
	repo := service.NewAccessReviewRepo(s.pool)

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
		if req.UserID == "" {
			writeJSONError(w, http.StatusBadRequest, "user_id required")
			return
		}
		// Default manager_id to the JWT subject if not provided.
		managerID := parseUUID(req.ManagerID)
		if managerID == uuid.Nil {
			managerID, _ = uuid.Parse(r.Header.Get("X-User-ID"))
		}
		review, err := repo.Create(r.Context(), tenantID, managerID, parseUUID(req.UserID), req.Roles)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		writeJSON(w, http.StatusCreated, review)

	case http.MethodGet:
		status := r.URL.Query().Get("status")
		managerID := parseUUID(r.URL.Query().Get("manager_id"))
		reviews, err := repo.ListPending(r.Context(), tenantID, managerID, status)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"reviews": reviews})

	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// POST /api/v1/audit/access-reviews/{id}/decision — submit approve/revoke decision
// Also handles legacy path /api/v1/audit/access-reviews/pending for backward compat.
func (s *HTTPServer) handleAccessReviewDecision(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	tenantID := accessReviewTenantID(r)

	var req struct {
		ReviewID string `json:"review_id"`
		Decision string `json:"decision"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.ReviewID == "" {
		writeJSONError(w, http.StatusBadRequest, "review_id required")
		return
	}

	// Also check path parameter: /api/v1/audit/access-reviews/{id}/decision
	reviewID := parseUUID(req.ReviewID)
	if pathID := extractReviewIDFromPath(r.URL.Path); pathID != uuid.Nil {
		reviewID = pathID
	}

	repo := service.NewAccessReviewRepo(s.pool)
	result, err := repo.SubmitDecision(r.Context(), tenantID, reviewID, req.Decision)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "bad request")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// extractReviewIDFromPath parses the review ID from a path like
// /api/v1/audit/access-reviews/{id}/decision.
func extractReviewIDFromPath(path string) uuid.UUID {
	// Expected pattern: .../access-reviews/{uuid}/decision
	parts := strings.Split(strings.TrimSuffix(path, "/"), "/")
	for i, p := range parts {
		if p == "access-reviews" && i+2 < len(parts) && parts[i+2] == "decision" {
			id, _ := uuid.Parse(parts[i+1])
			return id
		}
	}
	return uuid.Nil
}

// handlePendingReviews is kept for backward compatibility — delegates to decision handler.
func (s *HTTPServer) handlePendingReviews(w http.ResponseWriter, r *http.Request) {
	s.handleAccessReviewDecision(w, r)
}

func parseUUID(s string) uuid.UUID {
	id, _ := uuid.Parse(s)
	return id
}
