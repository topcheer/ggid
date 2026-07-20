package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// --- System Configuration ---

// handleSystemConfig handles GET/PUT /api/v1/system/config
// Reads or writes key-value system configuration from sys_config table.
func (h *HTTPHandler) handleSystemConfig(w http.ResponseWriter, r *http.Request) {
	pool := h.svc.Pool()
	if pool == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "database not available"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.systemConfigGet(w, r)
	case http.MethodPut:
		h.systemConfigPut(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (h *HTTPHandler) systemConfigGet(w http.ResponseWriter, r *http.Request) {
	// Fetch all config keys
	rows, err := h.svc.Pool().Query(r.Context(), `
		SELECT key, value::text, updated_at, COALESCE(updated_by::text, '')
		FROM sys_config ORDER BY key`)
	if err != nil {
		slog.Error("system config: query error", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to query config"})
		return
	}
	defer rows.Close()

	config := map[string]any{}
	for rows.Next() {
		var key, valueStr, updatedBy string
		var updatedAt time.Time
		if err := rows.Scan(&key, &valueStr, &updatedAt, &updatedBy); err != nil {
			continue
		}
		var value any
		_ = json.Unmarshal([]byte(valueStr), &value)
		config[key] = map[string]any{
			"value":      value,
			"updated_at": updatedAt,
			"updated_by": updatedBy,
		}
	}

	writeJSON(w, http.StatusOK, config)
}

func (h *HTTPHandler) systemConfigPut(w http.ResponseWriter, r *http.Request) {
	var req map[string]any
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}

	updatedByStr := r.Header.Get("X-User-ID")
	var updatedBy *uuid.UUID
	if uid, err := uuid.Parse(updatedByStr); err == nil {
		updatedBy = &uid
	}

	// Upsert each key-value pair
	for key, value := range req {
		valueJSON, err := json.Marshal(value)
		if err != nil {
			continue
		}

		if updatedBy != nil {
			_, err = h.svc.Pool().Exec(r.Context(), `
				INSERT INTO sys_config (key, value, updated_by)
				VALUES ($1, $2, $3)
				ON CONFLICT (key) DO UPDATE SET value = $2, updated_at = NOW(), updated_by = $3`,
				key, valueJSON, *updatedBy)
		} else {
			_, err = h.svc.Pool().Exec(r.Context(), `
				INSERT INTO sys_config (key, value)
				VALUES ($1, $2)
				ON CONFLICT (key) DO UPDATE SET value = $2, updated_at = NOW()`,
				key, valueJSON)
		}
		if err != nil {
			slog.Error("system config: upsert error", "key", key, "error", err)
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"updated": true, "keys": getMapKeys(req)})
}

func getMapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// GetWebAuthnConfig reads WebAuthn RP configuration from sys_config.
// Returns rp_id, rp_origins, rp_display_name with fallback to env vars.
// Used by auth service to dynamically configure WebAuthn without restart.
func GetWebAuthnConfig(pool QueryRower) (rpID string, rpOrigins []string, rpDisplayName string) {
	// Defaults
	rpDisplayName = "GGID"

	if pool == nil {
		return
	}

	var valueJSON string
	err := pool.QueryRow(`
		SELECT value::text FROM sys_config WHERE key = 'webauthn_config'`).Scan(&valueJSON)
	if err != nil {
		return // use defaults
	}

	var cfg struct {
		RPID          string   `json:"rp_id"`
		RPOrigins     []string `json:"rp_origins"`
		RPDisplayName string   `json:"rp_display_name"`
	}
	if err := json.Unmarshal([]byte(valueJSON), &cfg); err != nil {
		return
	}

	if cfg.RPID != "" {
		rpID = cfg.RPID
	}
	if len(cfg.RPOrigins) > 0 {
		rpOrigins = cfg.RPOrigins
	}
	if cfg.RPDisplayName != "" {
		rpDisplayName = cfg.RPDisplayName
	}
	return
}

// QueryRower is a minimal interface for DB query (pool or single connection).
type QueryRower interface {
	QueryRow(query string, args ...any) interface {
		Scan(dest ...any) error
	}
}
