package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
)

// scopeDeprecation tracks deprecated scopes.
type scopeDeprecation struct {
	ScopeName        string `json:"scope_name"`
	IsDeprecated     bool   `json:"is_deprecated"`
	SunsetDate       string `json:"sunset_date"`
	ReplacementScope string `json:"replacement_scope,omitempty"`
	DeprecatedAt     string `json:"deprecated_at,omitempty"`
	Reason           string `json:"reason,omitempty"`
}

var scopeDeprecationStore = struct {
	sync.RWMutex
	data map[string]*scopeDeprecation
}{data: map[string]*scopeDeprecation{
	"old_profile": {
		ScopeName: "old_profile", IsDeprecated: true,
		SunsetDate: time.Now().UTC().Add(30 * 24 * time.Hour).Format("2006-01-02"),
		ReplacementScope: "profile", DeprecatedAt: time.Now().UTC().Add(-7 * 24 * time.Hour).Format(time.RFC3339),
		Reason: "Merged into standard 'profile' scope",
	},
}}

// POST /api/v1/oauth/scopes/{name}/deprecate
// GET  /api/v1/oauth/scopes/deprecations — list all deprecated scopes
func handleScopeDeprecation(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		// Extract scope name from path
		path := r.URL.Path
		scopeName := ""
		if idx := strings.Index(path, "/scopes/"); idx >= 0 {
			rest := path[idx+len("/scopes/"):]
			if dIdx := strings.Index(rest, "/deprecate"); dIdx >= 0 {
				scopeName = rest[:dIdx]
			}
		}
		if scopeName == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "scope name is required"})
			return
		}

		var req struct {
			SunsetDate       string `json:"sunset_date"`
			ReplacementScope string `json:"replacement_scope"`
			Reason           string `json:"reason"`
		}
		if r.ContentLength > 0 {
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
				return
			}
		}

		dep := &scopeDeprecation{
			ScopeName:        scopeName,
			IsDeprecated:     true,
			SunsetDate:       req.SunsetDate,
			ReplacementScope: req.ReplacementScope,
			Reason:           req.Reason,
			DeprecatedAt:     time.Now().UTC().Format(time.RFC3339),
		}
		if dep.SunsetDate == "" {
			dep.SunsetDate = time.Now().UTC().Add(90 * 24 * time.Hour).Format("2006-01-02")
		}

		scopeDeprecationStore.Lock()
		scopeDeprecationStore.data[scopeName] = dep
		scopeDeprecationStore.Unlock()

		writeJSON(w, http.StatusOK, map[string]any{
			"scope_name":        scopeName,
			"is_deprecated":     true,
			"sunset_date":       dep.SunsetDate,
			"replacement_scope": dep.ReplacementScope,
			"deprecated_at":     dep.DeprecatedAt,
			"deprecation_warning": "Tokens containing this scope will include a deprecation_warning after this date",
		})

	case http.MethodGet:
		scopeDeprecationStore.RLock()
		result := []*scopeDeprecation{}
		for _, dep := range scopeDeprecationStore.data {
			result = append(result, dep)
		}
		scopeDeprecationStore.RUnlock()

		writeJSON(w, http.StatusOK, map[string]any{
			"deprecations": result,
			"total":        len(result),
		})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}
