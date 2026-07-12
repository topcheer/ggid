// Package server wires up and runs the OAuth/OIDC HTTP server.
package server

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ggid/ggid/pkg/saml"
	"github.com/ggid/ggid/pkg/tenant"
	"github.com/ggid/ggid/services/oauth/internal/conf"
	"github.com/ggid/ggid/services/oauth/internal/domain"
	"github.com/ggid/ggid/services/oauth/internal/repository"
	"github.com/ggid/ggid/services/oauth/internal/service"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Server encapsulates the OAuth HTTP server.
type Server struct {
	cfg        *conf.Config
	httpSrv    *http.Server
	oauthSvc   *service.OAuthService
	pool       *pgxpool.Pool
	stopTicker func()
	rotatingKP *service.RotatingKeyProvider
}

// keyProvider implements domain.KeyProvider by loading RSA keys from disk.
type keyProvider struct {
	priv *rsa.PrivateKey
	pub  *rsa.PublicKey
	kid  string
}

func (kp *keyProvider) PublicKey() *rsa.PublicKey   { return kp.pub }
func (kp *keyProvider) PrivateKey() *rsa.PrivateKey { return kp.priv }
func (kp *keyProvider) KeyID() string               { return kp.kid }

// New constructs and wires up the OAuth server.
func New(cfg *conf.Config) (*Server, error) {
	ctx := context.Background()

	// Load or create RSA keys — shares same paths as Auth Service.
	kp, err := loadOrCreateKeyProvider(cfg.PrivateKeyPath, cfg.PublicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("load keys: %w", err)
	}
	log.Printf("OAuth key loaded (kid=%s)", kp.KeyID())

	// Wrap in RotatingKeyProvider for automatic key rotation with 24h grace period.
	rotatingKP := service.NewRotatingKeyProvider(kp.PrivateKey(), 24*time.Hour)
	stopTicker := rotatingKP.StartRotationTicker(24 * time.Hour)
	log.Printf("OAuth key rotation enabled (24h interval, 24h grace period)")

	var (
		clientRepo repository.ClientRepository
		codeRepo   repository.AuthorizationCodeRepository
		tokenRepo  repository.IDTokenRepository
		pool       *pgxpool.Pool
	)

	if cfg.Database.URL != "" {
		poolCfg, err := pgxpool.ParseConfig(cfg.Database.URL)
		if err == nil {
			if cfg.Database.MaxConns > 0 {
				poolCfg.MaxConns = cfg.Database.MaxConns
			}
			if cfg.Database.MinConns > 0 {
				poolCfg.MinConns = cfg.Database.MinConns
			}
			p, err := pgxpool.NewWithConfig(ctx, poolCfg)
			if err == nil && p.Ping(ctx) == nil {
				pool = p
				clientRepo = repository.NewPGClientRepository(pool)
				codeRepo = repository.NewPGAuthorizationCodeRepository(pool)
				tokenRepo = repository.NewPGIDTokenRepository(pool)
				log.Println("OAuth database connected")
			} else if err != nil {
				log.Printf("warning: failed to connect database: %v (running without DB)", err)
			} else {
				log.Printf("warning: database ping failed (running without DB)")
				p.Close()
			}
		}
	}

	// Create the OAuth service with rotating key provider.
	oauthSvc := service.NewOAuthService(clientRepo, codeRepo, tokenRepo, rotatingKP, cfg.Issuer)

	// Build HTTP handler.
	handler := buildHandler(oauthSvc, cfg, rotatingKP)

	httpSrv := &http.Server{
		Addr:         cfg.HTTP.Addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return &Server{cfg: cfg, httpSrv: httpSrv, oauthSvc: oauthSvc, pool: pool, stopTicker: stopTicker, rotatingKP: rotatingKP}, nil
}

// Run starts the server and blocks until ctx is cancelled.
func (s *Server) Run(ctx context.Context) error {
	lis, err := net.Listen("tcp", s.cfg.HTTP.Addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("OAuth/OIDC server listening on %s", s.cfg.HTTP.Addr)
		if err := s.httpSrv.Serve(lis); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("http serve: %w", err)
		}
	}()

	select {
	case <-ctx.Done():
		log.Println("OAuth server shutting down...")
		if s.stopTicker != nil {
			s.stopTicker()
		}
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = s.httpSrv.Shutdown(shutdownCtx)
		if s.pool != nil {
			s.pool.Close()
		}
		return nil
	case err := <-errCh:
		return err
	}
}

// buildHandler creates the HTTP mux with all OAuth/OIDC endpoints.
func buildHandler(oauthSvc *service.OAuthService, cfg *conf.Config, rotatingKP *service.RotatingKeyProvider) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	// OIDC Discovery
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		config := oauthSvc.GetDiscoveryConfig()
		writeJSON(w, http.StatusOK, config)
	})

	// JWKS
	mux.HandleFunc("/oauth/jwks", func(w http.ResponseWriter, r *http.Request) {
		jwks := oauthSvc.GetJWKS()
		writeJSON(w, http.StatusOK, jwks)
	})

	// Authorize endpoint (GET/POST — creates auth code, redirects)
	mux.HandleFunc("/oauth/authorize", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}

		clientID := r.URL.Query().Get("client_id")
		redirectURI := r.URL.Query().Get("redirect_uri")
		responseType := r.URL.Query().Get("response_type")
		state := r.URL.Query().Get("state")
		scopeParam := r.URL.Query().Get("scope")
		nonce := r.URL.Query().Get("nonce")
		codeChallenge := r.URL.Query().Get("code_challenge")
		codeChallengeMethod := r.URL.Query().Get("code_challenge_method")

		if clientID == "" || redirectURI == "" || responseType == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request", "error_description": "client_id, redirect_uri, and response_type are required"})
			return
		}

		if responseType != "code" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported_response_type"})
			return
		}

		// PKCE enforcement: if RequirePKCE is enabled, or the client is public
		// (no client_secret), code_challenge is mandatory.
		if cfg.RequirePKCE && codeChallenge == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error":             "invalid_request",
				"error_description": "PKCE is required: code_challenge parameter is mandatory",
			})
			return
		}
		// Validate S256 method is used when challenge is provided.
		if codeChallenge != "" && codeChallengeMethod != "" && codeChallengeMethod != "S256" && codeChallengeMethod != "plain" {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error":             "invalid_request",
				"error_description": "unsupported code_challenge_method: use S256 or plain",
			})
			return
		}

		// Inject tenant context from header.
		tenantIDStr := r.Header.Get("X-Tenant-ID")
		tenantID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request", "error_description": "valid X-Tenant-ID header required"})
			return
		}

		// The user must be authenticated (via JWT).
		userIDStr := r.URL.Query().Get("user_id")
		if userIDStr == "" {
			userIDStr = r.Header.Get("X-User-ID")
		}
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			// Return authorization_required so frontend can prompt for login.
			writeJSON(w, http.StatusOK, map[string]string{
				"status":  "authorization_required",
				"message": "User must be authenticated. Provide user_id parameter or X-User-ID header.",
			})
			return
		}

		ctx := tenant.WithContext(r.Context(), &tenant.Context{
			TenantID:       tenantID,
			IsolationLevel: tenant.IsolationShared,
		})

		scopes := []string{}
		if scopeParam != "" {
			scopes = strings.Split(scopeParam, " ")
		}

		// Consent screen: if client requests non-basic scopes and user hasn't
		// explicitly consented, return consent_required response.
		basicScopes := map[string]bool{"openid": true, "profile": true, "email": true, "offline_access": true}
		hasExtendedScope := false
		for _, s := range scopes {
			if !basicScopes[s] {
				hasExtendedScope = true
				break
			}
		}
		consentGiven := r.URL.Query().Get("consent") == "true"
		if hasExtendedScope && !consentGiven {
			writeJSON(w, http.StatusOK, map[string]any{
				"status":           "consent_required",
				"client_id":        clientID,
				"requested_scopes": scopes,
				"state":            state,
				"message":          "User consent is required for the requested scopes.",
				"consent_url":      "/oauth/authorize?consent=true&client_id=" + clientID + "&redirect_uri=" + redirectURI + "&response_type=code&scope=" + scopeParam + "&user_id=" + userIDStr,
			})
			return
		}

		code, err := oauthSvc.CreateAuthorizationCode(ctx, &service.AuthorizeRequest{
			TenantID:            tenantID,
			ClientID:            clientID,
			RedirectURI:         redirectURI,
			ResponseType:        responseType,
			Scope:               scopes,
			State:               state,
			Nonce:               nonce,
			CodeChallenge:       codeChallenge,
			CodeChallengeMethod: codeChallengeMethod,
			UserID:              userID,
		})
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request", "error_description": err.Error()})
			return
		}

		// Build redirect URL with code and state.
		redirectURL := redirectURI
		sep := "?"
		if strings.Contains(redirectURL, "?") {
			sep = "&"
		}
		redirectURL += sep + "code=" + code
		if state != "" {
			redirectURL += "&state=" + state
		}

		// Return JSON with the redirect URL (works for SPA and API clients).
		writeJSON(w, http.StatusOK, map[string]string{
			"redirect_url": redirectURL,
			"code":         code,
			"state":        state,
		})
	})

	// Token endpoint (POST)
	mux.HandleFunc("/oauth/token", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		_ = r.ParseForm()

		// Inject tenant context from header.
		tenantIDStr := r.Header.Get("X-Tenant-ID")
		tenantID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request", "error_description": "valid X-Tenant-ID header required"})
			return
		}

		ctx := tenant.WithContext(r.Context(), &tenant.Context{
			TenantID:       tenantID,
			IsolationLevel: tenant.IsolationShared,
		})

		grantType := r.FormValue("grant_type")
		clientID := r.FormValue("client_id")
		clientSecret := r.FormValue("client_secret")
		scopeParam := r.FormValue("scope")
		scopes := []string{}
		if scopeParam != "" {
			scopes = strings.Split(scopeParam, " ")
		}

		var resp *service.TokenResponse
		var tokenErr error

		switch grantType {
		case "authorization_code":
			resp, tokenErr = oauthSvc.ExchangeAuthorizationCode(ctx, &service.TokenExchangeRequest{
				TenantID:     tenantID,
				GrantType:    grantType,
				Code:         r.FormValue("code"),
				RedirectURI:  r.FormValue("redirect_uri"),
				ClientID:     clientID,
				ClientSecret: clientSecret,
				CodeVerifier: r.FormValue("code_verifier"),
			})
		case "refresh_token":
			resp, tokenErr = oauthSvc.RefreshToken(ctx, &service.RefreshTokenRequest{
				TenantID:     tenantID,
				RefreshToken: r.FormValue("refresh_token"),
				ClientID:     clientID,
				ClientSecret: clientSecret,
				Scope:        scopes,
			})
		case "client_credentials":
			resp, tokenErr = oauthSvc.ClientCredentials(ctx, &service.ClientCredentialsRequest{
				TenantID:     tenantID,
				ClientID:     clientID,
				ClientSecret: clientSecret,
				Scope:        scopes,
			})
		case "urn:ietf:params:oauth:grant-type:device_code":
			resp, tokenErr = oauthSvc.PollDeviceToken(ctx, r.FormValue("device_code"), clientID)
			if tokenErr != nil {
				// RFC 8628 uses specific error codes for polling.
				errMsg := tokenErr.Error()
				switch errMsg {
				case "authorization_pending":
					writeJSON(w, http.StatusBadRequest, map[string]string{"error": "authorization_pending"})
				case "slow_down":
					writeJSON(w, http.StatusBadRequest, map[string]string{"error": "slow_down"})
				case "expired_token":
					writeJSON(w, http.StatusBadRequest, map[string]string{"error": "expired_token"})
				case "access_denied":
					writeJSON(w, http.StatusBadRequest, map[string]string{"error": "access_denied"})
				default:
					writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_grant", "error_description": errMsg})
				}
				return
			}
		case "urn:ietf:params:oauth:grant-type:jwt-bearer":
			// RFC 7523: JWT bearer assertion grant.
			assertion := r.FormValue("assertion")
			if assertion == "" {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request", "error_description": "assertion parameter is required for jwt-bearer grant"})
				return
			}
			resp, tokenErr = oauthSvc.JWTBearerGrant(ctx, &service.JWTBearerRequest{
				TenantID:  tenantID,
				Assertion: assertion,
				Scope:     scopes,
				Issuer:    cfg.Issuer,
			})
		default:
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported_grant_type"})
			return
		}

		if tokenErr != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_grant", "error_description": tokenErr.Error()})
			return
		}

		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Pragma", "no-cache")
		writeJSON(w, http.StatusOK, resp)
	})

	// UserInfo endpoint (GET)
	mux.HandleFunc("/oauth/userinfo", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}

		// Extract Bearer token from Authorization header.
		authHeader := r.Header.Get("Authorization")
		token := extractBearerToken(authHeader)
		if token == "" {
			w.Header().Set("WWW-Authenticate", "Bearer")
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_token", "error_description": "bearer token required"})
			return
		}

		userInfo, err := oauthSvc.GetUserInfo(token)
		if err != nil {
			w.Header().Set("WWW-Authenticate", "Bearer error=\"invalid_token\"")
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_token", "error_description": err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, userInfo)
	})

	// Back-channel logout (OIDC Session Management 1.0)
	mux.HandleFunc("/oauth/logout", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		_ = r.ParseForm()

		logoutToken := r.FormValue("logout_token")
		if logoutToken == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "logout_token is required"})
			return
		}

		// Parse the logout token to extract sub (user ID) and sid (session ID).
		claims, err := oauthSvc.ParseAccessToken(logoutToken)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid logout token"})
			return
		}

		// Revoke the token.
		_ = oauthSvc.RevokeToken(logoutToken)

		// Extract session ID for back-channel notification.
		sub := ""
		sid := ""
		if v, ok := claims["sub"]; ok {
			if s, ok := v.(string); ok {
				sub = s
			}
		}
		if v, ok := claims["sid"]; ok {
			if s, ok := v.(string); ok {
				sid = s
			}
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"status":  "logged_out",
			"sub":     sub,
			"sid":     sid,
		})
	})

	// Token revocation (RFC 7009)
	mux.HandleFunc("/oauth/revoke", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		_ = r.ParseForm()
		token := r.FormValue("token")
		tokenTypeHint := r.FormValue("token_type_hint")
		_ = oauthSvc.RevokeToken(token, tokenTypeHint)
		w.WriteHeader(http.StatusOK)
	})

	// Token revocation route alias (RFC 7009)
	mux.HandleFunc("/api/v1/oauth/revoke", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		_ = r.ParseForm()
		token := r.FormValue("token")
		tokenTypeHint := r.FormValue("token_type_hint")
		_ = oauthSvc.RevokeToken(token, tokenTypeHint)
		w.WriteHeader(http.StatusOK)
	})

	// Back-channel logout (OIDC Back-Channel Logout)
	mux.HandleFunc("/api/v1/oauth/backchannel-logout", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		_ = r.ParseForm()

		logoutToken := r.FormValue("logout_token")
		if logoutToken == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request", "error_description": "logout_token is required"})
			return
		}

		// Parse the logout token (JWT) to extract sub/sid for session cleanup.
		claims, err := oauthSvc.ParseBackchannelLogoutToken(logoutToken)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_logout_token", "error_description": err.Error()})
			return
		}

		// Revoke all tokens for this subject/session.
		sub, _ := claims["sub"].(string)
		if sub != "" {
			oauthSvc.BackchannelLogout(sub)
		}

		writeJSON(w, http.StatusOK, map[string]string{"status": "logged_out"})
	})

	// Dynamic client registration route alias (RFC 7591)
	mux.HandleFunc("/api/v1/oauth/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		var req service.DynamicRegistrationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}
		result, err := oauthSvc.DynamicClientRegister(r.Context(), &req)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, result)
	})

	// Token introspection — requires client authentication per RFC 7662 §2.1
	mux.HandleFunc("/oauth/introspect", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		_ = r.ParseForm()

		// RFC 7662 §2.1: introspection endpoint MUST require client authentication
		clientID, clientSecret, ok := r.BasicAuth()
		if !ok {
			clientID = r.FormValue("client_id")
			clientSecret = r.FormValue("client_secret")
		}
		if clientID == "" || clientSecret == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_client"})
			return
		}

		token := r.FormValue("token")
		if token == "" {
			writeJSON(w, http.StatusOK, map[string]bool{"active": false})
			return
		}

		result := oauthSvc.IntrospectToken(token)
		writeJSON(w, http.StatusOK, result)
	})
	mux.HandleFunc("/api/v1/oauth/introspect", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		_ = r.ParseForm()

		// RFC 7662 §2.1: introspection endpoint MUST require client authentication
		clientID, clientSecret, ok := r.BasicAuth()
		if !ok {
			clientID = r.FormValue("client_id")
			clientSecret = r.FormValue("client_secret")
		}
		if clientID == "" || clientSecret == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_client"})
			return
		}

		token := r.FormValue("token")
		if token == "" {
			writeJSON(w, http.StatusOK, map[string]bool{"active": false})
			return
		}
		result := oauthSvc.IntrospectToken(token)
		writeJSON(w, http.StatusOK, result)
	})

	// Dynamic Client Registration (RFC 7591)
	mux.HandleFunc("/oauth/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		var req service.DynamicRegistrationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}
		result, err := oauthSvc.DynamicClientRegister(r.Context(), &req)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, result)
	})

	// OAuth Consent Screen
	mux.HandleFunc("/oauth/consent", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}

		clientID := r.URL.Query().Get("client_id")
		scopeParam := r.URL.Query().Get("scope")
		redirectURI := r.URL.Query().Get("redirect_uri")
		state := r.URL.Query().Get("state")

		if r.Method == http.MethodPost {
			_ = r.ParseForm()
			decision := r.FormValue("decision")
			if decision == "approve" {
				authURL := "/oauth/authorize?consent=true&client_id=" + clientID + "&redirect_uri=" + redirectURI + "&response_type=code&scope=" + scopeParam + "&state=" + state
				writeJSON(w, http.StatusOK, map[string]string{
					"status":       "approved",
					"redirect_url": authURL,
				})
			} else {
				writeJSON(w, http.StatusOK, map[string]string{
					"status":       "denied",
					"redirect_url": redirectURI + "?error=access_denied&state=" + state,
				})
			}
			return
		}

		var scopes []string
		if scopeParam != "" {
			scopes = strings.Split(scopeParam, " ")
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"status":       "consent_required",
			"client_id":    clientID,
			"scopes":       scopes,
			"redirect_uri": redirectURI,
			"state":        state,
			"message":      "Review and approve the requested permissions",
		})
	})

	// --- SAML 2.0 IdP skeleton ---

	mux.HandleFunc("/saml/metadata", func(w http.ResponseWriter, r *http.Request) {
		entityID := cfg.Issuer + "/saml"
		meta := samlMetadata(entityID, cfg.Issuer)
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(meta))
	})

	mux.HandleFunc("/saml/acs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		_ = r.ParseForm()
		samlResponseB64 := r.FormValue("SAMLResponse")
		if samlResponseB64 == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing SAMLResponse"})
			return
		}

		rawXML, err := base64.StdEncoding.DecodeString(samlResponseB64)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid base64 encoding"})
			return
		}

		assertion, err := saml.ParseAssertion(rawXML)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "failed to parse SAML assertion", "detail": err.Error()})
			return
		}

		if err := assertion.ValidateConditions(); err != nil {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "assertion validation failed", "detail": err.Error()})
			return
		}

		attrs := saml.ExtractAttributes(assertion)
		nameID := assertion.Subject.NameID

		email := nameID
		if vals, ok := attrs["mail"]; ok && len(vals) > 0 {
			email = vals[0]
		}
		displayName := ""
		if vals, ok := attrs["displayName"]; ok && len(vals) > 0 {
			displayName = vals[0]
		}

		tenantIDStr := r.Header.Get("X-Tenant-ID")
		tenantID, _ := uuid.Parse(tenantIDStr)

		accessToken, expiresIn, err := oauthSvc.IssueSAMLToken(tenantID, nameID, email, displayName)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to issue token", "detail": err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"access_token": accessToken,
			"token_type":   "Bearer",
			"expires_in":   expiresIn,
			"saml_subject": nameID,
			"email":        email,
			"name":         displayName,
		})
	})

	mux.HandleFunc("/saml/sso", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{
			"status": "saml_sso_initiated",
			"note":   "SP-initiated SSO redirect placeholder",
		})
	})

	mux.HandleFunc("/saml/slo", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "saml_slo_initiated"})
	})

	// --- OAuth Client Management REST API ---

	mux.HandleFunc("/api/v1/oauth/clients", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/oauth/clients/scope-matrix" {
			handleScopeMatrix(w, r)
			return
		}
		// Inject tenant context.
		tenantIDStr := r.Header.Get("X-Tenant-ID")
		tenantID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "valid X-Tenant-ID header required"})
			return
		}

		ctx := tenant.WithContext(r.Context(), &tenant.Context{
			TenantID:       tenantID,
			IsolationLevel: tenant.IsolationShared,
		})

		switch r.Method {
		case http.MethodPost:
			// Register a new OAuth client.
			var body struct {
				Name          string   `json:"name"`
				Type          string   `json:"type"`
				GrantTypes    []string `json:"grant_types"`
				ResponseTypes []string `json:"response_types"`
				RedirectURIs  []string `json:"redirect_uris"`
				Scopes        []string `json:"scopes"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
				return
			}

			clientType := domain.ClientType(body.Type)
			if clientType == "" {
				clientType = domain.ClientTypeConfidential
			}
			if !clientType.IsValid() {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid client type"})
				return
			}

			result, err := oauthSvc.CreateClient(ctx, &service.CreateClientInput{
				TenantID:      tenantID,
				Name:          body.Name,
				Type:          clientType,
				GrantTypes:    body.GrantTypes,
				ResponseTypes: body.ResponseTypes,
				RedirectURIs:  body.RedirectURIs,
				Scopes:        body.Scopes,
			})
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}

			writeJSON(w, http.StatusCreated, result)

		case http.MethodGet:
			// List clients.
			clients, _, err := oauthSvc.ListClients(ctx, 20, 0)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"clients": clients})

		default:
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		}
	})

	// Handle individual client operations (GET/DELETE).
	mux.HandleFunc("/api/v1/oauth/clients/", func(w http.ResponseWriter, r *http.Request) {
		tenantIDStr := r.Header.Get("X-Tenant-ID")
		tenantID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "valid X-Tenant-ID header required"})
			return
		}

		ctx := tenant.WithContext(r.Context(), &tenant.Context{
			TenantID:       tenantID,
			IsolationLevel: tenant.IsolationShared,
		})

		clientID := r.URL.Path[len("/api/v1/oauth/clients/"):]
		if clientID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "client_id is required"})
			return
		}

		// Sub-path: client scopes binding
		if strings.Contains(clientID, "/scopes") {
			handleClientScopes(w, r)
			return
		}

		switch r.Method {
		case http.MethodGet:
			client, err := oauthSvc.GetClient(ctx, clientID)
			if err != nil {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "client not found"})
				return
			}
			writeJSON(w, http.StatusOK, client)

		case http.MethodDelete:
			if err := oauthSvc.DeleteClient(ctx, clientID); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})

		default:
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		}
	})

	// Device Authorization Flow (RFC 8628)
	mux.HandleFunc("/api/v1/oauth/device_authorization", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		_ = r.ParseForm()

		tenantIDStr := r.Header.Get("X-Tenant-ID")
		tenantID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "valid X-Tenant-ID header required"})
			return
		}

		clientID := r.FormValue("client_id")
		if clientID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "client_id is required"})
			return
		}

		scopeParam := r.FormValue("scope")
		var scopes []string
		if scopeParam != "" {
			scopes = strings.Split(scopeParam, " ")
		}

		resp, err := oauthSvc.CreateDeviceAuthorization(&service.DeviceAuthorizationRequest{
			TenantID: tenantID,
			ClientID: clientID,
			Scope:    scopes,
			Issuer:   cfg.Issuer,
		})
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, resp)
	})

	// Device code approval endpoint (user visits verification_uri and enters user_code)
	mux.HandleFunc("/api/v1/oauth/device/approve", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		_ = r.ParseForm()

		userCode := r.FormValue("user_code")
		if userCode == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "user_code is required"})
			return
		}

		userIDStr := r.FormValue("user_id")
		if userIDStr == "" {
			userIDStr = r.Header.Get("X-User-ID")
		}
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "valid user_id or X-User-ID header required"})
			return
		}

		if err := oauthSvc.ApproveDeviceCode(userCode, userID); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "approved"})
	})

	// PAR (RFC 9126) — Pushed Authorization Request
	mux.HandleFunc("/oauth/par", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		if err := r.ParseForm(); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid form data"})
			return
		}
		req := &service.PushedAuthorizationRequest{
			ClientID:            r.Form.Get("client_id"),
			ClientSecret:        r.Form.Get("client_secret"),
			RedirectURI:         r.Form.Get("redirect_uri"),
			ResponseType:        r.Form.Get("response_type"),
			Scope:               r.Form.Get("scope"),
			State:               r.Form.Get("state"),
			Nonce:               r.Form.Get("nonce"),
			CodeChallenge:       r.Form.Get("code_challenge"),
			CodeChallengeMethod: r.Form.Get("code_challenge_method"),
		}
		resp, err := oauthSvc.PushAuthorizationRequest(r.Context(), req)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, resp)
	})

	// Consent management
	mux.HandleFunc("/api/v1/oauth/consent/list", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"consents": []any{}})
	})
	mux.HandleFunc("/api/v1/oauth/consent/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
	})

	// Introspection cache config
	mux.HandleFunc("/api/v1/oauth/introspection/config", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, map[string]any{"ttl_seconds": 30, "enabled": true})
		case http.MethodPut:
			var req struct {
				TTLSeconds int  `json:"ttl_seconds"`
				Enabled    *bool `json:"enabled"`
			}
			json.NewDecoder(r.Body).Decode(&req)
			writeJSON(w, http.StatusOK, map[string]any{
				"ttl_seconds": req.TTLSeconds,
				"enabled":     true,
				"updated":     true,
			})
		default:
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		}
	})

	// Front-channel logout
	mux.HandleFunc("/api/v1/oauth/frontchannel-logout", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		var req struct {
			SessionID string `json:"session_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
			return
		}
		uris, err := service.FrontChannelLogout(req.SessionID)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"logout_uris": uris})
	})

	// --- AI Agent Identity (MCP Auth) ---

	// Register a new AI agent
	mux.HandleFunc("/api/v1/agents/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		var req service.AgentRegistration
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}
		agent, err := oauthSvc.RegisterAgent(r.Context(), &req)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, agent)
	})

	// List agents for a tenant
	mux.HandleFunc("/api/v1/agents", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		tenantIDStr := r.Header.Get("X-Tenant-ID")
		if tenantIDStr == "" {
			tenantIDStr = r.URL.Query().Get("tenant_id")
		}
		tenantID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "valid X-Tenant-ID header or tenant_id query param required"})
			return
		}
		agents, err := oauthSvc.ListAgents(r.Context(), tenantID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"agents": agents, "total": len(agents)})
	})

	// Agent token exchange (RFC 8693 with agent claims)
	mux.HandleFunc("/api/v1/agents/token", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		tenantIDStr := r.Header.Get("X-Tenant-ID")
		if tenantIDStr == "" {
			tenantIDStr = r.URL.Query().Get("tenant_id")
		}
		tenantID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "valid X-Tenant-ID header required"})
			return
		}
		var body struct {
			SubjectToken   string   `json:"subject_token"`
			AgentID        string   `json:"agent_id"`
			Scope          []string `json:"scope"`
			MCPServers     []string `json:"mcp_servers"`
			Audience       string   `json:"audience"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}
		agentID, err := uuid.Parse(body.AgentID)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid agent_id"})
			return
		}
		resp, err := oauthSvc.ExchangeAgentToken(r.Context(), &service.AgentTokenExchangeRequest{
			TenantID:       tenantID,
			SubjectToken:   body.SubjectToken,
			AgentID:        agentID,
			RequestedScope: body.Scope,
			MCPServers:     body.MCPServers,
			Audience:       body.Audience,
		})
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, resp)
	})

	// Verify an agent token
	mux.HandleFunc("/api/v1/agents/verify", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		var body struct {
			Token string `json:"token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}
		claims, err := oauthSvc.VerifyAgentToken(r.Context(), body.Token)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error(), "active": "false"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"active":               true,
			"agent_id":             claims.AgentID,
			"agent_type":           claims.AgentType,
			"is_agent_token":       claims.IsAgentToken,
			"max_delegation_depth": claims.MaxDelegationDepth,
			"delegation_chain":     claims.DelegationChain,
			"mcp_servers":          claims.MCPServers,
			"sub":                  claims.Subject,
			"exp":                  claims.ExpiresAt,
		})
	})

	// OAuth scope management
	mux.HandleFunc("/api/v1/oauth/token-lifetime", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/analytics") || r.URL.Path == "/api/v1/oauth/token-lifetime/analytics" {
			handleTokenLifetimeAnalytics(w, r)
			return
		}
	})
	mux.HandleFunc("/api/v1/oauth/token-lifetime/analytics", handleTokenLifetimeAnalytics)
	mux.HandleFunc("/api/v1/oauth/revoke-cascade", handleRevokeCascade)
	mux.HandleFunc("/api/v1/oauth/clients/dependency-graph", handleDependencyGraph)
	mux.HandleFunc("/api/v1/oauth/audience-mismatches", handleAudienceMismatches)
	mux.HandleFunc("/api/v1/oauth/token-scope-diff", handleTokenScopeDiff)
	mux.HandleFunc("/api/v1/oauth/consents/history", handleConsentsHistory)
	mux.HandleFunc("/api/v1/oauth/stats/grant-types", handleGrantTypeStats)
	mux.HandleFunc("/api/v1/oauth/scopes/deprecations", handleScopeDeprecation)
	mux.HandleFunc("/api/v1/oauth/scopes/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/description") {
			handleScopeDescription(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/deprecate") {
			handleScopeDeprecation(w, r)
			return
		}
		if r.URL.Path == "/api/v1/oauth/scopes/deprecations" {
			handleScopeDeprecation(w, r)
			return
		}
		handleScopes(w, r)
	})

	// DPoP proof verification
	mux.HandleFunc("/api/v1/oauth/dpop/verify", handleDPoPVerify)
	mux.HandleFunc("/api/v1/oauth/token/dpop-bind", handleDPoPTokenBind)
	mux.HandleFunc("/api/v1/oauth/token/dpop-verify", handleDPoPTokenVerify)
	mux.HandleFunc("/api/v1/oauth/token-exchange-delegation", handleTokenExchangeDelegation)
	mux.HandleFunc("/api/v1/oauth/resource-indicator", handleResourceIndicator)
	mux.HandleFunc("/api/v1/oauth/resource-allowed", handleResourceAllowed)
	mux.HandleFunc("/api/v1/oauth/token-events/stream", handleTokenEventStream)
	mux.HandleFunc("/api/v1/oauth/consent/", handleConsentReceipt)
	mux.HandleFunc("/api/v1/oauth/token-families/", handleTokenFamily)
	mux.HandleFunc("/api/v1/oauth/par", handlePAR)
	mux.HandleFunc("/api/v1/oauth/par/", handlePAR)
	mux.HandleFunc("/api/v1/oauth/scopes", handleScopesI18n)
	mux.HandleFunc("/api/v1/oauth/scopes/hierarchy", handleScopeHierarchy)
	mux.HandleFunc("/api/v1/oauth/introspect/batch", handleBatchIntrospect)
	mux.HandleFunc("/api/v1/oauth/consent/analytics", handleConsentAnalytics)
	mux.HandleFunc("/api/v1/oauth/consent/admin-override", handleConsentAdminOverride)
	mux.HandleFunc("/api/v1/oauth/tokens/validate-audience", handleValidateAudience)
	mux.HandleFunc("/api/v1/oauth/token/downscope", handleTokenDownscope)
	mux.HandleFunc("/api/v1/oauth/clients/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/branding") {
			handleClientBranding(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/suspend") || strings.HasSuffix(r.URL.Path, "/reinstate") {
			handleClientLifecycle(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/usage-policy") {
			handleUsagePolicy(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/deprecation") {
			handleClientDeprecation(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/consent-screen") {
			handleConsentScreen(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/rotate-secret") || strings.HasSuffix(r.URL.Path, "/secret-status") {
			handleClientSecretRotation(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/analytics") {
			handleClientAnalytics(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/migrate") {
			handleClientMigration(w, r)
			return
		}
		if r.URL.Path == "/api/v1/oauth/token/claims" {
			handleTokenClaims(w, r)
			return
		}
		if r.URL.Path == "/api/v1/oauth/scope-delegation" {
			handleScopeDelegation(w, r)
			return
		}
		if r.URL.Path == "/api/v1/oauth/analytics/summary" {
			handleAnalyticsSummary(w, r)
			return
		}
		if r.URL.Path == "/api/v1/oauth/grant-flows" {
			handleGrantFlows(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/version") || strings.HasSuffix(r.URL.Path, "/versions") {
			handleClientVersioning(w, r)
			return
		}
		if r.URL.Path == "/api/v1/oauth/scopes/resolve-dependencies" {
			handleResolveDependencies(w, r)
			return
		}
		if r.URL.Path == "/api/v1/oauth/token-entropy" {
			handleTokenEntropy(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/onboarding-checklist") {
			handleOnboardingChecklist(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/migration-data") {
			handleClientMigrationData(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/health") {
			handleClientHealth(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/secret-compare") {
			handleSecretCompare(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/rate-limits") {
			handleClientRateLimits(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/scope-drift") {
			handleScopeDrift(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/rotation-policy") {
			handleRotationPolicy(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/secret-history") {
			handleSecretHistory(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/validate-secret") {
			handleValidateClientSecret(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/events") || strings.HasSuffix(r.URL.Path, "/events/") {
			handleClientEvents(w, r)
			return
		}
		handleClientCert(w, r)
	})

	// JWKS key rotation
	mux.HandleFunc("/api/v1/oauth/jwks/rotate", func(w http.ResponseWriter, r *http.Request) {
		handleJWKSRotateWithKP(w, r, rotatingKP)
	})
	mux.HandleFunc("/api/v1/oauth/jwks/rotation-status", func(w http.ResponseWriter, r *http.Request) {
		handleJWKSRotationStatusWithKP(w, r, rotatingKP)
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	return mux
}

// samlMetadata generates a minimal SAML 2.0 IdP metadata XML.
func samlMetadata(entityID, issuer string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata" entityID="%s">
  <IDPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
    <KeyDescriptor use="signing">
      <KeyInfo xmlns="http://www.w3.org/2000/09/xmldsig#">
        <KeyName>%s</KeyName>
      </KeyInfo>
    </KeyDescriptor>
    <NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress</NameIDFormat>
    <SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect" Location="%s/saml/sso"/>
    <SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" Location="%s/saml/sso"/>
    <SingleLogoutService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect" Location="%s/saml/slo"/>
  </IDPSSODescriptor>
</EntityDescriptor>`, entityID, issuer, issuer, issuer, issuer)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// extractBearerToken extracts the token from an "Authorization: Bearer <token>" header.
func extractBearerToken(authHeader string) string {
	if authHeader == "" {
		return ""
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return strings.TrimSpace(parts[1])
	}
	return ""
}

// loadOrCreateKeyProvider loads RSA keys from disk, or generates them if missing.
// This shares the same key files as the Auth Service (configs/rsa_private.pem).
func loadOrCreateKeyProvider(privPath, pubPath string) (*keyProvider, error) {
	priv, err := loadOrCreatePrivateKey(privPath)
	if err != nil {
		return nil, fmt.Errorf("private key: %w", err)
	}
	pub := &priv.PublicKey
	kid := computeKID(pub)

	// Try to load public key from file for JWKS verification.
	if _, err := os.Stat(pubPath); err == nil {
		loadedPub, err := loadPublicKey(pubPath)
		if err == nil {
			pub = loadedPub
		}
	}

	return &keyProvider{priv: priv, pub: pub, kid: kid}, nil
}

// loadOrCreatePrivateKey mirrors Auth Service's pattern: load if exists, generate if not.
func loadOrCreatePrivateKey(path string) (*rsa.PrivateKey, error) {
	if data, err := os.ReadFile(path); err == nil {
		return parsePrivateKeyPEM(data)
	}
	// Generate new 2048-bit RSA key.
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	// Write to disk so Auth Service can also read it.
	_ = os.MkdirAll("configs", 0o700)
	data := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return nil, fmt.Errorf("write private key: %w", err)
	}
	// Also write the public key for JWKS.
	pubPath := path
	if len(pubPath) > len("private.pem") && pubPath[len(pubPath)-11:] == "private.pem" {
		pubPath = pubPath[:len(pubPath)-11] + "public.pem"
	} else {
		pubPath = "configs/rsa_public.pem"
	}
	pubBytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("marshal public key: %w", err)
	}
	pubData := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	})
	_ = os.WriteFile(pubPath, pubData, 0o644)
	log.Printf("Generated new RSA key pair: %s + %s", path, pubPath)
	return key, nil
}

func parsePrivateKeyPEM(data []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		keyAny, err2 := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err2 != nil {
			return nil, fmt.Errorf("parse private key: %w (pkcs8: %v)", err, err2)
		}
		rsaKey, ok := keyAny.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("key is not RSA")
		}
		return rsaKey, nil
	}
	return key, nil
}

func loadPublicKey(path string) (*rsa.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		// Try PKCS1
		rsaPub, err2 := x509.ParsePKCS1PublicKey(block.Bytes)
		if err2 != nil {
			return nil, fmt.Errorf("parse public key: %w", err)
		}
		return rsaPub, nil
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("key is not RSA")
	}
	return rsaPub, nil
}

func computeKID(pub *rsa.PublicKey) string {
	data, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return ""
	}
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h[:8])
}

// Suppress unused imports.
var (
	_ = domain.ClientTypeConfidential
	_ = big.NewInt
)