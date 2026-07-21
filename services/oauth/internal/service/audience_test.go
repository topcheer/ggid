package service

import (
	"context"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/ggid/ggid/services/oauth/internal/domain"
	"github.com/google/uuid"
)

// TestResolveAudience verifies the fallback behavior of the audience helper.
func TestResolveAudience(t *testing.T) {
	if got := resolveAudience("https://api.example.com", "client-1"); got != "https://api.example.com" {
		t.Errorf("resolveAudience with request = %q", got)
	}
	if got := resolveAudience("", "client-1"); got != "client-1" {
		t.Errorf("resolveAudience fallback = %q, want client-1", got)
	}
}

// TestClientCredentials_Audience verifies the Auth0-style audience parameter
// lands in the access token's aud claim (RFC 8707-style targeting), while the
// default (no audience) keeps the legacy client_id behavior.
func TestClientCredentials_Audience(t *testing.T) {
	svc, _, _, _ := newTestOAuthService()
	tenantID := uuid.New()

	result, _ := svc.CreateClient(context.Background(), &CreateClientInput{
		TenantID:   tenantID,
		Name:       "m2m-aud",
		Type:       domain.ClientTypeConfidential,
		GrantTypes: []string{"client_credentials"},
	})

	// 1. Explicit audience → aud = audience (Auth0 migration compat).
	resp, err := svc.ClientCredentials(context.Background(), &ClientCredentialsRequest{
		TenantID:     tenantID,
		ClientID:     result.Client.ClientID,
		ClientSecret: result.ClientSecret,
		Audience:     "https://api.example.com",
	})
	if err != nil {
		t.Fatalf("ClientCredentials: %v", err)
	}
	if got := parseAudClaim(t, resp.AccessToken); got != "https://api.example.com" {
		t.Errorf("aud = %q, want https://api.example.com", got)
	}

	// 2. No audience → aud defaults to client_id (unchanged legacy behavior).
	resp2, err := svc.ClientCredentials(context.Background(), &ClientCredentialsRequest{
		TenantID:     tenantID,
		ClientID:     result.Client.ClientID,
		ClientSecret: result.ClientSecret,
	})
	if err != nil {
		t.Fatalf("ClientCredentials: %v", err)
	}
	if got := parseAudClaim(t, resp2.AccessToken); got != result.Client.ClientID {
		t.Errorf("default aud = %q, want %q", got, result.Client.ClientID)
	}
}

func parseAudClaim(t *testing.T, token string) string {
	t.Helper()
	claims := jwt.MapClaims{}
	_, _, err := jwt.NewParser().ParseUnverified(token, claims)
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	aud, _ := claims["aud"].(string)
	return aud
}
