package service

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/ggid/ggid/services/oauth/internal/domain"
	"github.com/google/uuid"
)

// TestIDToken_CHashAndAuthTime verifies that the id_token issued during
// authorization code exchange includes c_hash (code hash) and auth_time
// per OIDC Core §3.1.3.6.
func TestIDToken_CHashAndAuthTime(t *testing.T) {
	svc, clientRepo, codeRepo, _ := newTestOAuthService()

	// Create client
	client := &domain.OAuthClient{
		ID:         uuid.New(),
		TenantID:   testTenantID,
		ClientID:   "gcid-chash-test",
		Name:       "CHash Test",
		Type:       domain.ClientTypePublic,
		GrantTypes: []string{"authorization_code", "refresh_token"},
		Enabled:    true,
	}
	_ = clientRepo.CreateClient(context.Background(), client)

	// Known auth time (5 minutes ago)
	authTime := time.Now().Add(-5 * time.Minute)

	// Create an authorization code with known values
	plaintextCode := "test-code-for-chash"
	code := &domain.AuthorizationCode{
		ID:                  uuid.New(),
		TenantID:            testTenantID,
		ClientID:            client.ID,
		UserID:              uuid.New(),
		RedirectURI:         "https://app.example.com/callback",
		Scope:               []string{"openid", "profile"},
		CodeChallenge:       "",
		CodeChallengeMethod: "",
		Nonce:               "test-nonce-12345",
		ExpiresAt:           time.Now().Add(5 * time.Minute),
		AMR:                 []string{"pwd"},
		ACR:                 "urn:mace:incommon:iap:silver",
		AuthTime:            authTime,
		CodeHash:            hashCode(plaintextCode),
	}
	_ = codeRepo.CreateCode(context.Background(), code)

	// Exchange the code
	resp, err := svc.ExchangeAuthorizationCode(context.Background(), &TokenExchangeRequest{
		TenantID:     testTenantID,
		GrantType:    "authorization_code",
		Code:         plaintextCode,
		RedirectURI:  "https://app.example.com/callback",
		ClientID:     "gcid-chash-test",
		CodeVerifier: "",
	})
	if err != nil {
		t.Fatalf("exchange failed: %v", err)
	}

	if resp.IDToken == "" {
		t.Fatal("id_token should be present for openid scope")
	}

	// Parse the id_token (without verification — just check claims)
	parts := strings.Split(resp.IDToken, ".")
	if len(parts) != 3 {
		t.Fatal("id_token should be a JWT with 3 parts")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		t.Fatalf("unmarshal claims: %v", err)
	}

	// Assert c_hash exists and is correct
	cHash, ok := claims["c_hash"]
	if !ok || cHash == "" {
		t.Error("FAIL: id_token missing c_hash (OIDC §3.1.3.6)")
	} else {
		// Verify c_hash = base64url(SHA256(code)[:16])
		expectedHash := sha256.Sum256([]byte(plaintextCode))
		expected := base64.RawURLEncoding.EncodeToString(expectedHash[:16])
		if cHash != expected {
			t.Errorf("c_hash mismatch: got %v, want %s", cHash, expected)
		}
	}

	// Assert auth_time exists and equals code.AuthTime
	authTimeClaim, ok := claims["auth_time"]
	if !ok {
		t.Error("FAIL: id_token missing auth_time")
	} else {
		authTimeFloat, _ := authTimeClaim.(float64)
		diff := int64(authTimeFloat) - authTime.Unix()
		if diff > 1 || diff < -1 {
			t.Errorf("auth_time mismatch: got %v, want ~%d (diff %d)", authTimeClaim, authTime.Unix(), diff)
		}
	}

	// Also assert at_hash exists
	if _, ok := claims["at_hash"]; !ok {
		t.Error("FAIL: id_token missing at_hash")
	}
}
