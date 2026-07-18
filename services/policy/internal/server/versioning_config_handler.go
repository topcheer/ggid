package httpserver

import (
	"encoding/json"
	"net/http"
)

type PolicyVersioningConfig struct {
	MaxVersionsPerPolicy    int    `json:"max_versions_per_policy"`
	AutoArchiveAfterDays    int    `json:"auto_archive_after_days"`
	RollbackWindowDays      int    `json:"rollback_window_days"`
	ChangeTrackingLevel     string `json:"change_tracking_level"`
	ApprovalRequiredForRevert bool  `json:"approval_required_for_revert"`
	DiffFormat              string `json:"diff_format"`
}

var globalPolicyVersioningConfig = &PolicyVersioningConfig{
	MaxVersionsPerPolicy:      50,
	AutoArchiveAfterDays:      90,
	RollbackWindowDays:        30,
	ChangeTrackingLevel:       "diff",
	ApprovalRequiredForRevert: true,
	DiffFormat:                "unified",
}

func (s *HTTPServer) handlePolicyVersioningConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(globalPolicyVersioningConfig)
	case http.MethodPut:
		var cfg PolicyVersioningConfig
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if cfg.MaxVersionsPerPolicy < 1 {
			writeJSONError(w, http.StatusBadRequest, "max_versions_per_policy must be at least 1")
			return
		}
		globalPolicyVersioningConfig = &cfg
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(cfg)
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
