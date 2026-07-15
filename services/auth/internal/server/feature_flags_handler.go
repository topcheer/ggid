package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
)

// FeatureFlag represents a toggleable feature flag.
type FeatureFlag struct {
	Name           string            `json:"name"`
	Enabled        bool              `json:"enabled"`
	RolloutPct     int               `json:"rollout_pct"`
	TargetAudience string            `json:"target_audience"`
	EnvOverride    map[string]bool   `json:"env_override"`
}

// FlagAuditEntry tracks changes to feature flags.
type FlagAuditEntry struct {
	FlagName  string    `json:"flag_name"`
	Action    string    `json:"action"`
	Timestamp time.Time `json:"timestamp"`
	Actor     string    `json:"actor"`
}

var (
	flagMu          sync.RWMutex
	featureFlags    = []FeatureFlag{
		{Name: "webauthn", Enabled: true, RolloutPct: 100, TargetAudience: "all", EnvOverride: map[string]bool{}},
		{Name: "scim_v2", Enabled: true, RolloutPct: 100, TargetAudience: "all", EnvOverride: map[string]bool{}},
		{Name: "passkey_autofill", Enabled: false, RolloutPct: 0, TargetAudience: "all", EnvOverride: map[string]bool{}},
	}
	flagAuditLog = []FlagAuditEntry{}
)

// GET/POST /api/v1/admin/feature-flags
// POST /api/v1/admin/feature-flags/{name}/toggle
func (h *Handler) handleFeatureFlags(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/api/v1/admin/feature-flags" && r.Method == http.MethodGet {
		flagMu.RLock()
		defer flagMu.RUnlock()
		writeJSON(w, http.StatusOK, map[string]any{
			"flags": featureFlags,
			"audit": flagAuditLog,
		})
		return
	}

	if r.URL.Path == "/api/v1/admin/feature-flags" && r.Method == http.MethodPost {
		var req FeatureFlag
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid body")
			return
		}
		if req.EnvOverride == nil {
			req.EnvOverride = map[string]bool{}
		}
		flagMu.Lock()
		featureFlags = append(featureFlags, req)
		flagAuditLog = append(flagAuditLog, FlagAuditEntry{
			FlagName: req.Name, Action: "created", Timestamp: time.Now(), Actor: "admin",
		})
		flagMu.Unlock()
		writeJSON(w, http.StatusCreated, req)
		return
	}

	// Toggle: /api/v1/admin/feature-flags/{name}/toggle
	if strings.HasPrefix(r.URL.Path, "/api/v1/admin/feature-flags/") && strings.HasSuffix(r.URL.Path, "/toggle") && r.Method == http.MethodPost {
		name := strings.TrimPrefix(r.URL.Path, "/api/v1/admin/feature-flags/")
		name = strings.TrimSuffix(name, "/toggle")
		flagMu.Lock()
		defer flagMu.Unlock()
		for i := range featureFlags {
			if featureFlags[i].Name == name {
				featureFlags[i].Enabled = !featureFlags[i].Enabled
				flagAuditLog = append(flagAuditLog, FlagAuditEntry{
					FlagName: name, Action: "toggled", Timestamp: time.Now(), Actor: "admin",
				})
				writeJSON(w, http.StatusOK, featureFlags[i])
				return
			}
		}
		writeError(w, http.StatusNotFound, "flag not found")
		return
	}

	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}
