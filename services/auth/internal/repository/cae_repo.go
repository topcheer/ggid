package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

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
