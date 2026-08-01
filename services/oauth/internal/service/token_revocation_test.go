package service

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func newTestOAuthServiceForRevocation() *OAuthService {
	return &OAuthService{}
}

func TestTokenRevocation_EmptyToken(t *testing.T) {
	svc := newTestOAuthServiceForRevocation()
	// RFC 7009: empty token should return nil (always 200)
	err := svc.RevokeToken("")
	if err != nil {
		t.Errorf("RevokeToken('') should be nil, got %v", err)
	}
}

func TestTokenRevocation_NotRevoked(t *testing.T) {
	svc := newTestOAuthServiceForRevocation()
	// A random token that was never revoked — IsTokenRevoked should return false.
	tokenStr := "test-token-" + uuid.New().String()
	if svc.IsTokenRevoked(tokenStr) {
		t.Error("unknown token should not be revoked")
	}
}

func TestTokenRevocation_ExpiredRevocation(t *testing.T) {
	svc := newTestOAuthServiceForRevocation()
	tokenStr := "expired-revocation-" + uuid.New().String()
	tokenHash := hashTokenSHA256(tokenStr)
	revokedTokens.Store(tokenHash, int64(time.Now().Add(-1*time.Hour).Unix()))

	// IsTokenRevoked checks the in-memory map without expiry validation.
	// Expired entries are cleaned up by the reaper goroutine, not IsTokenRevoked.
	// So an expired entry will still be reported as revoked until reaped.
	if !svc.IsTokenRevoked(tokenStr) {
		t.Error("in-memory revokedTokens map does not check expiry — expired entry still counts as revoked until reaped")
	}

	// Clean up
	revokedTokens.Delete(tokenHash)
}

func TestTokenRevocation_ActiveRevocation(t *testing.T) {
	svc := newTestOAuthServiceForRevocation()
	// Directly store a valid revocation entry in the blacklist.
	tokenStr := "active-revocation-" + uuid.New().String()
	tokenHash := hashTokenSHA256(tokenStr)
	revokedTokens.Store(tokenHash, int64(time.Now().Add(1*time.Hour).Unix()))

	// IsTokenRevoked should return true.
	if !svc.IsTokenRevoked(tokenStr) {
		t.Error("actively revoked token should be reported as revoked")
	}
}
