package server

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// geofenceRule defines geographic access restrictions.
type geofenceRule struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	AllowedCountries []string `json:"allowed_countries"`
	DeniedRegions   []string `json:"denied_regions"`
	Action          string   `json:"action"` // allow, deny, mfa
	Priority        int      `json:"priority"`
	Enabled         bool     `json:"enabled"`
	CreatedAt       string   `json:"created_at"`
}

var geofenceStore = struct {
	sync.RWMutex
	rules map[string]*geofenceRule
}{rules: make(map[string]*geofenceRule)}

// POST /api/v1/auth/geofencing — create a geofence rule
// GET  /api/v1/auth/geofencing — list all rules
// POST /api/v1/auth/geofencing/check — check a login against rules
func (h *Handler) handleGeofencing(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		geofenceStore.RLock()
		rules := []*geofenceRule{}
		for _, gr := range geofenceStore.rules {
			rules = append(rules, gr)
		}
		geofenceStore.RUnlock()

		writeJSON(w, http.StatusOK, map[string]any{
			"rules": rules,
			"total": len(rules),
		})

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
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
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
			Name:             req.Name,
			AllowedCountries: req.AllowedCountries,
			DeniedRegions:    req.DeniedRegions,
			Action:           req.Action,
			Priority:         req.Priority,
			Enabled:          true,
			CreatedAt:        time.Now().UTC().Format(time.RFC3339),
		}

		geofenceStore.Lock()
		geofenceStore.rules[rule.ID] = rule
		geofenceStore.Unlock()

		writeJSON(w, http.StatusCreated, rule)
	}
}

func (h *Handler) checkGeofence(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Country string `json:"country"`
		Region  string `json:"region"`
		IP      string `json:"ip"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	geofenceStore.RLock()
	defer geofenceStore.RUnlock()

	decision := "allow"
	matchedRule := ""
	var triggeredBy *geofenceRule

	for _, rule := range geofenceStore.rules {
		if !rule.Enabled {
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
