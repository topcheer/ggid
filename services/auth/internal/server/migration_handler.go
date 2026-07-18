package server

import (
	"encoding/json"
	"net/http"
	"strings"

	ggidtenant "github.com/ggid/ggid/pkg/tenant"
)

// handleMigrationConfig manages the legacy migration configuration.
// PUT /api/v1/admin/migration/config
// GET /api/v1/admin/migration/config
func (h *Handler) handleMigrationConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		if h.migrationEngine == nil {
			writeJSON(w, http.StatusOK, LegacyMigrationConfig{Enabled: false})
			return
		}
		writeJSON(w, http.StatusOK, h.migrationEngine.GetConfig())
		return
	}

	if r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if h.migrationEngine == nil {
		writeError(w, http.StatusServiceUnavailable, "migration engine not configured")
		return
	}

	var cfg LegacyMigrationConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if cfg.SourceDBConn == "" {
		writeError(w, http.StatusBadRequest, "source_db_conn is required")
		return
	}

	if cfg.HashFormat == "" {
		cfg.HashFormat = "auto"
	}

	// Set default attribute mapping if not provided.
	if cfg.AttributeMapping == nil {
		cfg.AttributeMapping = map[string]string{
			"username":     "username",
			"mail":         "email",
			"cn":           "display_name",
			"password_hash": "password_hash",
		}
	}

	if err := h.migrationEngine.SaveConfig(r.Context(), &cfg); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save config")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "configured",
		"enabled": cfg.Enabled,
	})
}

// handleMigrationStats returns migration statistics.
// GET /api/v1/admin/migration/stats
func (h *Handler) handleMigrationStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if h.migrationEngine == nil {
		writeError(w, http.StatusServiceUnavailable, "migration engine not configured")
		return
	}

	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "X-Tenant-ID header required")
		return
	}

	stats, err := h.migrationEngine.GetStats(r.Context(), tc.TenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get stats")
		return
	}

	writeJSON(w, http.StatusOK, stats)
}

// handleMigrationTest tests the legacy system connection.
// POST /api/v1/admin/migration/test
func (h *Handler) handleMigrationTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if h.migrationEngine == nil {
		writeError(w, http.StatusServiceUnavailable, "migration engine not configured")
		return
	}

	// Reload config from DB to ensure latest settings.
	if _, err := h.migrationEngine.LoadConfig(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load config")
		return
	}

	if err := h.migrationEngine.TestConnection(r.Context()); err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"connected": false,
			"error":     err.Error(),
		})
		return
	}

	// Mask the connection string for security.
	connStr := ""
	cfg := h.migrationEngine.GetConfig()
	if cfg != nil {
		connStr = maskConnStr(cfg.SourceDBConn)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"connected":  true,
		"connection": connStr,
	})
}

// maskConnStr masks password in connection string for display.
func maskConnStr(connStr string) string {
	idx := strings.Index(connStr, "password=")
	if idx == -1 {
		idx = strings.Index(connStr, "Password=")
	}
	if idx == -1 {
		return connStr
	}
	end := strings.Index(connStr[idx:], " ")
	if end == -1 {
		end = len(connStr) - idx
	}
	return connStr[:idx] + "password=***" + connStr[idx+end:]
}
