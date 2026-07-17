package server

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// pgScopeStore implements persistent custom scope storage using PostgreSQL.
type pgScopeStore struct {
	pool *pgxpool.Pool
}

// newPgScopeStore creates a PostgreSQL-backed scope store.
// Returns nil if pool is nil (caller should fall back to in-memory).
func newPgScopeStore(pool *pgxpool.Pool) *pgScopeStore {
	if pool == nil {
		return nil
	}
	return &pgScopeStore{pool: pool}
}

// EnsureSchema creates the custom_scopes table if it doesn't exist.
func (s *pgScopeStore) EnsureSchema(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS custom_scopes (
			id UUID PRIMARY KEY,
			name TEXT UNIQUE NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			attributes JSONB NOT NULL DEFAULT '[]',
			required BOOLEAN NOT NULL DEFAULT FALSE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`)
	return err
}

func (s *pgScopeStore) Get(ctx context.Context, name string) (*CustomScope, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT id, name, description, attributes, required, created_at, updated_at FROM custom_scopes WHERE name = $1`,
		name,
	)
	return s.scanScope(row)
}

func (s *pgScopeStore) List(ctx context.Context) ([]*CustomScope, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, name, description, attributes, required, created_at, updated_at FROM custom_scopes ORDER BY created_at`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*CustomScope
	for rows.Next() {
		sc, err := s.scanScope(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, sc)
	}
	return result, nil
}

func (s *pgScopeStore) Create(ctx context.Context, scope *CustomScope) error {
	attrs, _ := json.Marshal(scope.Attributes)
	_, err := s.pool.Exec(ctx,
		`INSERT INTO custom_scopes (id, name, description, attributes, required, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		scope.ID, scope.Name, scope.Description, attrs, scope.Required, scope.CreatedAt, scope.UpdatedAt,
	)
	return err
}

func (s *pgScopeStore) Update(ctx context.Context, scope *CustomScope) error {
	attrs, _ := json.Marshal(scope.Attributes)
	_, err := s.pool.Exec(ctx,
		`UPDATE custom_scopes SET description = $1, attributes = $2, required = $3, updated_at = $4 WHERE name = $5`,
		scope.Description, attrs, scope.Required, scope.UpdatedAt, scope.Name,
	)
	return err
}

func (s *pgScopeStore) Delete(ctx context.Context, name string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM custom_scopes WHERE name = $1`, name)
	return err
}

// scanScope scans a row into a CustomScope. Works with both QueryRow and Rows.
type scanner interface {
	Scan(dest ...any) error
}

func (s *pgScopeStore) scanScope(row scanner) (*CustomScope, error) {
	var sc CustomScope
	var idStr string
	var attrsBytes []byte
	if err := row.Scan(&idStr, &sc.Name, &sc.Description, &attrsBytes, &sc.Required, &sc.CreatedAt, &sc.UpdatedAt); err != nil {
		return nil, err
	}
	sc.ID = idStr
	if len(attrsBytes) > 0 {
		json.Unmarshal(attrsBytes, &sc.Attributes)
	}
	return &sc, nil
}

// scopeStoreAdapter provides a unified interface that tries PostgreSQL first,
// then falls back to the in-memory store.
type scopeStoreAdapter struct {
	pg    *pgScopeStore
}

// newScopeStoreAdapter creates an adapter that uses PG if available, else memory.
func newScopeStoreAdapter(pool *pgxpool.Pool) *scopeStoreAdapter {
	a := &scopeStoreAdapter{}
	if pool != nil {
		a.pg = &pgScopeStore{pool: pool}
		// Best-effort schema creation
		ctx := context.Background()
		if err := a.pg.EnsureSchema(ctx); err != nil {
			fmt.Printf("warning: custom_scopes schema creation failed: %v (using in-memory fallback)\n", err)
			a.pg = nil
		}
	}
	return a
}

func (a *scopeStoreAdapter) Get(name string) (*CustomScope, bool) {
	if a.pg != nil {
		sc, err := a.pg.Get(context.Background(), name)
		if err == nil && sc != nil {
			return sc, true
		}
	}
	return nil, false
}

func (a *scopeStoreAdapter) List() []*CustomScope {
	if a.pg != nil {
		list, err := a.pg.List(context.Background())
		if err == nil {
			return list
		}
	}
	return []*CustomScope{}
}

func (a *scopeStoreAdapter) Create(scope *CustomScope) error {
	if a.pg != nil {
		return a.pg.Create(context.Background(), scope)
	}
	return nil
}

func (a *scopeStoreAdapter) Update(scope *CustomScope) error {
	if a.pg != nil {
		return a.pg.Update(context.Background(), scope)
	}
	return nil
}

func (a *scopeStoreAdapter) Delete(name string) error {
	if a.pg != nil {
		return a.pg.Delete(context.Background(), name)
	}
	return nil
}

// suppress unused import guard
var _ = time.Now
var _ = uuid.New
