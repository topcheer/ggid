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

// PrivilegeBaseline defines standard permissions for a role.
type PrivilegeBaseline struct {
	RoleID               string   `json:"role_id"`
	TenantID             string   `json:"tenant_id"`
	StandardPermissions  []string `json:"standard_permissions"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// PrivilegeCreepAlert represents a detected privilege anomaly.
type PrivilegeCreepAlert struct {
	ID         string    `json:"id"`
	TenantID   string    `json:"tenant_id"`
	UserID     string    `json:"user_id"`
	Type       string    `json:"type"` // excess_permissions, privilege_growth, orphan_permissions
	Detail     map[string]any `json:"detail"`
	Severity   string    `json:"severity"` // low, medium, high
	DetectedAt time.Time `json:"detected_at"`
	Status     string    `json:"status"` // open, resolved, dismissed
}

// PrivilegeDiff is the comparison result for a single user.
type PrivilegeDiff struct {
	UserID            string   `json:"user_id"`
	Roles             []string `json:"roles"`
	ExpectedPermissions []string `json:"expected_permissions"`
	ActualPermissions   []string `json:"actual_permissions"`
	ExcessPermissions   []string `json:"excess_permissions"`
	MissingPermissions  []string `json:"missing_permissions"`
}

// privilegeCreepRepo manages privilege baselines and alerts.
type privilegeCreepRepo struct {
	pool *pgxpool.Pool
}

func newPrivilegeCreepRepo(pool *pgxpool.Pool) *privilegeCreepRepo {
	return &privilegeCreepRepo{pool: pool}
}

// NewPrivilegeCreepRepo is the exported constructor.
func NewPrivilegeCreepRepo(pool *pgxpool.Pool) *privilegeCreepRepo {
	return newPrivilegeCreepRepo(pool)
}

func (r *privilegeCreepRepo) EnsureSchema(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS privilege_baselines (
			role_id              TEXT NOT NULL,
			tenant_id            UUID NOT NULL,
			standard_permissions JSONB NOT NULL DEFAULT '[]'::jsonb,
			updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
			PRIMARY KEY (role_id, tenant_id)
		);
		CREATE TABLE IF NOT EXISTS privilege_creep_alerts (
			id          TEXT PRIMARY KEY,
			tenant_id   UUID NOT NULL,
			user_id     UUID NOT NULL,
			type        TEXT NOT NULL,
			detail      JSONB DEFAULT '{}'::jsonb,
			severity    TEXT NOT NULL DEFAULT 'medium',
			detected_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			status      TEXT NOT NULL DEFAULT 'open'
		);
		CREATE INDEX IF NOT EXISTS idx_pca_tenant ON privilege_creep_alerts(tenant_id);
		CREATE INDEX IF NOT EXISTS idx_pca_status ON privilege_creep_alerts(status);
	`)
	return err
}

// SetBaseline sets the standard permissions for a role.
func (r *privilegeCreepRepo) SetBaseline(ctx context.Context, b *PrivilegeBaseline) error {
	permsJSON, _ := json.Marshal(b.StandardPermissions)
	_, err := r.pool.Exec(ctx,
		`INSERT INTO privilege_baselines (role_id, tenant_id, standard_permissions, updated_at)
		 VALUES ($1, $2, $3, now())
		 ON CONFLICT (role_id, tenant_id) DO UPDATE SET
		   standard_permissions = $3, updated_at = now()`,
		b.RoleID, b.TenantID, permsJSON)
	return err
}

// GetBaseline retrieves the baseline for a role.
func (r *privilegeCreepRepo) GetBaseline(ctx context.Context, tenantID uuid.UUID, roleID string) (*PrivilegeBaseline, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT role_id, tenant_id::text, standard_permissions, updated_at
		 FROM privilege_baselines WHERE role_id = $1 AND tenant_id = $2`, roleID, tenantID)

	b := &PrivilegeBaseline{}
	var permsJSON []byte
	err := row.Scan(&b.RoleID, &b.TenantID, &permsJSON, &b.UpdatedAt)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(permsJSON, &b.StandardPermissions)
	return b, nil
}

// ListAlerts returns all open alerts for a tenant.
func (r *privilegeCreepRepo) ListAlerts(ctx context.Context, tenantID uuid.UUID) ([]*PrivilegeCreepAlert, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id::text, user_id::text, type, detail, severity, detected_at, status
		 FROM privilege_creep_alerts WHERE tenant_id = $1 AND status = 'open'
		 ORDER BY detected_at DESC LIMIT 100`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*PrivilegeCreepAlert
	for rows.Next() {
		a := &PrivilegeCreepAlert{}
		var detailJSON []byte
		if err := rows.Scan(&a.ID, &a.TenantID, &a.UserID, &a.Type, &detailJSON, &a.Severity, &a.DetectedAt, &a.Status); err != nil {
			continue
		}
		_ = json.Unmarshal(detailJSON, &a.Detail)
		result = append(result, a)
	}
	return result, nil
}

// CreateAlert records a new privilege creep alert.
func (r *privilegeCreepRepo) CreateAlert(ctx context.Context, a *PrivilegeCreepAlert) error {
	if a.ID == "" {
		a.ID = "pca-" + uuid.New().String()[:8]
	}
	if a.DetectedAt.IsZero() {
		a.DetectedAt = time.Now().UTC()
	}
	detailJSON, _ := json.Marshal(a.Detail)
	_, err := r.pool.Exec(ctx,
		`INSERT INTO privilege_creep_alerts (id, tenant_id, user_id, type, detail, severity, detected_at, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, COALESCE($8, 'open'))`,
		a.ID, a.TenantID, a.UserID, a.Type, detailJSON, a.Severity, a.DetectedAt, a.Status)
	return err
}

// ComputeDiff calculates the difference between expected and actual permissions.
// expected: union of all role baseline permissions for the user's roles.
// actual: the user's current effective permissions.
func ComputeDiff(userID string, roles []string, expected, actual []string) *PrivilegeDiff {
	expectedSet := toSet(expected)
	actualSet := toSet(actual)

	var excess, missing []string
	for p := range actualSet {
		if !expectedSet[p] {
			excess = append(excess, p)
		}
	}
	for p := range expectedSet {
		if !actualSet[p] {
			missing = append(missing, p)
		}
	}

	return &PrivilegeDiff{
		UserID:              userID,
		Roles:               roles,
		ExpectedPermissions: expected,
		ActualPermissions:   actual,
		ExcessPermissions:   excess,
		MissingPermissions:  missing,
	}
}

func toSet(items []string) map[string]bool {
	s := make(map[string]bool, len(items))
	for _, i := range items {
		s[i] = true
	}
	return s
}

// RunScan performs a privilege creep scan for all users in a tenant.
// It generates alerts for excess permissions and orphan permissions.
// In production this is called by a daily cron job.
func (r *privilegeCreepRepo) RunScan(ctx context.Context, tenantID uuid.UUID, userPermissions map[string][]string, userRoles map[string][]string, activeUserIDs []string) (int, error) {
	alertCount := 0

	// Build expected permissions from baselines for each user.
	baselineCache := make(map[string][]string)

	for _, userID := range activeUserIDs {
		_, err := uuid.Parse(userID)
		if err != nil {
			continue
		}

		roles := userRoles[userID]
		actualPerms := userPermissions[userID]
		activeSet := toSet(activeUserIDs)

		// Collect expected permissions from all role baselines.
		var expectedPerms []string
		seen := make(map[string]bool)
		for _, roleID := range roles {
			if bp, ok := baselineCache[roleID]; ok {
				for _, p := range bp {
					if !seen[p] {
						expectedPerms = append(expectedPerms, p)
						seen[p] = true
					}
				}
				continue
			}
			baseline, err := r.GetBaseline(ctx, tenantID, roleID)
			if err != nil || baseline == nil {
				baselineCache[roleID] = nil
				continue
			}
			baselineCache[roleID] = baseline.StandardPermissions
			for _, p := range baseline.StandardPermissions {
				if !seen[p] {
					expectedPerms = append(expectedPerms, p)
					seen[p] = true
				}
			}
		}

		// Compute diff.
		diff := ComputeDiff(userID, roles, expectedPerms, actualPerms)

		// Alert: excess permissions (permissions not in any role baseline).
		if len(diff.ExcessPermissions) > 0 {
			alert := &PrivilegeCreepAlert{
				TenantID: tenantID.String(),
				UserID:   userID,
				Type:     "excess_permissions",
				Detail:   map[string]any{"excess": diff.ExcessPermissions, "roles": roles},
				Severity: "high",
				Status:   "open",
			}
			if err := r.CreateAlert(ctx, alert); err != nil {
				slog.Warn("failed to create excess alert", "user_id", userID, "error", err)
			} else {
				alertCount++
			}
		}

		// Alert: orphan permissions — user has permissions but is not in active users list.
		if !activeSet[userID] && len(actualPerms) > 0 {
			alert := &PrivilegeCreepAlert{
				TenantID: tenantID.String(),
				UserID:   userID,
				Type:     "orphan_permissions",
				Detail:   map[string]any{"permissions": actualPerms},
				Severity: "high",
				Status:   "open",
			}
			if err := r.CreateAlert(ctx, alert); err != nil {
				slog.Warn("failed to create orphan alert", "user_id", userID, "error", err)
			} else {
				alertCount++
			}
		}
	}

	slog.Info("privilege creep scan completed",
		"tenant_id", tenantID, "users_scanned", len(activeUserIDs), "alerts", alertCount)

	return alertCount, nil
}

// ResolveAlert marks an alert as resolved.
func (r *privilegeCreepRepo) ResolveAlert(ctx context.Context, alertID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE privilege_creep_alerts SET status = 'resolved' WHERE id = $1`, alertID)
	return err
}

// Silence unused import in case fmt isn't needed in future refactors.
var _ = fmt.Sprintf
