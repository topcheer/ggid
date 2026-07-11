package ggid

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSDKAuthExt_IntrospectToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/oauth/introspect" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(IntrospectionResult{
			Active: true, ClientID: "c1", Scope: "read write",
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	result, err := c.IntrospectToken(context.Background(), "tok")
	if err != nil {
		t.Fatalf("IntrospectToken failed: %v", err)
	}
	if !result.Active {
		t.Error("expected active=true")
	}
	if result.ClientID != "c1" {
		t.Errorf("expected c1, got %s", result.ClientID)
	}
}

func TestSDKAuthExt_Logout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/logout" || r.Method != http.MethodPost {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	if err := c.Logout(context.Background(), "tok"); err != nil {
		t.Fatalf("Logout failed: %v", err)
	}
}

func TestSDKAuthExt_Impersonate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/impersonate" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(TokenSet{
			AccessToken: "impersonated-token", TokenType: "Bearer",
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	tokens, err := c.Impersonate(context.Background(), "admin-tok", "u1", "support ticket")
	if err != nil {
		t.Fatalf("Impersonate failed: %v", err)
	}
	if tokens.AccessToken != "impersonated-token" {
		t.Errorf("expected impersonated-token, got %s", tokens.AccessToken)
	}
}

func TestSDKAuthExt_RevokeImpersonation(t *testing.T) {
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	if err := c.RevokeImpersonation(context.Background(), "admin-tok", "s1"); err != nil {
		t.Fatalf("RevokeImpersonation failed: %v", err)
	}
	if path != "/api/v1/auth/impersonate/s1/revoke" {
		t.Errorf("unexpected path: %s", path)
	}
}

func TestSDKAuthExt_RevokeAllUserSessions(t *testing.T) {
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	if err := c.RevokeAllUserSessions(context.Background(), "admin-tok", "u1"); err != nil {
		t.Fatalf("RevokeAllUserSessions failed: %v", err)
	}
	if path != "/api/v1/users/u1/sessions/revoke" {
		t.Errorf("unexpected path: %s", path)
	}
}

func TestSDKAuthExt_CheckSoD(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/policies/sod/check" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode([]SoDViolation{
			{UserID: "u1", Roles: []string{"admin", "auditor"}, Reason: "conflict"},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	violations, err := c.CheckSoD(context.Background(), "tok", "u1", []string{"admin", "auditor"})
	if err != nil {
		t.Fatalf("CheckSoD failed: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
	if violations[0].Reason != "conflict" {
		t.Errorf("expected 'conflict', got '%s'", violations[0].Reason)
	}
}
