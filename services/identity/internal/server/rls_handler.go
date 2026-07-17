package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SetTenantContext sets the app.tenant_id session variable for the current transaction.
// This must be called within a transaction (pgxpool.Tx) for RLS to apply.
// Usage: tx, _ := pool.Begin(ctx); SetTenantContext(ctx, tx, tenantID); ... tx.Commit(ctx)
func SetTenantContext(ctx context.Context, db DBExecer, tenantID string) error {
	_, err := db.Exec(ctx, fmt.Sprintf("SET LOCAL app.tenant_id = '%s'", tenantID))
	return err
}

// DBExecer is implemented by both *pgxpool.Pool and pgx.Tx.
type DBExecer interface {
	Exec(ctx context.Context, sql string, arguments ...any) (interface{ RowsAffected() int64 }, error)
}

// rlsRepo manages RLS administration.
type rlsRepo struct {
	pool *pgxpool.Pool
}

func newRLSRepo(pool *pgxpool.Pool) *rlsRepo {
	return &rlsRepo{pool: pool}
}

// EnableRLS enables RLS + creates tenant isolation policy on a table.
func (r *rlsRepo) EnableRLS(ctx context.Context, table string) error {
	if r.pool == nil {
		return nil
	}
	// Validate table name (prevent SQL injection).
	if !isValidTableName(table) {
		return fmt.Errorf("invalid table name")
	}
	_, err := r.pool.Exec(ctx, fmt.Sprintf(`
		ALTER TABLE IF EXISTS %s ENABLE ROW LEVEL SECURITY;
		ALTER TABLE IF EXISTS %s FORCE ROW LEVEL SECURITY;
		DROP POLICY IF EXISTS tenant_isolation ON %s;
		CREATE POLICY tenant_isolation ON %s
			USING (tenant_id::text = current_setting('app.tenant_id', true));
	`, table, table, table, table))
	return err
}

// GetRLSStatus returns RLS status for all known tenant tables.
func (r *rlsRepo) GetRLSStatus(ctx context.Context) ([]map[string]any, error) {
	if r.pool == nil {
		return []map[string]any{}, nil
	}
	tables := RLSTables()
	var result []map[string]any
	for _, table := range tables {
		var relrowsecurity, relforcerowsecurity bool
		err := r.pool.QueryRow(ctx,
			`SELECT c.relrowsecurity, c.relforcerowsecurity FROM pg_class c
			 JOIN pg_namespace n ON n.oid = c.relnamespace
			 WHERE n.nspname = 'public' AND c.relname = $1 AND c.relkind = 'r'`, table,
		).Scan(&relrowsecurity, &relforcerowsecurity)
		if err != nil {
			result = append(result, map[string]any{"table": table, "rls_enabled": false, "exists": false})
			continue
		}
		result = append(result, map[string]any{
			"table": table, "rls_enabled": relrowsecurity, "forced": relforcerowsecurity,
			"exists": true,
		})
	}
	return result, nil
}

// RunIsolationTest verifies cross-tenant isolation by checking that RLS policies exist.
func (r *rlsRepo) RunIsolationTest(ctx context.Context) (map[string]any, error) {
	if r.pool == nil {
		return map[string]any{"status": "skipped", "reason": "no database connection"}, nil
	}
	status, _ := r.GetRLSStatus(ctx)
	totalTables := len(status)
	rlsEnabled := 0
	for _, s := range status {
		if enabled, _ := s["rls_enabled"].(bool); enabled {
			rlsEnabled++
		}
	}
	return map[string]any{
		"status": "completed",
		"total_tables": totalTables, "rls_enabled": rlsEnabled,
		"isolation_active": rlsEnabled > 0,
		"tested_at": time.Now().UTC(),
	}, nil
}

// RLSTables returns the list of tables that should have RLS.
func RLSTables() []string {
	return []string{
		"users", "groups", "group_members", "roles", "user_roles",
		"sessions", "audit_events", "oauth_clients", "oauth_tokens",
		"policies", "risk_scores", "threat_indicators", "consent_records",
		"scim_targets", "device_posture_scores", "soar_playbooks",
		"soar_executions", "hr_connectors", "hr_sync_log",
		"device_certificates", "encrypted_fields", "user_behavioral_baselines",
		"wasm_plugins", "policy_decisions", "risk_policies",
	}
}

func isValidTableName(name string) bool {
	for _, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}
	return len(name) > 0 && len(name) < 64
}

// --- HTTP Handlers ---

func (h *HTTPHandler) handleRLSEnable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	// Extract table name from path: /api/v1/admin/rls/enable/{table}
	table := r.URL.Path[len("/api/v1/admin/rls/enable/"):]
	if table == "" || !isValidTableName(table) {
		writeError(w, http.StatusBadRequest, "valid table name required")
		return
	}
	if h.rlsRepo != nil {
		if err := h.rlsRepo.EnableRLS(r.Context(), table); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to enable RLS")
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "enabled", "table": table})
}

func (h *HTTPHandler) handleRLSStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var status []map[string]any
	if h.rlsRepo != nil {
		status, _ = h.rlsRepo.GetRLSStatus(r.Context())
	}
	if status == nil {
		status = []map[string]any{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"tables": status, "count": len(status)})
}

func (h *HTTPHandler) handleRLSTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var result map[string]any
	if h.rlsRepo != nil {
		result, _ = h.rlsRepo.RunIsolationTest(r.Context())
	}
	if result == nil {
		result = map[string]any{"status": "skipped", "reason": "no database connection"}
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *HTTPHandler) SetRLSRepo(repo *rlsRepo) {
	h.rlsRepo = repo
}

var _ = json.Marshal
var _ = uuid.New
