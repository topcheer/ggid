package server

import (
	"fmt"
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type pgPARStore struct{ pool *pgxpool.Pool }

type parAdapter struct {
	pg  *pgPARStore
	mu  *sync.RWMutex
	mem map[string]*PAREntry
}

var parAdapterVar *parAdapter

func newPARAdapter(pool *pgxpool.Pool) *parAdapter {
	a := &parAdapter{mu: &sync.RWMutex{}, mem: make(map[string]*PAREntry)}
	if pool != nil {
		a.pg = &pgPARStore{pool: pool}
		ctx := context.Background()
		a.pg.EnsureSchema(ctx)
	}
	return a
}

func (s *pgPARStore) EnsureSchema(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS par_requests (request_uri TEXT PRIMARY KEY, client_id TEXT NOT NULL, params JSONB DEFAULT '{}', signed_request_object TEXT DEFAULT '', created_at TIMESTAMPTZ DEFAULT NOW(), expires_at TIMESTAMPTZ NOT NULL, used BOOLEAN DEFAULT FALSE)`)
	if err != nil { return fmt.Errorf("schema operation failed: %w", err) }
	return nil
}

func (s *pgPARStore) Put(ctx context.Context, e *PAREntry) error {
	params, _ := json.Marshal(e.Params)
	_, err := s.pool.Exec(ctx, `INSERT INTO par_requests (request_uri, client_id, params, signed_request_object, created_at, expires_at, used) VALUES ($1,$2,$3,$4,$5,$6,$7) ON CONFLICT (request_uri) DO UPDATE SET used=$7`, e.RequestURI, e.ClientID, params, e.SignedRequestObject, e.CreatedAt, e.ExpiresAt, e.Used)
	if err != nil { return fmt.Errorf("schema operation failed: %w", err) }
	return nil
}

func (s *pgPARStore) Get(ctx context.Context, requestURI string) (*PAREntry, bool) {
	var e PAREntry; var paramsBytes []byte
	err := s.pool.QueryRow(ctx, `SELECT request_uri, client_id, params, signed_request_object, created_at, expires_at, used FROM par_requests WHERE request_uri = $1`, requestURI).Scan(&e.RequestURI, &e.ClientID, &paramsBytes, &e.SignedRequestObject, &e.CreatedAt, &e.ExpiresAt, &e.Used)
	if err != nil { return nil, false }
	if len(paramsBytes) > 0 { json.Unmarshal(paramsBytes, &e.Params) }
	return &e, true
}

func (s *pgPARStore) MarkUsed(ctx context.Context, requestURI string) error {
	_, err := s.pool.Exec(ctx, `UPDATE par_requests SET used = TRUE WHERE request_uri = $1`, requestURI)
	if err != nil { return fmt.Errorf("schema operation failed: %w", err) }
	return nil
}

func (s *pgPARStore) CleanExpired(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM par_requests WHERE expires_at < NOW()`, )
	if err != nil { return fmt.Errorf("schema operation failed: %w", err) }
	return nil
}

func (a *parAdapter) Put(e *PAREntry) {
	if a.pg != nil { a.pg.Put(context.Background(), e); return }
	a.mu.Lock(); a.mem[e.RequestURI] = e; a.mu.Unlock()
}

func (a *parAdapter) Get(requestURI string) (*PAREntry, bool) {
	if a.pg != nil { e, ok := a.pg.Get(context.Background(), requestURI); if ok { return e, true } }
	a.mu.RLock(); e, ok := a.mem[requestURI]; a.mu.RUnlock()
	return e, ok
}

func (a *parAdapter) MarkUsed(requestURI string) {
	if a.pg != nil { a.pg.MarkUsed(context.Background(), requestURI); return }
	a.mu.Lock(); if e, ok := a.mem[requestURI]; ok { e.Used = true }; a.mu.Unlock()
}

var _ = time.Now
