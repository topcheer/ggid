package server

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DormantThresholdDays is the inactivity period before marking dormant.
const (
	DormantThresholdDays   = 90
	SuspendThresholdDays   = 120
	ArchiveThresholdDays   = 150
)

// UserLifecycleState tracks the dormant→suspend→archive progression.
type UserLifecycleState struct {
	UserID       string     `json:"user_id"`
	State        string     `json:"state"`
	DormantSince *time.Time `json:"dormant_since,omitempty"`
	SuspendedAt *time.Time `json:"suspended_at,omitempty"`
	ArchivedAt  *time.Time `json:"archived_at,omitempty"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
	NotifiedAt  *time.Time `json:"notified_at,omitempty"`
}

// GhostAccount is a user in GGID but not in HR active list.
type GhostAccount struct {
	UserID        string    `json:"user_id"`
	Email         string    `json:"email"`
	Recommendation string   `json:"recommendation"`
	Status        string    `json:"status"`
	DetectedAt    time.Time `json:"detected_at"`
}

// dormantRepo manages dormant lifecycle + ghost account state.
type dormantRepo struct {
	pool *pgxpool.Pool
}

func newDormantRepo(pool *pgxpool.Pool) *dormantRepo {
	return &dormantRepo{pool: pool}
}

func (r *dormantRepo) EnsureSchema(ctx context.Context) error {
	if r.pool == nil { return nil }
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS user_lifecycle_state (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id TEXT NOT NULL UNIQUE,
			tenant_id UUID, state TEXT DEFAULT 'active',
			dormant_since TIMESTAMPTZ, suspended_at TIMESTAMPTZ,
			archived_at TIMESTAMPTZ, last_login_at TIMESTAMPTZ,
			notified_at TIMESTAMPTZ, updated_at TIMESTAMPTZ DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_lifecycle_state ON user_lifecycle_state(state);
		CREATE INDEX IF NOT EXISTS idx_lifecycle_user ON user_lifecycle_state(user_id);
		CREATE TABLE IF NOT EXISTS ghost_accounts (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id TEXT NOT NULL, tenant_id UUID,
			email TEXT, recommendation TEXT DEFAULT 'disable',
			status TEXT DEFAULT 'flagged', detected_at TIMESTAMPTZ DEFAULT now(),
			actioned_at TIMESTAMPTZ, details JSONB DEFAULT '{}'
		);
		CREATE INDEX IF NOT EXISTS idx_ghost_status ON ghost_accounts(status);
	`)
	return err
}

// UpsertState creates or updates a user's lifecycle state.
func (r *dormantRepo) UpsertState(ctx context.Context, s *UserLifecycleState) error {
	if r.pool == nil { return nil }
	_, err := r.pool.Exec(ctx, `
		INSERT INTO user_lifecycle_state (user_id,state,dormant_since,suspended_at,archived_at,last_login_at,notified_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		ON CONFLICT (user_id) DO UPDATE SET state=EXCLUDED.state,
			dormant_since=EXCLUDED.dormant_since, suspended_at=EXCLUDED.suspended_at,
			archived_at=EXCLUDED.archived_at, last_login_at=EXCLUDED.last_login_at,
			notified_at=EXCLUDED.notified_at, updated_at=now()`,
		s.UserID, s.State, s.DormantSince, s.SuspendedAt, s.ArchivedAt, s.LastLoginAt, s.NotifiedAt)
	return err
}

// ListDormant returns users in dormant/suspended/archived state.
func (r *dormantRepo) ListDormant(ctx context.Context) ([]*UserLifecycleState, error) {
	if r.pool == nil { return []*UserLifecycleState{}, nil }
	rows, err := r.pool.Query(ctx, `SELECT user_id,state,dormant_since,suspended_at,archived_at,last_login_at,notified_at FROM user_lifecycle_state WHERE state != 'active' ORDER BY updated_at DESC`)
	if err != nil { return nil, err }
	defer rows.Close()
	var result []*UserLifecycleState
	for rows.Next() {
		s := &UserLifecycleState{}
		if err := rows.Scan(&s.UserID, &s.State, &s.DormantSince, &s.SuspendedAt, &s.ArchivedAt, &s.LastLoginAt, &s.NotifiedAt); err != nil { continue }
		result = append(result, s)
	}
	return result, nil
}

// CreateGhost flags a user as a potential ghost account.
func (r *dormantRepo) CreateGhost(ctx context.Context, userID, email, recommendation string) error {
	if r.pool == nil { return nil }
	_, err := r.pool.Exec(ctx, `INSERT INTO ghost_accounts (user_id,email,recommendation,status) VALUES ($1,$2,$3,'flagged') ON CONFLICT DO NOTHING`,
		userID, email, recommendation)
	return err
}

// ListGhosts returns flagged ghost accounts awaiting action.
func (r *dormantRepo) ListGhosts(ctx context.Context) ([]*GhostAccount, error) {
	if r.pool == nil { return []*GhostAccount{}, nil }
	rows, err := r.pool.Query(ctx, `SELECT user_id,email,recommendation,status,detected_at FROM ghost_accounts WHERE status='flagged' ORDER BY detected_at DESC`)
	if err != nil { return nil, err }
	defer rows.Close()
	var result []*GhostAccount
	for rows.Next() {
		g := &GhostAccount{}
		if err := rows.Scan(&g.UserID, &g.Email, &g.Recommendation, &g.Status, &g.DetectedAt); err != nil { continue }
		result = append(result, g)
	}
	return result, nil
}

// --- Dormant Detection Logic ---

// EvaluateDormantState determines the lifecycle state based on days since last login.
func EvaluateDormantState(daysSinceLogin int) (state string, action string) {
	switch {
	case daysSinceLogin >= ArchiveThresholdDays:
		return "archived", "archive"
	case daysSinceLogin >= SuspendThresholdDays:
		return "suspended", "suspend"
	case daysSinceLogin >= DormantThresholdDays:
		return "dormant", "notify"
	default:
		return "active", "none"
	}
}

// RunDormantScan checks all users and updates their lifecycle state.
// In production, this queries the user table for last_login_at.
func (r *dormantRepo) RunDormantScan(ctx context.Context, userLastLoginFn func(ctx context.Context) ([]UserLoginInfo, error)) (int, error) {
	if r.pool == nil || userLastLoginFn == nil {
		return 0, nil
	}
	users, err := userLastLoginFn(ctx)
	if err != nil {
		return 0, err
	}
	now := time.Now().UTC()
	updated := 0
	for _, u := range users {
		if u.LastLoginAt == nil {
			continue
		}
		days := int(now.Sub(*u.LastLoginAt).Hours() / 24)
		state, _ := EvaluateDormantState(days)
		if state == "active" {
			continue
		}
		ls := &UserLifecycleState{
			UserID: u.UserID, State: state, LastLoginAt: u.LastLoginAt,
		}
		if state == "dormant" || state == "suspended" || state == "archived" {
			ls.DormantSince = u.LastLoginAt
		}
		if state == "suspended" || state == "archived" {
			ls.SuspendedAt = &now
		}
		if state == "archived" {
			ls.ArchivedAt = &now
		}
		if err := r.UpsertState(ctx, ls); err != nil {
			continue
		}
		updated++
	}
	return updated, nil
}

// RunGhostReconciliation compares GGID users against HR active employees.
func (r *dormantRepo) RunGhostReconciliation(ctx context.Context, ggidUsers []UserInfo, hrActiveIDs map[string]bool) ([]*GhostAccount, error) {
	if r.pool == nil {
		return []*GhostAccount{}, nil
	}
	var ghosts []*GhostAccount
	for _, u := range ggidUsers {
		if !hrActiveIDs[u.EmployeeID] {
			// User in GGID but not in HR → potential ghost.
			rec := "disable"
			if u.Status == "disabled" || u.Status == "archived" {
				rec = "archive"
			}
			g := &GhostAccount{
				UserID: u.UserID, Email: u.Email,
				Recommendation: rec, Status: "flagged",
				DetectedAt: time.Now().UTC(),
			}
			r.CreateGhost(ctx, g.UserID, g.Email, g.Recommendation)
			ghosts = append(ghosts, g)
		}
	}
	return ghosts, nil
}

// ProcessHREventJML handles HR events and triggers JML lifecycle actions.
func (r *dormantRepo) ProcessHREventJML(ctx context.Context, event *HREvent) string {
	switch event.EventType {
	case "terminated":
		// JML: disable user account.
		return "jml:disable"
	case "hired":
		// JML: create/provision user.
		return "jml:create"
	case "dept_change":
		// JML: trigger access review for new department.
		return "jml:access_review"
	case "manager_change":
		// JML: update approvals/workflows.
		return "jml:update_manager"
	default:
		return "jml:none"
	}
}

// --- Supporting Types ---

type UserLoginInfo struct {
	UserID     string
	LastLoginAt *time.Time
}

type UserInfo struct {
	UserID     string
	Email      string
	EmployeeID string
	Status     string
}

var _ = json.Marshal
var _ = uuid.New
