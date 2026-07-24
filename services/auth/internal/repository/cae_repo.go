package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CAETrigger represents a runtime CAE monitoring trigger rule.
type CAETrigger struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	Event     string    `json:"event"`
	Condition string    `json:"condition"`
	Action    string    `json:"action"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
}

// CAEEvaluation represents a single continuous access evaluation result.
type CAEEvaluation struct {
	ID          uuid.UUID  `json:"id"`
	TenantID    uuid.UUID  `json:"tenant_id"`
	SessionID   string     `json:"session_id"`
	UserID      string     `json:"user_id"`
	Action      string     `json:"action"`
	PolicyName  string     `json:"policy_name,omitempty"`
	IPAddress   string     `json:"ip_address,omitempty"`
	RiskScore   int        `json:"risk_score"`
	EvaluatedAt time.Time  `json:"evaluated_at"`
}

// CAERepository manages CAE evaluation logs and provides the scan engine.
type CAERepository struct {
	pool *pgxpool.Pool
}

func NewCAERepository(pool *pgxpool.Pool) *CAERepository {
	return &CAERepository{pool: pool}
}

func (r *CAERepository) EnsureSchema(ctx context.Context) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS cae_evaluations (
			id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id    UUID NOT NULL,
			session_id   TEXT NOT NULL,
			user_id      TEXT NOT NULL,
			action       TEXT NOT NULL,
			policy_name  TEXT,
			ip_address   TEXT,
			risk_score   INT NOT NULL DEFAULT 0,
			evaluated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_cae_tenant_time ON cae_evaluations (tenant_id, evaluated_at DESC);
		CREATE INDEX IF NOT EXISTS idx_cae_session ON cae_evaluations (session_id);
		CREATE TABLE IF NOT EXISTS cae_triggers (
			id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id  UUID NOT NULL,
			event      TEXT NOT NULL,
			condition  TEXT DEFAULT '',
			action     TEXT NOT NULL,
			enabled    BOOLEAN NOT NULL DEFAULT true,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_cae_triggers_tenant ON cae_triggers(tenant_id);
	`)
	return err
}

// LogEvaluation records a CAE evaluation result.
func (r *CAERepository) LogEvaluation(ctx context.Context, eval *CAEEvaluation) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO cae_evaluations (id, tenant_id, session_id, user_id, action, policy_name, ip_address, risk_score)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		eval.ID, eval.TenantID, eval.SessionID, eval.UserID, eval.Action,
		eval.PolicyName, eval.IPAddress, eval.RiskScore,
	)
	return err
}

// ListByTenant returns recent CAE evaluations for a tenant.
func (r *CAERepository) ListByTenant(ctx context.Context, tenantID uuid.UUID, limit int) ([]*CAEEvaluation, error) {
	if r.pool == nil {
		return []*CAEEvaluation{}, nil
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, tenant_id, session_id, user_id, action, policy_name, ip_address, risk_score, evaluated_at
		FROM cae_evaluations
		WHERE tenant_id = $1
		ORDER BY evaluated_at DESC
		LIMIT $2`,
		tenantID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var evals []*CAEEvaluation
	for rows.Next() {
		var e CAEEvaluation
		if err := rows.Scan(&e.ID, &e.TenantID, &e.SessionID, &e.UserID, &e.Action,
			&e.PolicyName, &e.IPAddress, &e.RiskScore, &e.EvaluatedAt); err != nil {
			continue
		}
		evals = append(evals, &e)
	}
	return evals, nil
}

// CountRecent returns the number of evaluations in the last N minutes.
func (r *CAERepository) CountRecent(ctx context.Context, tenantID uuid.UUID, minutes int) (int, error) {
	if r.pool == nil {
		return 0, nil
	}
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT count(*) FROM cae_evaluations
		WHERE tenant_id = $1 AND evaluated_at > now() - ($2 || ' minutes')::interval`,
		tenantID, minutes,
	).Scan(&count)
	return count, err
}

// CountByAction returns counts grouped by action in the last N minutes.
func (r *CAERepository) CountByAction(ctx context.Context, tenantID uuid.UUID, minutes int) (map[string]int, error) {
	if r.pool == nil {
		return map[string]int{}, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT action, count(*) FROM cae_evaluations
		WHERE tenant_id = $1 AND evaluated_at > now() - ($2 || ' minutes')::interval
		GROUP BY action`,
		tenantID, minutes,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := map[string]int{}
	for rows.Next() {
		var action string
		var count int
		if err := rows.Scan(&action, &count); err != nil {
			continue
		}
		result[action] = count
	}
	return result, nil
}

// --- CAE Trigger CRUD ---

// ListTriggers returns all CAE triggers for a tenant.
func (r *CAERepository) ListTriggers(ctx context.Context, tenantID uuid.UUID) ([]*CAETrigger, error) {
	if r.pool == nil {
		return []*CAETrigger{}, nil
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, event, condition, action, enabled, created_at FROM cae_triggers WHERE tenant_id = $1 ORDER BY created_at DESC`,
		tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var triggers []*CAETrigger
	for rows.Next() {
		t := &CAETrigger{TenantID: tenantID}
		if err := rows.Scan(&t.ID, &t.Event, &t.Condition, &t.Action, &t.Enabled, &t.CreatedAt); err != nil {
			continue
		}
		triggers = append(triggers, t)
	}
	return triggers, nil
}

// CreateTrigger creates a new CAE trigger.
func (r *CAERepository) CreateTrigger(ctx context.Context, t *CAETrigger) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx,
		`INSERT INTO cae_triggers (id, tenant_id, event, condition, action, enabled) VALUES ($1, $2, $3, $4, $5, $6)`,
		t.ID, t.TenantID, t.Event, t.Condition, t.Action, t.Enabled,
	)
	return err
}

// UpdateTrigger updates a CAE trigger's enabled status and fields.
func (r *CAERepository) UpdateTrigger(ctx context.Context, tenantID, id uuid.UUID, event, condition, action string, enabled bool) error {
	if r.pool == nil {
		return nil
	}
	ct, err := r.pool.Exec(ctx,
		`UPDATE cae_triggers SET event = $1, condition = $2, action = $3, enabled = $4 WHERE id = $5 AND tenant_id = $6`,
		event, condition, action, enabled, id, tenantID,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("trigger not found")
	}
	return nil
}

// DeleteTrigger removes a CAE trigger.
func (r *CAERepository) DeleteTrigger(ctx context.Context, tenantID, id uuid.UUID) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx,
		`DELETE FROM cae_triggers WHERE id = $1 AND tenant_id = $2`,
		id, tenantID,
	)
	return err
}
