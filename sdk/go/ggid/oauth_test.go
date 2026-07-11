package ggid

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetOIDCDiscovery(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/.well-known/openid-configuration" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(DiscoveryConfig{
			Issuer:                "https://iam.example.com",
			AuthorizationEndpoint: "https://iam.example.com/oauth/authorize",
			TokenEndpoint:         "https://iam.example.com/oauth/token",
			ScopesSupported:       []string{"openid", "profile", "email"},
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL)
	cfg, err := client.GetOIDCDiscovery(context.Background())
	if err != nil {
		t.Fatalf("GetOIDCDiscovery failed: %v", err)
	}
	if cfg.Issuer != "https://iam.example.com" {
		t.Errorf("expected issuer 'https://iam.example.com', got %q", cfg.Issuer)
	}
	if len(cfg.ScopesSupported) != 3 {
		t.Errorf("expected 3 scopes, got %d", len(cfg.ScopesSupported))
	}
}

func TestGetJWKS(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth/jwks" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(JWKS{
			Keys: []JWK{
				{Kty: "RSA", Kid: "test-key-1", Use: "sig", Alg: "RS256"},
			},
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL)
	jwks, err := client.GetJWKS(context.Background())
	if err != nil {
		t.Fatalf("GetJWKS failed: %v", err)
	}
	if len(jwks.Keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(jwks.Keys))
	}
	if jwks.Keys[0].Kid != "test-key-1" {
		t.Errorf("expected kid 'test-key-1', got %q", jwks.Keys[0].Kid)
	}
}

func TestRegisterOAuthClient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/oauth/register" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(OAuthClient{
			ClientID:     "generated-client-id",
			ClientName:   "Test App",
			RedirectURIs: []string{"https://app.example.com/callback"},
			GrantTypes:   []string{"authorization_code", "refresh_token"},
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL)
	result, err := client.RegisterOAuthClient(context.Background(), OAuthClient{
		ClientName:   "Test App",
		RedirectURIs: []string{"https://app.example.com/callback"},
	})
	if err != nil {
		t.Fatalf("RegisterOAuthClient failed: %v", err)
	}
	if result.ClientID != "generated-client-id" {
		t.Errorf("expected client_id 'generated-client-id', got %q", result.ClientID)
	}
}

func TestListOAuthClients(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/oauth/clients" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("expected auth header")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]OAuthClient{
			{ClientID: "client-1", ClientName: "App 1"},
			{ClientID: "client-2", ClientName: "App 2"},
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL)
	clients, err := client.ListOAuthClients(context.Background(), "test-token")
	if err != nil {
		t.Fatalf("ListOAuthClients failed: %v", err)
	}
	if len(clients) != 2 {
		t.Fatalf("expected 2 clients, got %d", len(clients))
	}
}

func TestDeleteOAuthClient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/oauth/clients/client-123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := NewClient(srv.URL)
	err := client.DeleteOAuthClient(context.Background(), "test-token", "client-123")
	if err != nil {
		t.Fatalf("DeleteOAuthClient failed: %v", err)
	}
}

func TestGetUserInfo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth/userinfo" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer access-token-123" {
			t.Errorf("expected auth header")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(UserInfo{
			Sub:           "user-123",
			Email:         "user@example.com",
			EmailVerified: true,
			Name:          "Test User",
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL)
	info, err := client.GetUserInfo(context.Background(), "access-token-123")
	if err != nil {
		t.Fatalf("GetUserInfo failed: %v", err)
	}
	if info.Sub != "user-123" {
		t.Errorf("expected sub 'user-123', got %q", info.Sub)
	}
	if !info.EmailVerified {
		t.Error("expected email_verified=true")
	}
}

func TestRevokeToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/oauth/revoke" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := NewClient(srv.URL)
	err := client.RevokeToken(context.Background(), "token-to-revoke")
	if err != nil {
		t.Fatalf("RevokeToken failed: %v", err)
	}
}

func TestGetIntrospectionConfig(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/oauth/introspection/config" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(IntrospectionConfig{
			CacheTTL:     300,
			CacheEnabled: true,
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL)
	cfg, err := client.GetIntrospectionConfig(context.Background(), "admin-token")
	if err != nil {
		t.Fatalf("GetIntrospectionConfig failed: %v", err)
	}
	if cfg.CacheTTL != 300 {
		t.Errorf("expected TTL 300, got %d", cfg.CacheTTL)
	}
	if !cfg.CacheEnabled {
		t.Error("expected cache_enabled=true")
	}
}

func TestUpdateIntrospectionConfig(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := NewClient(srv.URL)
	err := client.UpdateIntrospectionConfig(context.Background(), "admin-token", IntrospectionConfig{
		CacheTTL:     600,
		CacheEnabled: true,
	})
	if err != nil {
		t.Fatalf("UpdateIntrospectionConfig failed: %v", err)
	}
}

func TestGenerateAuthorizeURL(t *testing.T) {
	client := NewClient("https://iam.example.com")
	url := client.GenerateAuthorizeURL(AuthorizeURLOptions{
		ClientID:            "my-client",
		RedirectURI:         "https://app.example.com/callback",
		ResponseType:        "code",
		Scope:               "openid profile email",
		State:               "random-state",
		Nonce:               "random-nonce",
		CodeChallenge:       "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
		CodeChallengeMethod: "S256",
	})

	if url == "" {
		t.Fatal("expected non-empty URL")
	}
	// Verify required parameters are present
	checks := []string{
		"client_id=my-client",
		"response_type=code",
		"scope=openid+profile+email",
		"state=random-state",
		"code_challenge_method=S256",
	}
	for _, check := range checks {
		if !contains(url, check) {
			t.Errorf("URL missing expected parameter: %s\nURL: %s", check, url)
		}
	}
}

func TestDeviceAuthorization(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/oauth/device_authorization" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if r.Form.Get("client_id") != "device-client" {
			t.Errorf("expected client_id 'device-client', got %q", r.Form.Get("client_id"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(DeviceAuthResponse{
			DeviceCode:      "device-code-123",
			UserCode:        "ABC-DEF",
			VerificationURI: "https://iam.example.com/device",
			ExpiresIn:       1800,
			Interval:        5,
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL)
	resp, err := client.DeviceAuthorization(context.Background(), "device-client", "openid")
	if err != nil {
		t.Fatalf("DeviceAuthorization failed: %v", err)
	}
	if resp.DeviceCode != "device-code-123" {
		t.Errorf("expected device_code 'device-code-123', got %q", resp.DeviceCode)
	}
	if resp.UserCode != "ABC-DEF" {
		t.Errorf("expected user_code 'ABC-DEF', got %q", resp.UserCode)
	}
}

func TestGetSAMLMetadata(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/saml/metadata" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(`<?xml version="1.0"?><EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata"></EntityDescriptor>`))
	}))
	defer srv.Close()

	client := NewClient(srv.URL)
	data, err := client.GetSAMLMetadata(context.Background())
	if err != nil {
		t.Fatalf("GetSAMLMetadata failed: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty metadata")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
