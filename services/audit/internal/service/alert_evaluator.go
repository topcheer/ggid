package service

import (
	"strings"
	"sync"
	"time"
)

type AlertRule struct {
	Name      string `json:"name"`
	Condition string `json:"condition"`
	Severity  string `json:"severity"`
	Channel   string `json:"channel"`
}

type AuditEvent struct {
	EventType string    `json:"event_type"`
	UserID    string    `json:"user_id"`
	TenantID  string    `json:"tenant_id"`
	Timestamp time.Time `json:"timestamp"`
	Details   string    `json:"details"`
}

type TriggeredAlert struct {
	RuleName    string      `json:"rule_name"`
	Severity    string      `json:"severity"`
	Channel     string      `json:"channel"`
	Events      []AuditEvent `json:"events"`
	TriggeredAt time.Time   `json:"triggered_at"`
	Fingerprint string      `json:"fingerprint"`
	Escalated   bool        `json:"escalated"`
}

type AlertEvaluator struct {
	mu    sync.RWMutex
	rules []AlertRule
}

func NewAlertEvaluator(rules []AlertRule) *AlertEvaluator {
	return &AlertEvaluator{rules: rules}
}

func (ae *AlertEvaluator) EvaluateRules(events []AuditEvent) []TriggeredAlert {
	ae.mu.RLock()
	defer ae.mu.RUnlock()
	var alerts []TriggeredAlert
	for _, rule := range ae.rules {
		var matched []AuditEvent
		for _, evt := range events {
			if matchCondition(rule.Condition, evt) {
				matched = append(matched, evt)
			}
		}
		if len(matched) > 0 {
			alerts = append(alerts, TriggeredAlert{
				RuleName:    rule.Name,
				Severity:    rule.Severity,
				Channel:     rule.Channel,
				Events:      matched,
				TriggeredAt: time.Now(),
				Fingerprint: fingerprint(rule.Name, matched),
			})
		}
	}
	return alerts
}

func (ae *AlertEvaluator) CorrelateEvents(window time.Duration, events []AuditEvent) [][]AuditEvent {
	var groups [][]AuditEvent
	used := make([]bool, len(events))
	for i, evt := range events {
		if used[i] {
			continue
		}
		var group []AuditEvent
		group = append(group, evt)
		used[i] = true
		for j := i + 1; j < len(events); j++ {
			if used[j] {
				continue
			}
			if events[j].UserID == evt.UserID && events[j].Timestamp.Sub(evt.Timestamp) <= window {
				group = append(group, events[j])
				used[j] = true
			}
		}
		if len(group) > 1 {
			groups = append(groups, group)
		}
	}
	return groups
}

func (ae *AlertEvaluator) DedupAlerts(alerts []TriggeredAlert) []TriggeredAlert {
	seen := make(map[string]bool)
	var deduped []TriggeredAlert
	for _, a := range alerts {
		if seen[a.Fingerprint] {
			continue
		}
		seen[a.Fingerprint] = true
		deduped = append(deduped, a)
	}
	return deduped
}

func (ae *AlertEvaluator) EscalateAlert(alert *TriggeredAlert, level string) {
	alert.Severity = level
	alert.Escalated = true
}

func matchCondition(condition string, evt AuditEvent) bool {
	// Simple condition matching: "failed_logins > 10 in 5m" matches event_type containing "failed_login"
	cond := strings.ToLower(condition)
	et := strings.ToLower(evt.EventType)
	if strings.Contains(cond, "failed_login") && strings.Contains(et, "failed_login") {
		return true
	}
	if strings.Contains(cond, "privilege_escalation") && strings.Contains(et, "role_change") {
		return true
	}
	if strings.Contains(cond, "brute_force") && strings.Contains(et, "auth_fail") {
		return true
	}
	return false
}

func fingerprint(ruleName string, events []AuditEvent) string {
	if len(events) == 0 {
		return ruleName
	}
	return ruleName + ":" + events[0].UserID
}