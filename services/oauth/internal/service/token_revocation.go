package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// RevocationStatus describes the revocation state of a token.
type RevocationStatus struct {
	TokenID   string    `json:"token_id"`
	Revoked   bool      `json:"revoked"`
	Reason    string    `json:"reason,omitempty"`
	RevokedAt time.Time `json:"revoked_at,omitempty"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
	ClientID  string    `json:"client_id,omitempty"`
	UserID    string    `json:"user_id,omitempty"`
	TokenType string    `json:"token_type,omitempty"` // access | refresh | session
}

// TokenRevocationService manages token revocation with Redis persistence.
// Falls back to in-memory map when Redis is unavailable.
type TokenRevocationService struct {
	rdb       *redis.Client
	pool      *pgxpool.Pool
	mu        sync.RWMutex
	blacklist map[string]*revocationEntry // memory fallback
}

type revocationEntry struct {
	Reason    string    `json:"reason"`
	RevokedAt time.Time `json:"revoked_at"`
	ExpiresAt time.Time `json:"expires_at"`
	ClientID  string    `json:"client_id"`
	UserID    string    `json:"user_id"`
	TokenType string    `json:"token_type"`
}

// NewTokenRevocationService creates a new TokenRevocationService.
// If rdb is nil, falls back to in-memory storage.
func NewTokenRevocationService(rdb *redis.Client) *TokenRevocationService {
	return &TokenRevocationService{
		rdb:       rdb,
		blacklist: make(map[string]*revocationEntry),
	}
}

// SetPool enables DB-level cascade revocation.
func (s *TokenRevocationService) SetPool(pool *pgxpool.Pool) {
	s.pool = pool
}

const revocationKeyPrefix = "ggid:revoked_token:"

// RevokeToken revokes a single token by its ID with a reason.
// The blacklist entry TTL is set to the remaining token lifetime.
func (s *TokenRevocationService) RevokeToken(ctx context.Context, tokenID, reason string, expiresAt time.Time) error {
	if tokenID == "" {
		return fmt.Errorf("tokenID is required")
	}
	return s.revokeWithMeta(ctx, tokenID, reason, expiresAt, "", "", "")
}

// RevokeByClient revokes all tokens for a given client ID.
// Returns the number of tokens revoked.
func (s *TokenRevocationService) RevokeByClient(ctx context.Context, clientID string, expiresAt time.Time) (int, error) {
	if clientID == "" {
		return 0, fmt.Errorf("clientID is required")
	}
	count := 0
	if s.pool != nil {
		tag, err := s.pool.Exec(ctx, `
			UPDATE refresh_tokens SET revoked = true, revoked_at = NOW()
			WHERE client_id = $1 AND revoked = false`, clientID)
		if err != nil {
			return 0, fmt.Errorf("revoke by client: %w", err)
		}
		count = int(tag.RowsAffected())
	}
	return count, nil
}

// RevokeByUser revokes all tokens for a given user ID (cascade: access + refresh + session).
func (s *TokenRevocationService) RevokeByUser(ctx context.Context, userID uuid.UUID, expiresAt time.Time) (int, error) {
	if userID == uuid.Nil {
		return 0, fmt.Errorf("userID is required")
	}
	count := 0
	if s.pool != nil {
		tag, err := s.pool.Exec(ctx, `
			UPDATE refresh_tokens SET revoked = true, revoked_at = NOW()
			WHERE user_id = $1 AND revoked = false`, userID)
		if err != nil {
			return 0, fmt.Errorf("revoke by user: %w", err)
		}
		count = int(tag.RowsAffected())
	}
	return count, nil
}

// GetRevocationStatus returns the revocation status of a token.
func (s *TokenRevocationService) GetRevocationStatus(ctx context.Context, tokenID string) (*RevocationStatus, error) {
	// Try Redis first
	if s.rdb != nil {
		data, err := s.rdb.Get(ctx, revocationKeyPrefix+tokenID).Bytes()
		if err == nil {
			var entry revocationEntry
			if json.Unmarshal(data, &entry) == nil {
				return &RevocationStatus{
					TokenID: tokenID, Revoked: true, Reason: entry.Reason,
					RevokedAt: entry.RevokedAt, ExpiresAt: entry.ExpiresAt,
					ClientID: entry.ClientID, UserID: entry.UserID, TokenType: entry.TokenType,
				}, nil
			}
		}
	}

	// Fallback to memory
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.blacklist[tokenID]
	if !ok {
		return &RevocationStatus{TokenID: tokenID, Revoked: false}, nil
	}
	return &RevocationStatus{
		TokenID: tokenID, Revoked: true, Reason: entry.Reason,
		RevokedAt: entry.RevokedAt, ExpiresAt: entry.ExpiresAt,
		ClientID: entry.ClientID, UserID: entry.UserID, TokenType: entry.TokenType,
	}, nil
}

// IsRevoked checks if a token is currently revoked.
func (s *TokenRevocationService) IsRevoked(ctx context.Context, tokenID string) bool {
	// Try Redis first
	if s.rdb != nil {
		n, err := s.rdb.Exists(ctx, revocationKeyPrefix+tokenID).Result()
		if err == nil && n > 0 {
			return true
		}
	}

	// Fallback to memory
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.blacklist[tokenID]
	if !ok {
		return false
	}
	if !entry.ExpiresAt.IsZero() && time.Now().After(entry.ExpiresAt) {
		return false
	}
	return true
}

// CascadeRevoke revokes access, refresh, and session tokens for a user.
func (s *TokenRevocationService) CascadeRevoke(ctx context.Context, userID uuid.UUID, tokenIDs map[string]string, reason string, expiresAt time.Time) error {
	if userID == uuid.Nil {
		return fmt.Errorf("userID is required")
	}
	uid := userID.String()
	for tokenType, tokenID := range tokenIDs {
		if tokenID == "" {
			continue
		}
		if err := s.revokeWithMeta(ctx, tokenID, reason, expiresAt, "", uid, tokenType); err != nil {
			return err
		}
	}
	return nil
}

// revokeWithMeta stores a revocation entry with full metadata.
func (s *TokenRevocationService) revokeWithMeta(ctx context.Context, tokenID, reason string, expiresAt time.Time, clientID, userID, tokenType string) error {
	entry := &revocationEntry{
		Reason:    reason,
		RevokedAt: time.Now(),
		ExpiresAt: expiresAt,
		ClientID:  clientID,
		UserID:    userID,
		TokenType: tokenType,
	}

	// Try Redis
	if s.rdb != nil {
		data, err := json.Marshal(entry)
		if err == nil {
			ttl := time.Until(expiresAt)
			if ttl > 0 {
				err = s.rdb.Set(ctx, revocationKeyPrefix+tokenID, data, ttl).Err()
				if err != nil {
					slog.Warn("Redis revocation write failed, using memory fallback", "error", err)
				}
			}
		}
	}

	// Always store in memory as fallback
	s.mu.Lock()
	defer s.mu.Unlock()
	s.blacklist[tokenID] = entry
	return nil
}

// CleanupExpired removes expired entries from the in-memory blacklist.
// Redis entries auto-expire via TTL.
func (s *TokenRevocationService) CleanupExpired() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	count := 0
	for tokenID, entry := range s.blacklist {
		if !entry.ExpiresAt.IsZero() && now.After(entry.ExpiresAt) {
			delete(s.blacklist, tokenID)
			count++
		}
	}
	return count
}

// Reset clears all revocation entries (for testing).
func (s *TokenRevocationService) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.blacklist = make(map[string]*revocationEntry)
	if s.rdb != nil {
		// Best-effort cleanup for tests
		ctx := context.Background()
		keys, err := s.rdb.Keys(ctx, revocationKeyPrefix+"*").Result()
		if err == nil && len(keys) > 0 {
			s.rdb.Del(ctx, keys...)
		}
	}
}
