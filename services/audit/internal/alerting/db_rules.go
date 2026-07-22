package alerting

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// LoadAlertRulesFromDB reads alert rules from the alert_rules table.
func LoadAlertRulesFromDB(ctx context.Context, pool *pgxpool.Pool) ([]*AlertRule, error) {
	if pool == nil {
		return nil, nil
	}
	rows, err := pool.Query(ctx, `
		SELECT id, COALESCE(tenant_id::text,''), rule_name, pattern,
		       threshold, window_minutes, severity, enabled
		FROM alert_rules WHERE enabled = true`)
	if err != nil {
		return nil, fmt.Errorf("query alert_rules: %w", err)
	}
	defer rows.Close()

	var rules []*AlertRule
	for rows.Next() {
		var id, tenantID, name, pattern, severity string
		var threshold, window int
		var enabled bool
		if err := rows.Scan(&id, &tenantID, &name, &pattern, &threshold, &window, &severity, &enabled); err != nil {
			continue
		}
		rules = append(rules, &AlertRule{
			ID:        id,
			Name:      name,
			TenantID:  tenantID,
			Condition: AlertCondition{Field: "action", Operator: "eq", Value: pattern},
			Threshold: threshold,
			Window:    time.Duration(window) * time.Minute,
			Actions:   []AlertAction{{Type: "webhook"}},
			Enabled:   enabled,
		})
	}
	return rules, rows.Err()
}
