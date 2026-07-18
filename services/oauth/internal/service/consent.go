package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// --- Consent Persistence (OIDC Core §13) ---

// ConsentRecord stores a user's consent for a client's requested scopes.
type ConsentRecord struct {
	ID        uuid.UUID
	TenantID  uuid.UUID
	UserID    uuid.UUID
	ClientID  string
	Scopes    []string
	GrantedAt time.Time
	ExpiresAt time.Time
}

// ConsentStore is an interface for consent persistence.
type ConsentStore interface {
	Get(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID, clientID string) (*ConsentRecord, error)
	Save(ctx context.Context, record *ConsentRecord) error
	Delete(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID, clientID string) error
}

// --- PostgreSQL ConsentStore implementation ---

// pgConsentStore persists consent records to PostgreSQL.
// Falls back to no-op when pool is nil (test/dev mode without DB).
type pgConsentStore struct {
	pool *pgxpool.Pool
}

// NewPGConsentStore creates a PostgreSQL-backed consent store.
func NewPGConsentStore(pool *pgxpool.Pool) ConsentStore {
	return &pgConsentStore{pool: pool}
}

func consentKey(tenantID uuid.UUID, userID uuid.UUID, clientID string) string {
	return fmt.Sprintf("%s:%s:%s", tenantID, userID, clientID)
}

func (s *pgConsentStore) EnsureSchema(ctx context.Context) error {
	if s.pool == nil {
		return nil
	}
	_, err := s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS oauth_consent_records (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id UUID NOT NULL,
			user_id UUID NOT NULL,
			client_id TEXT NOT NULL,
			scopes TEXT[] DEFAULT '{}',
			granted_at TIMESTAMPTZ DEFAULT now(),
			expires_at TIMESTAMPTZ,
			withdrawn BOOLEAN DEFAULT FALSE,
			withdrawn_at TIMESTAMPTZ
		);
		CREATE INDEX IF NOT EXISTS idx_oauth_consent_key ON oauth_consent_records(tenant_id, user_id, client_id, withdrawn);
	`)
	return err
}

func (s *pgConsentStore) Get(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID, clientID string) (*ConsentRecord, error) {
	if s.pool == nil {
		return nil, nil
	}
	var rec ConsentRecord
	var scopes []string
	var withdrawn bool
	err := s.pool.QueryRow(ctx, `
		SELECT id, tenant_id, user_id, client_id, scopes, granted_at, expires_at
		FROM oauth_consent_records
		WHERE tenant_id=$1 AND user_id=$2 AND client_id=$3 AND withdrawn=FALSE
		AND (expires_at IS NULL OR expires_at > now())
		ORDER BY granted_at DESC LIMIT 1`, tenantID, userID, clientID,
	).Scan(&rec.ID, &rec.TenantID, &rec.UserID, &rec.ClientID, &scopes, &rec.GrantedAt, &rec.ExpiresAt)
	if err != nil {
		return nil, nil
	}
	rec.Scopes = scopes
	_ = withdrawn
	return &rec, nil
}

func (s *pgConsentStore) Save(ctx context.Context, record *ConsentRecord) error {
	if s.pool == nil {
		return nil
	}
	if record.ID == uuid.Nil {
		record.ID = uuid.New()
	}
	record.GrantedAt = time.Now().UTC()
	_, err := s.pool.Exec(ctx, `
		INSERT INTO oauth_consent_records (id, tenant_id, user_id, client_id, scopes, granted_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		record.ID, record.TenantID, record.UserID, record.ClientID,
		record.Scopes, record.GrantedAt, record.ExpiresAt)
	return err
}

func (s *pgConsentStore) Delete(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID, clientID string) error {
	if s.pool == nil {
		return nil
	}
	_, err := s.pool.Exec(ctx, `
		UPDATE oauth_consent_records SET withdrawn=TRUE, withdrawn_at=now()
		WHERE tenant_id=$1 AND user_id=$2 AND client_id=$3 AND withdrawn=FALSE`,
		tenantID, userID, clientID)
	return err
}

// --- Distributed Token Revocation (RFC 7009) ---

// RevocationStore is an interface for distributed token revocation.
type RevocationStore interface {
	Revoke(ctx context.Context, tokenID string, expiresAt time.Time) error
	IsRevoked(ctx context.Context, tokenID string) bool
}

// --- In-memory RevocationStore (wraps sync.Map for interface compat) ---
// Note: This is acceptable for revocation since revoked tokens are short-lived
// (they expire when the original token would have expired). PG-backed
// revocation is handled by SessionRevocationManager in the auth service.



func (s *memRevocationStore) Revoke(ctx context.Context, tokenID string, expiresAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.revoked[tokenID] = expiresAt
	return nil
}

func (s *memRevocationStore) IsRevoked(ctx context.Context, tokenID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	exp, ok := s.revoked[tokenID]
	if !ok {
		return false
	}
	if time.Now().After(exp) {
		return false
	}
	return true
}

func (s *memRevocationStore) CleanupExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for tokenID, exp := range s.revoked {
		if now.After(exp) {
			delete(s.revoked, tokenID)
		}
	}
}
