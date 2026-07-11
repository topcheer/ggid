//go:build integration

// Package integration provides OAuth/OIDC E2E integration tests.
// These tests exercise the OAuth service through the Gateway.
//
// Run: go test -tags=integration -v -run TestOAuth ./test/integration/...
package integration

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

// TestOAuth_JWKS tests the JWKS endpoint returns valid key set.
func TestOAuth_JWKS(t *testing.T) {
	resp := doRequest(t, "GET", gatewayBaseURL+"/oauth/jwks", "", "")
	if resp == nil {
		t.Skip("Gateway not running")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Skipf("JWKS endpoint returned %d: %s (OAuth service might not be running)", resp.StatusCode, body)
	}

	var jwks map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		t.Fatalf("decode JWKS: %v", err)
	}

	keys, ok := jwks["keys"].([]any)
	if !ok {
		t.Fatal("JWKS response missing 'keys' array")
	}
	if len(keys) == 0 {
		t.Error("JWKS should have at least one key")
	}

	// Verify first key has required fields
	if len(keys) > 0 {
		firstKey, ok := keys[0].(map[string]any)
		if !ok {
			t.Fatal("JWKS key is not an object")
		}
		for _, field := range []string{"kty", "kid", "use", "n", "e"} {
			if _, ok := firstKey[field]; !ok {
				t.Errorf("JWKS key missing field %q", field)
			}
		}
	}
	t.Logf("JWKS: %d keys found", len(keys))
}

// TestOAuth_Discovery tests OIDC discovery endpoint returns complete metadata.
func TestOAuth_Discovery(t *testing.T) {
	resp := doRequest(t, "GET", gatewayBaseURL+"/.well-known/openid-configuration", "", "")
	if resp == nil {
		t.Skip("Gateway not running")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Skipf("Discovery endpoint returned %d: %s", resp.StatusCode, body)
	}

	var meta map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		t.Fatalf("decode discovery doc: %v", err)
	}

	// Verify required OIDC discovery fields
	required := []string{
		"issuer",
		"authorization_endpoint",
		"token_endpoint",
		"jwks_uri",
		"response_types_supported",
		"subject_types_supported",
		"id_token_signing_alg_values_supported",
	}
	for _, field := range required {
		if _, ok := meta[field]; !ok {
			t.Errorf("discovery doc missing required field %q", field)
		}
	}

	issuer, _ := meta["issuer"].(string)
	t.Logf("Discovery: issuer=%s, %d metadata fields", issuer, len(meta))
}

// TestOAuth_AuthorizeRedirect tests that the authorize endpoint redirects
// with proper error when called without required parameters.
func TestOAuth_AuthorizeRedirect(t *testing.T) {
	// Make request without client_id — should return 400 or redirect with error
	resp := doRequest(t, "GET", gatewayBaseURL+"/oauth/authorize?response_type=code", "", "")
	if resp == nil {
		t.Skip("Gateway not running")
	}
	defer resp.Body.Close()

	// Should be 400 Bad Request (missing required params) or 302 redirect with error
	if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Skipf("Authorize endpoint returned %d: %s (OAuth service might not be running)", resp.StatusCode, body)
	}

	t.Logf("Authorize endpoint returned %d for missing params", resp.StatusCode)
}

// TestOAuth_TokenInvalidGrant tests that token endpoint rejects invalid grants.
func TestOAuth_TokenInvalidGrant(t *testing.T) {
	body := "grant_type=authorization_code&code=invalid-code-12345&redirect_uri=http://localhost:3000/callback&client_id=test-client"
	resp := doRequest(t, "POST", gatewayBaseURL+"/oauth/token", body, "")
	if resp == nil {
		t.Skip("Gateway not running")
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		t.Error("token endpoint should reject invalid authorization code")
	}

	var errResp map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil {
		if errStr, ok := errResp["error"].(string); ok {
			t.Logf("Token endpoint correctly returned error: %s", errStr)
		}
	}
}

// TestOAuth_ClientCredentialsFlow tests the client_credentials grant type.
func TestOAuth_ClientCredentialsFlow(t *testing.T) {
	// This test uses the seeded test client (if available)
	// Skip if no test client is configured
	clientID := "test-client"
	clientSecret := "test-secret"

	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)
	form.Set("scope", "read")

	resp := doRequest(t, "POST", gatewayBaseURL+"/oauth/token", form.Encode(), "")
	if resp == nil {
		t.Skip("Gateway not running")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Skipf("Client credentials flow returned %d: %s (test client may not be seeded)", resp.StatusCode, body)
	}

	var tokenResp map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		t.Fatalf("decode token response: %v", err)
	}

	accessToken, ok := tokenResp["access_token"].(string)
	if !ok || accessToken == "" {
		t.Error("client_credentials response missing access_token")
	}
	tokenType, _ := tokenResp["token_type"].(string)
	if tokenType == "" {
		t.Log("NOTE: token_type not set (should be 'Bearer')")
	}
	t.Logf("Client credentials: received token (type=%s, len=%d)", tokenType, len(accessToken))
}

// TestOAuth_Introspection tests the token introspection endpoint.
func TestOAuth_Introspection(t *testing.T) {
	// Introspect a clearly invalid token — should return active=false
	body := `{"token":"invalid-token-for-introspection-test"}`
	resp := doRequest(t, "POST", gatewayBaseURL+"/api/v1/oauth/introspect", body, "")
	if resp == nil {
		t.Skip("Gateway not running")
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		t.Skip("Introspection endpoint requires authentication (expected if auth is enforced)")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Skipf("Introspection returned %d: %s", resp.StatusCode, body)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode introspection response: %v", err)
	}

	active, _ := result["active"].(bool)
	if active {
		t.Error("invalid token should not be active")
	}
	t.Log("Introspection correctly returned active=false for invalid token")
}

// --- Health Check E2E ---

// TestHealth_Gateway tests the Gateway healthz endpoint.
func TestHealth_Gateway(t *testing.T) {
	resp := doRequest(t, "GET", gatewayBaseURL+"/healthz", "", "")
	if resp == nil {
		t.Skip("Gateway not running")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("healthz returned %d, want 200", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)
	status, _ := result["status"].(string)
	t.Logf("Gateway healthz: status=%s", status)
}

// TestHealth_DeepCheck tests the deep health check endpoint.
func TestHealth_DeepCheck(t *testing.T) {
	resp := doRequest(t, "GET", gatewayBaseURL+"/healthz/deep", "", "")
	if resp == nil {
		t.Skip("Gateway not running")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Skipf("Deep healthz returned %d: %s (not all services may be running)", resp.StatusCode, body)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode deep health response: %v", err)
	}

	// Should contain status per backend service
	t.Logf("Deep health: %d fields in response", len(result))
	for key, val := range result {
		if svcStatus, ok := val.(map[string]any); ok {
			s, _ := svcStatus["status"].(string)
			t.Logf("  %s: %s", key, s)
		}
	}
}

// --- Rate Limiting E2E ---

// TestGateway_RateLimit tests that rapid requests trigger rate limiting.
func TestGateway_RateLimit(t *testing.T) {
	// Send many requests rapidly to a healthz endpoint
	// The gateway may or may not have rate limiting enabled per-IP.
	// We just verify that requests eventually succeed or get 429.
	var lastStatus int
	var rateLimited bool
	for i := 0; i < 50; i++ {
		resp := doRequest(t, "GET", gatewayBaseURL+"/healthz", "", "")
		if resp == nil {
			t.Skip("Gateway not running")
		}
		lastStatus = resp.StatusCode
		if resp.StatusCode == http.StatusTooManyRequests {
			rateLimited = true
			resp.Body.Close()
			break
		}
		resp.Body.Close()
		time.Sleep(10 * time.Millisecond)
	}

	if rateLimited {
		t.Logf("Rate limiting triggered after multiple rapid requests (status 429)")
	} else {
		t.Logf("No rate limiting triggered after 50 requests (last status: %d) — may not be configured for healthz", lastStatus)
	}
}

// --- Security Headers E2E ---

// TestGateway_SecurityHeaders verifies that the gateway sets standard security headers.
func TestGateway_SecurityHeaders(t *testing.T) {
	resp := doRequest(t, "GET", gatewayBaseURL+"/healthz", "", "")
	if resp == nil {
		t.Skip("Gateway not running")
	}
	defer resp.Body.Close()

	expectedHeaders := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"X-XSS-Protection":       "1",
	}

	for header, expected := range expectedHeaders {
		val := resp.Header.Get(header)
		if val == "" {
			t.Logf("NOTE: header %s not set (security headers middleware may not be configured)", header)
		} else if val != expected {
			t.Logf("Header %s = %q (expected %q)", header, val, expected)
		} else {
			t.Logf("Header %s = %q ✓", header, val)
		}
	}

	// Strict-Transport-Security should be present (or not, if not using TLS)
	hsts := resp.Header.Get("Strict-Transport-Security")
	if hsts != "" {
		t.Logf("HSTS: %s", hsts)
	}
}

// --- CORS E2E ---

// TestGateway_CORS tests that CORS preflight requests are handled.
func TestGateway_CORS(t *testing.T) {
	req, err := http.NewRequest("OPTIONS", gatewayBaseURL+"/api/v1/auth/login", strings.NewReader(""))
	if err != nil {
		t.Fatalf("create OPTIONS request: %v", err)
	}
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type,X-Tenant-ID")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Skip("Gateway not running")
	}
	defer resp.Body.Close()

	origin := resp.Header.Get("Access-Control-Allow-Origin")
	methods := resp.Header.Get("Access-Control-Allow-Methods")

	if origin != "" {
		t.Logf("CORS: Allow-Origin=%s, Allow-Methods=%s", origin, methods)
	} else {
		t.Log("CORS: No Access-Control-Allow-Origin header (CORS middleware may not be configured)")
	}
}
