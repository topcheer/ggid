package httpserver

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ReviewCampaign represents an access review campaign.
type ReviewCampaign struct {
	ID         string    `json:"id"`
	TenantID   string    `json:"tenant_id"`
	Scope      string    `json:"scope"`       // "org", "role", "department"
	ScopeID    string    `json:"scope_id"`    // org ID, role ID, or dept ID
	ReviewerID string    `json:"reviewer_id"`
	Deadline   time.Time `json:"deadline"`
	Status     string    `json:"status"`      // "active", "completed", "expired"
	Decision   string    `json:"decision,omitempty"` // "approve", "revoke", "modify"
	Notes      string    `json:"notes,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	SubmittedAt *time.Time `json:"submitted_at,omitempty"`
}

// campaignStore holds review campaigns in memory.
type campaignStore struct {
	mu        sync.RWMutex
	campaigns map[string]*ReviewCampaign
}

var reviewCampaigns = &campaignStore{campaigns: make(map[string]*ReviewCampaign)}

// POST /api/v1/policies/access-reviews/campaigns          — create campaign
// GET  /api/v1/policies/access-reviews/campaigns/active   — list active campaigns
// POST /api/v1/policies/access-reviews/campaigns/{id}/submit — submit review
func (s *HTTPServer) handleReviewCampaigns(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/policies/access-reviews/campaigns")

	if path == "" || path == "/" {
		if r.Method == http.MethodPost {
			s.createReviewCampaign(w, r)
			return
		}
		if r.Method == http.MethodGet {
			s.listReviewCampaigns(w, r)
			return
		}
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 2 && parts[1] == "submit" {
		s.submitReviewCampaign(w, r, parts[0])
		return
	}
	if len(parts) == 2 && parts[1] == "results" {
		s.handleCampaignResults(w, r, parts[0])
		return
	}

	writeJSONError(w, http.StatusNotFound, "not found")
}

func (s *HTTPServer) handleReviewCampaignsActive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	tenantID := r.URL.Query().Get("tenant_id")

	reviewCampaigns.mu.RLock()
	defer reviewCampaigns.mu.RUnlock()

	result := []*ReviewCampaign{}
	for _, c := range reviewCampaigns.campaigns {
		if c.Status != "active" {
			continue
		}
		if tenantID != "" && c.TenantID != tenantID {
			continue
		}
		result = append(result, c)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"campaigns": result,
		"count":     len(result),
	})
}

func (s *HTTPServer) createReviewCampaign(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TenantID   string `json:"tenant_id"`
		Scope      string `json:"scope"`
		ScopeID    string `json:"scope_id"`
		ReviewerID string `json:"reviewer_id"`
		DeadlineHours int  `json:"deadline_hours"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Scope == "" {
		writeJSONError(w, http.StatusBadRequest, "scope is required")
		return
	}
	if req.ReviewerID == "" {
		writeJSONError(w, http.StatusBadRequest, "reviewer_id is required")
		return
	}
	if req.DeadlineHours <= 0 {
		req.DeadlineHours = 168 // default 7 days
	}

	now := time.Now().UTC()
	c := &ReviewCampaign{
		ID:         uuid.New().String(),
		TenantID:   req.TenantID,
		Scope:      req.Scope,
		ScopeID:    req.ScopeID,
		ReviewerID: req.ReviewerID,
		Deadline:   now.Add(time.Duration(req.DeadlineHours) * time.Hour),
		Status:     "active",
		CreatedAt:  now,
	}

	if s.campaignRepo != nil {
		if err := s.campaignRepo.Create(r.Context(), c); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "internal server error")
			return
		}
	} else {
		reviewCampaigns.mu.Lock()
		reviewCampaigns.campaigns[c.ID] = c
		reviewCampaigns.mu.Unlock()
	}

	writeJSON(w, http.StatusCreated, c)
}

// SetCampaignRepo injects a DB-backed campaign store.
func (s *HTTPServer) SetCampaignRepo(repo *CampaignRepo) {
	s.campaignRepo = repo
}

// executeCampaignRevoke executes revoke decisions for each campaign item.
// Iterates items with decision=revoke and calls roleSvc.RevokeRole for each
// (user_id, role_id) pair. The reviewer's own permissions are never touched.
func (s *HTTPServer) executeCampaignRevoke(ctx context.Context, c *ReviewCampaign) {
	if s.roleSvc == nil || c == nil {
		return
	}

	// Get items with decision=revoke from DB.
	var items []*CampaignItem
	if s.campaignRepo != nil {
		var err error
		items, err = s.campaignRepo.ListRevokeItems(ctx, c.ID)
		if err != nil {
			log.Printf("campaign revoke: failed to list items for campaign %s: %v", c.ID, err)
			return
		}
	}

	// If no DB items, fall back to single-user revoke using scope_id as role
	// and reviewer_id as the reviewed user (legacy behavior, will be removed).
	if len(items) == 0 {
		roleID, err := uuid.Parse(c.ScopeID)
		if err != nil {
			return // silently skip — no valid target
		}
		// BUGFIX: The reviewed user should NOT be the reviewer.
		// Without items, we cannot determine the target user, so skip.
		_ = roleID
		return
	}

	// Execute revoke for each item.
	for _, item := range items {
		userID, err := uuid.Parse(item.UserID)
		if err != nil {
			log.Printf("campaign revoke: invalid user_id %s: %v", item.UserID, err)
			continue
		}
		roleID, err := uuid.Parse(item.RoleID)
		if err != nil {
			log.Printf("campaign revoke: invalid role_id %s: %v", item.RoleID, err)
			continue
		}

		if err := s.roleSvc.RevokeRole(ctx, userID, roleID, "tenant", uuid.Nil); err != nil {
			log.Printf("campaign revoke: failed to revoke role %s for user %s: %v",
				item.RoleID, item.UserID, err)
		}
	}
}

func (s *HTTPServer) listReviewCampaigns(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")

	if s.campaignRepo != nil {
		active, err := s.campaignRepo.ListActive(r.Context(), "")
		if err == nil {
			if status != "" {
				var filtered []*ReviewCampaign
				for _, c := range active {
					if c.Status == status {
						filtered = append(filtered, c)
					}
				}
				active = filtered
			}
			writeJSON(w, http.StatusOK, map[string]any{"campaigns": active, "count": len(active)})
			return
		}
	}

	reviewCampaigns.mu.RLock()
	defer reviewCampaigns.mu.RUnlock()

	result := []*ReviewCampaign{}
	for _, c := range reviewCampaigns.campaigns {
		if status != "" && c.Status != status {
			continue
		}
		result = append(result, c)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"campaigns": result,
		"count":     len(result),
	})
}

func (s *HTTPServer) submitReviewCampaign(w http.ResponseWriter, r *http.Request, campaignID string) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		Decision string `json:"decision"`
		Notes    string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Decision != "approve" && req.Decision != "revoke" && req.Decision != "modify" {
		writeJSONError(w, http.StatusBadRequest, "decision must be approve, revoke, or modify")
		return
	}

	var c *ReviewCampaign
	if s.campaignRepo != nil {
		var err error
		c, err = s.campaignRepo.GetByID(r.Context(), campaignID)
		if err != nil || c == nil {
			writeJSONError(w, http.StatusNotFound, "campaign not found")
			return
		}
		if c.Status != "active" {
			writeJSONError(w, http.StatusConflict, "campaign already completed")
			return
		}
		if err := s.campaignRepo.Submit(r.Context(), campaignID, req.Decision, req.Notes); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		c.Status = "completed"
		c.Decision = req.Decision
		c.Notes = req.Notes
		now := time.Now().UTC()
		c.SubmittedAt = &now
	} else {
		reviewCampaigns.mu.Lock()
		defer reviewCampaigns.mu.Unlock()
		var ok bool
		c, ok = reviewCampaigns.campaigns[campaignID]
		if !ok {
			writeJSONError(w, http.StatusNotFound, "campaign not found")
			return
		}
		if c.Status != "active" {
			writeJSONError(w, http.StatusConflict, "campaign already completed")
			return
		}
		now := time.Now().UTC()
		c.Status = "completed"
		c.Decision = req.Decision
		c.Notes = req.Notes
		c.SubmittedAt = &now
	}

	// Execute revoke: if decision is "revoke", call roleSvc.RevokeRole.
	if req.Decision == "revoke" && s.roleSvc != nil {
		s.executeCampaignRevoke(r.Context(), c)
	}

	writeJSON(w, http.StatusOK, c)
}
