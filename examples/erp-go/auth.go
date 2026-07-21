package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	ggid "github.com/ggid/ggid/sdk/go"
)

// PKCE session store (in-memory, per-instance)
var (
	pkceSessions   = make(map[string]*pkceSession)
	pkceSessionsMu sync.Mutex
)

type pkceSession struct {
	Verifier   string
	Redirect   string
	CreatedAt  time.Time
}

// handleOAuthLogin redirects user to GGID OAuth2 authorize endpoint with PKCE
func handleOAuthLogin(w http.ResponseWriter, r *http.Request) {
	clientID := getEnv("OAUTH_CLIENT_ID", "erp-go-demo")
	redirectURI := getEnv("OAUTH_REDIRECT_URI", fmt.Sprintf("http://%s/api/auth/callback", r.Host))
	scopes := "openid profile email"

	// Generate PKCE verifier + challenge
	verifier := generateCodeVerifier()
	challenge := generateCodeChallenge(verifier)
	state := randomString(16)

	// Store PKCE session
	pkceSessionsMu.Lock()
	pkceSessions[state] = &pkceSession{
		Verifier:  verifier,
		Redirect:  redirectURI,
		CreatedAt: time.Now(),
	}
	pkceSessionsMu.Unlock()

	// Clean old sessions (> 10 min)
	go cleanPKCESessions()

	// Build authorize URL
	params := url.Values{
		"response_type":         {"code"},
		"client_id":             {clientID},
		"redirect_uri":          {redirectURI},
		"scope":                 {scopes},
		"state":                 {state},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
	}

	authURL := fmt.Sprintf("%s/api/v1/oauth/authorize?%s", ggidURL, params.Encode())
	http.Redirect(w, r, authURL, http.StatusFound)
}

// handleOAuthCallback handles the OAuth2 callback, exchanges code for token
func handleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	errorParam := r.URL.Query().Get("error")

	if errorParam != "" {
		writeJSON(w, 400, map[string]string{"error": errorParam})
		return
	}
	if code == "" || state == "" {
		writeJSON(w, 400, map[string]string{"error": "missing code or state"})
		return
	}

	// Retrieve PKCE session
	pkceSessionsMu.Lock()
	session, ok := pkceSessions[state]
	if ok {
		delete(pkceSessions, state)
	}
	pkceSessionsMu.Unlock()

	if !ok {
		writeJSON(w, 400, map[string]string{"error": "invalid or expired state"})
		return
	}

	clientID := getEnv("OAUTH_CLIENT_ID", "erp-go-demo")
	clientSecret := getEnv("OAUTH_CLIENT_SECRET", "")

	// Exchange code for token
	tokenReq := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {session.Redirect},
		"client_id":     {clientID},
		"code_verifier": {session.Verifier},
	}
	if clientSecret != "" {
		tokenReq.Set("client_secret", clientSecret)
	}

	req, err := http.NewRequestWithContext(r.Context(), "POST",
		fmt.Sprintf("%s/api/v1/oauth/token", ggidURL),
		strings.NewReader(tokenReq.Encode()))
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "failed to create token request"})
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Tenant-ID", tenantID)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		writeJSON(w, 502, map[string]string{"error": "failed to call GGID token endpoint"})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var tokens map[string]interface{}
	if err := json.Unmarshal(body, &tokens); err != nil {
		writeJSON(w, 500, map[string]string{"error": "failed to parse token response"})
		return
	}

	if resp.StatusCode != 200 {
		writeJSON(w, resp.StatusCode, tokens)
		return
	}

	// Return tokens to client (in production, set httpOnly cookies)
	writeJSON(w, 200, tokens)
}

// handleLogin — keep for backward compat but delegate to OAuth flow info
func handleLogin(w http.ResponseWriter, r *http.Request) {
	if !methodAllowed(w, r, http.MethodPost) {
		return
	}
	// Legacy: direct username/password login (for testing only)
	var req struct {
		Username, Password string
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, "invalid body")
		return
	}
	tokens, err := ggidClient.Login(r.Context(), &ggid.LoginRequest{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		writeJSON(w, 401, map[string]string{"error": "login failed"})
		return
	}
	writeJSON(w, 200, tokens)
}

func handleRefresh(w http.ResponseWriter, r *http.Request) {
	if !methodAllowed(w, r, http.MethodPost) {
		return
	}
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, "invalid body")
		return
	}
	tokens, err := ggidClient.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		writeJSON(w, 401, map[string]string{"error": "refresh failed"})
		return
	}
	writeJSON(w, 200, tokens)
}

func handleVerify(w http.ResponseWriter, r *http.Request) {
	if !methodAllowed(w, r, http.MethodPost) {
		return
	}
	var req struct {
		Token string
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, "invalid body")
		return
	}
	info, err := ggidClient.VerifyToken(r.Context(), req.Token)
	if err != nil {
		writeJSON(w, 401, map[string]string{"error": "invalid token"})
		return
	}
	writeJSON(w, 200, info)
}

// PKCE helpers

func generateCodeVerifier() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func generateCodeChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

func randomString(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func cleanPKCESessions() {
	pkceSessionsMu.Lock()
	defer pkceSessionsMu.Unlock()
	for k, v := range pkceSessions {
		if time.Since(v.CreatedAt) > 10*time.Minute {
			delete(pkceSessions, k)
		}
	}
}

var _ = context.Background // keep import