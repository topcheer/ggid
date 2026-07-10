package service

import (
	"context"
	"testing"

	"github.com/ggid/ggid/pkg/tenant"
	"github.com/ggid/ggid/services/oauth/internal/domain"
	"github.com/google/uuid"
)

func TestListClients(t *testing.T) {
	svc, clientRepo, _, _ := newTestOAuthService()

	tenantID := uuid.New()
	ctx := tenant.WithContext(context.Background(), &tenant.Context{
		TenantID:       tenantID,
		IsolationLevel: tenant.IsolationShared,
	})

	// Add a client.
	clientRepo.CreateClient(ctx, &domain.OAuthClient{
		ID:        uuid.New(),
		TenantID:  tenantID,
		ClientID:  "test-client-1",
		Name:      "Test Client",
		Type:      domain.ClientTypeConfidential,
		Enabled:   true,
	})

	clients, total, err := svc.ListClients(ctx, 10, 0)
	if err != nil {
		t.Fatalf("ListClients: %v", err)
	}
	if total < 1 {
		t.Errorf("expected total >= 1, got %d", total)
	}
	if len(clients) < 1 {
		t.Errorf("expected >= 1 client, got %d", len(clients))
	}
}

func TestDeleteClient(t *testing.T) {
	svc, clientRepo, _, _ := newTestOAuthService()

	tenantID := uuid.New()
	ctx := tenant.WithContext(context.Background(), &tenant.Context{
		TenantID:       tenantID,
		IsolationLevel: tenant.IsolationShared,
	})

	clientRepo.CreateClient(ctx, &domain.OAuthClient{
		ID:        uuid.New(),
		TenantID:  tenantID,
		ClientID:  "test-client-del",
		Name:      "Delete Me",
		Type:      domain.ClientTypeConfidential,
		Enabled:   true,
	})

	err := svc.DeleteClient(ctx, "test-client-del")
	if err != nil {
		t.Fatalf("DeleteClient: %v", err)
	}

	// Verify deletion.
	_, err = svc.GetClient(ctx, "test-client-del")
	if err == nil {
		t.Error("expected error after deletion")
	}
}

func TestRevokeToken_Invalid(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	// RFC 7009: invalid token returns nil (200 OK).
	err := svc.RevokeToken("invalid-token-to-revoke")
	if err != nil {
		t.Errorf("expected nil for invalid token (RFC 7009), got: %v", err)
	}
}

func TestRevokeToken_CoverageBoost_Empty(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	// Empty token returns nil (RFC 7009).
	err := svc.RevokeToken("")
	if err != nil {
		t.Errorf("expected nil for empty token, got: %v", err)
	}
}

func TestParseAccessToken_Invalid(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	_, err := svc.ParseAccessToken("not-a-jwt")
	if err == nil {
		t.Error("expected error for non-JWT token")
	}
}

func TestParseAccessToken_Empty(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	_, err := svc.ParseAccessToken("")
	if err == nil {
		t.Error("expected error for empty token")
	}
}

func TestGetUserInfo_NoToken(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	// With no valid token, should return error.
	_, err := svc.GetUserInfo("invalid-token")
	if err == nil {
		t.Error("expected error for invalid token in GetUserInfo")
	}
}

func TestRefreshToken_Invalid(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()

	_, err := svc.RefreshToken(context.Background(), &RefreshTokenRequest{
		RefreshToken: "invalid-refresh-token",
		ClientID:     "test-client",
	})
	if err == nil {
		t.Error("expected error for invalid refresh token")
	}
}

func TestGetStringClaim(t *testing.T) {
	claims := map[string]any{"sub": "user123", "name": 123}
	if v := getStringClaim(claims, "sub"); v != "user123" {
		t.Errorf("expected 'user123', got '%s'", v)
	}
	if v := getStringClaim(claims, "missing"); v != "" {
		t.Errorf("expected empty, got '%s'", v)
	}
	// Non-string value should return empty.
	if v := getStringClaim(claims, "name"); v != "" {
		t.Errorf("expected empty for non-string, got '%s'", v)
	}
}

func TestGetInt64Claim(t *testing.T) {
	claims := map[string]any{"exp": float64(1234567890), "n": int64(42)}
	if v := getInt64Claim(claims, "exp"); v != 1234567890 {
		t.Errorf("expected 1234567890, got %d", v)
	}
	if v := getInt64Claim(claims, "n"); v != 42 {
		t.Errorf("expected 42, got %d", v)
	}
	if v := getInt64Claim(claims, "missing"); v != 0 {
		t.Errorf("expected 0, got %d", v)
	}
}
