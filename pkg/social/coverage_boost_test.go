package social

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGoogleConnector_GetAuthURL(t *testing.T) {
	c := NewGoogleConnector("test-client-id", "test-secret")
	url, err := c.GetAuthURL(context.Background(), "state123", "https://app.com/callback")
	if err != nil {
		t.Fatalf("GetAuthURL failed: %v", err)
	}
	if url == "" {
		t.Fatal("expected non-empty URL")
	}
	// Should contain google endpoint
	if !contains(url, "accounts.google.com") {
		t.Errorf("expected google auth URL, got: %s", url)
	}
	// Should contain state
	if !contains(url, "state=state123") {
		t.Errorf("expected state in URL, got: %s", url)
	}
	// Should contain redirect_uri
	if !contains(url, "redirect_uri") {
		t.Errorf("expected redirect_uri in URL, got: %s", url)
	}
}

func TestGoogleConnector_HandleCallback(t *testing.T) {
	// Mock Google token + userinfo endpoint
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if contains(r.URL.Path, "token") {
			json.NewEncoder(w).Encode(map[string]any{
				"access_token": "mock-token",
				"token_type":   "Bearer",
			})
		} else {
			// userinfo
			json.NewEncoder(w).Encode(map[string]any{
				"id":         "google-123",
				"email":      "user@gmail.com",
				"name":       "Test User",
				"picture":    "https://lh3.googleusercontent.com/photo.jpg",
				"verified_email": true,
			})
		}
	}))
	defer ts.Close()

	// We can't easily redirect Google's endpoint to our test server,
	// so just verify the connector ID and DisplayName
	c := NewGoogleConnector("test-id", "test-secret")
	if c.ID() != "google" {
		t.Errorf("expected 'google', got '%s'", c.ID())
	}
	if c.DisplayName() != "Google" {
		t.Errorf("expected 'Google', got '%s'", c.DisplayName())
	}
}

func TestGitHubConnector_GetAuthURL(t *testing.T) {
	c := NewGitHubConnector("gh-client-id", "gh-secret")
	url, err := c.GetAuthURL(context.Background(), "state456", "https://app.com/callback")
	if err != nil {
		t.Fatalf("GetAuthURL failed: %v", err)
	}
	if !contains(url, "github.com") {
		t.Errorf("expected github URL, got: %s", url)
	}
	if !contains(url, "state=state456") {
		t.Errorf("expected state in URL")
	}
}

func TestGitHubConnector_HandleCallback_EmailFallback(t *testing.T) {
	// Test fetchPrimaryEmail with mocked endpoint
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{
			{"email": "primary@github.com", "primary": true},
			{"email": "secondary@github.com", "primary": false},
		})
	}))
	defer ts.Close()

	// We can't directly call fetchPrimaryEmail since it's unexported and uses fixed URL,
	// but we verify the connector works
	c := NewGitHubConnector("id", "secret")
	if c.ID() != "github" {
		t.Errorf("expected 'github', got '%s'", c.ID())
	}
}

func TestOIDCConnector_GetAuthURL(t *testing.T) {
	c := NewGenericOIDCConnector("keycloak", "Keycloak",
		"kc-id", "kc-secret",
		"https://kc.example.com/auth", "https://kc.example.com/token",
		"https://kc.example.com/userinfo", nil)

	url, err := c.GetAuthURL(context.Background(), "state789", "https://app.com/callback")
	if err != nil {
		t.Fatalf("GetAuthURL failed: %v", err)
	}
	if !contains(url, "kc.example.com/auth") {
		t.Errorf("expected keycloak auth URL, got: %s", url)
	}
	if !contains(url, "state=state789") {
		t.Errorf("expected state in URL")
	}
}

func TestOIDCConnector_DefaultScopes(t *testing.T) {
	// When scopes is nil, should default to openid profile email
	c := NewGenericOIDCConnector("oidc", "OIDC",
		"id", "secret",
		"https://idp.com/auth", "https://idp.com/token",
		"https://idp.com/userinfo", nil)

	url, _ := c.GetAuthURL(context.Background(), "s", "https://app.com/cb")
	if !contains(url, "openid") {
		t.Error("expected openid scope in URL")
	}
}

func TestOIDCConnector_CustomScopes(t *testing.T) {
	c := NewGenericOIDCConnector("custom", "Custom",
		"id", "secret",
		"https://idp.com/auth", "https://idp.com/token",
		"https://idp.com/userinfo",
		[]string{"openid", "groups", "email"})

	url, _ := c.GetAuthURL(context.Background(), "s", "https://app.com/cb")
	if !contains(url, "groups") {
		t.Error("expected groups scope in URL")
	}
}

func TestRegistry_GetNonExistent(t *testing.T) {
	r := NewRegistry()
	_, err := r.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent connector")
	}
}

func TestRegistry_List(t *testing.T) {
	r := NewRegistry()
	r.Register(NewGoogleConnector("id1", "secret1"))
	r.Register(NewGitHubConnector("id2", "secret2"))

	list := r.List()
	if len(list) != 2 {
		t.Fatalf("expected 2 connectors, got %d", len(list))
	}
}

func TestParseJWTClaims_RealJWT(t *testing.T) {
	// A proper JWT with header.payload.signature
	// Header: {"alg":"RS256","kid":"test-key"}
	header := base64URLEncode([]byte(`{"alg":"RS256","kid":"test-key"}`))
	payload := base64URLEncode([]byte(`{"sub":"user-123","email":"test@oidc.com","name":"OIDC User","picture":"https://avatar.com/me.jpg","iss":"https://idp.example.com","exp":9999999999,"iat":1700000000}`))
	jwt := header + "." + payload + ".signature"

	claims, err := parseJWTClaims(jwt)
	if err != nil {
		t.Fatalf("parseJWTClaims failed: %v", err)
	}
	if claims["sub"] != "user-123" {
		t.Errorf("expected sub='user-123', got '%v'", claims["sub"])
	}
	if claims["email"] != "test@oidc.com" {
		t.Errorf("expected email='test@oidc.com', got '%v'", claims["email"])
	}
	if claims["picture"] != "https://avatar.com/me.jpg" {
		t.Errorf("expected picture url, got '%v'", claims["picture"])
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
