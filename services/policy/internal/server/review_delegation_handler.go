package httpserver

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// reviewDelegation represents a delegated access review.
type reviewDelegation struct {
	ID               string `json:"id"`
	CampaignID       string `json:"campaign_id"`
	OriginalReviewer string `json:"original_reviewer"`
	DelegatedTo      string `json:"delegated_to"`
	Scope            string `json:"scope"`
	ExpiresAt        string `json:"expires_at"`
	Status           string `json:"status"` // active, expired, revoked
	CreatedAt        string `json:"created_at"`
}

var reviewDelegationStore = struct {
	sync.RWMutex
	data map[string]*reviewDelegation
}{data: make(map[string]*reviewDelegation)}

// POST /api/v1/policies/access-reviews/delegate — create delegation
// GET  /api/v1/policies/access-reviews/delegated — list delegations
func (s *HTTPServer) handleReviewDelegation(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req struct {
			CampaignID       string `json:"campaign_id"`
			OriginalReviewer string `json:"original_reviewer"`
			DelegatedTo      string `json:"delegated_to"`
			Scope            string `json:"scope"`
			ExpiresAt        string `json:"expires_at"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if req.OriginalReviewer == "" || req.DelegatedTo == "" {
			writeJSONError(w, http.StatusBadRequest, "original_reviewer and delegated_to are required")
			return
		}
		if req.OriginalReviewer == req.DelegatedTo {
			writeJSONError(w, http.StatusBadRequest, "cannot delegate to self")
			return
		}

		expiresAt := req.ExpiresAt
		if expiresAt == "" {
			expiresAt = time.Now().UTC().Add(7 * 24 * time.Hour).Format(time.RFC3339)
		}

		deg := &reviewDelegation{
			ID: uuid.New().String(), CampaignID: req.CampaignID,
			OriginalReviewer: req.OriginalReviewer, DelegatedTo: req.DelegatedTo,
			Scope: req.Scope, ExpiresAt: expiresAt,
			Status: "active", CreatedAt: time.Now().UTC().Format(time.RFC3339),
		}

		reviewDelegationStore.Lock()
		reviewDelegationStore.data[deg.ID] = deg
		reviewDelegationStore.Unlock()

		writeJSON(w, http.StatusCreated, deg)

	case http.MethodGet:
		reviewer := r.URL.Query().Get("reviewer")
		status := r.URL.Query().Get("status")

		reviewDelegationStore.RLock()
		result := []*reviewDelegation{}
		now := time.Now().UTC()
		for _, d := range reviewDelegationStore.data {
			// Auto-expire
			if d.Status == "active" {
				if exp, err := time.Parse(time.RFC3339, d.ExpiresAt); err == nil && now.After(exp) {
					d.Status = "expired"
				}
			}
			if reviewer != "" && d.OriginalReviewer != reviewer && d.DelegatedTo != reviewer {
				continue
			}
			if status != "" && d.Status != status {
				continue
			}
			result = append(result, d)
		}
		reviewDelegationStore.RUnlock()

		writeJSON(w, http.StatusOK, map[string]any{
			"delegations": result,
			"total":       len(result),
		})

	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
