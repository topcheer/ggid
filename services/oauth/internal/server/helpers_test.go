package server

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	pkgcrypto "github.com/ggid/ggid/pkg/crypto"
	"github.com/ggid/ggid/services/oauth/internal/domain"
	"github.com/google/uuid"
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

func TestIntrospectRequestAuthenticated(t *testing.T) {
	tenantID := uuid.New()
	secretHash, err := pkgcrypto.HashPassword("s3cret")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	svc := newTestOAuthService([]*domain.OAuthClient{{
		ClientID:         "rs-client",
		ClientSecretHash: secretHash,
		TenantID:         tenantID,
		Type:             domain.ClientTypeConfidential,
		Enabled:          true,
	}})

	newReq := func(auth, body string, withTenant bool) *http.Request {
		req := httptest.NewRequest(http.MethodPost, "/oauth/introspect", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		if auth != "" {
			req.Header.Set("Authorization", auth)
		}
		if withTenant {
			req.Header.Set("X-Tenant-ID", tenantID.String())
		}
		return req
	}

	cases := []struct {
		name       string
		auth       string
		body       string
		withTenant bool
		want       bool
	}{
		{"valid basic creds", "Basic " + base64.StdEncoding.EncodeToString([]byte("rs-client:s3cret")), "", true, true},
		{"wrong secret", "Basic " + base64.StdEncoding.EncodeToString([]byte("rs-client:wrong")), "", true, false},
		{"unknown client", "Basic " + base64.StdEncoding.EncodeToString([]byte("ghost:s3cret")), "", true, false},
		{"basic creds without tenant header", "Basic " + base64.StdEncoding.EncodeToString([]byte("rs-client:s3cret")), "", false, false},
		{"valid form creds", "", "client_id=rs-client&client_secret=s3cret", true, true},
		{"form creds wrong secret", "", "client_id=rs-client&client_secret=wrong", true, false},
		{"garbage bearer token", "Bearer not-a-real-token", "", false, false},
		{"no auth", "", "", false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := introspectRequestAuthenticated(svc, newReq(tc.auth, tc.body, tc.withTenant))
			if got != tc.want {
				t.Fatalf("introspectRequestAuthenticated: got %v, want %v", got, tc.want)
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
