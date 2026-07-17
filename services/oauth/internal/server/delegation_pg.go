package server

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type pgDelegationStore struct{ pool *pgxpool.Pool }

type delegationAdapter struct {
	pg    *pgDelegationStore
	mu    *sync.RWMutex
	chains map[string][]DelegationEntry
}

var delegationAdapterVar *delegationAdapter

func newDelegationAdapter(pool *pgxpool.Pool) *delegationAdapter {
	a := &delegationAdapter{}
	if pool != nil {
		a.pg = &pgDelegationStore{pool: pool}
		ctx := context.Background()
		a.pg.EnsureSchema(ctx)
	}
	return a
}

func (s *pgDelegationStore) EnsureSchema(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS delegation_chains (token_id TEXT PRIMARY KEY, chain JSONB NOT NULL DEFAULT '[]', created_at TIMESTAMPTZ DEFAULT NOW())`)
	return err
}

func (s *pgDelegationStore) Put(ctx context.Context, tokenID string, chain []DelegationEntry) error {
	chainJSON, _ := json.Marshal(chain)
	_, err := s.pool.Exec(ctx, `INSERT INTO delegation_chains (token_id, chain, created_at) VALUES ($1,$2,NOW()) ON CONFLICT (token_id) DO UPDATE SET chain=$2`, tokenID, chainJSON)
	return err
}

func (s *pgDelegationStore) Get(ctx context.Context, tokenID string) ([]DelegationEntry, bool) {
	var chainBytes []byte
	err := s.pool.QueryRow(ctx, `SELECT chain FROM delegation_chains WHERE token_id = $1`, tokenID).Scan(&chainBytes)
	if err != nil { return nil, false }
	var chain []DelegationEntry
	json.Unmarshal(chainBytes, &chain)
	return chain, true
}

func (a *delegationAdapter) Put(tokenID string, chain []DelegationEntry) {
	if a.pg != nil { a.pg.Put(context.Background(), tokenID, chain); return }
	a.mu.Lock(); a.chains[tokenID] = chain; a.mu.Unlock()
}

func (a *delegationAdapter) Get(tokenID string) ([]DelegationEntry, bool) {
	if a.pg != nil { c, ok := a.pg.Get(context.Background(), tokenID); if ok { return c, true } }
	a.mu.RLock(); c, ok := a.chains[tokenID]; a.mu.RUnlock()
	return c, ok
}

var _ = time.Now
