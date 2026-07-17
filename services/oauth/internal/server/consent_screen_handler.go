package server

import (
	"encoding/json"
	"net/http"
	"strings"
)

type ConsentScreenConfig struct {
	Title              string `json:"title"`
	Description        string `json:"description"`
	DataSharingSummary string `json:"data_sharing_summary"`
	PrivacyURL         string `json:"privacy_url"`
	TermsURL           string `json:"terms_url"`
}

// GET/PUT /api/v1/oauth/clients/{id}/consent-screen
func handleConsentScreen(w http.ResponseWriter, r *http.Request) {
	if !strings.HasSuffix(r.URL.Path, "/consent-screen") {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
		return
	}

	clientID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/v1/oauth/clients/"), "/consent-screen")
	if clientID == "" || strings.Contains(clientID, "/") {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid client_id"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		if mapRepoVar != nil {
			data, err := mapRepoVar.Get(r.Context(), "oauth_consent_screens", clientID)
			if err == nil {
				title, _ := data["title"].(string)
				description, _ := data["description"].(string)
				dataSharingSummary, _ := data["data_sharing_summary"].(string)
				privacyURL, _ := data["privacy_url"].(string)
				termsURL, _ := data["terms_url"].(string)
				writeJSON(w, http.StatusOK, map[string]any{
					"client_id":          clientID,
					"configured":         true,
					"title":              title,
					"description":        description,
					"data_sharing_summary": dataSharingSummary,
					"privacy_url":        privacyURL,
					"terms_url":          termsURL,
				})
				return
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"client_id":   clientID,
			"title":       "Authorize Application",
			"description": "This application is requesting access to your account.",
			"configured":  false,
		})

	case http.MethodPut, http.MethodPost:
		var req ConsentScreenConfig
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON"})
			return
		}
		// Sanitize: strip script tags from description
		req.Description = strings.ReplaceAll(req.Description, "<script>", "")
		req.Description = strings.ReplaceAll(req.Description, "</script>", "")
		req.DataSharingSummary = strings.ReplaceAll(req.DataSharingSummary, "<script>", "")
		req.DataSharingSummary = strings.ReplaceAll(req.DataSharingSummary, "</script>", "")

		if mapRepoVar != nil {
			b, _ := json.Marshal(req)
			var dataMap map[string]any
			json.Unmarshal(b, &dataMap)
			dataMap["client_id"] = clientID
			mapRepoVar.Store(r.Context(), "oauth_consent_screens", clientID, dataMap)
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"status":    "updated",
			"client_id": clientID,
			"config":    req,
		})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
	}
}
