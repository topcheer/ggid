package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// TokenRevocationService is a standalone in-memory token revocation checker.
// Used by token_revocation_test.go. The OAuthService embeds revocation
// via Redis directly (RevokeToken/IsTokenRevoked methods), but this type
// provides a simpler API for testing and standalone use.
type TokenRevocationService struct {
	rdb interface{} // *redis.Client (nil for in-memory)
	mu  sync.RWMutex
	m   map[string]time.Time
}

// NewTokenRevocationService creates a standalone revocation service.
func NewTokenRevocationService(rdb interface{}) *TokenRevocationService {
	return &TokenRevocationService{rdb: rdb, m: make(map[string]time.Time)}
}

func (s *TokenRevocationService) RevokeToken(ctx context.Context, tokenID, reason string, expires time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[tokenID] = expires
	return nil
}

func (s *TokenRevocationService) IsRevoked(ctx context.Context, tokenID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	exp, ok := s.m[tokenID]
	if !ok {
		return false
	}
	// Expired entries are not considered revoked
	return time.Now().Before(exp)
}

type RevocationStatus struct {
	Revoked bool
	Reason  string
}

func (s *TokenRevocationService) GetRevocationStatus(ctx context.Context, tokenID string) (*RevocationStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.m[tokenID]; ok {
		return &RevocationStatus{Revoked: true, Reason: "revoked"}, nil
	}
	return &RevocationStatus{Revoked: false}, nil
}

func (s *TokenRevocationService) CleanupExpired() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	removed := 0
	for k, exp := range s.m {
		if now.After(exp) {
			delete(s.m, k)
			removed++
		}
	}
	return removed
}

func (s *TokenRevocationService) RevokeByClient(ctx context.Context, clientID string, expires time.Time) (int, error) {
	if clientID == "" {
		return 0, fmt.Errorf("clientID is required")
	}
	return 0, nil
}

func (s *TokenRevocationService) RevokeByUser(ctx context.Context, userID uuid.UUID, expires time.Time) (int, error) {
	if userID == uuid.Nil {
		return 0, fmt.Errorf("userID is required")
	}
	return 0, nil
}
