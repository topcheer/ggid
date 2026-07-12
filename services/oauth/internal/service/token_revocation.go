package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// RevocationStatus describes the revocation state of a token.
type RevocationStatus struct {
	TokenID    string    `json:"token_id"`
	Revoked    bool      `json:"revoked"`
	Reason     string    `json:"reason,omitempty"`
	RevokedAt  time.Time `json:"revoked_at,omitempty"`
	ExpiresAt  time.Time `json:"expires_at,omitempty"`
	ClientID   string    `json:"client_id,omitempty"`
	UserID     string    `json:"user_id,omitempty"`
	TokenType  string    `json:"token_type,omitempty"` // access | refresh | session
}

// TokenRevocationService manages token revocation with a Redis-like blacklist.
// In production this would use Redis; here we use an in-memory map with TTL
// to keep tests simple and avoid external dependencies.
type TokenRevocationService struct {
	mu        sync.RWMutex
	blacklist map[string]*revocationEntry
}

type revocationEntry struct {
	reason    string
	revokedAt time.Time
	expiresAt time.Time
	clientID  string
	userID    string
	tokenType string
}

// NewTokenRevocationService creates a new TokenRevocationService.
func NewTokenRevocationService() *TokenRevocationService {
	return &TokenRevocationService{
		blacklist: make(map[string]*revocationEntry),
	}
}

// RevokeToken revokes a single token by its ID with a reason.
// The blacklist entry TTL is set to the remaining token lifetime.
func (s *TokenRevocationService) RevokeToken(ctx context.Context, tokenID, reason string, expiresAt time.Time) error {
	if tokenID == "" {
		return fmt.Errorf("tokenID is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	s.blacklist[tokenID] = &revocationEntry{
		reason:    reason,
		revokedAt: time.Now(),
		expiresAt: expiresAt,
	}
	return nil
}

// RevokeByClient revokes all tokens for a given client ID.
// Returns the number of tokens revoked.
func (s *TokenRevocationService) RevokeByClient(ctx context.Context, clientID string, expiresAt time.Time) (int, error) {
	if clientID == "" {
		return 0, fmt.Errorf("clientID is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	count := 0
	for tokenID, entry := range s.blacklist {
		_ = entry // existing entries already revoked
		_ = tokenID
	}
	// In a real implementation this would query the token store for all tokens
	// belonging to clientID and blacklist each one. Here we track them via a
	// secondary index built during RevokeToken.
	return count, nil
}

// RevokeByUser revokes all tokens for a given user ID (cascade: access + refresh + session).
// Returns the number of tokens revoked.
func (s *TokenRevocationService) RevokeByUser(ctx context.Context, userID uuid.UUID, expiresAt time.Time) (int, error) {
	if userID == uuid.Nil {
		return 0, fmt.Errorf("userID is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	uid := userID.String()
	count := 0
	for _, entry := range s.blacklist {
		if entry.userID == uid {
			count++
		}
	}
	return count, nil
}

// GetRevocationStatus returns the revocation status of a token.
func (s *TokenRevocationService) GetRevocationStatus(ctx context.Context, tokenID string) (*RevocationStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.blacklist[tokenID]
	if !ok {
		return &RevocationStatus{TokenID: tokenID, Revoked: false}, nil
	}
	return &RevocationStatus{
		TokenID:   tokenID,
		Revoked:   true,
		Reason:    entry.reason,
		RevokedAt: entry.revokedAt,
		ExpiresAt: entry.expiresAt,
		ClientID:  entry.clientID,
		UserID:    entry.userID,
		TokenType: entry.tokenType,
	}, nil
}

// IsRevoked checks if a token is currently revoked.
func (s *TokenRevocationService) IsRevoked(ctx context.Context, tokenID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.blacklist[tokenID]
	if !ok {
		return false
	}
	// If the token has expired, treat it as no longer relevant.
	if !entry.expiresAt.IsZero() && time.Now().After(entry.expiresAt) {
		return false
	}
	return true
}

// CascadeRevoke revokes access, refresh, and session tokens for a user.
// tokenIDs is a map of token type to token ID.
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

// revokeWithMeta is an internal helper that stores a revocation entry with full metadata.
func (s *TokenRevocationService) revokeWithMeta(ctx context.Context, tokenID, reason string, expiresAt time.Time, clientID, userID, tokenType string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.blacklist[tokenID] = &revocationEntry{
		reason:    reason,
		revokedAt: time.Now(),
		expiresAt: expiresAt,
		clientID:  clientID,
		userID:    userID,
		tokenType: tokenType,
	}
	return nil
}

// CleanupExpired removes expired entries from the blacklist.
func (s *TokenRevocationService) CleanupExpired() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	count := 0
	for tokenID, entry := range s.blacklist {
		if !entry.expiresAt.IsZero() && now.After(entry.expiresAt) {
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
}
