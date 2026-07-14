package server

import (
	"context"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

// brandingStoreAdapter: PG-first with in-memory fallback.

type pgBrandingStore struct {
	pool *pgxpool.Pool
}

func newBrandingAdapter(pool *pgxpool.Pool) *brandingAdapter {
	a := &brandingAdapter{mem: brandingStore, mu: &brandingMu}
	if pool != nil {
		a.pg = &pgBrandingStore{pool: pool}
		ctx := context.Background()
		a.pg.EnsureSchema(ctx)
	}
	return a
}

func (s *pgBrandingStore) EnsureSchema(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS client_branding (
		client_id TEXT PRIMARY KEY, logo_url TEXT DEFAULT '', primary_color TEXT DEFAULT '', background_url TEXT DEFAULT '', custom_css TEXT DEFAULT '', updated_at TIMESTAMPTZ DEFAULT NOW())`)
	return err
}

func (s *pgBrandingStore) Get(ctx context.Context, clientID string) (*ClientBranding, bool) {
	var b ClientBranding
	err := s.pool.QueryRow(ctx, `SELECT logo_url, primary_color, background_url, custom_css FROM client_branding WHERE client_id = $1`, clientID).Scan(&b.LogoURL, &b.PrimaryColor, &b.BackgroundURL, &b.CustomCSS)
	if err != nil { return nil, false }
	return &b, true
}

func (s *pgBrandingStore) Put(ctx context.Context, clientID string, b *ClientBranding) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO client_branding (client_id, logo_url, primary_color, background_url, custom_css, updated_at) VALUES ($1,$2,$3,$4,$5,NOW()) ON CONFLICT (client_id) DO UPDATE SET logo_url=$2, primary_color=$3, background_url=$4, custom_css=$5, updated_at=NOW()`, clientID, b.LogoURL, b.PrimaryColor, b.BackgroundURL, b.CustomCSS)
	return err
}

type brandingAdapter struct {
	pg  *pgBrandingStore
	mem map[string]*ClientBranding
	mu  *sync.RWMutex
}

func (a *brandingAdapter) Get(clientID string) (*ClientBranding, bool) {
	if a.pg != nil {
		b, ok := a.pg.Get(context.Background(), clientID)
		if ok { return b, true }
	}
	a.mu.RLock(); b, ok := a.mem[clientID]; a.mu.RUnlock()
	return b, ok
}

func (a *brandingAdapter) Put(clientID string, b *ClientBranding) {
	if a.pg != nil { a.pg.Put(context.Background(), clientID, b); return }
	a.mu.Lock(); a.mem[clientID] = b; a.mu.Unlock()
}

var brandingAdapterVar *brandingAdapter
