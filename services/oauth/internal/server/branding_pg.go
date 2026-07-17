package server

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// brandingStoreAdapter: PG-first with in-memory fallback.

type pgBrandingStore struct {
	pool *pgxpool.Pool
}

func newBrandingAdapter(pool *pgxpool.Pool) *brandingAdapter {
	a := &brandingAdapter{}
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
	if err != nil { return fmt.Errorf("schema operation failed: %w", err) }
	return nil
}

func (s *pgBrandingStore) Get(ctx context.Context, clientID string) (*ClientBranding, bool) {
	var b ClientBranding
	err := s.pool.QueryRow(ctx, `SELECT logo_url, primary_color, background_url, custom_css FROM client_branding WHERE client_id = $1`, clientID).Scan(&b.LogoURL, &b.PrimaryColor, &b.BackgroundURL, &b.CustomCSS)
	if err != nil { return nil, false }
	return &b, true
}

func (s *pgBrandingStore) Put(ctx context.Context, clientID string, b *ClientBranding) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO client_branding (client_id, logo_url, primary_color, background_url, custom_css, updated_at) VALUES ($1,$2,$3,$4,$5,NOW()) ON CONFLICT (client_id) DO UPDATE SET logo_url=$2, primary_color=$3, background_url=$4, custom_css=$5, updated_at=NOW()`, clientID, b.LogoURL, b.PrimaryColor, b.BackgroundURL, b.CustomCSS)
	if err != nil { return fmt.Errorf("schema operation failed: %w", err) }
	return nil
}

type brandingAdapter struct {
	pg  *pgBrandingStore
	mem map[string]*ClientBranding // test/dev fallback when no PG or mapRepoVar
}

func (a *brandingAdapter) Get(clientID string) (*ClientBranding, bool) {
	if a.pg != nil {
		b, ok := a.pg.Get(context.Background(), clientID)
		if ok { return b, true }
	}
	if mapRepoVar != nil {
		if row, _ := mapRepoVar.Get(context.Background(), "oauth_branding", clientID); row != nil {
			return &ClientBranding{
				LogoURL: omGetString(row, "logo_url"),
				PrimaryColor: omGetString(row, "primary_color"),
				BackgroundURL: omGetString(row, "background_url"),
				CustomCSS: omGetString(row, "custom_css"),
			}, true
		}
	}
	if a.mem != nil {
		b, ok := a.mem[clientID]
		return b, ok
	}
	return nil, false
}

func (a *brandingAdapter) Put(clientID string, b *ClientBranding) {
	if a.pg != nil { a.pg.Put(context.Background(), clientID, b); return }
	if mapRepoVar != nil {
		mapRepoVar.Store(context.Background(), "oauth_branding", clientID, map[string]any{
			"logo_url": b.LogoURL, "primary_color": b.PrimaryColor,
			"background_url": b.BackgroundURL, "custom_css": b.CustomCSS,
		})
		return
	}
	if a.mem == nil { a.mem = make(map[string]*ClientBranding) }
	a.mem[clientID] = b
}

var brandingAdapterVar *brandingAdapter
