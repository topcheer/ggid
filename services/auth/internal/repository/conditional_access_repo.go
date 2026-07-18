package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ConditionalAccessPolicy defines a rule that evaluates context signals
// (device posture, risk score, geo, time, auth method, IP) and returns
// an action: allow, require_mfa, require_step_up, or block.
type ConditionalAccessPolicy struct {
	ID          uuid.UUID  `json:"id"`
	TenantID    uuid.UUID  `json:"tenant_id"`
	Name        string     `json:"name"`
	Conditions  Conditions `json:"conditions"`
	Action      string     `json:"action"`
	Priority    int        `json:"priority"`
	Enabled     bool       `json:"enabled"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// Conditions defines the matching criteria for a conditional access policy.
// All specified conditions must match (AND logic) for the policy to trigger.
// Empty/zero fields are ignored (not evaluated).
type Conditions struct {
	DevicePostureLessThan *int     `json:"device_posture_less_than,omitempty"`
	RiskScoreGreaterThan  *int     `json:"risk_score_greater_than,omitempty"`
	GeoCountries          []string `json:"geo_countries,omitempty"`
	TimeNotInRange        *TimeRange `json:"time_not_in_range,omitempty"`
	AuthMethodNotIn       []string `json:"auth_method_not_in,omitempty"`
	IPNotInAllowlist      []string `json:"ip_not_in_allowlist,omitempty"`
}

// TimeRange defines an allowed time window.
type TimeRange struct {
	Start string `json:"start"` // "09:00"
	End   string `json:"end"`   // "17:00"
}

// EvalContext contains the signals that conditions are evaluated against.
type EvalContext struct {
	DevicePosture int
	RiskScore     int
	GeoCountry    string
	AuthMethod    string
	IPAddress     string
}

// Action constants.
const (
	ActionAllow          = "allow"
	ActionRequireMFA     = "require_mfa"
	ActionRequireStepUp  = "require_step_up"
	ActionBlock          = "block"
)

// ConditionalAccessRepository manages conditional access policies in PostgreSQL.
type ConditionalAccessRepository struct {
	pool *pgxpool.Pool
}

func NewConditionalAccessRepository(pool *pgxpool.Pool) *ConditionalAccessRepository {
	return &ConditionalAccessRepository{pool: pool}
}

func (r *ConditionalAccessRepository) EnsureSchema(ctx context.Context) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS conditional_access_policies (
			id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id   UUID NOT NULL,
			name        TEXT NOT NULL,
			conditions  JSONB NOT NULL DEFAULT '{}',
			action      TEXT NOT NULL DEFAULT 'allow',
			priority    INT NOT NULL DEFAULT 0,
			enabled     BOOLEAN NOT NULL DEFAULT true,
			created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_cap_tenant_priority ON conditional_access_policies (tenant_id, priority DESC);
		CREATE INDEX IF NOT EXISTS idx_cap_tenant_enabled ON conditional_access_policies (tenant_id, enabled) WHERE enabled = true;
	`)
	return err
}

// Create inserts a new policy.
func (r *ConditionalAccessRepository) Create(ctx context.Context, p *ConditionalAccessPolicy) error {
	if r.pool == nil {
		return nil
	}
	conditionsJSON, _ := json.Marshal(p.Conditions)
	_, err := r.pool.Exec(ctx, `
		INSERT INTO conditional_access_policies (id, tenant_id, name, conditions, action, priority, enabled)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		p.ID, p.TenantID, p.Name, conditionsJSON, p.Action, p.Priority, p.Enabled,
	)
	return err
}

// ListByTenant returns all policies for a tenant, ordered by priority descending.
func (r *ConditionalAccessRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*ConditionalAccessPolicy, error) {
	if r.pool == nil {
		return []*ConditionalAccessPolicy{}, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, tenant_id, name, conditions, action, priority, enabled, created_at, updated_at
		FROM conditional_access_policies
		WHERE tenant_id = $1
		ORDER BY priority DESC`,
		tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var policies []*ConditionalAccessPolicy
	for rows.Next() {
		p, err := scanCAPRow(rows)
		if err != nil {
			continue
		}
		policies = append(policies, p)
	}
	return policies, nil
}

// Update modifies an existing policy.
func (r *ConditionalAccessRepository) Update(ctx context.Context, p *ConditionalAccessPolicy) error {
	if r.pool == nil {
		return nil
	}
	conditionsJSON, _ := json.Marshal(p.Conditions)
	_, err := r.pool.Exec(ctx, `
		UPDATE conditional_access_policies
		SET name = $3, conditions = $4, action = $5, priority = $6, enabled = $7, updated_at = now()
		WHERE id = $1 AND tenant_id = $2`,
		p.ID, p.TenantID, p.Name, conditionsJSON, p.Action, p.Priority, p.Enabled,
	)
	return err
}

// Delete removes a policy.
func (r *ConditionalAccessRepository) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `DELETE FROM conditional_access_policies WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	return err
}

// Evaluate checks all enabled policies for a tenant against the given context.
// Returns the action from the first matching policy (highest priority),
// or ActionAllow if no policies match.
func (r *ConditionalAccessRepository) Evaluate(ctx context.Context, tenantID uuid.UUID, evalCtx EvalContext) (string, *ConditionalAccessPolicy) {
	policies, err := r.ListByTenant(ctx, tenantID)
	if err != nil || len(policies) == 0 {
		return ActionAllow, nil
	}
	for _, p := range policies {
		if !p.Enabled {
			continue
		}
		if matchesConditions(p.Conditions, evalCtx) {
			return p.Action, p
		}
	}
	return ActionAllow, nil
}

// MatchesConditionsPublic is the exported version of matchesConditions for external/testing use.
func MatchesConditionsPublic(conds Conditions, evalCtx EvalContext) bool {
	return matchesConditions(conds, evalCtx)
}

// matchesConditions evaluates whether the eval context satisfies all
// specified conditions (AND logic). Empty conditions = no match.
func matchesConditions(conds Conditions, evalCtx EvalContext) bool {
	matched := false

	if conds.DevicePostureLessThan != nil {
		matched = true
		if evalCtx.DevicePosture >= *conds.DevicePostureLessThan {
			return false
		}
	}

	if conds.RiskScoreGreaterThan != nil {
		matched = true
		if evalCtx.RiskScore <= *conds.RiskScoreGreaterThan {
			return false
		}
	}

	if len(conds.GeoCountries) > 0 {
		matched = true
		found := false
		for _, c := range conds.GeoCountries {
			if c == evalCtx.GeoCountry {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if len(conds.AuthMethodNotIn) > 0 {
		matched = true
		inForbiddenList := false
		for _, m := range conds.AuthMethodNotIn {
			if m == evalCtx.AuthMethod {
				inForbiddenList = true
				break
			}
		}
		if !inForbiddenList {
			return false // method is NOT in forbidden list → condition not triggered.
		}
	}

	if len(conds.IPNotInAllowlist) > 0 {
		matched = true
		for _, ip := range conds.IPNotInAllowlist {
			if ip == evalCtx.IPAddress {
				return false // IP is in allowlist → condition not met.
			}
		}
	}

	return matched
}

// String returns a human-readable description of the action decision.
func ActionDescription(action string) string {
	switch action {
	case ActionBlock:
		return "access blocked by conditional access policy"
	case ActionRequireMFA:
		return "multi-factor authentication required"
	case ActionRequireStepUp:
		return "step-up authentication required"
	default:
		return "allowed"
	}
}

func scanCAPRow(row interface {
	Scan(dest ...any) error
}) (*ConditionalAccessPolicy, error) {
	var p ConditionalAccessPolicy
	var conditionsJSON []byte
	err := row.Scan(
		&p.ID, &p.TenantID, &p.Name, &conditionsJSON, &p.Action, &p.Priority, &p.Enabled,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if len(conditionsJSON) > 0 {
		_ = json.Unmarshal(conditionsJSON, &p.Conditions)
	}
	return &p, nil
}

// ValidateAction checks if the action string is valid.
func ValidateAction(action string) error {
	switch action {
	case ActionAllow, ActionRequireMFA, ActionRequireStepUp, ActionBlock:
		return nil
	default:
		return fmt.Errorf("invalid action: %s (must be allow, require_mfa, require_step_up, block)", action)
	}
}
