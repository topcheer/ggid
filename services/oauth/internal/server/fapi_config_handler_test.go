package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/ggid/ggid/pkg/tenant"
	"github.com/ggid/ggid/services/oauth/internal/domain"
	"github.com/ggid/ggid/services/oauth/internal/service"
	"github.com/google/uuid"
)

func TestFAPI2_0ClientMetadata(t *testing.T) {
	c := &domain.OAuthClient{
		ClientID: "c-fapi",
		Name:     "FAPI Client",
		Metadata: map[string]any{},
	}
	if c.FAPI2_0() {
		t.Fatal("expected FAPI 2.0 to default to false")
	}
	c.SetFAPI2_0(true)
	if !c.FAPI2_0() {
		t.Fatal("expected FAPI 2.0 to be true after SetFAPI2_0")
	}
	if c.Metadata["fapi_2_0"] != true {
		t.Fatalf("expected metadata fapi_2_0=true, got %v", c.Metadata["fapi_2_0"])
	}
}

func TestHandleFAPIConfigGet(t *testing.T) {
	clients := []*domain.OAuthClient{
		{
			ClientID: "c-regular",
			Name:     "Regular Client",
			Metadata: map[string]any{},
		},
		{
			ClientID: "c-fapi",
			Name:     "FAPI Client",
			Metadata: map[string]any{"fapi_2_0": true},
		},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/oauth/fapi-config", handleFAPIConfig(newTestOAuthService(clients)))

	req := newTestRequest(http.MethodGet, "/api/v1/oauth/fapi-config", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var resp FAPIConfigResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if !resp.Enabled {
		t.Fatal("expected FAPI config enabled=true")
	}
	if len(resp.EnabledClients) != 1 {
		t.Fatalf("expected 1 enabled client, got %d", len(resp.EnabledClients))
	}
	if resp.EnabledClients[0].ClientID != "c-fapi" {
		t.Fatalf("expected c-fapi, got %s", resp.EnabledClients[0].ClientID)
	}
}

func TestHandleFAPIConfigPut(t *testing.T) {
	clients := []*domain.OAuthClient{
		{
			ClientID: "c-001",
			Name:     "Client One",
			Metadata: map[string]any{},
		},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/oauth/fapi-config", handleFAPIConfig(newTestOAuthService(clients)))

	body := `{"client_id":"c-001","enabled":true}`
	req := newTestRequest(http.MethodPut, "/api/v1/oauth/fapi-config", []byte(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var result map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result["client_id"] != "c-001" {
		t.Fatalf("expected client_id c-001, got %v", result["client_id"])
	}
	if result["fapi_2_0"] != true {
		t.Fatalf("expected fapi_2_0 true, got %v", result["fapi_2_0"])
	}
}

func TestHandleFAPIConfigPut_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/oauth/fapi-config", handleFAPIConfig(newTestOAuthService(nil)))

	body := `{"client_id":"c-missing","enabled":true}`
	req := newTestRequest(http.MethodPut, "/api/v1/oauth/fapi-config", []byte(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestEnforceFAPIAuthorize_NonFAPIAllowed(t *testing.T) {
	c := &domain.OAuthClient{ClientID: "c-regular", Metadata: map[string]any{}}
	req := httptest.NewRequest(http.MethodGet, "/oauth/authorize?client_id=c-regular&response_type=code", nil)
	if err := enforceFAPIAuthorize(c, req); err != nil {
		t.Fatalf("non-FAPI client should not be enforced: %v", err)
	}
}

func TestEnforceFAPIAuthorize_FAPIRequiresS256(t *testing.T) {
	c := &domain.OAuthClient{ClientID: "c-fapi", Metadata: map[string]any{"fapi_2_0": true}}
	req := httptest.NewRequest(http.MethodGet, "/oauth/authorize?client_id=c-fapi&response_type=code&request_uri=urn:ietf:params:oauth:request_uri:abc", nil)
	req.Header.Set("DPoP", "proof")
	if err := enforceFAPIAuthorize(c, req); err == nil {
		t.Fatal("expected FAPI authorize to require PKCE S256")
	}
}

func TestEnforceFAPIAuthorize_FAPIRequiresPAR(t *testing.T) {
	c := &domain.OAuthClient{ClientID: "c-fapi", Metadata: map[string]any{"fapi_2_0": true}}
	req := httptest.NewRequest(http.MethodGet, "/oauth/authorize?client_id=c-fapi&response_type=code&code_challenge=abc&code_challenge_method=S256", nil)
	req.Header.Set("DPoP", "proof")
	if err := enforceFAPIAuthorize(c, req); err == nil {
		t.Fatal("expected FAPI authorize to require PAR")
	}
}

func TestEnforceFAPIAuthorize_FAPIRequiresDPoP(t *testing.T) {
	c := &domain.OAuthClient{ClientID: "c-fapi", Metadata: map[string]any{"fapi_2_0": true}}
	req := httptest.NewRequest(http.MethodGet, "/oauth/authorize?client_id=c-fapi&response_type=code&code_challenge=abc&code_challenge_method=S256&request_uri=urn:abc", nil)
	if err := enforceFAPIAuthorize(c, req); err == nil {
		t.Fatal("expected FAPI authorize to require DPoP")
	}
}

func TestEnforceFAPIAuthorize_FAPICompliant(t *testing.T) {
	c := &domain.OAuthClient{ClientID: "c-fapi", Metadata: map[string]any{"fapi_2_0": true}}
	req := httptest.NewRequest(http.MethodGet, "/oauth/authorize?client_id=c-fapi&response_type=code&code_challenge=abc&code_challenge_method=S256&request_uri=urn:abc", nil)
	req.Header.Set("DPoP", "proof")
	if err := enforceFAPIAuthorize(c, req); err != nil {
		t.Fatalf("expected compliant FAPI request to pass: %v", err)
	}
}

func TestEnforceFAPIToken_NonFAPIAllowed(t *testing.T) {
	c := &domain.OAuthClient{ClientID: "c-regular", Metadata: map[string]any{}}
	form := url.Values{"grant_type": {"refresh_token"}}
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err := enforceFAPIToken(c, req); err != nil {
		t.Fatalf("non-FAPI client should not be enforced: %v", err)
	}
}

func TestEnforceFAPIToken_FAPIRequiresAuthorizationCode(t *testing.T) {
	c := &domain.OAuthClient{ClientID: "c-fapi", Metadata: map[string]any{"fapi_2_0": true}}
	form := url.Values{"grant_type": {"client_credentials"}}
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err := enforceFAPIToken(c, req); err == nil {
		t.Fatal("expected FAPI token to require authorization_code grant")
	}
}

func TestEnforceFAPIToken_FAPIRequiresDPoP(t *testing.T) {
	c := &domain.OAuthClient{ClientID: "c-fapi", Metadata: map[string]any{"fapi_2_0": true}}
	form := url.Values{"grant_type": {"authorization_code"}, "code_verifier": {"verifier"}}
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err := enforceFAPIToken(c, req); err == nil {
		t.Fatal("expected FAPI token to require DPoP")
	}
}

func TestEnforceFAPIToken_FAPIRequiresCodeVerifier(t *testing.T) {
	c := &domain.OAuthClient{ClientID: "c-fapi", Metadata: map[string]any{"fapi_2_0": true}}
	form := url.Values{"grant_type": {"authorization_code"}}
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("DPoP", "proof")
	if err := enforceFAPIToken(c, req); err == nil {
		t.Fatal("expected FAPI token to require code_verifier")
	}
}

func TestEnforceFAPIToken_FAPICompliant(t *testing.T) {
	c := &domain.OAuthClient{ClientID: "c-fapi", Metadata: map[string]any{"fapi_2_0": true}}
	form := url.Values{"grant_type": {"authorization_code"}, "code_verifier": {"verifier"}}
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("DPoP", "proof")
	if err := enforceFAPIToken(c, req); err != nil {
		t.Fatalf("expected compliant FAPI token request to pass: %v", err)
	}
}

func TestUpdateClientMetadata_FAPI2_0(t *testing.T) {
	clients := []*domain.OAuthClient{
		{ClientID: "c-001", Name: "Test", Metadata: map[string]any{}},
	}
	svc := newTestOAuthService(clients)

	ctx := tenant.WithContext(context.Background(), &tenant.Context{
		TenantID:       uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		IsolationLevel: tenant.IsolationShared,
	})

	updated, err := svc.UpdateClientMetadata(ctx, "c-001", &service.ClientMetadataUpdate{
		Metadata: map[string]any{"fapi_2_0": true},
	})
	if err != nil {
		t.Fatalf("update metadata failed: %v", err)
	}
	if !updated.FAPI2_0() {
		t.Fatal("expected FAPI 2.0 to be enabled after update")
	}
}
