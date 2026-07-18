package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	ggidtenant "github.com/ggid/ggid/pkg/tenant"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PrivilegedOperation records a privileged API action for compliance audit.
type PrivilegedOperation struct {
	ID            string                 `json:"id"`
	TenantID      string                 `json:"tenant_id"`
	OperatorID    string                 `json:"operator_id"`
	TargetID      string                 `json:"target_id,omitempty"`
	Action        string                 `json:"action"` // e.g. break_glass, jit_elevate, user_delete, policy_change
	ElevatedRole  string                 `json:"elevated_role,omitempty"`
	ScopesDelta   []string               `json:"scopes_delta,omitempty"`
	Duration      int                    `json:"duration_seconds,omitempty"`
	IPAddress     string                 `json:"ip_address"`
	UserAgent     string                 `json:"user_agent,omitempty"`
	BeforePerms   []string               `json:"before_perms,omitempty"`
	AfterPerms    []string               `json:"after_perms,omitempty"`
	Metadata      map[string]any         `json:"metadata,omitempty"`
	Timestamp     time.Time              `json:"timestamp"`
}

// privilegedOpRepo manages privileged_operations in PostgreSQL.
type privilegedOpRepo struct {
	pool *pgxpool.Pool
}

func newPrivilegedOpRepo(pool *pgxpool.Pool) *privilegedOpRepo {
	return &privilegedOpRepo{pool: pool}
}

// NewPrivilegedOpRepo is the exported constructor.
func NewPrivilegedOpRepo(pool *pgxpool.Pool) *privilegedOpRepo {
	return newPrivilegedOpRepo(pool)
}

func (r *privilegedOpRepo) EnsureSchema(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS privileged_operations (
			id            TEXT PRIMARY KEY,
			tenant_id     UUID NOT NULL,
			operator_id   UUID NOT NULL,
			target_id     UUID,
			action        TEXT NOT NULL,
			elevated_role TEXT DEFAULT '',
			scopes_delta  TEXT[] DEFAULT '{}',
			duration_seconds INTEGER DEFAULT 0,
			ip_address    TEXT DEFAULT '',
			user_agent    TEXT DEFAULT '',
			before_perms  TEXT[] DEFAULT '{}',
			after_perms   TEXT[] DEFAULT '{}',
			metadata      JSONB DEFAULT '{}'::jsonb,
			timestamp     TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_privil_op_tenant ON privileged_operations(tenant_id);
		CREATE INDEX IF NOT EXISTS idx_privil_op_operator ON privileged_operations(operator_id);
		CREATE INDEX IF NOT EXISTS idx_privil_op_timestamp ON privileged_operations(timestamp DESC);
	`)
	return err
}

// Record logs a privileged operation.
func (r *privilegedOpRepo) Record(ctx context.Context, op *PrivilegedOperation) error {
	if op.ID == "" {
		op.ID = "prv-" + uuid.New().String()[:8]
	}
	if op.Timestamp.IsZero() {
		op.Timestamp = time.Now().UTC()
	}
	metaJSON, _ := json.Marshal(op.Metadata)
	_, err := r.pool.Exec(ctx,
		`INSERT INTO privileged_operations
		 (id, tenant_id, operator_id, target_id, action, elevated_role, scopes_delta,
		  duration_seconds, ip_address, user_agent, before_perms, after_perms, metadata, timestamp)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		op.ID, op.TenantID, op.OperatorID, nilIfEmpty(op.TargetID),
		op.Action, op.ElevatedRole, op.ScopesDelta, op.Duration,
		op.IPAddress, op.UserAgent, op.BeforePerms, op.AfterPerms, metaJSON, op.Timestamp)
	return err
}

// List returns privileged operations for a tenant with optional filters.
func (r *privilegedOpRepo) List(ctx context.Context, tenantID uuid.UUID, operatorID string, action string, limit int) ([]*PrivilegedOperation, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}

	q := `SELECT id, tenant_id::text, operator_id::text, COALESCE(target_id::text,''),
	             action, elevated_role, scopes_delta, duration_seconds,
	             ip_address, user_agent, before_perms, after_perms, metadata, timestamp
	      FROM privileged_operations WHERE tenant_id = $1`
	args := []any{tenantID}

	if operatorID != "" {
		args = append(args, operatorID)
		q += " AND operator_id = $" + intToStr(len(args))
	}
	if action != "" {
		args = append(args, action)
		q += " AND action = $" + intToStr(len(args))
	}
	q += " ORDER BY timestamp DESC LIMIT " + intToStr(limit)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*PrivilegedOperation
	for rows.Next() {
		op := &PrivilegedOperation{}
		var metaJSON []byte
		if err := rows.Scan(&op.ID, &op.TenantID, &op.OperatorID, &op.TargetID,
			&op.Action, &op.ElevatedRole, &op.ScopesDelta, &op.Duration,
			&op.IPAddress, &op.UserAgent, &op.BeforePerms, &op.AfterPerms,
			&metaJSON, &op.Timestamp); err != nil {
			slog.Warn("privileged op scan error", "error", err)
			continue
		}
		if len(metaJSON) > 0 {
			_ = json.Unmarshal(metaJSON, &op.Metadata)
		}
		result = append(result, op)
	}
	return result, nil
}

// handlePrivilegedOperations handles GET /api/v1/identity/privileged-operations.
func (h *HTTPHandler) handlePrivilegedOperations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.privilOpRepo == nil {
		writeJSON(w, http.StatusOK, map[string]any{"operations": []any{}, "count": 0})
		return
	}
	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "X-Tenant-ID header required")
		return
	}

	operatorID := r.URL.Query().Get("operator_id")
	action := r.URL.Query().Get("action")
	limit := 100

	ops, err := h.privilOpRepo.List(r.Context(), tc.TenantID, operatorID, action, limit)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to list operations")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"operations": ops,
		"count":      len(ops),
	})
}

// RecordPrivilegedOp is a helper for other handlers to log privileged operations.
func (h *HTTPHandler) RecordPrivilegedOp(ctx context.Context, op *PrivilegedOperation) {
	if h.privilOpRepo == nil {
		return
	}
	if err := h.privilOpRepo.Record(ctx, op); err != nil {
		slog.Error("failed to record privileged op", "error", err, "action", op.Action)
	}
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[pos:])
}
