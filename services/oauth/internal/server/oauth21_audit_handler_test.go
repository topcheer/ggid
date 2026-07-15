package server

import (
	"bytes"
	"context"
	"crypto"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	ggidcrypto "github.com/ggid/ggid/pkg/crypto"
	"github.com/ggid/ggid/pkg/tenant"
	"github.com/ggid/ggid/services/oauth/internal/domain"
	"github.com/ggid/ggid/services/oauth/internal/service"
	"github.com/google/uuid"
)

type testClientRepo struct {
	clients []*domain.OAuthClient
}

func (m *testClientRepo) CreateClient(_ context.Context, _ *domain.OAuthClient) error { return nil }
func (m *testClientRepo) GetClientByID(_ context.Context, _ uuid.UUID, _ string) (*domain.OAuthClient, error) {
	return nil, nil
}
func (m *testClientRepo) ListClients(_ context.Context, _ uuid.UUID, _, _ int) ([]*domain.OAuthClient, int, error) {
	return m.clients, len(m.clients), nil
}
func (m *testClientRepo) UpdateClient(_ context.Context, _ uuid.UUID, _ string, _ *domain.OAuthClient) (*domain.OAuthClient, error) {
	return nil, nil
}
func (m *testClientRepo) DeleteClient(_ context.Context, _ uuid.UUID, _ string) error { return nil }

type testCodeRepo struct{}

func (m *testCodeRepo) CreateCode(_ context.Context, _ *domain.AuthorizationCode) error { return nil }
func (m *testCodeRepo) ConsumeCode(_ context.Context, _ string) (*domain.AuthorizationCode, error) { return nil, nil }

type testTokenRepo struct{}

func (m *testTokenRepo) RecordIDToken(_ context.Context, _ *domain.IDTokenRecord) error { return nil }
func (m *testTokenRepo) StoreRefreshToken(_ context.Context, _ *domain.RefreshTokenRecord) error { return nil }
func (m *testTokenRepo) GetRefreshToken(_ context.Context, _ uuid.UUID, _ string) (*domain.RefreshTokenRecord, error) {
	return nil, nil
}
func (m *testTokenRepo) RevokeRefreshToken(_ context.Context, _ uuid.UUID, _ string) error { return nil }
func (m *testTokenRepo) RevokeAllRefreshTokens(_ context.Context, _, _ uuid.UUID) error { return nil }

type testKeyProvider struct{}

func (testKeyProvider) Metadata() ggidcrypto.KeyMetadata { return ggidcrypto.KeyMetadata{} }
func (testKeyProvider) Public() crypto.PublicKey       { return nil }
func (testKeyProvider) Signer() crypto.Signer          { return nil }
func (testKeyProvider) Close() error                     { return nil }

func newTestOAuthService(clients []*domain.OAuthClient) *service.OAuthService {
	return service.NewOAuthService(
		&testClientRepo{clients: clients},
		&testCodeRepo{},
		&testTokenRepo{},
		testKeyProvider{},
		"https://test.example.com",
	)
}

func newTestRequest(method, path string, body []byte) *http.Request {
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	tc := &tenant.Context{TenantID: uuid.MustParse("00000000-0000-0000-0000-000000000001")}
	req = req.Clone(tenant.WithContext(req.Context(), tc))
	return req
}

func TestAuditClient_Compliant(t *testing.T) {
	c := &domain.OAuthClient{
		ClientID:                "c-compliant",
		Name:                    "Compliant App",
		Type:                    domain.ClientTypeConfidential,
		GrantTypes:              []string{"authorization_code", "refresh_token"},
		RedirectURIs:            []string{"https://app.example.com/callback"},
		TokenEndpointAuthMethod: "client_secret_basic",
		RequirePKCE:             true,
	}
	issues := auditClient(c)
	if len(issues) != 0 {
		t.Fatalf("expected compliant client, got issues: %v", issues)
	}
}

func TestAuditClient_NonCompliant(t *testing.T) {
	c := &domain.OAuthClient{
		ClientID:                "c-bad",
		Name:                    "Legacy App",
		Type:                    domain.ClientTypePublic,
		GrantTypes:              []string{"implicit", "password", "authorization_code"},
		RedirectURIs:            []string{"http://app.example.com/callback", "https://app.example.com/*"},
		TokenEndpointAuthMethod: "none",
		RequirePKCE:             false,
	}
	issues := auditClient(c)
	expected := map[string]bool{
		"implicit_grant_enabled":             true,
		"password_grant_enabled":             true,
		"non_https_redirect_uri":             true,
		"wildcard_redirect_uri":              true,
		"public_client_without_pkce":         true,
		"invalid_token_endpoint_auth_method": true,
	}
	if len(issues) != len(expected) {
		t.Fatalf("expected %d issues, got %d: %v", len(expected), len(issues), issues)
	}
	for _, issue := range issues {
		if !expected[issue] {
			t.Fatalf("unexpected issue: %s", issue)
		}
	}
}

func TestHandleOAuth21Audit_Compliant(t *testing.T) {
	clients := []*domain.OAuthClient{
		{
			ClientID:                "c-001",
			Name:                    "Good App",
			Type:                    domain.ClientTypeConfidential,
			GrantTypes:              []string{"authorization_code"},
			RedirectURIs:            []string{"https://app.example.com/callback"},
			TokenEndpointAuthMethod: "client_secret_basic",
			RequirePKCE:             true,
		},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/oauth/stats/oauth-2-1-audit", handleOAuth21Audit(newTestOAuthService(clients)))

	req := newTestRequest(http.MethodGet, "/api/v1/oauth/stats/oauth-2-1-audit", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var result OAuth21AuditResult
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result.TotalClientsAudited != 1 {
		t.Fatalf("expected 1 client audited, got %d", result.TotalClientsAudited)
	}
	if result.OverallCompliancePct != 100.0 {
		t.Fatalf("expected 100%% compliance, got %f", result.OverallCompliancePct)
	}
	if len(result.NonCompliantClients) != 0 {
		t.Fatalf("expected 0 non-compliant clients, got %d", len(result.NonCompliantClients))
	}
}

func TestHandleOAuth21Audit_NonCompliant(t *testing.T) {
	clients := []*domain.OAuthClient{
		{
			ClientID:                "c-002",
			Name:                    "Legacy App",
			Type:                    domain.ClientTypePublic,
			GrantTypes:              []string{"password"},
			RedirectURIs:            []string{"http://legacy.example.com/callback"},
			TokenEndpointAuthMethod: "none",
			RequirePKCE:             false,
		},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/oauth/stats/oauth-2-1-audit", handleOAuth21Audit(newTestOAuthService(clients)))

	req := newTestRequest(http.MethodGet, "/api/v1/oauth/stats/oauth-2-1-audit", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var result OAuth21AuditResult
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result.TotalClientsAudited != 1 {
		t.Fatalf("expected 1 client audited, got %d", result.TotalClientsAudited)
	}
	if result.OverallCompliancePct != 0.0 {
		t.Fatalf("expected 0%% compliance, got %f", result.OverallCompliancePct)
	}
	if len(result.NonCompliantClients) != 1 {
		t.Fatalf("expected 1 non-compliant client, got %d", len(result.NonCompliantClients))
	}

	nc := result.NonCompliantClients[0]
	if nc.ClientID != "c-002" {
		t.Fatalf("expected client c-002, got %s", nc.ClientID)
	}
	requiredIssues := map[string]bool{
		"password_grant_enabled":             true,
		"non_https_redirect_uri":             true,
		"public_client_without_pkce":         true,
		"invalid_token_endpoint_auth_method": true,
	}
	for issue := range requiredIssues {
		found := false
		for _, got := range nc.Issues {
			if got == issue {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected issue %s, got issues %v", issue, nc.Issues)
		}
	}
}

func TestHandleOAuth21Audit_MethodNotAllowed(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/oauth/stats/oauth-2-1-audit", handleOAuth21Audit(newTestOAuthService(nil)))

	req := newTestRequest(http.MethodPost, "/api/v1/oauth/stats/oauth-2-1-audit", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleOAuth21Audit_MixedCompliance(t *testing.T) {
	clients := []*domain.OAuthClient{
		{
			ClientID:                "c-good",
			Name:                    "Good App",
			Type:                    domain.ClientTypeConfidential,
			GrantTypes:              []string{"authorization_code"},
			RedirectURIs:            []string{"https://good.example.com/callback"},
			TokenEndpointAuthMethod: "client_secret_basic",
			RequirePKCE:             true,
		},
		{
			ClientID:                "c-bad",
			Name:                    "Bad App",
			Type:                    domain.ClientTypePublic,
			GrantTypes:              []string{"authorization_code", "implicit"},
			RedirectURIs:            []string{"https://bad.example.com/callback"},
			TokenEndpointAuthMethod: "client_secret_post",
			RequirePKCE:             false,
		},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/oauth/stats/oauth-2-1-audit", handleOAuth21Audit(newTestOAuthService(clients)))

	req := newTestRequest(http.MethodGet, "/api/v1/oauth/stats/oauth-2-1-audit", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	var result OAuth21AuditResult
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result.TotalClientsAudited != 2 {
		t.Fatalf("expected 2 clients audited, got %d", result.TotalClientsAudited)
	}
	if result.OverallCompliancePct != 50.0 {
		t.Fatalf("expected 50%% compliance, got %f", result.OverallCompliancePct)
	}
	if len(result.NonCompliantClients) != 1 {
		t.Fatalf("expected 1 non-compliant client, got %d", len(result.NonCompliantClients))
	}
	if result.NonCompliantClients[0].ClientID != "c-bad" {
		t.Fatalf("expected c-bad non-compliant, got %s", result.NonCompliantClients[0].ClientID)
	}
}
