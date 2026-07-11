package httpserver

import (
	"encoding/csv"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// CampaignResult tracks individual review decisions within a campaign.
type CampaignResult struct {
	Reviewer  string    `json:"reviewer"`
	Decision  string    `json:"decision"` // certify, revoke, modify
	UserID    string    `json:"user_id"`
	RoleID    string    `json:"role_id"`
	Notes     string    `json:"notes,omitempty"`
	DecidedAt time.Time `json:"decided_at"`
}

var (
	campaignResultMu sync.RWMutex
	campaignResults  = make(map[string][]CampaignResult) // campaign_id → results
)

// RecordCampaignResult stores a review decision for a campaign.
func RecordCampaignResult(campaignID, reviewer, decision, userID, roleID, notes string) {
	campaignResultMu.Lock()
	campaignResults[campaignID] = append(campaignResults[campaignID], CampaignResult{
		Reviewer: reviewer, Decision: decision, UserID: userID,
		RoleID: roleID, Notes: notes, DecidedAt: time.Now().UTC(),
	})
	campaignResultMu.Unlock()
}

// handleCampaignResults is called from handleReviewCampaigns for /{id}/results sub-path.
func (s *HTTPServer) handleCampaignResults(w http.ResponseWriter, r *http.Request, campaignID string) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	format := r.URL.Query().Get("format")

	campaignResultMu.RLock()
	results := campaignResults[campaignID]
	campaignResultMu.RUnlock()

	if format == "csv" {
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename=campaign_"+campaignID+"_results.csv")
		writer := csv.NewWriter(w)
		writer.Write([]string{"reviewer", "decision", "user_id", "role_id", "notes", "decided_at"})
		for _, res := range results {
			writer.Write([]string{res.Reviewer, res.Decision, res.UserID, res.RoleID, res.Notes, res.DecidedAt.Format(time.RFC3339)})
		}
		writer.Flush()
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"campaign_id": campaignID,
		"results":     results,
		"count":       len(results),
	})
}

// Ensure uuid import used
var _ = uuid.New
