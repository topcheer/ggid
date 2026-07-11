package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
)

type ClientBranding struct {
	LogoURL       string `json:"logo_url"`
	PrimaryColor  string `json:"primary_color"`
	BackgroundURL string `json:"background_url"`
	CustomCSS     string `json:"custom_css"`
}

var (brandingMu sync.RWMutex; brandingStore = make(map[string]*ClientBranding))

// PUT /api/v1/oauth/clients/{id}/branding — set client branding.
// GET /api/v1/oauth/clients/{id}/branding — get client branding.
func handleClientBranding(w http.ResponseWriter, r *http.Request) {
	clientID := extractClientIDFromPath(r.URL.Path, "/branding")
	if clientID == "" { writeJSON(w, http.StatusBadRequest, map[string]any{"error": "client_id required"}); return }
	switch r.Method {
	case http.MethodPut, http.MethodPost:
		var b ClientBranding
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil { writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON"}); return }
		// Sanitize custom_css: strip script tags
		b.CustomCSS = strings.ReplaceAll(b.CustomCSS, "<script>", "")
		b.CustomCSS = strings.ReplaceAll(b.CustomCSS, "</script>", "")
		brandingMu.Lock(); brandingStore[clientID] = &b; brandingMu.Unlock()
		writeJSON(w, http.StatusOK, map[string]any{"status": "updated", "client_id": clientID, "branding": b})
	case http.MethodGet:
		brandingMu.RLock(); b, ok := brandingStore[clientID]; brandingMu.RUnlock()
		if !ok { writeJSON(w, http.StatusOK, map[string]any{"client_id": clientID, "branding": nil}); return }
		writeJSON(w, http.StatusOK, map[string]any{"client_id": clientID, "branding": b})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
	}
}

func extractClientIDFromPath(path, suffix string) string {
	path = strings.TrimPrefix(path, "/api/v1/oauth/clients/")
	path = strings.TrimSuffix(path, suffix)
	if strings.Contains(path, "/") { return "" }
	return path
}
