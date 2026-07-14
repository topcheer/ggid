package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/ggid/ggid/pkg/sysconfig"
)

// handleAdminConfig handles GET/PUT for system configuration.
// GET  /api/v1/admin/config          — list all config keys
// PUT  /api/v1/admin/config/{key}    — update a config key
// POST /api/v1/admin/config/{key}/reset — reset a key to default
func (h *Handler) handleAdminConfig(w http.ResponseWriter, r *http.Request) {
	tenantID := tenantIDFromRequest(r)

	// GET — list all
	if r.Method == http.MethodGet {
		keys := h.sysconfigStore.GetAll(tenantID)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"config": keys,
			"total":  len(keys),
		})
		return
	}

	// PUT — update single key
	if r.Method == http.MethodPut {
		key := strings.TrimPrefix(r.URL.Path, "/api/v1/admin/config/")
		key = strings.TrimSuffix(key, "/")
		if key == "" || key == "/api/v1/admin/config" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "config key required in path"})
			return
		}

		var body struct {
			Value string `json:"value"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}
		if body.Value == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "value is required"})
			return
		}

		if err := h.sysconfigStore.Set(r.Context(), tenantID, key, body.Value); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update config: " + err.Error()})
			return
		}

		// Return updated config
		cfg := h.sysconfigStore.Get(tenantID)
		keys := sysconfig.AllKeys(cfg)
		for _, k := range keys {
			if k.Key == key {
				writeJSON(w, http.StatusOK, map[string]interface{}{
					"message": "config updated",
					"key":     k,
				})
				return
			}
		}
		writeJSON(w, http.StatusOK, map[string]string{"message": "config updated", "key": key})
		return
	}

	// POST — reset to default
	if r.Method == http.MethodPost {
		key := strings.TrimPrefix(r.URL.Path, "/api/v1/admin/config/")
		key = strings.TrimSuffix(key, "/reset")
		if key == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "config key required"})
			return
		}

		if err := h.sysconfigStore.Reset(r.Context(), tenantID, key); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to reset config: " + err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{
			"message": "config reset to default",
			"key":     key,
		})
		return
	}

	writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
}

// tenantIDFromRequest extracts the tenant ID from the request context
// or falls back to the X-Tenant-ID header.
func tenantIDFromRequest(r *http.Request) string {
	// Try header first
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID != "" {
		return tenantID
	}
	// Default tenant for admin operations
	return "00000000-0000-0000-0000-000000000001"
}
