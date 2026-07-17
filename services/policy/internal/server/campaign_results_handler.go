package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
)

type CampaignResult struct {
	CampaignID string `json:"campaign_id"`
	ItemID     string `json:"item_id"`
	UserID     string `json:"user_id"`
	RoleID     string `json:"role_id"`
	Decision   string `json:"decision"`
}

func (s *HTTPServer) handleCampaignResults(w http.ResponseWriter, r *http.Request, campaignID string) {
	switch r.Method {
	case http.MethodPost:
		var req CampaignResult
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		req.ItemID = uuid.New().String()
		if s.policyMap != nil {
			s.policyMap.Store(r.Context(), "policy_campaign_results", req.ItemID, map[string]any{
				"campaign_id": req.CampaignID, "user_id": req.UserID,
				"role_id": req.RoleID, "decision": req.Decision,
			})
		}
		writeJSON(w, http.StatusCreated, req)
	case http.MethodGet:
		campaignID := r.URL.Query().Get("campaign_id")
		var result []map[string]any
		if s.policyMap != nil {
			rows, _ := s.policyMap.List(r.Context(), "policy_campaign_results")
			for _, row := range rows {
				if campaignID != "" && pmGetString(row, "campaign_id") != campaignID {
					continue
				}
				result = append(result, row)
			}
		}
		if result == nil { result = []map[string]any{} }
		writeJSON(w, http.StatusOK, map[string]any{"results": result, "count": len(result)})
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
