package httpserver

import (
	"encoding/json"
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

	reviewCampaigns.mu.Lock()
	reviewCampaigns.campaigns[c.ID] = c
	reviewCampaigns.mu.Unlock()

	writeJSON(w, http.StatusCreated, c)
}

func (s *HTTPServer) listReviewCampaigns(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")

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

	reviewCampaigns.mu.Lock()
	defer reviewCampaigns.mu.Unlock()

	c, ok := reviewCampaigns.campaigns[campaignID]
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

	writeJSON(w, http.StatusOK, c)
}
