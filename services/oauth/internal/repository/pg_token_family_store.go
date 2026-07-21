package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// pg_token_family_store.go — Task-E: PostgreSQL-backed refresh-token family
// registry. Shares the oauth_token_families JSONB table (created by the
// server map repo's EnsureSchema) so the /token-families API view and the
// service-layer reuse detection read/write the same records.

// PGTokenFamilyStore implements service.TokenFamilyStore over
// oauth_token_families (id = family_id, data = family record JSON).
type PGTokenFamilyStore struct {
	pool *pgxpool.Pool
}

// NewPGTokenFamilyStore creates the store and ensures the table exists.
func NewPGTokenFamilyStore(pool *pgxpool.Pool) *PGTokenFamilyStore {
	return &PGTokenFamilyStore{pool: pool}
}

// EnsureSchema creates the backing table (idempotent).
func (s *PGTokenFamilyStore) EnsureSchema(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS oauth_token_families (
			id TEXT PRIMARY KEY, data JSONB DEFAULT '{}', created_at TIMESTAMPTZ DEFAULT now()
		);
		ALTER TABLE oidc_refresh_tokens ADD COLUMN IF NOT EXISTS family_id TEXT;
		CREATE INDEX IF NOT EXISTS idx_oidc_refresh_tokens_family ON oidc_refresh_tokens(family_id);
	`)
	return err
}

func (s *PGTokenFamilyStore) get(ctx context.Context, familyID string) (map[string]any, error) {
	var raw []byte
	err := s.pool.QueryRow(ctx, `SELECT data FROM oauth_token_families WHERE id = $1`, familyID).Scan(&raw)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func (s *PGTokenFamilyStore) put(ctx context.Context, familyID string, data map[string]any) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO oauth_token_families (id, data, created_at) VALUES ($1, $2, now())
		ON CONFLICT (id) DO UPDATE SET data = EXCLUDED.data
	`, familyID, raw)
	return err
}

// RegisterRotation records oldTokenID → newTokenID. A second rotation of the
// same old token flags theft_detected (defense in depth alongside the
// Used/Revoked check in the refresh flow).
func (s *PGTokenFamilyStore) RegisterRotation(ctx context.Context, familyID, oldTokenID, newTokenID string) error {
	existing, _ := s.get(ctx, familyID)
	data := familyRecordToJSONRepo(existing, familyID, oldTokenID, newTokenID, time.Now())
	return s.put(ctx, familyID, data)
}

// MarkTheft flags the family and marks every member revoked.
func (s *PGTokenFamilyStore) MarkTheft(ctx context.Context, familyID string) error {
	existing, _ := s.get(ctx, familyID)
	data := familyRecordMarkTheftRepo(existing, familyID, time.Now())
	return s.put(ctx, familyID, data)
}

// GetFamily returns the family record for the API view.
func (s *PGTokenFamilyStore) GetFamily(ctx context.Context, familyID string) (map[string]any, error) {
	m, err := s.get(ctx, familyID)
	if err != nil {
		return nil, fmt.Errorf("token family not found: %w", err)
	}
	return m, nil
}

// --- record shaping (mirrors service/token_family.go to keep one JSON shape) ---

func familyRecordToJSONRepo(existing map[string]any, familyID, oldTokenID, newTokenID string, now time.Time) map[string]any {
	data := existing
	if data == nil {
		data = map[string]any{
			"family_id":      familyID,
			"created_at":     now.UTC(),
			"tokens":         []any{},
			"theft_detected": false,
		}
	}
	tokens, _ := data["tokens"].([]any)
	found := false
	for i, t := range tokens {
		tm, ok := t.(map[string]any)
		if !ok {
			continue
		}
		if tid, _ := tm["token_id"].(string); tid == oldTokenID {
			if status, _ := tm["status"].(string); status == "rotated" {
				data["theft_detected"] = true
			}
			tm["status"] = "rotated"
			tm["rotated_to"] = newTokenID
			tokens[i] = tm
			found = true
			break
		}
	}
	if !found {
		tokens = append(tokens, map[string]any{
			"token_id":   oldTokenID,
			"issued_at":  now.UTC(),
			"status":     "rotated",
			"rotated_to": newTokenID,
		})
	}
	tokens = append(tokens, map[string]any{
		"token_id":  newTokenID,
		"issued_at": now.UTC(),
		"status":    "active",
	})
	data["tokens"] = tokens
	return data
}

func familyRecordMarkTheftRepo(existing map[string]any, familyID string, now time.Time) map[string]any {
	data := existing
	if data == nil {
		data = map[string]any{
			"family_id":  familyID,
			"created_at": now.UTC(),
			"tokens":     []any{},
		}
	}
	data["theft_detected"] = true
	tokens, _ := data["tokens"].([]any)
	for i, t := range tokens {
		if tm, ok := t.(map[string]any); ok {
			tm["status"] = "revoked"
			tokens[i] = tm
		}
	}
	data["tokens"] = tokens
	return data
}
