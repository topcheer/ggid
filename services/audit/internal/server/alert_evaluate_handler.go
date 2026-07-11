package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ggid/ggid/services/audit/internal/domain"
	"github.com/google/uuid"
)

// POST /api/v1/audit/alerts/evaluate — manually trigger alert rule evaluation.
// Body: {"rule_id": "...", "time_range": "1h", "tenant_id": "..."}
// Returns triggered alerts with details.
func (s *HTTPServer) handleAlertEvaluate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		RuleID    string `json:"rule_id"`
		TimeRange string `json:"time_range"`
		TenantID  string `json:"tenant_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.TenantID == "" {
		writeJSONError(w, http.StatusBadRequest, "tenant_id is required")
		return
	}
	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid tenant_id")
		return
	}

	if req.TimeRange == "" {
		req.TimeRange = "1h"
	}
	dur, err := time.ParseDuration(req.TimeRange)
	if err != nil || dur <= 0 {
		dur = time.Hour
	}

	since := time.Now().UTC().Add(-dur)
	var triggered []map[string]any

	if req.RuleID != "" {
		// Evaluate specific rule
		for _, rule := range anomalyRules {
			ruleID, _ := rule["id"].(string)
			if ruleID != req.RuleID {
				continue
			}
			alert := s.evaluateRule(r, rule, tenantID, since)
			if alert != nil {
				triggered = append(triggered, alert)
			}
			break
		}
	} else {
		// Evaluate all rules
		for _, rule := range anomalyRules {
			alert := s.evaluateRule(r, rule, tenantID, since)
			if alert != nil {
				triggered = append(triggered, alert)
			}
		}
	}

	if triggered == nil {
		triggered = []map[string]any{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"rule_id":     req.RuleID,
		"time_range":  req.TimeRange,
		"evaluated_at": time.Now().UTC().Format(time.RFC3339),
		"total":       len(triggered),
		"alerts":      triggered,
	})
}

// evaluateRule evaluates a single anomaly rule against recent events.
func (s *HTTPServer) evaluateRule(r *http.Request, rule map[string]any, tenantID uuid.UUID, since time.Time) map[string]any {
	action, _ := rule["action"].(string)
	threshold, _ := rule["threshold"].(int)
	windowMins, _ := rule["window_minutes"].(int)
	if threshold == 0 {
		threshold = 5
	}
	if windowMins == 0 {
		windowMins = 5
	}

	windowStart := time.Now().UTC().Add(-time.Duration(windowMins) * time.Minute)
	if since.After(windowStart) {
		windowStart = since
	}

	events, _, err := s.svc.ListEvents(r.Context(), domain.ListFilter{
		TenantID:  tenantID,
		Action:    action,
		StartTime: &windowStart,
	}, 1, threshold+10)
	if err != nil {
		return nil
	}

	if len(events) >= threshold {
		alert := map[string]any{
			"rule_id":      rule["id"],
			"rule_name":    rule["name"],
			"severity":     rule["severity"],
			"count":        len(events),
			"threshold":    threshold,
			"window_mins":  windowMins,
			"action":       action,
			"triggered_at": time.Now().UTC().Format(time.RFC3339),
			"message":      formatAlertMessage(len(events), action, windowMins, threshold),
		}
		s.dispatchAlert(alert)
		return alert
	}

	return nil
}

func formatAlertMessage(count int, action string, windowMins, threshold int) string {
	return fmt.Sprintf("%d '%s' events in %d minutes (threshold: %d)", count, action, windowMins, threshold)
}
