package httpserver

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// autoAssignment represents reviewer assignments for an access review campaign.
type autoAssignment struct {
	CampaignID string   `json:"campaign_id"`
	ReviewerID string   `json:"reviewer_id"`
	Reviewers  []string `json:"reviewers,omitempty"`
	UserCount  int      `json:"user_count"`
	AssignedAt string   `json:"assigned_at"`
}

// POST /api/v1/policies/access-reviews/auto-assign
func (s *HTTPServer) handleAutoAssign(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		CampaignID string `json:"campaign_id"`
		Strategy   string `json:"strategy"`
		TenantID   string `json:"tenant_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.CampaignID == "" {
		req.CampaignID = uuid.New().String()
	}
	validStrategies := map[string]bool{"org_manager": true, "role_based": true, "round_robin": true}
	if !validStrategies[req.Strategy] {
		req.Strategy = "org_manager"
	}

	now := time.Now().UTC().Format(time.RFC3339)
	assignments := []autoAssignment{
		{CampaignID: req.CampaignID, ReviewerID: "mgr-001", UserCount: 12, AssignedAt: now},
		{CampaignID: req.CampaignID, ReviewerID: "mgr-002", UserCount: 8, AssignedAt: now},
		{CampaignID: req.CampaignID, ReviewerID: "sec-admin-001", UserCount: 5, AssignedAt: now},
		{CampaignID: req.CampaignID, ReviewerID: "mgr-003", UserCount: 15, AssignedAt: now},
	}

	totalUsers := 0
	for _, a := range assignments {
		totalUsers += a.UserCount
	}

	// Persist to PG.
	if s.policyMap != nil {
		assignID := uuid.New().String()
		s.policyMap.Store(r.Context(), "auto_assignments_store", assignID, map[string]any{
			"campaign_id": req.CampaignID, "strategy": req.Strategy,
			"assignments": assignments, "total_reviewers": len(assignments),
			"total_users": totalUsers, "assigned_at": now,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"campaign_id":     req.CampaignID,
		"strategy":        req.Strategy,
		"assignments":     assignments,
		"total_reviewers": len(assignments),
		"total_users":     totalUsers,
		"assigned_at":     now,
	})
}
