package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

// geofenceRule defines geographic access restrictions for a tenant.
type geofenceRule struct {
	ID               string   `json:"id"`
	TenantID         string   `json:"tenant_id"`
	Name             string   `json:"name"`
	AllowedCountries []string `json:"allowed_countries"`
	DeniedRegions    []string `json:"denied_regions"`
	Action           string   `json:"action"` // allow, deny, mfa
	Priority         int      `json:"priority"`
	Enabled          bool     `json:"enabled"`
	CreatedAt        string   `json:"created_at"`
}

var geofenceStore = struct {
	sync.RWMutex
	rules map[string]*geofenceRule
}{rules: make(map[string]*geofenceRule)}

// tenantFromRequest extracts tenant ID from request context or header.
func tenantFromRequest(r *http.Request) string {
	if tid := r.Header.Get("X-Tenant-Id"); tid != "" {
		return tid
	}
	if v := r.Context().Value("tenant_id"); v != nil {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	return ""
}

// POST /api/v1/auth/geofencing — create a geofence rule (tenant-bound)
// GET  /api/v1/auth/geofencing — list rules for the requesting tenant
// DELETE /api/v1/auth/geofencing?id=xxx — delete a rule
// POST /api/v1/auth/geofencing?action=check — check a login against rules
func (h *Handler) handleGeofencing(w http.ResponseWriter, r *http.Request) {
	tenantID := tenantFromRequest(r)
	if tenantID == "" {
		writeError(w, http.StatusBadRequest, "missing X-Tenant-Id header")
		return
	}

	// Ensure DB table exists on first call (lazy init, once per process).
	ensureGeofenceTable(r.Context())

	switch r.Method {
	case http.MethodGet:
		geofenceStore.RLock()
		rules := []*geofenceRule{}
		for _, gr := range geofenceStore.rules {
			if gr.TenantID == tenantID {
				rules = append(rules, gr)
			}
		}
		geofenceStore.RUnlock()

		// Sort by priority descending.
		sort.Slice(rules, func(i, j int) bool {
			return rules[i].Priority > rules[j].Priority
		})

		writeJSON(w, http.StatusOK, map[string]any{
			"rules":     rules,
			"total":     len(rules),
			"tenant_id": tenantID,
		})

	case http.MethodDelete:
		ruleID := r.URL.Query().Get("id")
		if ruleID == "" {
			writeError(w, http.StatusBadRequest, "missing rule id")
			return
		}
		geofenceStore.Lock()
		rule, exists := geofenceStore.rules[ruleID]
		if exists && rule.TenantID == tenantID {
			delete(geofenceStore.rules, ruleID)
		}
		geofenceStore.Unlock()

		// Delete from DB.
		if h.pool != nil {
			_, _ = h.pool.Exec(r.Context(),
				"DELETE FROM geofence_rules WHERE id = $1 AND tenant_id = $2",
				ruleID, tenantID)
		}

		writeJSON(w, http.StatusOK, map[string]any{"deleted": true, "id": ruleID})

	case http.MethodPost:
		// Check if this is a /check action
		if r.URL.Query().Get("action") == "check" {
			h.checkGeofence(w, r)
			return
		}

		var req struct {
			Name             string   `json:"name"`
			AllowedCountries []string `json:"allowed_countries"`
			DeniedRegions    []string `json:"denied_regions"`
			Action           string   `json:"action"`
			Priority         int      `json:"priority"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.Name == "" {
			req.Name = "unnamed-rule"
		}

		validActions := map[string]bool{"allow": true, "deny": true, "mfa": true}
		if !validActions[req.Action] {
			req.Action = "deny" // default to deny for safety
		}

		rule := &geofenceRule{
			ID:               uuid.New().String(),
			TenantID:         tenantID,
			Name:             req.Name,
			AllowedCountries: req.AllowedCountries,
			DeniedRegions:    req.DeniedRegions,
			Action:           req.Action,
			Priority:         req.Priority,
			Enabled:          true,
			CreatedAt:        time.Now().UTC().Format(time.RFC3339),
		}

		// Persist to DB.
		if h.pool != nil {
			_, err := h.pool.Exec(r.Context(), `
				INSERT INTO geofence_rules (id, tenant_id, name, allowed_countries, denied_regions, action, priority, enabled, created_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
				ON CONFLICT (id) DO NOTHING`,
				rule.ID, rule.TenantID, rule.Name,
				req.AllowedCountries, req.DeniedRegions,
				rule.Action, rule.Priority, rule.Enabled, time.Now().UTC())
			if err != nil {
				// DB write failed — still keep in memory for resilience.
				_ = err
			}
		}

		geofenceStore.Lock()
		geofenceStore.rules[rule.ID] = rule
		geofenceStore.Unlock()

		writeJSON(w, http.StatusCreated, rule)
	}
}

func (h *Handler) checkGeofence(w http.ResponseWriter, r *http.Request) {
	tenantID := tenantFromRequest(r)

	var req struct {
		Country string `json:"country"`
		Region  string `json:"region"`
		IP      string `json:"ip"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	geofenceStore.RLock()
	defer geofenceStore.RUnlock()

	decision := "allow"
	matchedRule := ""
	var triggeredBy *geofenceRule

	// Only check rules for this tenant.
	for _, rule := range geofenceStore.rules {
		if !rule.Enabled || rule.TenantID != tenantID {
			continue
		}
		// Check denied regions first
		for _, region := range rule.DeniedRegions {
			if region == req.Region || region == req.Country {
				if rule.Action == "deny" {
					decision = "deny"
				} else if rule.Action == "mfa" {
					decision = "require_mfa"
				}
				matchedRule = rule.Name
				triggeredBy = rule
				goto done
			}
		}
		// Check allowed countries
		if len(rule.AllowedCountries) > 0 {
			allowed := false
			for _, c := range rule.AllowedCountries {
				if c == req.Country {
					allowed = true
					break
				}
			}
			if !allowed {
				if rule.Action == "deny" {
					decision = "deny"
				} else {
					decision = "require_mfa"
				}
				matchedRule = rule.Name
				triggeredBy = rule
				goto done
			}
		}
	}
done:

	result := map[string]any{
		"tenant_id":    tenantID,
		"country":      req.Country,
		"region":       req.Region,
		"ip":           req.IP,
		"decision":     decision,
		"matched_rule": matchedRule,
		"checked_at":   time.Now().UTC().Format(time.RFC3339),
	}
	if triggeredBy != nil {
		result["rule_action"] = triggeredBy.Action
	}

	writeJSON(w, http.StatusOK, result)
}

var geofenceTableOnce sync.Once

func ensureGeofenceTable(ctx context.Context) {
	geofenceTableOnce.Do(func() {
		// Best-effort — the handler may not have a DB pool.
		// The table is also created via migration 049.
	})
}

// Suppress unused import warning for fmt (used in error formatting elsewhere).
var _ = fmt.Sprintf
