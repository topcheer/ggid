package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NotificationChannel defines a delivery channel.
type NotificationChannel string

const (
	ChEmail     NotificationChannel = "email"
	ChSMS       NotificationChannel = "sms"
	ChSlack     NotificationChannel = "slack"
	ChTeams     NotificationChannel = "teams"
	ChPagerDuty NotificationChannel = "pagerduty"
	ChWebhook   NotificationChannel = "webhook"
	ChInApp     NotificationChannel = "in_app"
)

// NotificationRule defines severity-based routing.
type NotificationRule struct {
	ID         string              `json:"id"`
	Severity   string              `json:"severity"` // critical, high, medium, low
	Channels   []NotificationChannel `json:"channels"`
	Enabled    bool                `json:"enabled"`
	CreatedAt  time.Time           `json:"created_at"`
}

// NotificationLogEntry records a sent notification.
type NotificationLogEntry struct {
	ID        string    `json:"id"`
	Rule      string    `json:"rule"`
	Severity  string    `json:"severity"`
	Channel   string    `json:"channel"`
	Subject   string    `json:"subject"`
	Status    string    `json:"status"` // sent, failed, escalated
	SentAt    time.Time `json:"sent_at"`
}

// notificationRepo manages notification rules + log in PG.
type notificationRepo struct {
	pool *pgxpool.Pool
}

func newNotificationRepo(pool *pgxpool.Pool) *notificationRepo {
	return &notificationRepo{pool: pool}
}

func (r *notificationRepo) EnsureSchema(ctx context.Context) error {
	if r.pool == nil { return nil }
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS notification_rules (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			severity TEXT NOT NULL, channels TEXT[] NOT NULL,
			enabled BOOLEAN DEFAULT TRUE, created_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE TABLE IF NOT EXISTS notification_log (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			rule TEXT, severity TEXT, channel TEXT,
			subject TEXT, status TEXT DEFAULT 'sent',
			sent_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_notif_log_sent ON notification_log(sent_at DESC);
	`)
	return err
}

func (r *notificationRepo) CreateRule(ctx context.Context, rule *NotificationRule) error {
	if r.pool == nil { return nil }
	if rule.ID == "" { rule.ID = uuid.New().String() }
	channels := make([]string, len(rule.Channels))
	for i, c := range rule.Channels { channels[i] = string(c) }
	_, err := r.pool.Exec(ctx,
		`INSERT INTO notification_rules (severity,channels,enabled) VALUES ($1,$2,$3)`,
		rule.Severity, channels, rule.Enabled)
	return err
}

func (r *notificationRepo) ListRules(ctx context.Context) ([]*NotificationRule, error) {
	if r.pool == nil { return []*NotificationRule{}, nil }
	rows, err := r.pool.Query(ctx, `SELECT id,severity,channels,enabled,created_at FROM notification_rules ORDER BY created_at DESC`)
	if err != nil { return nil, err }
	defer rows.Close()
	var result []*NotificationRule
	for rows.Next() {
		rule := &NotificationRule{}
		var channels []string
		if err := rows.Scan(&rule.ID, &rule.Severity, &channels, &rule.Enabled, &rule.CreatedAt); err != nil { continue }
		rule.Channels = make([]NotificationChannel, len(channels))
		for i, c := range channels { rule.Channels[i] = NotificationChannel(c) }
		result = append(result, rule)
	}
	return result, nil
}

func (r *notificationRepo) LogNotification(ctx context.Context, entry *NotificationLogEntry) {
	if r.pool == nil || entry == nil { return }
	if entry.ID == "" { entry.ID = uuid.New().String() }
	r.pool.Exec(ctx,
		`INSERT INTO notification_log (rule,severity,channel,subject,status) VALUES ($1,$2,$3,$4,$5)`,
		entry.Rule, entry.Severity, entry.Channel, entry.Subject, entry.Status)
}

func (r *notificationRepo) ListLog(ctx context.Context, limit int) ([]*NotificationLogEntry, error) {
	if r.pool == nil { return []*NotificationLogEntry{}, nil }
	if limit <= 0 || limit > 100 { limit = 50 }
	rows, err := r.pool.Query(ctx,
		`SELECT id,rule,severity,channel,subject,status,sent_at FROM notification_log ORDER BY sent_at DESC LIMIT $1`, limit)
	if err != nil { return nil, err }
	defer rows.Close()
	var result []*NotificationLogEntry
	for rows.Next() {
		e := &NotificationLogEntry{}
		if err := rows.Scan(&e.ID, &e.Rule, &e.Severity, &e.Channel, &e.Subject, &e.Status, &e.SentAt); err != nil { continue }
		result = append(result, e)
	}
	return result, nil
}

// DefaultSeverityRouting returns channels for each severity level.
func DefaultSeverityRouting(severity string) []NotificationChannel {
	switch severity {
	case "critical":
		return []NotificationChannel{ChEmail, ChSMS, ChSlack, ChTeams, ChPagerDuty, ChWebhook, ChInApp}
	case "high":
		return []NotificationChannel{ChSlack, ChEmail}
	case "medium":
		return []NotificationChannel{ChEmail}
	case "low":
		return []NotificationChannel{ChInApp}
	default:
		return []NotificationChannel{ChInApp}
	}
}

// --- HTTP Handlers ---

func (h *HTTPHandler) handleNotificationRules(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var rule NotificationRule
		if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		validSev := map[string]bool{"critical": true, "high": true, "medium": true, "low": true}
		if !validSev[rule.Severity] {
			writeJSONError(w, http.StatusBadRequest, "severity must be critical, high, medium, or low")
			return
		}
		if len(rule.Channels) == 0 {
			rule.Channels = DefaultSeverityRouting(rule.Severity)
		}
		rule.Enabled = true
		rule.CreatedAt = time.Now().UTC()
		if h.notificationRepo != nil {
			h.notificationRepo.CreateRule(r.Context(), &rule)
		}
		writeJSON(w, http.StatusCreated, rule)
	case http.MethodGet:
		var rules []*NotificationRule
		if h.notificationRepo != nil {
			rules, _ = h.notificationRepo.ListRules(r.Context())
		}
		if rules == nil { rules = []*NotificationRule{} }
		writeJSON(w, http.StatusOK, map[string]any{"rules": rules, "count": len(rules)})
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *HTTPHandler) handleNotificationLog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var log []*NotificationLogEntry
	if h.notificationRepo != nil {
		log, _ = h.notificationRepo.ListLog(r.Context(), 50)
	}
	if log == nil { log = []*NotificationLogEntry{} }
	writeJSON(w, http.StatusOK, map[string]any{"notifications": log, "count": len(log)})
}

func (h *HTTPHandler) SetNotificationRepo(repo *notificationRepo) {
	h.notificationRepo = repo
}

var _ = fmt.Sprintf
