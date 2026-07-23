package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	ggidcrypto "github.com/ggid/ggid/pkg/crypto"
	"github.com/ggid/ggid/pkg/social"
	"github.com/ggid/ggid/pkg/tenant"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// socialState holds the OAuth state context for CSRF protection.
type socialState struct {
	TenantID    uuid.UUID
	Provider    string
	RedirectURI string
	CreatedAt   time.Time
}

// socialStateStore is a simple in-memory TTL store for OAuth state.
type socialStateStore struct {
	mu      sync.RWMutex
	entries map[string]socialStateEntry
}

type socialStateEntry struct {
	value    *socialState
	expireAt time.Time
}

var socialStates = &socialStateStore{entries: make(map[string]socialStateEntry)}

// isAllowedRedirectURI validates that the redirect_uri is a safe destination.
// Prevents open redirect attacks (P2-11).
func isAllowedRedirectURI(uri string) bool {
	parsed, err := url.Parse(uri)
	if err != nil {
		return false
	}
	if parsed.Scheme != "https" {
		if parsed.Host != "localhost" && !strings.HasPrefix(parsed.Host, "localhost:") {
			return false
		}
	}
	if parsed.Host == "" {
		return false
	}
	return true
}

func (s *socialStateStore) Set(key string, val *socialState, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries[key] = socialStateEntry{value: val, expireAt: time.Now().Add(ttl)}
}

func (s *socialStateStore) Get(key string) (*socialState, bool) {
	s.mu.RLock()
	e, ok := s.entries[key]
	s.mu.RUnlock()
	if !ok || time.Now().After(e.expireAt) {
		s.mu.Lock()
		delete(s.entries, key)
		s.mu.Unlock()
		return nil, false
	}
	return e.value, true
}

func (s *socialStateStore) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.entries, key)
}

// socialLoginConfig is the JSON shape stored in idp_config.config_json.
type socialLoginConfig struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RedirectURI  string `json:"redirect_uri"` // optional override
}

// handleSocialLogin initiates the OAuth flow for a social provider.
// GET /api/v1/auth/social/{provider}?redirect_uri=https://console.ggid.iot2.win/auth/callback
func (h *Handler) handleSocialLogin(w http.ResponseWriter, r *http.Request) {
	provider, redirectURI, err := parseSocialPath(r.URL.Path)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	// Resolve tenant from header or query param (social login is pre-auth).
	tenantID, err := resolveSocialTenant(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "tenant context required: set X-Tenant-ID header or ?tenant_id= param"})
		return
	}

	// Load connector config from DB.
	cfg, err := h.loadSocialConfig(r.Context(), tenantID, provider)
	if err != nil {
		slog.Error("social login: load config", "provider", provider, "err", err)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": fmt.Sprintf("social provider %q is not configured for this tenant", provider),
		})
		return
	}

	// Create connector instance.
	connector, err := createSocialConnector(provider, cfg)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	// Generate state for CSRF protection.
	state := uuid.New().String()

	// Build callback URI.
	callbackURI := cfg.RedirectURI
	if callbackURI == "" {
		callbackURI = redirectURI
	}
	if callbackURI == "" {
		scheme := "https"
		if r.TLS == nil {
			scheme = "http"
		}
		callbackURI = fmt.Sprintf("%s://%s/api/v1/auth/social/%s/callback", scheme, r.Host, provider)
	}

	// P2-11 fix: Validate redirect_uri against allowlist to prevent open redirect.
	if redirectURI != "" && !isAllowedRedirectURI(redirectURI) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "redirect_uri not allowed"})
		return
	}

	// Store state for CSRF protection (5 min TTL).
	socialStates.Set(state, &socialState{
		TenantID:    tenantID,
		Provider:    provider,
		RedirectURI: redirectURI,
		CreatedAt:   time.Now(),
	}, 5*time.Minute)

	authURL, err := connector.GetAuthURL(r.Context(), state, callbackURI)
	if err != nil {
		slog.Error("social login: get auth URL", "provider", provider, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to generate authorization URL"})
		return
	}

	slog.Info("social login: redirecting", "provider", provider, "tenant", tenantID)
	http.Redirect(w, r, authURL, http.StatusFound)
}

// handleSocialCallback handles the IdP callback, exchanges the code for user
// info, JIT-provisions or links the user, and issues a JWT.
// GET /api/v1/auth/social/{provider}/callback?code=...&state=...
func (h *Handler) handleSocialCallback(w http.ResponseWriter, r *http.Request) {
	provider, _, err := parseSocialPath(r.URL.Path)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if code == "" || state == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing code or state parameter"})
		return
	}

	// Validate state.
	st, ok := socialStates.Get(state)
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid or expired state"})
		return
	}
	socialStates.Delete(state)

	if st.Provider != provider {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "provider mismatch in state"})
		return
	}

	// Load connector config.
	cfg, err := h.loadSocialConfig(r.Context(), st.TenantID, provider)
	if err != nil {
		slog.Error("social callback: load config", "provider", provider, "err", err)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "social provider not configured"})
		return
	}

	connector, err := createSocialConnector(provider, cfg)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	// Build callback URI (same as in handleSocialLogin).
	callbackURI := cfg.RedirectURI
	if callbackURI == "" {
		scheme := "https"
		if r.TLS == nil {
			scheme = "http"
		}
		callbackURI = fmt.Sprintf("%s://%s/api/v1/auth/social/%s/callback", scheme, r.Host, provider)
	}

	// Exchange code for user info.
	userInfo, err := connector.HandleCallback(r.Context(), code, state, callbackURI)
	if err != nil {
		slog.Error("social callback: exchange", "provider", provider, "err", err)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "failed to exchange authorization code"})
		return
	}

	// Set tenant context for downstream calls.
	tc := &tenant.Context{
		TenantID:       st.TenantID,
		IsolationLevel: tenant.IsolationShared,
	}
	r = r.WithContext(tenant.WithContext(r.Context(), tc))

	// JIT provision: find or create user.
	userID, err := h.jitProvisionUser(r.Context(), st.TenantID, userInfo)
	if err != nil {
		slog.Error("social callback: JIT provision", "provider", provider, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to provision user"})
		return
	}

	// Issue JWT.
	token, err := h.issueSocialJWT(st.TenantID, userID, provider)
	if err != nil {
		slog.Error("social callback: issue JWT", "provider", provider, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to issue token"})
		return
	}

	// Audit: social login success.
	h.publishAuditEvent("user.social_login", "success", st.TenantID, userID)

	// Redirect to frontend with token in URL fragment (not query param — avoids logging).
	frontendURL := st.RedirectURI
	if frontendURL == "" {
		frontendURL = "/auth/callback"
	}
	redirectURL := fmt.Sprintf("%s#access_token=%s&token_type=Bearer&provider=%s", frontendURL, token, provider)
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

// jitProvisionUser finds an existing user by external identity or creates a new one.
func (h *Handler) jitProvisionUser(ctx context.Context, tenantID uuid.UUID, info *social.UserInfo) (uuid.UUID, error) {
	// 1. Check if external identity already linked.
	if h.pool != nil {
		var userIDStr string
		err := h.pool.QueryRow(ctx,
			`SELECT user_id::text FROM user_external_identities
			 WHERE provider = $1 AND external_id = $2
			 AND user_id IN (SELECT id FROM users WHERE tenant_id = $3)`,
			info.Provider, info.ExternalID, tenantID).Scan(&userIDStr)
		if err == nil {
			return uuid.Parse(userIDStr)
		}
	}

	// 2. Check if email matches an existing user in the tenant.
	// P2-13 fix: Only merge by email when the IdP has verified the email ownership.
	// Without this, an attacker can register a social account with a victim's email
	// and gain access to the victim's existing account (account takeover).
	if h.pool != nil && info.Email != "" && info.EmailVerified {
		var userIDStr string
		err := h.pool.QueryRow(ctx,
			`SELECT id::text FROM users WHERE email = $1 AND tenant_id = $2`,
			info.Email, tenantID).Scan(&userIDStr)
		if err == nil {
			userID, _ := uuid.Parse(userIDStr)
			h.linkExternalIdentity(ctx, tenantID, userID, info)
			return userID, nil
		}
	}

	// If email is not verified and matches no external identity, create a new
	// account instead of merging — prevents account takeover via unverified email.
	if !info.EmailVerified && info.Email != "" {
		slog.Warn("social JIT: email not verified by IdP, creating new account instead of merging",
			"provider", info.Provider, "email", info.Email)
	}

	// 3. Create new user via identity client (JIT provisioning).
	username := info.Email
	if username == "" {
		username = fmt.Sprintf("%s_%s", info.Provider, info.ExternalID)
	}
	displayName := info.Name
	if displayName == "" {
		displayName = username
	}

	user, err := h.authSvc.IdentityClient().CreateUserFromSocial(
		ctx, tenantID, username, info.Email, displayName,
		info.Provider, info.ExternalID, info.RawClaims,
	)
	if err != nil {
		return uuid.Nil, fmt.Errorf("create user from social: %w", err)
	}

	return user.ID, nil
}

// linkExternalIdentity inserts a record into user_external_identities.
func (h *Handler) linkExternalIdentity(ctx context.Context, tenantID, userID uuid.UUID, info *social.UserInfo) {
	if h.pool == nil {
		return
	}
	metaJSON, _ := json.Marshal(info.RawClaims)
	_, err := h.pool.Exec(ctx,
		`INSERT INTO user_external_identities (id, user_id, provider, external_id, metadata)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (provider, external_id) DO NOTHING`,
		uuid.New(), userID, info.Provider, info.ExternalID, metaJSON)
	if err != nil {
		slog.Warn("link external identity", "err", err)
	}
}

// issueSocialJWT signs a JWT for the social-login user.
func (h *Handler) issueSocialJWT(tenantID, userID uuid.UUID, provider string) (string, error) {
	kp := h.authSvc.KeyProvider()
	if kp == nil {
		return "", fmt.Errorf("key provider not available")
	}

	meta := kp.Metadata()
	now := time.Now()
	roles, permissions := h.authSvc.GetUserScopesAndPermissions(context.Background(), tenantID, userID)

	claims := jwt.MapClaims{
		"iss":         h.authSvc.JWTIssuer(),
		"aud":         h.authSvc.JWTAudience(),
		"sub":         userID.String(),
		"tenant_id":   tenantID.String(),
		"roles":       roles,
		"permissions": permissions,
		"amr":         []string{"social", provider},
		"acr":         "AAL1",
		"iat":         now.Unix(),
		"exp":         now.Add(h.authSvc.JWTTTL()).Unix(),
		"jti":         uuid.New().String(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = meta.KeyID

	signed, err := token.SignedString(kp.Signer())
	if err != nil {
		return "", fmt.Errorf("sign JWT: %w", err)
	}
	return signed, nil
}

// loadSocialConfig reads the IdP configuration from the database.
func (h *Handler) loadSocialConfig(ctx context.Context, tenantID uuid.UUID, provider string) (*socialLoginConfig, error) {
	if h.pool == nil {
		return nil, fmt.Errorf("database not available")
	}

	var configJSON string
	err := h.pool.QueryRow(ctx,
		`SELECT config_json FROM tenant_idp_configs
		 WHERE tenant_id = $1 AND idp_type = 'oidc' AND name = $2 AND enabled = true`,
		tenantID, provider).Scan(&configJSON)
	if err != nil {
		return nil, fmt.Errorf("no configured social provider %q: %w", provider, err)
	}

	var cfg socialLoginConfig
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return nil, fmt.Errorf("invalid config JSON for %q: %w", provider, err)
	}
	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		return nil, fmt.Errorf("incomplete config for %q: client_id and client_secret required", provider)
	}
	return &cfg, nil
}

// parseSocialPath extracts the provider name from the URL path.
func parseSocialPath(path string) (provider string, callbackURI string, err error) {
	prefix := "/api/v1/auth/social/"
	if !strings.HasPrefix(path, prefix) {
		return "", "", fmt.Errorf("invalid social login path")
	}
	rest := strings.TrimPrefix(path, prefix)
	parts := strings.Split(rest, "/")
	if len(parts) == 0 || parts[0] == "" {
		return "", "", fmt.Errorf("provider not specified")
	}
	return parts[0], "", nil
}

// resolveSocialTenant extracts tenant ID from header or query param.
func resolveSocialTenant(r *http.Request) (uuid.UUID, error) {
	idStr := r.Header.Get("X-Tenant-ID")
	if idStr == "" {
		idStr = r.URL.Query().Get("tenant_id")
	}
	if idStr == "" {
		return uuid.Nil, fmt.Errorf("missing tenant context")
	}
	return uuid.Parse(idStr)
}

// createSocialConnector creates a connector instance for the given provider.
func createSocialConnector(provider string, cfg *socialLoginConfig) (social.Connector, error) {
	switch strings.ToLower(provider) {
	case "google":
		return social.NewGoogleConnector(cfg.ClientID, cfg.ClientSecret), nil
	case "github":
		return social.NewGitHubConnector(cfg.ClientID, cfg.ClientSecret), nil
	case "discord":
		return social.NewDiscordConnector(cfg.ClientID, cfg.ClientSecret), nil
	case "microsoft":
		return social.NewMicrosoftConnector(cfg.ClientID, cfg.ClientSecret), nil
	case "gitlab":
		return social.NewGitLabConnector(cfg.ClientID, cfg.ClientSecret, "https://gitlab.com"), nil
	case "linkedin":
		return social.NewLinkedInConnector(cfg.ClientID, cfg.ClientSecret), nil
	case "apple":
		return social.NewAppleConnector(cfg.ClientID, cfg.ClientSecret), nil
	case "slack":
		return social.NewSlackConnector(cfg.ClientID, cfg.ClientSecret), nil
	default:
		return nil, fmt.Errorf("unsupported social provider: %s", provider)
	}
}

// Ensure ggidcrypto import is used (referenced in issueSocialJWT via kp.Metadata()).
var _ ggidcrypto.KeyProvider = (ggidcrypto.KeyProvider)(nil)
