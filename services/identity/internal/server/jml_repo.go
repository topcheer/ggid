package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// LifecycleExecution logs a single action execution.
type LifecycleExecution struct {
	ID           uuid.UUID      `json:"id"`
	TenantID     uuid.UUID      `json:"tenant_id"`
	RuleID       string         `json:"rule_id"`
	UserID       uuid.UUID      `json:"user_id"`
	Trigger      string         `json:"trigger"`
	ActionType   string         `json:"action_type"`
	ActionParams map[string]any `json:"action_params,omitempty"`
	Result       string         `json:"result"` // success, failed, skipped
	Error        string         `json:"error,omitempty"`
	ExecutedAt   time.Time      `json:"executed_at"`
}

// lifecycleRepo manages JML rules and execution logs in Postgres.
type lifecycleRepo struct {
	pool *pgxpool.Pool
}

func newLifecycleRepo(pool *pgxpool.Pool) *lifecycleRepo {
	return &lifecycleRepo{pool: pool}
}

func (r *lifecycleRepo) EnsureSchema(ctx context.Context) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS lifecycle_rules (
			id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id   UUID NOT NULL,
			name        TEXT NOT NULL,
			trigger     TEXT NOT NULL,
			conditions  JSONB NOT NULL DEFAULT '{}',
			actions     JSONB NOT NULL DEFAULT '[]',
			priority    INT NOT NULL DEFAULT 100,
			enabled     BOOLEAN NOT NULL DEFAULT TRUE,
			created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_jml_rules_trigger ON lifecycle_rules (tenant_id, trigger) WHERE enabled = TRUE;
		CREATE TABLE IF NOT EXISTS lifecycle_executions (
			id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id     UUID NOT NULL,
			rule_id       TEXT NOT NULL,
			user_id       UUID NOT NULL,
			trigger       TEXT NOT NULL,
			action_type   TEXT NOT NULL,
			action_params JSONB,
			result        TEXT NOT NULL,
			error         TEXT,
			executed_at   TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_jml_exec_user ON lifecycle_executions (tenant_id, user_id, executed_at DESC);
	`)
	return err
}

// CreateRule stores a lifecycle rule in the DB.
func (r *lifecycleRepo) CreateRule(ctx context.Context, tenantID uuid.UUID, rule *LifecycleRule) error {
	if r.pool == nil {
		return nil
	}
	if rule.ID == "" {
		rule.ID = uuid.New().String()
	}
	conditionsJSON, _ := json.Marshal(rule.Conditions)
	actionsJSON, _ := json.Marshal(rule.Actions)
	_, err := r.pool.Exec(ctx, `
		INSERT INTO lifecycle_rules (id, tenant_id, name, trigger, conditions, actions, enabled)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		rule.ID, tenantID, rule.Name, rule.Trigger, conditionsJSON, actionsJSON, rule.Enabled,
	)
	return err
}

// ListRules returns rules for a tenant, optionally filtered by trigger.
func (r *lifecycleRepo) ListRules(ctx context.Context, tenantID uuid.UUID, trigger string) ([]*LifecycleRule, error) {
	if r.pool == nil {
		return []*LifecycleRule{}, nil
	}
	query := `SELECT id::text, name, trigger, conditions, actions, enabled, created_at FROM lifecycle_rules WHERE tenant_id = $1`
	args := []any{tenantID}
	if trigger != "" {
		query += ` AND trigger = $2 AND enabled = TRUE`
		args = append(args, trigger)
	} else {
		query += ` AND enabled = TRUE`
	}
	query += ` ORDER BY created_at DESC LIMIT 100`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*LifecycleRule
	for rows.Next() {
		var rule LifecycleRule
		var conditionsJSON, actionsJSON []byte
		if err := rows.Scan(&rule.ID, &rule.Name, &rule.Trigger, &conditionsJSON, &actionsJSON, &rule.Enabled, &rule.CreatedAt); err != nil {
			continue
		}
		if len(conditionsJSON) > 0 {
			json.Unmarshal(conditionsJSON, &rule.Conditions)
		}
		if len(actionsJSON) > 0 {
			json.Unmarshal(actionsJSON, &rule.Actions)
		}
		rule.TenantID = tenantID.String()
		rules = append(rules, &rule)
	}
	return rules, nil
}

// FindMatchingRules returns rules matching trigger + user attributes.
func (r *lifecycleRepo) FindMatchingRules(ctx context.Context, tenantID uuid.UUID, trigger string, userAttrs map[string]any) ([]*LifecycleRule, error) {
	rules, err := r.ListRules(ctx, tenantID, trigger)
	if err != nil {
		return nil, err
	}
	var matched []*LifecycleRule
	for _, rule := range rules {
		if matchConditions(rule.Conditions, userAttrs) {
			matched = append(matched, rule)
		}
	}
	return matched, nil
}

// matchConditions checks if userAttrs satisfy all condition key-value pairs.
func matchConditions(conditions, userAttrs map[string]any) bool {
	if len(conditions) == 0 {
		return true
	}
	for key, expected := range conditions {
		actual, ok := userAttrs[key]
		if !ok {
			return false
		}
		expectedStr := fmt.Sprintf("%v", expected)
		if expectedStr == "*" {
			continue
		}
		if fmt.Sprintf("%v", actual) != expectedStr {
			return false
		}
	}
	return true
}

// LogExecution records an action execution result.
func (r *lifecycleRepo) LogExecution(ctx context.Context, exec *LifecycleExecution) {
	if r.pool == nil {
		return
	}
	if exec.ID == uuid.Nil {
		exec.ID = uuid.New()
	}
	paramsJSON, _ := json.Marshal(exec.ActionParams)
	_, err := r.pool.Exec(ctx, `
		INSERT INTO lifecycle_executions (id, tenant_id, rule_id, user_id, trigger, action_type, action_params, result, error)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		exec.ID, exec.TenantID, exec.RuleID, exec.UserID, exec.Trigger, exec.ActionType, paramsJSON, exec.Result, exec.Error,
	)
	if err != nil {
		slog.Error("lifecycle: failed to log execution", "error", err)
	}
}

// ListExecutions returns recent executions for a user.
func (r *lifecycleRepo) ListExecutions(ctx context.Context, tenantID, userID uuid.UUID) ([]*LifecycleExecution, error) {
	if r.pool == nil {
		return []*LifecycleExecution{}, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, tenant_id, rule_id, user_id, trigger, action_type, action_params, result, error, executed_at
		FROM lifecycle_executions WHERE tenant_id = $1 AND user_id = $2
		ORDER BY executed_at DESC LIMIT 50`,
		tenantID, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var execs []*LifecycleExecution
	for rows.Next() {
		var exec LifecycleExecution
		var paramsJSON []byte
		if err := rows.Scan(&exec.ID, &exec.TenantID, &exec.RuleID, &exec.UserID, &exec.Trigger, &exec.ActionType, &paramsJSON, &exec.Result, &exec.Error, &exec.ExecutedAt); err != nil {
			continue
		}
		if len(paramsJSON) > 0 {
			json.Unmarshal(paramsJSON, &exec.ActionParams)
		}
		execs = append(execs, &exec)
	}
	return execs, nil
}
