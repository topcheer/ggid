package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type AccessReviewCampaign struct {
	CampaignID         string   `json:"campaign_id"`
	CampaignName       string   `json:"campaign_name"`
	Scope              string   `json:"scope"`
	Reviewers          []string `json:"reviewers"`
	Deadline           string   `json:"deadline"`
	RemindersEnabled   bool     `json:"reminders_enabled"`
	AutoRevokeOnExpiry bool     `json:"auto_revoke_on_expiry"`
	CompletionPct      float64  `json:"completion_pct"`
	Status             string   `json:"status"`
	CreatedAt          string   `json:"created_at"`
	ItemsTotal         int      `json:"items_total"`
	ItemsReviewed      int      `json:"items_reviewed"`
}

type CampaignRequest struct {
	CampaignName       string   `json:"campaign_name"`
	Scope              string   `json:"scope"`
	Reviewers          []string `json:"reviewers"`
	Deadline           string   `json:"deadline"`
	RemindersEnabled   bool     `json:"reminders_enabled"`
	AutoRevokeOnExpiry bool     `json:"auto_revoke_on_expiry"`
}

func (h *HTTPHandler) handleAccessReviewCampaigns(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req CampaignRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.CampaignName == "" {
			req.CampaignName = "Untitled Campaign"
		}
		if req.Scope == "" {
			req.Scope = "org"
		}
		if len(req.Reviewers) == 0 {
			req.Reviewers = []string{"admin@ggid.dev"}
		}
		if req.Deadline == "" {
			req.Deadline = time.Now().Add(30 * 24 * time.Hour).UTC().Format("2006-01-02")
		}
		camp := AccessReviewCampaign{
			CampaignID:         uuid.New().String(),
			CampaignName:       req.CampaignName,
			Scope:              req.Scope,
			Reviewers:          req.Reviewers,
			Deadline:           req.Deadline,
			RemindersEnabled:   req.RemindersEnabled,
			AutoRevokeOnExpiry: req.AutoRevokeOnExpiry,
			CompletionPct:      0,
			Status:             "active",
			CreatedAt:          time.Now().UTC().Format(time.RFC3339),
			ItemsTotal:         0,
			ItemsReviewed:      0,
		}
		// Persist to PG.
		if h.identityPolicyMap != nil {
			data := map[string]any{
				"campaign_name": camp.CampaignName, "scope": camp.Scope,
				"reviewers": camp.Reviewers, "deadline": camp.Deadline,
				"reminders_enabled": camp.RemindersEnabled, "auto_revoke_on_expiry": camp.AutoRevokeOnExpiry,
				"status": camp.Status, "created_at": camp.CreatedAt,
			}
			h.identityPolicyMap.Store(r.Context(), "review_campaigns_store", camp.CampaignID, data)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(camp)
	case http.MethodGet:
		var campaigns []AccessReviewCampaign
		if h.identityPolicyMap != nil {
			rows, _ := h.identityPolicyMap.List(r.Context(), "review_campaigns_store")
			for _, row := range rows {
				camp := AccessReviewCampaign{
					CampaignID:   getString(row, "id"),
					CampaignName: getString(row, "campaign_name"),
					Scope:        getString(row, "scope"),
					Deadline:     getString(row, "deadline"),
					Status:       getString(row, "status"),
					CreatedAt:    getString(row, "created_at"),
				}
				campaigns = append(campaigns, camp)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"campaigns": campaigns, "count": len(campaigns)})
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
