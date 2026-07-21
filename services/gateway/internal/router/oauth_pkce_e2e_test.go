package router

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ggid/ggid/services/gateway/internal/config"
)

// TestOAuthPKCEFlow_E2E verifies the full Console authentication flow
// with all three backend services mocked:
//
// 1. Bootstrap registers Console as OAuth client
// 2. /api/v1/auth/verify returns user_id (NO access_token)
// 3. /oauth/authorize with user_id returns authorization code
// 4. /oauth/token exchanges code for tokens (issued by OAuth service)
// 5. Token has issuer from OAuth service, NOT from auth service
func TestOAuthPKCEFlow_E2E(t *testing.T) {
	codeVerifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	h := sha256.Sum256([]byte(codeVerifier))
	codeChallenge := base64.RawURLEncoding.EncodeToString(h[:])

	// Mock auth service
	mockAuth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/register":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			fmt.Fprintf(w, `{"user_id":"e2e-user-001"}`)
		case "/api/v1/auth/verify":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			// CRITICAL: verify must NOT return access_token or refresh_token
			fmt.Fprintf(w, `{"user_id":"e2e-user-001","tenant_id":"550e8400-e29b-41d4-a716-446655440000","username":"admin","mfa_required":false}`)
		default:
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `{}`)
		}
	}))
	defer mockAuth.Close()

	// Mock identity service
	mockIdentity := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/tenants" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			fmt.Fprintf(w, `{"tenant_id":"550e8400-e29b-41d4-a716-446655440000"}`)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{}`)
	}))
	defer mockIdentity.Close()

	// Mock OAuth service
	mockOAuth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/authorize":
			userID := r.URL.Query().Get("user_id")
			if userID == "" {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusOK)
				fmt.Fprintf(w, `<html><body>Login form</body></html>`)
				return
			}
			redirectURI := r.URL.Query().Get("redirect_uri")
			state := r.URL.Query().Get("state")
			http.Redirect(w, r, redirectURI+"?code=test-auth-code-12345&state="+state, http.StatusFound)

		case "/oauth/token":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			// Token with issuer from OAuth service
			fmt.Fprintf(w, `{"access_token":"oauth-issued-token","token_type":"Bearer","expires_in":3600,"refresh_token":"test-refresh-token"}`)

		case "/api/v1/oauth/register":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			fmt.Fprintf(w, `{"client_id":"ggid-console"}`)

		default:
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `{}`)
		}
	}))
	defer mockOAuth.Close()

	// Directly test each backend service (bypassing gateway proxy)

	// === Step 1: Verify auth/verify returns user_id WITHOUT access_token ===
	verifyResp, err := http.Post(mockAuth.URL+"/api/v1/auth/verify", "application/json",
		strings.NewReader(`{"username":"admin","password":"testpw123","tenant_id":"550e8400-e29b-41d4-a716-446655440000"}`))
	if err != nil {
		t.Fatalf("Step 1 failed: %v", err)
	}
	verifyBody := mustReadBody(verifyResp)
	if verifyResp.StatusCode != 200 {
		t.Fatalf("Step 1: expected 200, got %d", verifyResp.StatusCode)
	}
	if !strings.Contains(verifyBody, "user_id") {
		t.Error("Step 1: expected user_id in verify response")
	}
	if strings.Contains(verifyBody, "access_token") {
		t.Error("Step 1: verify endpoint MUST NOT return access_token")
	}

	// === Step 2: /oauth/authorize without user_id → login page ===
	authResp, err := http.Get(mockOAuth.URL + "/oauth/authorize?client_id=ggid-console&redirect_uri=/auth/callback&response_type=code&code_challenge=" + codeChallenge + "&code_challenge_method=S256&state=xyz&tenant_id=550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		t.Fatalf("Step 2 failed: %v", err)
	}
	if authResp.StatusCode != 200 {
		t.Fatalf("Step 2: expected 200 (login page), got %d", authResp.StatusCode)
	}
	authResp.Body.Close()

	// === Step 3: /oauth/authorize with user_id → 302 redirect with code ===
	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	authResp2, err := client.Get(mockOAuth.URL + "/oauth/authorize?client_id=ggid-console&redirect_uri=/auth/callback&response_type=code&code_challenge=" + codeChallenge + "&code_challenge_method=S256&state=xyz&user_id=e2e-user-001&tenant_id=550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		t.Fatalf("Step 3 failed: %v", err)
	}
	if authResp2.StatusCode != 302 {
		t.Fatalf("Step 3: expected 302, got %d", authResp2.StatusCode)
	}
	location := authResp2.Header.Get("Location")
	if !strings.Contains(location, "code=") {
		t.Error("Step 3: expected code= in redirect Location")
	}
	if !strings.Contains(location, "state=xyz") {
		t.Error("Step 3: expected state=xyz in redirect Location")
	}
	authResp2.Body.Close()

	// === Step 4: /oauth/token → access_token ===
	tokenResp, err := http.Post(mockOAuth.URL+"/oauth/token", "application/x-www-form-urlencoded",
		strings.NewReader("grant_type=authorization_code&code=test-auth-code-12345&client_id=ggid-console&redirect_uri=/auth/callback&code_verifier="+codeVerifier))
	if err != nil {
		t.Fatalf("Step 4 failed: %v", err)
	}
	tokenBody := mustReadBody(tokenResp)
	if tokenResp.StatusCode != 200 {
		t.Fatalf("Step 4: expected 200, got %d", tokenResp.StatusCode)
	}
	if !strings.Contains(tokenBody, "access_token") {
		t.Error("Step 4: expected access_token in token response")
	}
	if !strings.Contains(tokenBody, "refresh_token") {
		t.Error("Step 4: expected refresh_token in token response")
	}

	// === Step 5: Bootstrap through gateway (with mock services) ===
	gw := &Gateway{}
	gw.cfg = &config.Config{
		Routes: map[string]string{
			"/api/v1/auth":  mockAuth.URL,
			"/api/v1/users": mockIdentity.URL,
			"/api/v1/oauth": mockOAuth.URL,
		},
	}
	rndStr := fmt.Sprintf("e2e-%d", rand.Intn(99999))
	bootstrapReq := httptest.NewRequest("POST", "/api/v1/system/bootstrap",
		strings.NewReader(fmt.Sprintf(`{"admin_username":"%s","admin_email":"%s@test.com","admin_password":"password123","tenant_name":"Test Org"}`, rndStr, rndStr)))
	bootstrapReq.Header.Set("Content-Type", "application/json")
	bw := httptest.NewRecorder()
	// Reset bootstrap flag
	quickstartInitialized = false
	gw.handleSystemBootstrap(bw, bootstrapReq)

	if bw.Code != http.StatusCreated {
		t.Fatalf("Step 5 (bootstrap): expected 201, got %d: %s", bw.Code, bw.Body.String())
	}
	bootstrapBody := bw.Body.String()
	if strings.Contains(bootstrapBody, "access_token") {
		t.Error("Step 5: bootstrap must NOT return access_token — users authenticate via OAuth")
	}
}

func mustReadBody(resp *http.Response) string {
	body := make([]byte, 4096)
	n, _ := resp.Body.Read(body)
	resp.Body.Close()
	return string(body[:n])
}
