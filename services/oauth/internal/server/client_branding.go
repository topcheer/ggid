package server

import (
	"encoding/json"
	"net/http"
	"strings"
)

type ClientBranding struct {
	LogoURL       string `json:"logo_url"`
	PrimaryColor  string `json:"primary_color"`
	BackgroundURL string `json:"background_url"`
	CustomCSS     string `json:"custom_css"`
}

// PUT /api/v1/oauth/clients/{id}/branding — set client branding.
// GET /api/v1/oauth/clients/{id}/branding — get client branding.
func handleClientBranding(w http.ResponseWriter, r *http.Request) {
	clientID := extractClientIDFromPath(r.URL.Path, "/branding")
	if clientID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "client_id required"})
		return
	}
	switch r.Method {
	case http.MethodPut, http.MethodPost:
		var b ClientBranding
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON"})
			return
		}
		// Sanitize custom_css: strip script tags
		b.CustomCSS = strings.ReplaceAll(b.CustomCSS, "<script>", "")
		b.CustomCSS = strings.ReplaceAll(b.CustomCSS, "</script>", "")
		if brandingAdapterVar != nil {
			brandingAdapterVar.Put(clientID, &b)
		} else if mapRepoVar != nil {
			mapRepoVar.Store(r.Context(), "oauth_branding", clientID, map[string]any{
				"logo_url": b.LogoURL, "primary_color": b.PrimaryColor,
				"background_url": b.BackgroundURL, "custom_css": b.CustomCSS,
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "updated", "client_id": clientID, "branding": b})
	case http.MethodGet:
		if brandingAdapterVar != nil {
			b, ok := brandingAdapterVar.Get(clientID)
			if !ok {
				writeJSON(w, http.StatusOK, map[string]any{"client_id": clientID, "branding": nil})
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"client_id": clientID, "branding": b})
			return
		}
		if mapRepoVar != nil {
			if row, _ := mapRepoVar.Get(r.Context(), "oauth_branding", clientID); row != nil {
				b := ClientBranding{
					LogoURL: omGetString(row, "logo_url"),
					PrimaryColor: omGetString(row, "primary_color"),
					BackgroundURL: omGetString(row, "background_url"),
					CustomCSS: omGetString(row, "custom_css"),
				}
				writeJSON(w, http.StatusOK, map[string]any{"client_id": clientID, "branding": b})
				return
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{"client_id": clientID, "branding": nil})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
	}
}

func extractClientIDFromPath(path, suffix string) string {
	path = strings.TrimPrefix(path, "/api/v1/oauth/clients/")
	path = strings.TrimSuffix(path, suffix)
	if strings.Contains(path, "/") {
		return ""
	}
	return path
}
