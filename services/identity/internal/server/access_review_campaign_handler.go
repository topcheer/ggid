package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
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

var campaignStore sync.Map

func (h *HTTPHandler) handleAccessReviewCampaigns(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req CampaignRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
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
			CampaignID:         fmt.Sprintf("arc-%d", time.Now().UnixNano()%100000),
			CampaignName:       req.CampaignName,
			Scope:              req.Scope,
			Reviewers:          req.Reviewers,
			Deadline:           req.Deadline,
			RemindersEnabled:   req.RemindersEnabled,
			AutoRevokeOnExpiry: req.AutoRevokeOnExpiry,
			CompletionPct:      0,
			Status:             "active",
			CreatedAt:          time.Now().UTC().Format(time.RFC3339),
			ItemsTotal:         50,
			ItemsReviewed:      0,
		}
		campaignStore.Store(camp.CampaignID, camp)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(camp)
	case http.MethodGet:
		var campaigns []AccessReviewCampaign
		campaignStore.Range(func(_, v any) bool {
			campaigns = append(campaigns, v.(AccessReviewCampaign))
			return true
		})
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"campaigns": campaigns, "count": len(campaigns)})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
