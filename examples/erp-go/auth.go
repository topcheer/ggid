package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
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

// handleOAuthLogin redirects user to GGID OAuth2 authorize endpoint with PKCE.
// Uses SDK GetAuthorizeURL() instead of manual URL construction.
func handleOAuthLogin(w http.ResponseWriter, r *http.Request) {
	clientID := getEnv("OAUTH_CLIENT_ID", "erp-go-demo")
	redirectURI := getEnv("OAUTH_REDIRECT_URI", fmt.Sprintf("http://%s/api/auth/callback", r.Host))

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

	// Use SDK to build the authorize URL
	authURL := ggidClient.GetAuthorizeURL(clientID, redirectURI, tenantID,
		ggid.WithState(state),
		ggid.WithCodeChallenge(challenge),
	)
	http.Redirect(w, r, authURL, http.StatusFound)
}

// handleOAuthCallback handles the OAuth2 callback.
// Uses SDK ExchangeCode() instead of manual HTTP POST to token endpoint.
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

	// Use SDK to exchange authorization code for tokens
	tokens, err := ggidClient.ExchangeCode(r.Context(), code, session.Redirect, clientID, session.Verifier, tenantID)
	if err != nil {
		writeJSON(w, 502, map[string]string{"error": "token exchange failed: " + err.Error()})
		return
	}

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
		ClientID: getEnv("OAUTH_CLIENT_ID", "erp-go-demo"),
		TenantID: tenantID,
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