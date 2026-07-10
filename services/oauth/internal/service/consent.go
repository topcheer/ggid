package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
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
	ExpiresAt time.Time // optional expiry, zero = no expiry
}

// ConsentStore is an interface for consent persistence.
// Implementations: in-memory (default), PostgreSQL, Redis.
type ConsentStore interface {
	Get(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID, clientID string) (*ConsentRecord, error)
	Save(ctx context.Context, record *ConsentRecord) error
	Delete(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID, clientID string) error
}

// --- In-memory ConsentStore implementation ---

type memConsentStore struct {
	mu      sync.RWMutex
	records map[string]*ConsentRecord // key: tenantID:userID:clientID
}

func newMemConsentStore() *memConsentStore {
	return &memConsentStore{records: make(map[string]*ConsentRecord)}
}

func consentKey(tenantID uuid.UUID, userID uuid.UUID, clientID string) string {
	return fmt.Sprintf("%s:%s:%s", tenantID, userID, clientID)
}

func (s *memConsentStore) Get(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID, clientID string) (*ConsentRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.records[consentKey(tenantID, userID, clientID)]
	if !ok {
		return nil, nil
	}
	// Check expiry
	if !rec.ExpiresAt.IsZero() && time.Now().After(rec.ExpiresAt) {
		return nil, nil
	}
	return rec, nil
}

func (s *memConsentStore) Save(ctx context.Context, record *ConsentRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if record.ID == uuid.Nil {
		record.ID = uuid.New()
	}
	record.GrantedAt = time.Now()
	s.records[consentKey(record.TenantID, record.UserID, record.ClientID)] = record
	return nil
}

func (s *memConsentStore) Delete(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID, clientID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.records, consentKey(tenantID, userID, clientID))
	return nil
}

// --- Distributed Token Revocation (RFC 7009) ---

// RevocationStore is an interface for distributed token revocation.
// Implementations: in-memory (default), Redis (for multi-instance).
type RevocationStore interface {
	Revoke(ctx context.Context, tokenID string, expiresAt time.Time) error
	IsRevoked(ctx context.Context, tokenID string) bool
}

// --- In-memory RevocationStore (wraps sync.Map for interface compat) ---

type memRevocationStore struct {
	mu      sync.RWMutex
	revoked map[string]time.Time // tokenID -> expiry
}

func newMemRevocationStore() *memRevocationStore {
	return &memRevocationStore{revoked: make(map[string]time.Time)}
}

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
	// Auto-expire revocation entries after token expiry
	if time.Now().After(exp) {
		return false
	}
	return true
}

// CleanupExpired removes expired revocation entries to prevent unbounded growth.
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
