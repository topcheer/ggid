package service

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/ggid/ggid/services/oauth/internal/domain"
	"github.com/google/uuid"
)

// TestRevokeToken_RefreshTokenUpdatesDB is a regression test for a P2 bug
// where RevokeToken only blacklisted refresh token hashes in Redis but
// never updated the DB record's revoked field. This meant the RefreshToken
// flow could still use the "revoked" token to mint new access tokens.
//
// Fix: RevokeToken now checks tokenTypeHint and non-JWT format to detect
// refresh tokens, then calls tokenRepo.RevokeRefreshToken to mark
// revoked=true in the DB.
func TestRevokeToken_RefreshTokenUpdatesDB(t *testing.T) {
	svc := NewOAuthService(newMockClientRepo(), newMockCodeRepo(), &mockTokenRepo{}, newMockKeyProvider(), "https://test.ggid.dev")

	// Generate a refresh token (random string, not a JWT)
	refreshToken := "test-refresh-token-for-revoke-regression"
	tokenHash := hashTokenSHA256Helper(refreshToken)

	// Store a refresh token record
	mockRepo := svc.tokenRepo.(*mockTokenRepo)
	uid := uuid.New()
	mockRepo.refreshTokens = append(mockRepo.refreshTokens, &domain.RefreshTokenRecord{
		ID:        uid,
		TokenHash: tokenHash,
		Revoked:   false,
	})

	// Verify it's not revoked yet
	record, _ := svc.tokenRepo.GetRefreshToken(nil, uuid.Nil, tokenHash)
	if record.Revoked {
		t.Fatal("refresh token should start as not revoked")
	}

	// Revoke with hint=refresh_token
	err := svc.RevokeToken(refreshToken, "refresh_token")
	if err != nil {
		t.Fatalf("RevokeToken failed: %v", err)
	}

	// Verify DB record is now revoked=true
	record, _ = svc.tokenRepo.GetRefreshToken(nil, uuid.Nil, tokenHash)
	if !record.Revoked {
		t.Error("FAIL: refresh token not marked as revoked in DB after RevokeToken with hint=refresh_token")
	}
}

// TestRevokeToken_NonJWTTokenFormatDetectsRefreshToken verifies that
// RevokeToken detects refresh tokens by format (no dots = not a JWT)
// even without an explicit token_type_hint.
func TestRevokeToken_NonJWTFormatTreatedAsRefresh(t *testing.T) {
	svc := NewOAuthService(newMockClientRepo(), newMockCodeRepo(), &mockTokenRepo{}, newMockKeyProvider(), "https://test.ggid.dev")

	refreshToken := "plain-opaque-token-no-dots"
	tokenHash := hashTokenSHA256Helper(refreshToken)

	mockRepo := svc.tokenRepo.(*mockTokenRepo)
	uid := uuid.New()
	mockRepo.refreshTokens = append(mockRepo.refreshTokens, &domain.RefreshTokenRecord{
		ID:        uid,
		TokenHash: tokenHash,
		Revoked:   false,
	})

	// Revoke WITHOUT hint — should still detect as refresh token (no dots)
	err := svc.RevokeToken(refreshToken)
	if err != nil {
		t.Fatalf("RevokeToken failed: %v", err)
	}

	record, _ := svc.tokenRepo.GetRefreshToken(nil, uuid.Nil, tokenHash)
	if !record.Revoked {
		t.Error("FAIL: non-JWT token not treated as refresh token for revocation")
	}
}

func hashTokenSHA256Helper(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
