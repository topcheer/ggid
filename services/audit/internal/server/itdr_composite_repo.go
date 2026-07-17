package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// compositeRuleRepo manages composite detection rules in PostgreSQL.
type compositeRuleRepo struct {
	pool *pgxpool.Pool
}

func newCompositeRuleRepo(pool *pgxpool.Pool) *compositeRuleRepo {
	return &compositeRuleRepo{pool: pool}
}

func (r *compositeRuleRepo) EnsureSchema(ctx context.Context) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS composite_rules (
			id          TEXT PRIMARY KEY,
			tenant_id   UUID NOT NULL,
			name        TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			signals     JSONB NOT NULL DEFAULT '[]',
			min_signals INT NOT NULL DEFAULT 1,
			window_min  INT NOT NULL DEFAULT 30,
			severity    TEXT NOT NULL DEFAULT 'critical',
			actions     JSONB NOT NULL DEFAULT '[]',
			enabled     BOOLEAN NOT NULL DEFAULT TRUE,
			created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_composite_rules_tenant ON composite_rules(tenant_id, enabled);

		CREATE TABLE IF NOT EXISTS security_incidents (
			id              TEXT PRIMARY KEY,
			tenant_id       UUID NOT NULL,
			title           TEXT NOT NULL,
			severity        TEXT NOT NULL DEFAULT 'high',
			status          TEXT NOT NULL DEFAULT 'open',
			triggered_rules JSONB NOT NULL DEFAULT '[]',
			user_ids        JSONB DEFAULT '[]',
			ip_addresses    JSONB DEFAULT '[]',
			detection_count INT NOT NULL DEFAULT 0,
			first_detected  TIMESTAMPTZ NOT NULL DEFAULT now(),
			last_updated    TIMESTAMPTZ NOT NULL DEFAULT now(),
			timeline        JSONB DEFAULT '[]'
		);
		CREATE INDEX IF NOT EXISTS idx_incidents_tenant ON security_incidents(tenant_id, status);
	`)
	return err
}

func (r *compositeRuleRepo) Create(ctx context.Context, rule *CompositeRule) error {
	if r.pool == nil {
		return nil
	}
	signalsJSON, _ := json.Marshal(rule.Signals)
	actionsJSON, _ := json.Marshal(rule.Actions)
	_, err := r.pool.Exec(ctx, `
		INSERT INTO composite_rules (id, tenant_id, name, description, signals, min_signals, window_min, severity, actions, enabled, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		rule.ID, uuid.Nil, rule.Name, rule.Description, signalsJSON, rule.MinSignals, rule.WindowMin, rule.Severity, actionsJSON, rule.Enabled, rule.CreatedAt)
	return err
}

func (r *compositeRuleRepo) List(ctx context.Context) ([]*CompositeRule, error) {
	if r.pool == nil {
		return []*CompositeRule{}, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, description, signals, min_signals, window_min, severity, actions, enabled, created_at
		FROM composite_rules ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*CompositeRule
	for rows.Next() {
		var rule CompositeRule
		var signalsJSON, actionsJSON []byte
		if err := rows.Scan(&rule.ID, &rule.Name, &rule.Description, &signalsJSON, &rule.MinSignals, &rule.WindowMin, &rule.Severity, &actionsJSON, &rule.Enabled, &rule.CreatedAt); err != nil {
			continue
		}
		json.Unmarshal(signalsJSON, &rule.Signals)
		json.Unmarshal(actionsJSON, &rule.Actions)
		result = append(result, &rule)
	}
	return result, nil
}

func (r *compositeRuleRepo) Get(ctx context.Context, id string) (*CompositeRule, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("not found")
	}
	row := r.pool.QueryRow(ctx, `SELECT id, name, description, signals, min_signals, window_min, severity, actions, enabled, created_at FROM composite_rules WHERE id=$1`, id)
	var rule CompositeRule
	var signalsJSON, actionsJSON []byte
	if err := row.Scan(&rule.ID, &rule.Name, &rule.Description, &signalsJSON, &rule.MinSignals, &rule.WindowMin, &rule.Severity, &actionsJSON, &rule.Enabled, &rule.CreatedAt); err != nil {
		return nil, fmt.Errorf("not found")
	}
	json.Unmarshal(signalsJSON, &rule.Signals)
	json.Unmarshal(actionsJSON, &rule.Actions)
	return &rule, nil
}

func (r *compositeRuleRepo) Update(ctx context.Context, rule *CompositeRule) error {
	if r.pool == nil {
		return nil
	}
	signalsJSON, _ := json.Marshal(rule.Signals)
	actionsJSON, _ := json.Marshal(rule.Actions)
	_, err := r.pool.Exec(ctx, `UPDATE composite_rules SET name=$2, description=$3, signals=$4, min_signals=$5, window_min=$6, severity=$7, actions=$8, enabled=$9 WHERE id=$1`,
		rule.ID, rule.Name, rule.Description, signalsJSON, rule.MinSignals, rule.WindowMin, rule.Severity, actionsJSON, rule.Enabled)
	return err
}

func (r *compositeRuleRepo) Delete(ctx context.Context, id string) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `DELETE FROM composite_rules WHERE id=$1`, id)
	return err
}

// --- Incidents repo ---

func (r *compositeRuleRepo) CreateIncident(ctx context.Context, inc *IncidentListEntry) error {
	if r.pool == nil {
		return nil
	}
	rulesJSON, _ := json.Marshal(inc.TriggeredRules)
	usersJSON, _ := json.Marshal(inc.UserIDs)
	ipsJSON, _ := json.Marshal(inc.IPAddresses)
	timelineJSON, _ := json.Marshal(inc.Timeline)
	_, err := r.pool.Exec(ctx, `
		INSERT INTO security_incidents (id, tenant_id, title, severity, status, triggered_rules, user_ids, ip_addresses, detection_count, first_detected, last_updated, timeline)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$10,$11)`,
		inc.ID, uuid.Nil, inc.Title, inc.Severity, inc.Status, rulesJSON, usersJSON, ipsJSON, inc.DetectionCount, inc.FirstDetected, timelineJSON)
	return err
}

func (r *compositeRuleRepo) ListIncidents(ctx context.Context, status string) ([]*IncidentListEntry, error) {
	if r.pool == nil {
		return []*IncidentListEntry{}, nil
	}
	q := `SELECT id, title, severity, status, triggered_rules, user_ids, ip_addresses, detection_count, first_detected, last_updated, timeline FROM security_incidents`
	args := []any{}
	if status != "" {
		q += ` WHERE status = $1`
		args = append(args, status)
	}
	q += ` ORDER BY first_detected DESC LIMIT 100`
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*IncidentListEntry
	for rows.Next() {
		var inc IncidentListEntry
		var rulesJSON, usersJSON, ipsJSON, timelineJSON []byte
		if err := rows.Scan(&inc.ID, &inc.Title, &inc.Severity, &inc.Status, &rulesJSON, &usersJSON, &ipsJSON, &inc.DetectionCount, &inc.FirstDetected, &inc.LastUpdated, &timelineJSON); err != nil {
			continue
		}
		json.Unmarshal(rulesJSON, &inc.TriggeredRules)
		json.Unmarshal(usersJSON, &inc.UserIDs)
		json.Unmarshal(ipsJSON, &inc.IPAddresses)
		json.Unmarshal(timelineJSON, &inc.Timeline)
		result = append(result, &inc)
	}
	return result, nil
}

func (r *compositeRuleRepo) GetIncident(ctx context.Context, id string) (*IncidentListEntry, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("not found")
	}
	row := r.pool.QueryRow(ctx, `SELECT id, title, severity, status, triggered_rules, user_ids, ip_addresses, detection_count, first_detected, last_updated, timeline FROM security_incidents WHERE id=$1`, id)
	var inc IncidentListEntry
	var rulesJSON, usersJSON, ipsJSON, timelineJSON []byte
	if err := row.Scan(&inc.ID, &inc.Title, &inc.Severity, &inc.Status, &rulesJSON, &usersJSON, &ipsJSON, &inc.DetectionCount, &inc.FirstDetected, &inc.LastUpdated, &timelineJSON); err != nil {
		return nil, fmt.Errorf("not found")
	}
	json.Unmarshal(rulesJSON, &inc.TriggeredRules)
	json.Unmarshal(usersJSON, &inc.UserIDs)
	json.Unmarshal(ipsJSON, &inc.IPAddresses)
	json.Unmarshal(timelineJSON, &inc.Timeline)
	return &inc, nil
}

// EvaluateComposite checks if N of the rule's signals have triggered within the time window.
// Returns true if the composite rule threshold is met.
func EvaluateComposite(rule *CompositeRule, triggeredSignals map[string]time.Time) bool {
	window := time.Duration(rule.WindowMin) * time.Minute
	now := time.Now()
	count := 0
	for _, signalID := range rule.Signals {
		if triggerTime, ok := triggeredSignals[signalID]; ok {
			if now.Sub(triggerTime) <= window {
				count++
			}
		}
	}
	return count >= rule.MinSignals
}
