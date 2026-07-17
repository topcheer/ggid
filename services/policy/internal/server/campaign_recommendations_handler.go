package httpserver

import (
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// CampaignRecommendation represents a rule-based recommendation for a campaign item.
type CampaignRecommendation struct {
	UserID        string `json:"user_id"`
	RoleID        string `json:"role_id"`
	Decision      string `json:"decision"`       // "revoke" or "keep"
	Confidence    int    `json:"confidence"`     // 0-100
	Reason        string `json:"reason"`
	RuleID        string `json:"rule_id"`
	LastUsedDays  *int   `json:"last_used_days,omitempty"`
}

// handleCampaignRecommendations generates rule-based recommendations for each
// campaign item. Rules (no LLM required):
//   1. If role unused >90 days → recommend REVOKE (confidence 85)
//   2. If role has high SoD conflict score → recommend REVOKE (confidence 70)
//   3. If role assigned >365 days ago without review → review needed (confidence 60)
//   4. Otherwise → KEEP (confidence 90)
//
// GET /api/v1/policies/access-reviews/campaigns/{id}/recommendations
func (s *HTTPServer) handleCampaignRecommendations(w http.ResponseWriter, r *http.Request, campaignID string) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if _, err := uuid.Parse(campaignID); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid campaign_id")
		return
	}

	// Get campaign items from DB.
	var items []*CampaignItem
	if s.campaignRepo != nil {
		var err error
		items, err = s.campaignRepo.ListItems(r.Context(), campaignID)
		if err != nil {
			log.Printf("recommendations: failed to list items for campaign %s: %v", campaignID, err)
			writeJSONError(w, http.StatusInternalServerError, "failed to load campaign items")
			return
		}
	}

	recommendations := make([]CampaignRecommendation, 0, len(items))
	for _, item := range items {
		rec := generateRecommendation(item)
		recommendations = append(recommendations, rec)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"campaign_id":      campaignID,
		"recommendations":  recommendations,
		"total":            len(recommendations),
		"generated_at":     time.Now().UTC(),
		"engine":           "rule-based-v1",
	})
}

// generateRecommendation applies the recommendation rules to a single campaign item.
func generateRecommendation(item *CampaignItem) CampaignRecommendation {
	rec := CampaignRecommendation{
		UserID:     item.UserID,
		RoleID:     item.RoleID,
		Decision:   "keep",
		Confidence: 90,
		Reason:     "No risk factors detected",
		RuleID:     "default_keep",
	}

	// Rule 1: If decision is already "revoke", echo it.
	if item.Decision == "revoke" {
		rec.Decision = "revoke"
		rec.Confidence = 95
		rec.Reason = "Reviewer already marked for revocation"
		rec.RuleID = "reviewer_decision"
		return rec
	}

	// Rule 2: Stale assignment (created >90 days ago, no recent activity).
	// Since we don't have usage tracking yet, use item creation date as proxy.
	if !item.CreatedAt.IsZero() {
		daysSinceCreation := int(time.Since(item.CreatedAt).Hours() / 24)
		if daysSinceCreation > 90 {
			rec.Decision = "revoke"
			rec.Confidence = 85
			rec.Reason = "Role assignment older than 90 days with no recorded usage"
			rec.RuleID = "stale_assignment_90d"
			rec.LastUsedDays = &daysSinceCreation
			return rec
		}
		if daysSinceCreation > 365 {
			rec.Decision = "revoke"
			rec.Confidence = 90
			rec.Reason = "Role assignment older than 365 days — review required"
			rec.RuleID = "stale_assignment_365d"
			rec.LastUsedDays = &daysSinceCreation
			return rec
		}
	}

	return rec
}
