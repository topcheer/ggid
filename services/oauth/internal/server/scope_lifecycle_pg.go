package server

import (
	"fmt"
	"encoding/json"
	"context"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

type pgScopeLifecycleStore struct{ pool *pgxpool.Pool }

type scopeLifecycleAdapter struct {
	pg   *pgScopeLifecycleStore
	mem  *sync.Map
}

var scopeLifecycleAdapterVar *scopeLifecycleAdapter

func newScopeLifecycleAdapter(pool *pgxpool.Pool) *scopeLifecycleAdapter {
	a := &scopeLifecycleAdapter{mem: &scopeLifecycleStore}
	if pool != nil {
		a.pg = &pgScopeLifecycleStore{pool: pool}
		ctx := context.Background()
		a.pg.EnsureSchema(ctx)
	}
	return a
}

func (s *pgScopeLifecycleStore) EnsureSchema(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS scope_lifecycle (scope_id TEXT PRIMARY KEY, requested_scope TEXT NOT NULL, requester TEXT DEFAULT '', approver_chain JSONB DEFAULT '[]', status TEXT DEFAULT 'pending', risk_level TEXT DEFAULT 'low', auto_expire_days INT DEFAULT 0, created_at TIMESTAMPTZ DEFAULT NOW(), updated_at TIMESTAMPTZ DEFAULT NOW())`)
	if err != nil { return fmt.Errorf("schema operation failed: %w", err) }
	return nil
}

func (s *pgScopeLifecycleStore) Put(ctx context.Context, sl *ScopeLifecycle) error {
	approverJSON, _ := json.Marshal(sl.ApproverChain)
	_, err := s.pool.Exec(ctx, `INSERT INTO scope_lifecycle (scope_id, requested_scope, requester, approver_chain, status, risk_level, auto_expire_days, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NOW()) ON CONFLICT (scope_id) DO UPDATE SET status=$5, risk_level=$6, auto_expire_days=$7, updated_at=NOW()`, sl.ScopeID, sl.RequestedScope, sl.Requester, approverJSON, sl.Status, sl.RiskLevel, sl.AutoExpireDays, sl.CreatedAt)
	if err != nil { return fmt.Errorf("schema operation failed: %w", err) }
	return nil
}

func (s *pgScopeLifecycleStore) Get(ctx context.Context, scopeID string) (*ScopeLifecycle, bool) {
	var sl ScopeLifecycle; var approverBytes []byte
	err := s.pool.QueryRow(ctx, `SELECT scope_id, requested_scope, requester, approver_chain, status, risk_level, auto_expire_days, created_at FROM scope_lifecycle WHERE scope_id = $1`, scopeID).Scan(&sl.ScopeID, &sl.RequestedScope, &sl.Requester, &approverBytes, &sl.Status, &sl.RiskLevel, &sl.AutoExpireDays, &sl.CreatedAt)
	if err != nil { return nil, false }
	if len(approverBytes) > 0 { json.Unmarshal(approverBytes, &sl.ApproverChain) }
	return &sl, true
}

func (s *pgScopeLifecycleStore) List(ctx context.Context) ([]*ScopeLifecycle, error) {
	rows, err := s.pool.Query(ctx, `SELECT scope_id, requested_scope, requester, approver_chain, status, risk_level, auto_expire_days, created_at FROM scope_lifecycle ORDER BY created_at DESC`)
	if err != nil { return nil, err }
	defer rows.Close()
	var result []*ScopeLifecycle
	for rows.Next() {
		var sl ScopeLifecycle; var approverBytes []byte
		if err := rows.Scan(&sl.ScopeID, &sl.RequestedScope, &sl.Requester, &approverBytes, &sl.Status, &sl.RiskLevel, &sl.AutoExpireDays, &sl.CreatedAt); err != nil { return nil, err }
		if len(approverBytes) > 0 { json.Unmarshal(approverBytes, &sl.ApproverChain) }
		result = append(result, &sl)
	}
	return result, nil
}

func (a *scopeLifecycleAdapter) Put(sl *ScopeLifecycle) {
	if a.pg != nil { a.pg.Put(context.Background(), sl); return }
	a.mem.Store(sl.ScopeID, sl)
}

func (a *scopeLifecycleAdapter) Get(scopeID string) (*ScopeLifecycle, bool) {
	if a.pg != nil { sl, ok := a.pg.Get(context.Background(), scopeID); if ok { return sl, true } }
	v, ok := a.mem.Load(scopeID)
	if !ok { return nil, false }
	return v.(*ScopeLifecycle), true
}

func (a *scopeLifecycleAdapter) List() []*ScopeLifecycle {
	if a.pg != nil { list, _ := a.pg.List(context.Background()); if list != nil { return list } }
	var result []*ScopeLifecycle
	a.mem.Range(func(_, v any) bool { result = append(result, v.(*ScopeLifecycle)); return true })
	return result
}
