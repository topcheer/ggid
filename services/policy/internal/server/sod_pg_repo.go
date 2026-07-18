package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SoDRulePG represents a segregation-of-duties rule in PostgreSQL.
type SoDRulePG struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	RoleA       string    `json:"role_a"`
	RoleB       string    `json:"role_b"`
	Description string    `json:"description"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
}

// SoDViolationPG represents a detected SoD violation in PostgreSQL.
type SoDViolationPG struct {
	ID         string    `json:"id"`
	TenantID   string    `json:"tenant_id"`
	UserID     string    `json:"user_id"`
	RoleA      string    `json:"role_a"`
	RoleB      string    `json:"role_b"`
	RuleID     string    `json:"rule_id"`
	Reason     string    `json:"reason"`
	DetectedAt time.Time `json:"detected_at"`
	Status     string    `json:"status"` // open, resolved, dismissed
}

// sodPGRepo manages sod_rules and sod_violations in PostgreSQL.
type sodPGRepo struct {
	pool *pgxpool.Pool
}

// NewSodPGRepo creates a new PG-backed SoD repo (exported for wiring).
func NewSodPGRepo(pool *pgxpool.Pool) *sodPGRepo {
	return &sodPGRepo{pool: pool}
}

func (r *sodPGRepo) EnsureSchema(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS sod_rules (
			id          TEXT PRIMARY KEY,
			tenant_id   UUID NOT NULL,
			role_a      TEXT NOT NULL,
			role_b      TEXT NOT NULL,
			description TEXT DEFAULT '',
			enabled     BOOLEAN NOT NULL DEFAULT true,
			created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_sod_rules_tenant ON sod_rules(tenant_id);
		CREATE TABLE IF NOT EXISTS sod_violations (
			id          TEXT PRIMARY KEY,
			tenant_id   UUID NOT NULL,
			user_id     UUID NOT NULL,
			role_a      TEXT NOT NULL,
			role_b      TEXT NOT NULL,
			rule_id     TEXT DEFAULT '',
			reason      TEXT DEFAULT '',
			detected_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			status      TEXT NOT NULL DEFAULT 'open'
		);
		CREATE INDEX IF NOT EXISTS idx_sod_violations_tenant ON sod_violations(tenant_id);
		CREATE INDEX IF NOT EXISTS idx_sod_violations_user ON sod_violations(user_id);
		CREATE INDEX IF NOT EXISTS idx_sod_violations_status ON sod_violations(status);
	`)
	return err
}

// SeedDefaults inserts default SoD rules if none exist for the tenant.
func (r *sodPGRepo) SeedDefaults(ctx context.Context, tenantID uuid.UUID) error {
	existing, err := r.ListRules(ctx, tenantID)
	if err != nil {
		return err
	}
	if len(existing) > 0 {
		return nil // already seeded
	}

	defaults := []struct {
		roleA, roleB, desc string
	}{
		{"admin", "auditor", "admin + auditor mutually exclusive"},
		{"admin", "compliance", "admin + compliance mutually exclusive"},
	}
	for _, d := range defaults {
		rule := &SoDRulePG{
			ID:          "sod-" + uuid.New().String()[:8],
			TenantID:    tenantID.String(),
			RoleA:       d.roleA,
			RoleB:       d.roleB,
			Description: d.desc,
			Enabled:     true,
			CreatedAt:   time.Now().UTC(),
		}
		if err := r.CreateRule(ctx, rule); err != nil {
			return fmt.Errorf("seed default rule: %w", err)
		}
	}
	return nil
}

func (r *sodPGRepo) CreateRule(ctx context.Context, rule *SoDRulePG) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO sod_rules (id, tenant_id, role_a, role_b, description, enabled, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		rule.ID, rule.TenantID, rule.RoleA, rule.RoleB, rule.Description, rule.Enabled, rule.CreatedAt)
	return err
}

func (r *sodPGRepo) ListRules(ctx context.Context, tenantID uuid.UUID) ([]*SoDRulePG, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id::text, role_a, role_b, description, enabled, created_at
		 FROM sod_rules WHERE tenant_id = $1 ORDER BY created_at`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*SoDRulePG
	for rows.Next() {
		rule := &SoDRulePG{}
		if err := rows.Scan(&rule.ID, &rule.TenantID, &rule.RoleA, &rule.RoleB,
			&rule.Description, &rule.Enabled, &rule.CreatedAt); err != nil {
			continue
		}
		result = append(result, rule)
	}
	return result, nil
}

func (r *sodPGRepo) DeleteRule(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM sod_rules WHERE id = $1`, id)
	return err
}

func (r *sodPGRepo) RecordViolation(ctx context.Context, v *SoDViolationPG) error {
	if v.ID == "" {
		v.ID = "sdv-" + uuid.New().String()[:8]
	}
	if v.DetectedAt.IsZero() {
		v.DetectedAt = time.Now().UTC()
	}
	if v.Status == "" {
		v.Status = "open"
	}
	_, err := r.pool.Exec(ctx,
		`INSERT INTO sod_violations (id, tenant_id, user_id, role_a, role_b, rule_id, reason, detected_at, status)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		v.ID, v.TenantID, v.UserID, v.RoleA, v.RoleB, v.RuleID, v.Reason, v.DetectedAt, v.Status)
	return err
}

func (r *sodPGRepo) ListViolations(ctx context.Context, tenantID uuid.UUID) ([]*SoDViolationPG, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id::text, user_id::text, role_a, role_b, rule_id, reason, detected_at, status
		 FROM sod_violations WHERE tenant_id = $1 AND status = 'open'
		 ORDER BY detected_at DESC LIMIT 100`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*SoDViolationPG
	for rows.Next() {
		v := &SoDViolationPG{}
		if err := rows.Scan(&v.ID, &v.TenantID, &v.UserID, &v.RoleA, &v.RoleB,
			&v.RuleID, &v.Reason, &v.DetectedAt, &v.Status); err != nil {
			continue
		}
		result = append(result, v)
	}
	return result, nil
}

func (r *sodPGRepo) ResolveViolation(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `UPDATE sod_violations SET status = 'resolved' WHERE id = $1`, id)
	return err
}

// CheckSoDFromDB evaluates user roles against DB-backed SoD rules.
// Returns violations found.
func (r *sodPGRepo) CheckSoD(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID, userRoles []string) ([]*SoDViolationPG, error) {
	rules, err := r.ListRules(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	roleSet := make(map[string]bool)
	for _, role := range userRoles {
		roleSet[role] = true
	}

	var violations []*SoDViolationPG
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		if roleSet[rule.RoleA] && roleSet[rule.RoleB] {
			v := &SoDViolationPG{
				TenantID:   tenantID.String(),
				UserID:     userID.String(),
				RoleA:      rule.RoleA,
				RoleB:      rule.RoleB,
				RuleID:     rule.ID,
				Reason:     rule.Description,
				DetectedAt: time.Now().UTC(),
				Status:     "open",
			}
			violations = append(violations, v)
			// Persist the violation.
			_ = r.RecordViolation(ctx, v)
		}
	}
	return violations, nil
}

// Ensure json import is used (for future metadata fields).
var _ = json.Marshal
