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
	Name           string          `json:"name"`
	Key            string          `json:"key"`
	Description    string          `json:"description"`
	Enabled        bool            `json:"enabled"`
	RolloutPct     int             `json:"rollout_pct"`
	TargetAudience string          `json:"target_audience"`
	EnvOverride    map[string]bool `json:"env_override"`
}

// flagName returns the identifier (key if set, else name).
func (f FeatureFlag) flagName() string {
	if f.Key != "" {
		return f.Key
	}
	return f.Name
}

// FlagAuditEntry tracks changes to feature flags.
type FlagAuditEntry struct {
	FlagName  string    `json:"flag_name"`
	Action    string    `json:"action"`
	Timestamp time.Time `json:"timestamp"`
	Actor     string    `json:"actor"`
}

var (
	flagMu       sync.RWMutex
	featureFlags = []FeatureFlag{
		{Name: "webauthn", Key: "webauthn", Enabled: true, RolloutPct: 100, TargetAudience: "all", EnvOverride: map[string]bool{}},
		{Name: "scim_v2", Key: "scim_v2", Enabled: true, RolloutPct: 100, TargetAudience: "all", EnvOverride: map[string]bool{}},
		{Name: "passkey_autofill", Key: "passkey_autofill", Enabled: false, RolloutPct: 0, TargetAudience: "all", EnvOverride: map[string]bool{}},
	}
	flagAuditLog = []FlagAuditEntry{}
)

// GET/POST /api/v1/admin/feature-flags
// PUT /api/v1/admin/feature-flags/{name} (toggle via PUT — Console-compatible)
// POST /api/v1/admin/feature-flags/{name}/toggle
// DELETE /api/v1/admin/feature-flags/{name}
// hasAdminScope checks if the request has tenant:admin or platform:admin scope.
// Defense-in-depth: the gateway adminOnlyPaths also enforces this.
func hasAdminScope(r *http.Request) bool {
	scopes := r.Header.Get("X-Scopes")
	for _, s := range strings.Split(scopes, ",") {
		s = strings.TrimSpace(s)
		if s == "platform:admin" || s == "tenant:admin" {
			return true
		}
	}
	return false
}

// hasPlatformAdminScope checks ONLY for platform:admin (not tenant:admin).
// Use this for operations that affect global/platform-level settings.
func hasPlatformAdminScope(r *http.Request) bool {
	scopes := r.Header.Get("X-Scopes")
	for _, s := range strings.Split(scopes, ",") {
		if strings.TrimSpace(s) == "platform:admin" {
			return true
		}
	}
	return false
}

func (h *Handler) handleFeatureFlags(w http.ResponseWriter, r *http.Request) {
	// SECURITY: admin scope required (defense-in-depth alongside gateway adminOnlyPaths)
	if !hasAdminScope(r) {
		writeError(w, http.StatusForbidden, "admin scope required")
		return
	}
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
		// Console sends key, not name — sync both fields.
		if req.Name == "" {
			req.Name = req.Key
		}
		if req.Key == "" {
			req.Key = req.Name
		}
		if req.EnvOverride == nil {
			req.EnvOverride = map[string]bool{}
		}
		flagMu.Lock()
		featureFlags = append(featureFlags, req)
		flagAuditLog = append(flagAuditLog, FlagAuditEntry{
			FlagName: req.flagName(), Action: "created", Timestamp: time.Now(), Actor: "admin",
		})
		flagMu.Unlock()
		writeJSON(w, http.StatusCreated, req)
		return
	}

	// Sub-path: /api/v1/admin/feature-flags/{name}[/toggle]
	if strings.HasPrefix(r.URL.Path, "/api/v1/admin/feature-flags/") {
		sub := strings.TrimPrefix(r.URL.Path, "/api/v1/admin/feature-flags/")
		isToggle := strings.HasSuffix(sub, "/toggle")
		name := strings.TrimSuffix(sub, "/toggle")

		// PUT /{name} — toggle (Console calls PUT)
		if r.Method == http.MethodPut {
			flagMu.Lock()
			defer flagMu.Unlock()
			for i := range featureFlags {
				if featureFlags[i].flagName() == name || featureFlags[i].Name == name {
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

		// POST /{name}/toggle — toggle (original API)
		if isToggle && r.Method == http.MethodPost {
			flagMu.Lock()
			defer flagMu.Unlock()
			for i := range featureFlags {
				if featureFlags[i].flagName() == name || featureFlags[i].Name == name {
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

		// DELETE /{name} — remove flag
		if r.Method == http.MethodDelete {
			flagMu.Lock()
			defer flagMu.Unlock()
			for i := range featureFlags {
				if featureFlags[i].flagName() == name || featureFlags[i].Name == name {
					featureFlags = append(featureFlags[:i], featureFlags[i+1:]...)
					flagAuditLog = append(flagAuditLog, FlagAuditEntry{
						FlagName: name, Action: "deleted", Timestamp: time.Now(), Actor: "admin",
					})
					writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "name": name})
					return
				}
			}
			writeError(w, http.StatusNotFound, "flag not found")
			return
		}
	}

	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}
