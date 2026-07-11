package service

import (
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// makeLogoutToken builds a JWT with the claims required for OIDC back-channel logout (RFC 8417).
func makeLogoutToken(kp *mockKeyProvider, claims jwt.MapClaims) string {
	if _, ok := claims["events"]; !ok {
		claims["events"] = map[string]any{
			"http://schemas.openid.net/event/backchannel-logout": map[string]any{},
		}
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kp.KeyID()
	signed, _ := token.SignedString(kp.PrivateKey())
	return signed
}

// TestBackchannelLogout_ValidToken tests the full backchannel logout flow:
// submit a valid logout_token -> session is invalidated.
func TestBackchannelLogout_ValidToken(t *testing.T) {
	kp := newMockKeyProvider()
	svc := NewOAuthService(newMockClientRepo(), newMockCodeRepo(), &mockTokenRepo{}, kp, "https://test.ggid.dev")

	sub := uuid.New().String()
	logoutToken := makeLogoutToken(kp, jwt.MapClaims{
		"sub": sub,
		"iss": "https://test.ggid.dev",
		"aud": "ggid",
		"iat": 1700000000,
		"jti": uuid.New().String(),
	})

	// Before logout, the subject should not be in the logout list.
	key := "ggid:backchannel_logout:" + sub
	if _, seen := backchannelLogoutList.Load(key); seen {
		t.Fatal("subject should not be logged out yet")
	}

	// Submit the logout token.
	err := svc.BackchannelLogoutEndpoint(logoutToken)
	if err != nil {
		t.Fatalf("BackchannelLogoutEndpoint: %v", err)
	}

	// After logout, the subject should be in the logout list.
	if _, seen := backchannelLogoutList.Load(key); !seen {
		t.Fatal("subject should be marked as logged out")
	}
}

// TestBackchannelLogout_EmptyToken verifies that empty token is rejected.
func TestBackchannelLogout_EmptyToken(t *testing.T) {
	svc := NewOAuthService(newMockClientRepo(), newMockCodeRepo(), &mockTokenRepo{}, newMockKeyProvider(), "https://test.ggid.dev")

	err := svc.BackchannelLogoutEndpoint("")
	if err == nil {
		t.Fatal("expected error for empty logout_token")
	}
	if err.Error() != "logout_token is required" {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestBackchannelLogout_MissingSubAndSid verifies that a token without sub or sid is rejected.
func TestBackchannelLogout_MissingSubAndSid(t *testing.T) {
	kp := newMockKeyProvider()
	svc := NewOAuthService(newMockClientRepo(), newMockCodeRepo(), &mockTokenRepo{}, kp, "https://test.ggid.dev")

	logoutToken := makeLogoutToken(kp, jwt.MapClaims{
		"iss": "https://test.ggid.dev",
		"jti": uuid.New().String(),
	})

	err := svc.BackchannelLogoutEndpoint(logoutToken)
	if err == nil {
		t.Fatal("expected error for missing sub and sid")
	}
}

// TestBackchannelLogout_MissingEvents verifies that a token without events claim is rejected.
func TestBackchannelLogout_MissingEvents(t *testing.T) {
	kp := newMockKeyProvider()
	svc := NewOAuthService(newMockClientRepo(), newMockCodeRepo(), &mockTokenRepo{}, kp, "https://test.ggid.dev")

	// Build token WITHOUT the events claim.
	claims := jwt.MapClaims{
		"sub": uuid.New().String(),
		"iss": "https://test.ggid.dev",
		"jti": uuid.New().String(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kp.KeyID()
	signed, _ := token.SignedString(kp.PrivateKey())

	err := svc.BackchannelLogoutEndpoint(signed)
	if err == nil {
		t.Fatal("expected error for missing events claim")
	}
}

// TestBackchannelLogout_WithNonce verifies that a token with nonce is rejected per spec.
func TestBackchannelLogout_WithNonce(t *testing.T) {
	kp := newMockKeyProvider()
	svc := NewOAuthService(newMockClientRepo(), newMockCodeRepo(), &mockTokenRepo{}, kp, "https://test.ggid.dev")

	logoutToken := makeLogoutToken(kp, jwt.MapClaims{
		"sub":   uuid.New().String(),
		"iss":   "https://test.ggid.dev",
		"jti":   uuid.New().String(),
		"nonce": "should-not-be-here",
	})

	err := svc.BackchannelLogoutEndpoint(logoutToken)
	if err == nil {
		t.Fatal("expected error for token containing nonce")
	}
}

// TestBackchannelLogout_ReplayPrevention verifies that the same jti is rejected on second submission.
func TestBackchannelLogout_ReplayPrevention(t *testing.T) {
	kp := newMockKeyProvider()
	svc := NewOAuthService(newMockClientRepo(), newMockCodeRepo(), &mockTokenRepo{}, kp, "https://test.ggid.dev")

	jti := uuid.New().String()
	logoutToken := makeLogoutToken(kp, jwt.MapClaims{
		"sub": uuid.New().String(),
		"iss": "https://test.ggid.dev",
		"jti": jti,
	})

	// First submission should succeed.
	err := svc.BackchannelLogoutEndpoint(logoutToken)
	if err != nil {
		t.Fatalf("first submission should succeed: %v", err)
	}

	// Second submission with same jti should fail (replay detected).
	err = svc.BackchannelLogoutEndpoint(logoutToken)
	if err == nil {
		t.Fatal("expected replay detection error on second submission")
	}
}

// TestBackchannelLogout_WithSessionID tests logout with sid instead of sub.
func TestBackchannelLogout_WithSessionID(t *testing.T) {
	kp := newMockKeyProvider()
	svc := NewOAuthService(newMockClientRepo(), newMockCodeRepo(), &mockTokenRepo{}, kp, "https://test.ggid.dev")

	sid := "session-abc-123"
	logoutToken := makeLogoutToken(kp, jwt.MapClaims{
		"sid": sid,
		"iss": "https://test.ggid.dev",
		"jti": uuid.New().String(),
	})

	err := svc.BackchannelLogoutEndpoint(logoutToken)
	if err != nil {
		t.Fatalf("BackchannelLogoutEndpoint with sid: %v", err)
	}

	// Verify session-based key is set.
	key := "ggid:backchannel_logout:" + sid
	if _, seen := backchannelLogoutList.Load(key); !seen {
		t.Fatal("session should be marked as logged out")
	}
}
