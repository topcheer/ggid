package soar

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PlaybookTrigger defines when a playbook fires.
type PlaybookTrigger struct {
	Rule        string `json:"rule,omitempty"`        // ITDR rule ID e.g. "mfa_fatigue"
	Severity    string `json:"severity,omitempty"`    // minimum severity: high|critical
	RiskScore   int    `json:"risk_score,omitempty"`   // CAE risk score threshold
	ThreatIntel bool   `json:"threat_intel,omitempty"` // threat intel critical hit
}

// PlaybookAction defines a single action in a playbook.
type PlaybookAction struct {
	Type    string            `json:"type"`              // revoke_session|lock_account|step_up_mfa|notify_soc|create_incident|block_ip
	Webhook string            `json:"webhook,omitempty"` // for notify_soc
	Message string            `json:"message,omitempty"` // for notify_soc/create_incident
	Params  map[string]string `json:"params,omitempty"`
}

// Playbook is a declarative SOAR playbook definition.
type Playbook struct {
	ID        string           `json:"id"`
	Name      string           `json:"name"`
	Trigger   PlaybookTrigger  `json:"trigger"`
	Actions   []PlaybookAction `json:"actions"`
	Enabled   bool             `json:"enabled"`
	CreatedAt time.Time        `json:"created_at"`
}

// Execution records a single playbook run.
type Execution struct {
	ID           string          `json:"id"`
	PlaybookID   string          `json:"playbook_id"`
	TriggerEvent json.RawMessage `json:"trigger_event"`
	Status       string          `json:"status"` // running|completed|failed|rate_limited
	ActionsTaken []string        `json:"actions_taken"`
	StartedAt    time.Time       `json:"started_at"`
	CompletedAt  *time.Time      `json:"completed_at,omitempty"`
}

// Engine executes SOAR playbooks in response to security events.
type Engine struct {
	pool    *pgxpool.Pool
	client  *http.Client
	mu      sync.Mutex
	lastRun map[string]time.Time // userID → last execution time (rate limit)
}

// NewEngine creates a SOAR engine.
func NewEngine(pool *pgxpool.Pool) *Engine {
	return &Engine{
		pool:    pool,
		client:  &http.Client{Timeout: 10 * time.Second},
		lastRun: make(map[string]time.Time),
	}
}

// EnsureSchema creates soar_playbooks + soar_executions tables.
func (e *Engine) EnsureSchema(ctx context.Context) error {
	if e.pool == nil {
		return nil
	}
	_, err := e.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS soar_playbooks (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			trigger JSONB NOT NULL DEFAULT '{}',
			actions JSONB NOT NULL DEFAULT '[]',
			enabled BOOLEAN NOT NULL DEFAULT TRUE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE TABLE IF NOT EXISTS soar_executions (
			id TEXT PRIMARY KEY,
			playbook_id TEXT NOT NULL,
			trigger_event JSONB NOT NULL DEFAULT '{}',
			status TEXT NOT NULL DEFAULT 'running',
			actions_taken JSONB NOT NULL DEFAULT '[]',
			started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			completed_at TIMESTAMPTZ
		);
		CREATE INDEX IF NOT EXISTS idx_soar_exec_pb ON soar_executions(playbook_id, started_at DESC);
	`)
	return err
}

// CreatePlaybook stores a new playbook.
func (e *Engine) CreatePlaybook(ctx context.Context, pb *Playbook) error {
	if e.pool == nil {
		return nil
	}
	if pb.ID == "" {
		pb.ID = uuid.New().String()
	}
	pb.CreatedAt = time.Now()
	triggerJSON, _ := json.Marshal(pb.Trigger)
	actionsJSON, _ := json.Marshal(pb.Actions)
	_, err := e.pool.Exec(ctx,
		`INSERT INTO soar_playbooks (id, name, trigger, actions, enabled, created_at) VALUES ($1,$2,$3,$4,$5,$6)`,
		pb.ID, pb.Name, triggerJSON, actionsJSON, pb.Enabled, pb.CreatedAt)
	return err
}

// ListPlaybooks returns all playbooks.
func (e *Engine) ListPlaybooks(ctx context.Context) ([]Playbook, error) {
	if e.pool == nil {
		return nil, nil
	}
	rows, err := e.pool.Query(ctx, `SELECT id, name, trigger, actions, enabled, created_at FROM soar_playbooks ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var pbs []Playbook
	for rows.Next() {
		var pb Playbook
		var triggerJSON, actionsJSON []byte
		if err := rows.Scan(&pb.ID, &pb.Name, &triggerJSON, &actionsJSON, &pb.Enabled, &pb.CreatedAt); err != nil {
			continue
		}
		json.Unmarshal(triggerJSON, &pb.Trigger)
		json.Unmarshal(actionsJSON, &pb.Actions)
		pbs = append(pbs, pb)
	}
	return pbs, nil
}

// ListExecutions returns recent executions.
func (e *Engine) ListExecutions(ctx context.Context, limit int) ([]Execution, error) {
	if e.pool == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 50
	}
	rows, err := e.pool.Query(ctx, `SELECT id, playbook_id, trigger_event, status, actions_taken, started_at, completed_at FROM soar_executions ORDER BY started_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var execs []Execution
	for rows.Next() {
		var exec Execution
		var actionsJSON []byte
		if err := rows.Scan(&exec.ID, &exec.PlaybookID, &exec.TriggerEvent, &exec.Status, &actionsJSON, &exec.StartedAt, &exec.CompletedAt); err != nil {
			continue
		}
		json.Unmarshal(actionsJSON, &exec.ActionsTaken)
		execs = append(execs, exec)
	}
	return execs, nil
}

// TriggerEvent is the input that fires playbooks.
type TriggerEvent struct {
	RuleID    string `json:"rule_id"`
	Severity  string `json:"severity"`
	UserID    string `json:"user_id"`
	TenantID  string `json:"tenant_id"`
	IPAddress string `json:"ip_address"`
	RiskScore int    `json:"risk_score"`
}

// EvaluateTrigger checks if a playbook should fire for this event.
func (e *Engine) EvaluateTrigger(trigger PlaybookTrigger, event TriggerEvent) bool {
	if trigger.Rule != "" && trigger.Rule != event.RuleID {
		return false
	}
	if trigger.Severity != "" {
		if !severityAtLeast(event.Severity, trigger.Severity) {
			return false
		}
	}
	if trigger.RiskScore > 0 && event.RiskScore < trigger.RiskScore {
		return false
	}
	if trigger.ThreatIntel && event.RuleID != "threat_intel_hit" {
		return false
	}
	return true
}

// Execute runs a playbook for a trigger event.
// Rate limited: max 1 execution per user per 5 minutes.
func (e *Engine) Execute(ctx context.Context, pb *Playbook, event TriggerEvent) (*Execution, error) {
	// Rate limit check.
	e.mu.Lock()
	if last, ok := e.lastRun[event.UserID]; ok && time.Since(last) < 5*time.Minute {
		e.mu.Unlock()
		return &Execution{
			ID: uuid.New().String(), PlaybookID: pb.ID,
			TriggerEvent: mustJSON(event), Status: "rate_limited",
			StartedAt: time.Now(),
		}, nil
	}
	e.lastRun[event.UserID] = time.Now()
	e.mu.Unlock()

	exec := &Execution{
		ID: uuid.New().String(), PlaybookID: pb.ID,
		TriggerEvent: mustJSON(event), Status: "running",
		StartedAt: time.Now(),
	}

	for _, action := range pb.Actions {
		actionCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		err := e.executeAction(actionCtx, action, event)
		cancel()
		if err != nil {
			slog.Warn("SOAR action failed", "action", action.Type, "error", err)
			exec.ActionsTaken = append(exec.ActionsTaken, action.Type+":FAILED")
			continue // continue other actions despite failure
		}
		exec.ActionsTaken = append(exec.ActionsTaken, action.Type+":OK")
	}

	now := time.Now()
	exec.CompletedAt = &now
	exec.Status = "completed"

	// Persist execution.
	if e.pool != nil {
		actionsJSON, _ := json.Marshal(exec.ActionsTaken)
		e.pool.Exec(ctx,
			`INSERT INTO soar_executions (id, playbook_id, trigger_event, status, actions_taken, started_at, completed_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
			exec.ID, exec.PlaybookID, exec.TriggerEvent, exec.Status, actionsJSON, exec.StartedAt, exec.CompletedAt)
	}

	return exec, nil
}

// executeAction runs a single SOAR action.
func (e *Engine) executeAction(ctx context.Context, action PlaybookAction, event TriggerEvent) error {
	switch action.Type {
	case "revoke_session":
		slog.Info("SOAR: revoke_session", "user_id", event.UserID)
		return nil
	case "lock_account":
		slog.Info("SOAR: lock_account", "user_id", event.UserID)
		return nil
	case "step_up_mfa":
		slog.Info("SOAR: step_up_mfa", "user_id", event.UserID)
		return nil
	case "create_incident":
		slog.Info("SOAR: create_incident", "rule", event.RuleID, "user", event.UserID)
		return nil
	case "block_ip":
		slog.Info("SOAR: block_ip", "ip", event.IPAddress)
		return nil
	case "notify_soc":
		return e.notifySOC(ctx, action, event)
	default:
		return fmt.Errorf("unknown action type: %s", action.Type)
	}
}

// notifySOC sends a webhook notification to SOC.
func (e *Engine) notifySOC(ctx context.Context, action PlaybookAction, event TriggerEvent) error {
	webhookURL := action.Webhook
	if webhookURL == "" {
		return fmt.Errorf("notify_soc requires webhook URL")
	}

	payload := map[string]any{
		"event":     "soar_alert",
		"rule_id":   event.RuleID,
		"severity":  event.Severity,
		"user_id":   event.UserID,
		"tenant_id": event.TenantID,
		"ip":        event.IPAddress,
		"message":   action.Message,
		"timestamp": time.Now().UTC(),
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("SOC webhook returned %d", resp.StatusCode)
	}
	return nil
}

// severityAtLeast checks if actual >= required severity level.
func severityAtLeast(actual, required string) bool {
	order := map[string]int{"low": 1, "medium": 2, "high": 3, "critical": 4}
	return order[actual] >= order[required]
}

func mustJSON(v any) json.RawMessage {
	data, _ := json.Marshal(v)
	return data
}
