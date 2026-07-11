package service

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestDeviceAuthorization_FullFlow exercises the complete RFC 8628 device flow:
// create device_code -> user approves -> poll -> receive JWT -> verify JWT.
func TestDeviceAuthorization_FullFlow(t *testing.T) {
	svc := NewOAuthService(newMockClientRepo(), newMockCodeRepo(), &mockTokenRepo{}, newMockKeyProvider(), "https://test.ggid.dev")

	userID := uuid.New()

	// Step 1: Create device authorization
	resp, err := svc.CreateDeviceAuthorization(&DeviceAuthorizationRequest{
		TenantID: testTenantID,
		ClientID: "test-client",
		Scope:    []string{"openid", "profile"},
		Issuer:   "https://test.ggid.dev",
	})
	if err != nil {
		t.Fatalf("CreateDeviceAuthorization: %v", err)
	}

	if resp.DeviceCode == "" {
		t.Fatal("device_code is empty")
	}
	if resp.UserCode == "" {
		t.Fatal("user_code is empty")
	}
	if resp.ExpiresIn != 900 {
		t.Fatalf("expected expires_in=900, got %d", resp.ExpiresIn)
	}
	if resp.Interval != 5 {
		t.Fatalf("expected interval=5, got %d", resp.Interval)
	}
	if resp.VerificationURI != "https://test.ggid.dev/device" {
		t.Fatalf("unexpected verification_uri: %s", resp.VerificationURI)
	}

	// Step 2: Poll before approval -> should get authorization_pending
	_, err = svc.PollDeviceToken(t.Context(), resp.DeviceCode, "test-client")
	if err == nil {
		t.Fatal("expected authorization_pending error")
	}
	if err.Error() != "authorization_pending" {
		t.Fatalf("expected authorization_pending, got: %v", err)
	}

	// Step 3: User approves via user_code
	if err := svc.ApproveDeviceCode(resp.UserCode, userID); err != nil {
		t.Fatalf("ApproveDeviceCode: %v", err)
	}

	// Step 4: Poll again -> should get access token
	tokenResp, err := svc.PollDeviceToken(t.Context(), resp.DeviceCode, "test-client")
	if err != nil {
		t.Fatalf("PollDeviceToken after approval: %v", err)
	}

	if tokenResp.AccessToken == "" {
		t.Fatal("access token is empty")
	}
	if tokenResp.TokenType != "Bearer" {
		t.Fatalf("expected token_type=Bearer, got %s", tokenResp.TokenType)
	}
	if tokenResp.ExpiresIn <= 0 {
		t.Fatalf("expected positive expires_in, got %d", tokenResp.ExpiresIn)
	}
	if tokenResp.Scope != "openid profile" {
		t.Fatalf("expected scope 'openid profile', got '%s'", tokenResp.Scope)
	}

	// Step 5: Verify the JWT is valid via ParseAccessToken
	claims, err := svc.ParseAccessToken(tokenResp.AccessToken)
	if err != nil {
		t.Fatalf("ParseAccessToken: %v", err)
	}

	if getStringClaim(claims, "sub") != userID.String() {
		t.Fatalf("expected sub=%s, got %s", userID.String(), getStringClaim(claims, "sub"))
	}
	if getStringClaim(claims, "iss") != "https://test.ggid.dev" {
		t.Fatalf("unexpected iss: %s", getStringClaim(claims, "iss"))
	}
	if getStringClaim(claims, "tenant_id") != testTenantID.String() {
		t.Fatalf("unexpected tenant_id: %s", getStringClaim(claims, "tenant_id"))
	}
}

// TestDeviceAuthorization_DeniedFlow tests that a denied device code returns access_denied.
func TestDeviceAuthorization_DeniedFlow(t *testing.T) {
	svc := NewOAuthService(newMockClientRepo(), newMockCodeRepo(), &mockTokenRepo{}, newMockKeyProvider(), "https://test.ggid.dev")

	resp, err := svc.CreateDeviceAuthorization(&DeviceAuthorizationRequest{
		TenantID: testTenantID,
		ClientID: "test-client",
	})
	if err != nil {
		t.Fatalf("CreateDeviceAuthorization: %v", err)
	}

	// Deny by setting status to denied via the code store directly.
	deviceCodeMu.Lock()
	if info, ok := deviceCodeStore[resp.DeviceCode]; ok {
		info.Status = "denied"
	}
	deviceCodeMu.Unlock()

	_, err = svc.PollDeviceToken(t.Context(), resp.DeviceCode, "test-client")
	if err == nil || err.Error() != "access_denied" {
		t.Fatalf("expected access_denied, got: %v", err)
	}
}

// TestDeviceAuthorization_InvalidCode tests polling with invalid device_code.
func TestDeviceAuthorization_InvalidCode(t *testing.T) {
	svc := NewOAuthService(newMockClientRepo(), newMockCodeRepo(), &mockTokenRepo{}, newMockKeyProvider(), "https://test.ggid.dev")

	_, err := svc.PollDeviceToken(t.Context(), "nonexistent-code", "test-client")
	if err == nil || err.Error() != "invalid_device_code" {
		t.Fatalf("expected invalid_device_code, got: %v", err)
	}
}

// TestDeviceAuthorization_ExpiredCode tests that expired codes return expired_token.
func TestDeviceAuthorization_ExpiredCode(t *testing.T) {
	svc := NewOAuthService(newMockClientRepo(), newMockCodeRepo(), &mockTokenRepo{}, newMockKeyProvider(), "https://test.ggid.dev")

	resp, err := svc.CreateDeviceAuthorization(&DeviceAuthorizationRequest{
		TenantID: testTenantID,
		ClientID: "test-client",
	})
	if err != nil {
		t.Fatalf("CreateDeviceAuthorization: %v", err)
	}

	// Expire the code by setting ExpiresAt in the past.
	deviceCodeMu.Lock()
	if info, ok := deviceCodeStore[resp.DeviceCode]; ok {
		info.ExpiresAt = time.Now().Add(-1 * time.Minute)
	}
	deviceCodeMu.Unlock()

	_, err = svc.PollDeviceToken(t.Context(), resp.DeviceCode, "test-client")
	if err == nil || err.Error() != "expired_token" {
		t.Fatalf("expected expired_token, got: %v", err)
	}
}

// TestDeviceAuthorization_SlowDown tests that polling too fast returns slow_down.
func TestDeviceAuthorization_SlowDown(t *testing.T) {
	svc := NewOAuthService(newMockClientRepo(), newMockCodeRepo(), &mockTokenRepo{}, newMockKeyProvider(), "https://test.ggid.dev")

	resp, err := svc.CreateDeviceAuthorization(&DeviceAuthorizationRequest{
		TenantID: testTenantID,
		ClientID: "test-client",
	})
	if err != nil {
		t.Fatalf("CreateDeviceAuthorization: %v", err)
	}

	// First poll sets LastPoll.
	_, _ = svc.PollDeviceToken(t.Context(), resp.DeviceCode, "test-client")

	// Immediate second poll should return slow_down.
	_, err = svc.PollDeviceToken(t.Context(), resp.DeviceCode, "test-client")
	if err == nil || err.Error() != "slow_down" {
		t.Fatalf("expected slow_down, got: %v", err)
	}
}

// TestDeviceAuthorization_ApproveInvalidUserCode tests error on bad user_code.
func TestDeviceAuthorization_ApproveInvalidUserCode(t *testing.T) {
	svc := NewOAuthService(newMockClientRepo(), newMockCodeRepo(), &mockTokenRepo{}, newMockKeyProvider(), "https://test.ggid.dev")

	err := svc.ApproveDeviceCode("BAD-CODE", uuid.New())
	if err == nil {
		t.Fatal("expected error for invalid user_code")
	}
}
