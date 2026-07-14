package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestExtractBearerToken(t *testing.T) {
	cases := []struct {
		name   string
		header string
		want   string
	}{
		{"valid bearer", "Bearer abc123", "abc123"},
		{"lowercase bearer", "bearer abc123", "abc123"},
		{"missing bearer", "Basic abc123", ""},
		{"empty header", "", ""},
		{"only bearer", "Bearer", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractBearerToken(tc.header)
			if got != tc.want {
				t.Fatalf("extractBearerToken(%q) = %q, want %q", tc.header, got, tc.want)
			}
		})
	}
}

func TestShouldOverrideIssuer(t *testing.T) {
	cases := []struct {
		issuer string
		want   bool
	}{
		{"http://localhost:9000", true},
		{"http://127.0.0.1:9000", true},
		{"http://192.168.1.1:9000", true},
		{"http://10.0.0.1:9000", true},
		{"http://172.16.0.1:9000", true},
		{"https://idp.example.com", false},
		{"", true},
	}
	for _, tc := range cases {
		t.Run(tc.issuer, func(t *testing.T) {
			got := shouldOverrideIssuer(tc.issuer)
			if got != tc.want {
				t.Fatalf("shouldOverrideIssuer(%q) = %v, want %v", tc.issuer, got, tc.want)
			}
		})
	}
}

func TestIsClientAuthenticated(t *testing.T) {
	cases := []struct {
		name string
		auth string
		body string
		want bool
	}{
		{"basic auth", "Basic Y2xpZW50OnNlY3JldA==", "", true},
		{"bearer token", "Bearer abc123", "", true},
		{"form credentials", "", "client_id=client&client_secret=secret", true},
		{"no auth", "", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/oauth/introspect", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			if tc.auth != "" {
				req.Header.Set("Authorization", tc.auth)
			}
			got := isClientAuthenticated(req)
			if got != tc.want {
				t.Fatalf("isClientAuthenticated: got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestInjectTenantContext(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/oauth/authorize?tenant_id=00000000-0000-0000-0000-000000000001", nil)
	ctx, err := injectTenantContext(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ctx == nil {
		t.Fatal("expected non-nil context")
	}

	req2 := httptest.NewRequest(http.MethodGet, "/oauth/authorize", nil)
	if _, err := injectTenantContext(req2); err == nil {
		t.Fatal("expected error for missing tenant")
	}
}
