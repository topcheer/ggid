package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
)

type ConsentScreenConfig struct {
	Title               string `json:"title"`
	Description         string `json:"description"`
	DataSharingSummary  string `json:"data_sharing_summary"`
	PrivacyURL          string `json:"privacy_url"`
	TermsURL            string `json:"terms_url"`
}

var (
	consentScreenMu sync.RWMutex
	consentScreens  = make(map[string]*ConsentScreenConfig)
)

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
		consentScreenMu.RLock()
		cfg, ok := consentScreens[clientID]
		consentScreenMu.RUnlock()
		if !ok {
			writeJSON(w, http.StatusOK, map[string]any{
				"client_id":       clientID,
				"title":           "Authorize Application",
				"description":     "This application is requesting access to your account.",
				"configured":      false,
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"client_id":       clientID,
			"configured":      true,
			"title":           cfg.Title,
			"description":     cfg.Description,
			"data_sharing_summary": cfg.DataSharingSummary,
			"privacy_url":     cfg.PrivacyURL,
			"terms_url":       cfg.TermsURL,
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

		consentScreenMu.Lock()
		consentScreens[clientID] = &req
		consentScreenMu.Unlock()
		writeJSON(w, http.StatusOK, map[string]any{
			"status":    "updated",
			"client_id": clientID,
			"config":    req,
		})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
	}
}
