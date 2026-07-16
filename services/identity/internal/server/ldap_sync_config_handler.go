package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// LDAPSyncConfig stores LDAP directory sync configuration.
type LDAPSyncConfig struct {
	ServerURL        string            `json:"server_url"`
	BindDN           string            `json:"bind_dn"`
	BindPassword     string            `json:"bind_password,omitempty"`
	BaseDN           string            `json:"base_dn"`
	UserFilter       string            `json:"user_filter"`
	GroupFilter      string            `json:"group_filter"`
	AttributeMapping map[string]string `json:"attribute_mapping"`
	SyncSchedule     string            `json:"sync_schedule"`
	StartTLS         bool              `json:"start_tls"`
	AutoProvision    bool              `json:"auto_provision"`
}

// ldapConfigStore persists LDAP sync configuration in memory.
// In production this would be backed by a DB table (idp_sync_configs).
var ldapConfigStore = struct {
	sync.RWMutex
	config *LDAPSyncConfig
}{config: nil}

// ldapSyncState tracks the last sync run result.
var ldapSyncState = struct {
	sync.RWMutex
	lastRun    time.Time
	status     string // success, failed, never
	synced     int
	totalFound int
	errs       []map[string]any
}{status: "never", errs: []map[string]any{}}

func (h *HTTPHandler) handleLDAPSyncConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		ldapConfigStore.RLock()
		cfg := ldapConfigStore.config
		ldapConfigStore.RUnlock()

		if cfg == nil {
			// Return empty default config so frontend can populate the form.
			cfg = &LDAPSyncConfig{
				UserFilter:       "(objectClass=person)",
				GroupFilter:      "(objectClass=groupOfNames)",
				AttributeMapping: map[string]string{},
				SyncSchedule:     "0 */6 * * *",
				StartTLS:         true,
				AutoProvision:    true,
			}
		}

		ldapSyncState.RLock()
		result := map[string]any{
			"config": cfg,
			"last_sync": map[string]any{
				"timestamp": ldapSyncState.lastRun.Format(time.RFC3339),
				"status":    ldapSyncState.status,
				"errors":    len(ldapSyncState.errs),
				"synced":    ldapSyncState.synced,
			},
		}
		ldapSyncState.RUnlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)

	case http.MethodPut:
		var cfg LDAPSyncConfig
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		// Basic validation
		if cfg.ServerURL == "" {
			writeError(w, http.StatusBadRequest, "server_url is required")
			return
		}
		if cfg.BaseDN == "" {
			writeError(w, http.StatusBadRequest, "base_dn is required")
			return
		}
		if cfg.UserFilter == "" {
			cfg.UserFilter = "(objectClass=person)"
		}
		if cfg.GroupFilter == "" {
			cfg.GroupFilter = "(objectClass=groupOfNames)"
		}
		if cfg.SyncSchedule == "" {
			cfg.SyncSchedule = "0 */6 * * *"
		}

		ldapConfigStore.Lock()
		ldapConfigStore.config = &cfg
		ldapConfigStore.Unlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": "saved", "server_url": cfg.ServerURL})

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleLDAPSyncConfigTest tests the LDAP connection and returns diagnostic info.
// POST /api/v1/identity/ldap/sync-config/test
func (h *HTTPHandler) handleLDAPSyncConfigTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var cfg LDAPSyncConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		// If body is empty, use stored config
		ldapConfigStore.RLock()
		stored := ldapConfigStore.config
		ldapConfigStore.RUnlock()
		if stored == nil {
			writeError(w, http.StatusBadRequest, "no LDAP config provided and none saved")
			return
		}
		cfg = *stored
	}

	start := time.Now()

	// Attempt real LDAP connection using pkg/authprovider
	result, err := h.testLDAPConnection(r.Context(), &cfg)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":      "failed",
			"error":       err.Error(),
			"latency_ms":  latency,
			"users_found": 0,
			"groups_found": 0,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":       "ok",
		"latency_ms":   latency,
		"users_found":  result.usersFound,
		"groups_found": result.groupsFound,
		"server_url":   cfg.ServerURL,
	})
}

type ldapTestResult struct {
	usersFound  int
	groupsFound int
}

// handleLDAPSync triggers an LDAP user/group sync run.
// POST /api/v1/identity/ldap/sync
func (h *HTTPHandler) handleLDAPSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ldapConfigStore.RLock()
	cfg := ldapConfigStore.config
	ldapConfigStore.RUnlock()

	if cfg == nil {
		writeError(w, http.StatusBadRequest, "LDAP sync config not set — configure and save first")
		return
	}

	// Set sync state to in_progress
	ldapSyncState.Lock()
	ldapSyncState.status = "in_progress"
	ldapSyncState.errs = []map[string]any{}
	ldapSyncState.Unlock()

	// Run sync synchronously (could be background job in production)
	result, errs := h.runLDAPSync(r.Context(), cfg)

	ldapSyncState.Lock()
	ldapSyncState.lastRun = time.Now()
	ldapSyncState.synced = result.usersFound
	ldapSyncState.totalFound = result.usersFound
	ldapSyncState.errs = errs
	if len(errs) > 0 && result.usersFound == 0 {
		ldapSyncState.status = "failed"
	} else {
		ldapSyncState.status = "success"
	}
	finalStatus := ldapSyncState.status
	finalSynced := ldapSyncState.synced
	finalErrors := ldapSyncState.errs
	ldapSyncState.Unlock()

	// Record in sync history
	addSyncHistory(syncHistoryEntry{
		ID:          fmt.Sprintf("sync-%d", time.Now().UnixNano()),
		StartedAt:   time.Now().UTC(),
		Status:      finalStatus,
		SyncedUsers: finalSynced,
		Errors:      len(finalErrors),
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"status":       finalStatus,
		"synced_users": finalSynced,
		"errors":       finalErrors,
		"synced_at":    time.Now().UTC().Format(time.RFC3339),
	})
}

// handleLDAPSyncStatus returns the current LDAP sync state.
// GET /api/v1/identity/ldap/sync-status
func (h *HTTPHandler) handleLDAPSyncStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ldapSyncState.RLock()
	resp := map[string]any{
		"provider":    "ldap",
		"status":      ldapSyncState.status,
		"last_sync":   ldapSyncState.lastRun.Format(time.RFC3339),
		"synced_users": ldapSyncState.synced,
		"total_users":  ldapSyncState.totalFound,
		"errors":       ldapSyncState.errs,
	}
	ldapSyncState.RUnlock()

	ldapConfigStore.RLock()
	if ldapConfigStore.config != nil {
		resp["configured"] = true
		resp["server_url"] = ldapConfigStore.config.ServerURL
		resp["schedule"] = ldapConfigStore.config.SyncSchedule
	} else {
		resp["configured"] = false
	}
	ldapConfigStore.RUnlock()

	writeJSON(w, http.StatusOK, resp)
}
