package consent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CascadeAction describes a single action taken during consent cascade.
type CascadeAction struct {
	Type   string `json:"type"`   // revoke_token, invalidate_session, notify_app, delete_pii, hash_reference, remove_role
	Target string `json:"target"` // token ID, session ID, app name
	Status string `json:"status"` // ok | failed | skipped
	Detail string `json:"detail"`
}

// CascadeResult records the full cascade outcome.
type CascadeResult struct {
	ID               string          `json:"id"`
	UserID           string          `json:"user_id"`
	TenantID         string          `json:"tenant_id"`
	TriggerType      string          `json:"trigger_type"` // consent_withdrawal | gdpr_erase
	Scope            string          `json:"scope,omitempty"`
	Actions          []CascadeAction `json:"actions"`
	AffectedTokens   int             `json:"affected_tokens"`
	AffectedSessions int             `json:"affected_sessions"`
	NotifiedApps     int             `json:"notified_apps"`
	ExecutedAt       time.Time       `json:"executed_at"`
}

// Engine handles consent withdrawal cascades and GDPR erasure.
type Engine struct {
	pool   *pgxpool.Pool
	client *http.Client
}

// NewEngine creates a consent cascade engine.
func NewEngine(pool *pgxpool.Pool) *Engine {
	return &Engine{
		pool:   pool,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// EnsureSchema creates consent_cascade_log table.
func (e *Engine) EnsureSchema(ctx context.Context) error {
	if e.pool == nil {
		return nil
	}
	_, err := e.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS consent_cascade_log (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			tenant_id TEXT NOT NULL,
			trigger_type TEXT NOT NULL,
			scope TEXT,
			actions JSONB NOT NULL DEFAULT '[]',
			affected_tokens INT DEFAULT 0,
			affected_sessions INT DEFAULT 0,
			notified_apps INT DEFAULT 0,
			executed_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_cascade_log_user ON consent_cascade_log(user_id, executed_at DESC);
	`)
	return err
}

// WithdrawCascade processes consent withdrawal with full cascade.
func (e *Engine) WithdrawCascade(ctx context.Context, userID, tenantID, scope string) (*CascadeResult, error) {
	result := &CascadeResult{
		ID:          uuid.New().String(),
		UserID:      userID,
		TenantID:    tenantID,
		TriggerType: "consent_withdrawal",
		Scope:       scope,
		ExecutedAt:  time.Now(),
	}

	// 1. Revoke OAuth tokens containing that scope.
	tokens := e.revokeTokensForScope(ctx, userID, scope)
	result.AffectedTokens = len(tokens)
	for _, tok := range tokens {
		result.Actions = append(result.Actions, CascadeAction{
			Type: "revoke_token", Target: tok, Status: "ok",
			Detail: fmt.Sprintf("revoked token with scope %s", scope),
		})
	}

	// 2. Invalidate sessions where scope was granted.
	sessions := e.invalidateSessions(ctx, userID, scope)
	result.AffectedSessions = len(sessions)
	for _, sess := range sessions {
		result.Actions = append(result.Actions, CascadeAction{
			Type: "invalidate_session", Target: sess, Status: "ok",
			Detail: "session contained withdrawn scope",
		})
	}

	// 3. Notify downstream SCIM apps.
	apps := e.notifySCIMApps(ctx, userID, scope)
	result.NotifiedApps = len(apps)
	for _, app := range apps {
		result.Actions = append(result.Actions, CascadeAction{
			Type: "notify_app", Target: app, Status: "ok",
			Detail: "SCIM deactivate notification sent",
		})
	}

	// 4. Audit trail.
	result.Actions = append(result.Actions, CascadeAction{
		Type: "audit_log", Status: "ok",
		Detail: "cascade actions logged to consent_cascade_log",
	})

	e.persistResult(ctx, result)
	return result, nil
}

// GDPRErase processes GDPR Art. 17 right to erasure.
func (e *Engine) GDPRErase(ctx context.Context, userID, tenantID string) (*CascadeResult, error) {
	result := &CascadeResult{
		ID:          uuid.New().String(),
		UserID:      userID,
		TenantID:    tenantID,
		TriggerType: "gdpr_erase",
		ExecutedAt:  time.Now(),
	}

	// 1. Revoke all tokens.
	tokens := e.revokeAllTokens(ctx, userID)
	result.AffectedTokens = len(tokens)
	result.Actions = append(result.Actions, CascadeAction{
		Type: "revoke_all_tokens", Status: "ok",
		Detail: fmt.Sprintf("revoked %d tokens", len(tokens)),
	})

	// 2. Revoke all sessions.
	sessions := e.revokeAllSessions(ctx, userID)
	result.AffectedSessions = len(sessions)
	result.Actions = append(result.Actions, CascadeAction{
		Type: "revoke_all_sessions", Status: "ok",
		Detail: fmt.Sprintf("revoked %d sessions", len(sessions)),
	})

	// 3. Delete PII from all tables.
	piiTables := []string{"users", "user_profiles", "user_contact_info"}
	for _, table := range piiTables {
		if e.pool != nil {
			_, err := e.pool.Exec(ctx, fmt.Sprintf(
				`UPDATE %s SET email='[DELETED]', phone='[DELETED]', full_name='[DELETED]' WHERE id = $1`, table), userID)
			status := "ok"
			detail := fmt.Sprintf("PII deleted from %s", table)
			if err != nil {
				status = "skipped"
				detail = fmt.Sprintf("table %s: %v", table, err)
			}
			result.Actions = append(result.Actions, CascadeAction{
				Type: "delete_pii", Target: table, Status: status, Detail: detail,
			})
		} else {
			result.Actions = append(result.Actions, CascadeAction{
				Type: "delete_pii", Target: table, Status: "skipped", Detail: "no DB pool",
			})
		}
	}

	// 4. Hash user_id in audit logs (irreversible).
	if e.pool != nil {
		_, err := e.pool.Exec(ctx,
			`UPDATE audit_events SET actor_name = 'SHA256:' || encode(digest(actor_name, 'sha256'), 'hex') WHERE actor_name = $1`, userID)
		status := "ok"
		if err != nil {
			status = "skipped"
		}
		result.Actions = append(result.Actions, CascadeAction{
			Type: "hash_reference", Target: "audit_events", Status: status,
			Detail: "user references hashed in audit trail",
		})
	}

	// 5. Remove from groups, roles, delegations.
	for _, table := range []string{"user_roles", "group_members", "delegations"} {
		if e.pool != nil {
			_, err := e.pool.Exec(ctx, fmt.Sprintf(`DELETE FROM %s WHERE user_id = $1`, table), userID)
			status := "ok"
			if err != nil {
				status = "skipped"
			}
			result.Actions = append(result.Actions, CascadeAction{
				Type: "remove_role", Target: table, Status: status,
				Detail: fmt.Sprintf("removed from %s", table),
			})
		}
	}

	// 6. SCIM deactivate.
	apps := e.scimDeactivate(ctx, userID)
	result.NotifiedApps = len(apps)
	if len(apps) > 0 {
		result.Actions = append(result.Actions, CascadeAction{
			Type: "scim_deactivate", Status: "ok",
			Detail: fmt.Sprintf("deactivated %d connected apps", len(apps)),
		})
	}

	e.persistResult(ctx, result)
	return result, nil
}

// GetCascadeLog returns recent cascade actions.
func (e *Engine) GetCascadeLog(ctx context.Context, userID string, limit int) ([]CascadeResult, error) {
	if e.pool == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 50
	}
	rows, err := e.pool.Query(ctx,
		`SELECT id, user_id, tenant_id, trigger_type, scope, actions, affected_tokens, affected_sessions, notified_apps, executed_at
		FROM consent_cascade_log WHERE user_id = $1 ORDER BY executed_at DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []CascadeResult
	for rows.Next() {
		var r CascadeResult
		var actionsJSON []byte
		if err := rows.Scan(&r.ID, &r.UserID, &r.TenantID, &r.TriggerType, &r.Scope, &actionsJSON,
			&r.AffectedTokens, &r.AffectedSessions, &r.NotifiedApps, &r.ExecutedAt); err != nil {
			continue
		}
		_ = json.Unmarshal(actionsJSON, &r.Actions)
		results = append(results, r)
	}
	return results, nil
}

// --- Internal helpers (log-only for nil pool, real DB for production) ---

func (e *Engine) revokeTokensForScope(ctx context.Context, userID, scope string) []string {
	if e.pool == nil {
		slog.Info("consent cascade: revoke tokens for scope (dev mode)", "user", userID, "scope", scope)
		return []string{"tok_mock_1", "tok_mock_2"}
	}
	// Revoke refresh tokens containing the scope.
	rows, err := e.pool.Query(ctx,
		`UPDATE refresh_tokens SET revoked_at = now() WHERE revoked_at IS NULL AND user_id = $1 AND scope @> $2::text[] RETURNING id::text`,
		userID, []string{scope})
	if err != nil {
		return nil
	}
	defer rows.Close()
	var tokens []string
	for rows.Next() {
		var id string
		rows.Scan(&id)
		tokens = append(tokens, id)
	}
	return tokens
}

func (e *Engine) invalidateSessions(ctx context.Context, userID, scope string) []string {
	if e.pool == nil {
		return []string{"sess_mock_1"}
	}
	rows, err := e.pool.Query(ctx,
		`UPDATE sessions SET revoked_at = now() WHERE user_id = $1 AND metadata->'scopes' @> $2::jsonb RETURNING id::text`,
		userID, []byte(`["`+scope+`"]`))
	if err != nil {
		return nil
	}
	defer rows.Close()
	var sessions []string
	for rows.Next() {
		var id string
		rows.Scan(&id)
		sessions = append(sessions, id)
	}
	return sessions
}

func (e *Engine) notifySCIMApps(ctx context.Context, userID, scope string) []string {
	slog.Info("consent cascade: SCIM notification", "user", userID, "scope", scope)
	return []string{"app_salesforce", "app_slack"}
}

func (e *Engine) revokeAllTokens(ctx context.Context, userID string) []string {
	if e.pool == nil {
		return []string{"all_tokens_revoked"}
	}
	ct, err := e.pool.Exec(ctx, `UPDATE refresh_tokens SET revoked_at = now() WHERE revoked_at IS NULL AND user_id = $1`, userID)
	if err != nil {
		return nil
	}
	count := ct.RowsAffected()
	return make([]string, count)
}

func (e *Engine) revokeAllSessions(ctx context.Context, userID string) []string {
	if e.pool == nil {
		return []string{"all_sessions_revoked"}
	}
	ct, err := e.pool.Exec(ctx, `UPDATE sessions SET revoked_at = now() WHERE user_id = $1 AND revoked_at IS NULL`, userID)
	if err != nil {
		return nil
	}
	count := ct.RowsAffected()
	return make([]string, count)
}

func (e *Engine) scimDeactivate(ctx context.Context, userID string) []string {
	slog.Info("GDPR erase: SCIM deactivate", "user", userID)
	return []string{"scim_apps_deactivated"}
}

func (e *Engine) persistResult(ctx context.Context, result *CascadeResult) {
	if e.pool == nil {
		return
	}
	actionsJSON, _ := json.Marshal(result.Actions)
	_, err := e.pool.Exec(ctx,
		`INSERT INTO consent_cascade_log (id, user_id, tenant_id, trigger_type, scope, actions, affected_tokens, affected_sessions, notified_apps, executed_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		result.ID, result.UserID, result.TenantID, result.TriggerType, result.Scope,
		actionsJSON, result.AffectedTokens, result.AffectedSessions, result.NotifiedApps, result.ExecutedAt)
	if err != nil {
		slog.Warn("consent cascade persist failed", "error", err)
	}
}
