package server

import (
	"net/http"
	"sync"
	"time"
)

// passwordHistoryConfig holds password history policy settings.
type passwordHistoryConfig struct {
	HistoryCount      int  `json:"history_count"`
	RotationRequired  bool `json:"rotation_required"`
	MinRotationDays   int  `json:"min_rotation_days"`
	EnforceComplexity bool `json:"enforce_complexity"`
}

var passwordHistoryCfgStore = struct {
	sync.RWMutex
	cfg passwordHistoryConfig
}{cfg: passwordHistoryConfig{
	HistoryCount: 5, RotationRequired: true, MinRotationDays: 90, EnforceComplexity: true,
}}

// GET/PUT /api/v1/auth/password-history/config
func (h *Handler) handlePasswordHistoryConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		passwordHistoryCfgStore.RLock()
		cfg := passwordHistoryCfgStore.cfg
		passwordHistoryCfgStore.RUnlock()
		writeJSON(w, http.StatusOK, cfg)

	case http.MethodPut, http.MethodPost:
		var req struct {
			HistoryCount      int  `json:"history_count"`
			RotationRequired  *bool `json:"rotation_required"`
			MinRotationDays   int  `json:"min_rotation_days"`
			EnforceComplexity *bool `json:"enforce_complexity"`
		}
		if err := readJSONBody(r, &req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		passwordHistoryCfgStore.Lock()
		if req.HistoryCount > 0 {
			passwordHistoryCfgStore.cfg.HistoryCount = req.HistoryCount
		}
		if req.MinRotationDays > 0 {
			passwordHistoryCfgStore.cfg.MinRotationDays = req.MinRotationDays
		}
		if req.RotationRequired != nil {
			passwordHistoryCfgStore.cfg.RotationRequired = *req.RotationRequired
		}
		if req.EnforceComplexity != nil {
			passwordHistoryCfgStore.cfg.EnforceComplexity = *req.EnforceComplexity
		}
		cfg := passwordHistoryCfgStore.cfg
		passwordHistoryCfgStore.Unlock()

		writeJSON(w, http.StatusOK, map[string]any{
			"config":    cfg,
			"updated":   true,
			"updated_at": time.Now().UTC().Format(time.RFC3339),
		})

	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
