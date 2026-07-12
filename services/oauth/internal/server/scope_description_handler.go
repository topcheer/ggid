package server

import (
	"encoding/json"
	"net/http"
	"strings"
)

// PUT  /api/v1/oauth/scopes/{name}/description — set multi-lang description
// GET  /api/v1/oauth/scopes/{name}/description?lang=en — get description for a language
// DELETE /api/v1/oauth/scopes/{name}/description?lang=zh — delete a language entry
// Uses existing scopeDescStore from scope_i18n.go
func handleScopeDescription(w http.ResponseWriter, r *http.Request) {
	// Extract scope name from path
	path := r.URL.Path
	scopeName := ""
	if idx := strings.Index(path, "/scopes/"); idx >= 0 {
		rest := path[idx+len("/scopes/"):]
		if dIdx := strings.Index(rest, "/description"); dIdx >= 0 {
			scopeName = rest[:dIdx]
		}
	}
	if scopeName == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "scope name is required"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		lang := r.URL.Query().Get("lang")
		if lang == "" {
			lang = "en"
		}

		scopeDescMu.RLock()
		desc, exists := scopeDescStore[scopeName]
		scopeDescMu.RUnlock()

		if !exists {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "scope not found"})
			return
		}

		description, ok := desc.Descriptions[lang]
		if !ok {
			description = desc.Descriptions["en"]
			if description == "" {
				description = "No description available"
			}
		}

		availableLangs := make([]string, 0, len(desc.Descriptions))
		for l := range desc.Descriptions {
			availableLangs = append(availableLangs, l)
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"scope_name":      scopeName,
			"lang":            lang,
			"description":     description,
			"available_langs": availableLangs,
		})

	case http.MethodPut, http.MethodPost:
		var req struct {
			Description string `json:"description"`
			Lang        string `json:"lang"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}
		if req.Description == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "description is required"})
			return
		}
		if req.Lang == "" {
			req.Lang = "en"
		}

		scopeDescMu.Lock()
		desc, exists := scopeDescStore[scopeName]
		if !exists {
			desc = &scopeDesc{Name: scopeName, Descriptions: map[string]string{}}
			scopeDescStore[scopeName] = desc
		}
		desc.Descriptions[req.Lang] = req.Description
		scopeDescMu.Unlock()

		writeJSON(w, http.StatusOK, map[string]any{
			"scope_name":  scopeName,
			"lang":        req.Lang,
			"description": req.Description,
			"updated":     true,
		})

	case http.MethodDelete:
		lang := r.URL.Query().Get("lang")
		if lang == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "lang query param is required for deletion"})
			return
		}

		scopeDescMu.Lock()
		desc, exists := scopeDescStore[scopeName]
		if exists {
			delete(desc.Descriptions, lang)
		}
		scopeDescMu.Unlock()

		writeJSON(w, http.StatusOK, map[string]any{
			"scope_name": scopeName,
			"lang":       lang,
			"deleted":    true,
		})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}
