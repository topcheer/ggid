package server

import (
	"fmt"
	"context"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type pgDPoPStore struct{ pool *pgxpool.Pool }

type dpopAdapter struct {
	pg  *pgDPoPStore
	mu  *sync.RWMutex
	binds map[string]string
}

var dpopAdapterVar *dpopAdapter

func newDPoPAdapter(pool *pgxpool.Pool) *dpopAdapter {
	a := &dpopAdapter{mu: &dpopBindings.mu, binds: dpopBindings.binds}
	if pool != nil {
		a.pg = &pgDPoPStore{pool: pool}
		ctx := context.Background()
		a.pg.EnsureSchema(ctx)
	}
	return a
}

func (s *pgDPoPStore) EnsureSchema(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS dpop_bindings (token_hash TEXT PRIMARY KEY, jkt TEXT NOT NULL, created_at TIMESTAMPTZ DEFAULT NOW())`)
	if err != nil { return fmt.Errorf("schema operation failed: %w", err) }
	return nil
}

func (s *pgDPoPStore) Put(ctx context.Context, tokenHash, jkt string) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO dpop_bindings (token_hash, jkt, created_at) VALUES ($1,$2,NOW()) ON CONFLICT (token_hash) DO UPDATE SET jkt=$2`, tokenHash, jkt)
	if err != nil { return fmt.Errorf("schema operation failed: %w", err) }
	return nil
}

func (s *pgDPoPStore) Get(ctx context.Context, tokenHash string) (string, bool) {
	var jkt string
	err := s.pool.QueryRow(ctx, `SELECT jkt FROM dpop_bindings WHERE token_hash = $1`, tokenHash).Scan(&jkt)
	if err != nil { return "", false }
	return jkt, true
}

func (a *dpopAdapter) Bind(token, jkt string) {
	if a.pg != nil { a.pg.Put(context.Background(), token, jkt); return }
	a.mu.Lock(); a.binds[token] = jkt; a.mu.Unlock()
}

func (a *dpopAdapter) Get(token string) string {
	if a.pg != nil { jkt, ok := a.pg.Get(context.Background(), token); if ok { return jkt } }
	a.mu.RLock(); defer a.mu.RUnlock()
	return a.binds[token]
}

var _ = time.Now
