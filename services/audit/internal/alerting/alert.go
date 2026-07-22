// Package alerting provides real-time alerting for audit events.
//
// An AlertEngine evaluates incoming audit events against AlertRules.
// When a rule's condition is met (e.g., threshold exceeded), the engine
// fires notifications to configured channels (webhook, email).
package alerting

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// AlertCondition defines the event field and comparison operator.
type AlertCondition struct {
	Field    string `json:"field"`    // e.g., "action", "user_id", "ip_address"
	Operator string `json:"operator"` // "eq", "neq", "contains", "gt", "lt"
	Value    any    `json:"value"`    // comparison value
}

// AlertAction defines what happens when a rule fires.
type AlertAction struct {
	Type   string            `json:"type"`   // "webhook", "email"
	Target string            `json:"target"` // URL or email address
	Params map[string]string `json:"params"` // extra headers, subject, etc.
}

// AlertRule defines a single alerting rule.
type AlertRule struct {
	ID         string          `json:"id"`
	TenantID   string          `json:"tenant_id"`
	Name       string          `json:"name"`
	Condition  AlertCondition  `json:"condition"`
	Threshold  int             `json:"threshold"`     // fire after N matches
	Window     time.Duration   `json:"window"`        // within this time window
	Actions    []AlertAction   `json:"actions"`
	Enabled    bool            `json:"enabled"`
}

// AlertEvent represents an audit event to evaluate.
type AlertEvent struct {
	TenantID string         `json:"tenant_id"`
	Action   string         `json:"action"`
	UserID   string         `json:"user_id"`
	IPAddress string        `json:"ip_address"`
	Timestamp time.Time     `json:"timestamp"`
	Fields    map[string]any `json:"fields"`
}

// Alert represents a fired alert.
type Alert struct {
	RuleID    string    `json:"rule_id"`
	RuleName  string    `json:"rule_name"`
	TenantID  string    `json:"tenant_id"`
	Trigger   string    `json:"trigger"` // what matched
	Count     int       `json:"count"`
	FiredAt   time.Time `json:"fired_at"`
}

// Notifier sends alert notifications.
type Notifier interface {
	Notify(ctx context.Context, alert *Alert, actions []AlertAction) error
}

// WebhookNotifier sends alerts via HTTP POST with HMAC-SHA256 signature.
// The signature header X-GGID-Signature allows receivers to verify authenticity.
type WebhookNotifier struct {
	URL    string
	Secret string // HMAC signing secret (from ALERT_WEBHOOK_SECRET env)
	client *http.Client
}

func (w *WebhookNotifier) Notify(ctx context.Context, alert *Alert, actions []AlertAction) error {
	if w.URL == "" {
		return nil
	}
	if w.client == nil {
		w.client = &http.Client{Timeout: 10 * time.Second}
	}

	payload := map[string]any{
		"rule_id":   alert.RuleID,
		"rule_name": alert.RuleName,
		"tenant_id": alert.TenantID,
		"trigger":   alert.Trigger,
		"count":     alert.Count,
		"fired_at":  alert.FiredAt.Format(time.RFC3339),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal alert: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", w.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// HMAC-SHA256 signature for webhook verification.
	if w.Secret != "" {
		mac := hmac.New(sha256.New, []byte(w.Secret))
		mac.Write(body)
		sig := hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-GGID-Signature", "sha256="+sig)
	}

	resp, err := w.client.Do(req)
	if err != nil {
		slog.Error("webhook alert failed", "url", w.URL, "error", err)
		return err
	}
	resp.Body.Close()

	if resp.StatusCode >= 400 {
		slog.Warn("webhook returned error status", "url", w.URL, "status", resp.StatusCode)
	} else {
		slog.Info("webhook alert sent", "rule", alert.RuleName, "url", w.URL, "status", resp.StatusCode)
	}
	return nil
}

// AlertEngine evaluates events against rules.
type AlertEngine struct {
	mu     sync.RWMutex
	rules  map[string]*AlertRule
	counts map[string][]time.Time // rule_id → match timestamps
	notifier Notifier
}

// NewAlertEngine creates a new engine.
func NewAlertEngine(notifier Notifier) *AlertEngine {
	return &AlertEngine{
		rules:    make(map[string]*AlertRule),
		counts:   make(map[string][]time.Time),
		notifier: notifier,
	}
}

// AddRule registers an alert rule.
func (e *AlertEngine) AddRule(rule *AlertRule) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.rules[rule.ID] = rule
}

// RemoveRule removes a rule by ID.
func (e *AlertEngine) RemoveRule(ruleID string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.rules, ruleID)
	delete(e.counts, ruleID)
}

// ListRules returns all rules for a tenant.
func (e *AlertEngine) ListRules(tenantID string) []*AlertRule {
	e.mu.RLock()
	defer e.mu.RUnlock()
	var result []*AlertRule
	for _, r := range e.rules {
		if r.TenantID == tenantID {
			result = append(result, r)
		}
	}
	return result
}

// Evaluate checks an event against all rules. Returns any alerts fired.
func (e *AlertEngine) Evaluate(ctx context.Context, event *AlertEvent) []*Alert {
	e.mu.Lock()
	defer e.mu.Unlock()

	var fired []*Alert
	now := time.Now()

	for _, rule := range e.rules {
		if !rule.Enabled {
			continue
		}
		if rule.TenantID != event.TenantID {
			continue
		}
		if !matchesCondition(rule.Condition, event) {
			continue
		}

		// Track match timestamp
		e.counts[rule.ID] = append(e.counts[rule.ID], now)

		// Prune old matches outside window
		windowStart := now.Add(-rule.Window)
		filtered := e.counts[rule.ID][:0]
		for _, ts := range e.counts[rule.ID] {
			if ts.After(windowStart) {
				filtered = append(filtered, ts)
			}
		}
		e.counts[rule.ID] = filtered

		// Check threshold
		if len(e.counts[rule.ID]) >= rule.Threshold {
			alert := &Alert{
				RuleID:   rule.ID,
				RuleName: rule.Name,
				TenantID: rule.TenantID,
				Trigger:  fmt.Sprintf("%s %s %v", rule.Condition.Field, rule.Condition.Operator, rule.Condition.Value),
				Count:    len(e.counts[rule.ID]),
				FiredAt:  now,
			}
			fired = append(fired, alert)

			// Fire notifications
			if e.notifier != nil {
				if err := e.notifier.Notify(ctx, alert, rule.Actions); err != nil {
					slog.Error("alert notification failed", "rule", rule.Name, "error", err)
				}
			}

			// Reset counter after firing
			e.counts[rule.ID] = nil
		}
	}

	return fired
}

// matchesCondition checks if an event matches a condition.
func matchesCondition(cond AlertCondition, event *AlertEvent) bool {
	var val any
	switch cond.Field {
	case "action":
		val = event.Action
	case "user_id":
		val = event.UserID
	case "ip_address":
		val = event.IPAddress
	default:
		if event.Fields != nil {
			val = event.Fields[cond.Field]
		}
	}

	switch cond.Operator {
	case "eq":
		return fmt.Sprintf("%v", val) == fmt.Sprintf("%v", cond.Value)
	case "neq":
		return fmt.Sprintf("%v", val) != fmt.Sprintf("%v", cond.Value)
	case "contains":
		return contains(fmt.Sprintf("%v", val), fmt.Sprintf("%v", cond.Value))
	default:
		return false
	}
}

func contains(s, substr string) bool {
	if substr == "" {
		return true
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
