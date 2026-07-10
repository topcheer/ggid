// Package service implements the OAuth2/OIDC business logic.
package service

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/ggid/ggid/pkg/crypto"
	"github.com/ggid/ggid/pkg/errors"
	"github.com/ggid/ggid/pkg/tenant"
	"github.com/ggid/ggid/services/oauth/internal/domain"
	"github.com/ggid/ggid/services/oauth/internal/repository"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// OAuthService implements OAuth2 client management and the authorization code flow.
type OAuthService struct {
	clientRepo repository.ClientRepository
	codeRepo   repository.AuthorizationCodeRepository
	tokenRepo  repository.IDTokenRepository
	keyProvider domain.KeyProvider
	issuer      string
}

// NewOAuthService creates a new OAuthService.
func NewOAuthService(
	clientRepo repository.ClientRepository,
	codeRepo repository.AuthorizationCodeRepository,
	tokenRepo repository.IDTokenRepository,
	keyProvider domain.KeyProvider,
	issuer string,
) *OAuthService {
	return &OAuthService{
		clientRepo:  clientRepo,
		codeRepo:    codeRepo,
		tokenRepo:   tokenRepo,
		keyProvider: keyProvider,
		issuer:      issuer,
	}
}

// --- Client Management ---

// CreateClientInput holds parameters for registering a new OAuth client.
type CreateClientInput struct {
	TenantID                uuid.UUID
	Name                    string
	Type                    domain.ClientType
	GrantTypes              []string
	ResponseTypes           []string
	RedirectURIs            []string
	Scopes                  []string
	TokenEndpointAuthMethod string
	Metadata                map[string]any
}

// CreateClientResult contains the new client and the plaintext secret (shown once).
type CreateClientResult struct {
	Client       *domain.OAuthClient
	ClientSecret string // plaintext secret — only returned on creation
}

// CreateClient registers a new OAuth2 client application.
func (s *OAuthService) CreateClient(ctx context.Context, input *CreateClientInput) (*CreateClientResult, error) {
	clientID := generateClientID()
	client := &domain.OAuthClient{
		ID:                      uuid.New(),
		TenantID:                input.TenantID,
		ClientID:                clientID,
		Name:                    input.Name,
		Type:                    input.Type,
		GrantTypes:              input.GrantTypes,
		ResponseTypes:           input.ResponseTypes,
		RedirectURIs:            input.RedirectURIs,
		Scopes:                  input.Scopes,
		TokenEndpointAuthMethod: defaultIfEmpty(input.TokenEndpointAuthMethod, "client_secret_basic"),
		Metadata:                input.Metadata,
		Enabled:                 true,
	}

	var plaintextSecret string
	if client.IsConfidential() {
		plaintextSecret = generateClientSecret()
		hash, err := crypto.HashPassword(plaintextSecret)
		if err != nil {
			return nil, errors.Internal("hash client secret", err)
		}
		client.ClientSecretHash = hash
	}

	if err := s.clientRepo.CreateClient(ctx, client); err != nil {
		return nil, err
	}

	return &CreateClientResult{Client: client, ClientSecret: plaintextSecret}, nil
}

// GetClient retrieves a client by its public client_id.
func (s *OAuthService) GetClient(ctx context.Context, clientID string) (*domain.OAuthClient, error) {
	tc, err := tenant.FromContext(ctx)
	if err != nil {
		return nil, errors.New(errors.ErrFailedPrecondition, "missing tenant context")
	}
	return s.clientRepo.GetClientByID(ctx, tc.TenantID, clientID)
}

// ListClients returns a paginated list of OAuth clients.
func (s *OAuthService) ListClients(ctx context.Context, pageSize, offset int) ([]*domain.OAuthClient, int, error) {
	tc, err := tenant.FromContext(ctx)
	if err != nil {
		return nil, 0, errors.New(errors.ErrFailedPrecondition, "missing tenant context")
	}
	return s.clientRepo.ListClients(ctx, tc.TenantID, pageSize, offset)
}

// DeleteClient removes a client registration.
func (s *OAuthService) DeleteClient(ctx context.Context, clientID string) error {
	tc, err := tenant.FromContext(ctx)
	if err != nil {
		return errors.New(errors.ErrFailedPrecondition, "missing tenant context")
	}
	return s.clientRepo.DeleteClient(ctx, tc.TenantID, clientID)
}

// --- Authorization Code Flow ---

// AuthorizeRequest holds parameters for the /oauth/authorize endpoint.
type AuthorizeRequest struct {
	TenantID            uuid.UUID
	ClientID            string
	RedirectURI         string
	ResponseType        string // "code"
	Scope               []string
	State               string
	Nonce               string
	CodeChallenge       string // PKCE
	CodeChallengeMethod string // "S256" or "plain"
	UserID              uuid.UUID // the authenticated user
}

// CreateAuthorizationCode creates a short-lived authorization code.
func (s *OAuthService) CreateAuthorizationCode(ctx context.Context, req *AuthorizeRequest) (string, error) {
	client, err := s.clientRepo.GetClientByID(ctx, req.TenantID, req.ClientID)
	if err != nil {
		return "", err
	}

	if !client.ValidateRedirectURI(req.RedirectURI) {
		return "", errors.InvalidArgument("redirect_uri not registered for this client")
	}

	if client.ResponseTypes != nil && len(client.ResponseTypes) > 0 {
		if !contains(client.ResponseTypes, req.ResponseType) {
			return "", errors.InvalidArgument("response_type not allowed for this client")
		}
	}

	plaintextCode, err := crypto.GenerateRandomToken(32)
	if err != nil {
		return "", errors.Internal("generate auth code", err)
	}

	code := &domain.AuthorizationCode{
		ID:                  uuid.New(),
		TenantID:            req.TenantID,
		CodeHash:            hashCode(plaintextCode),
		ClientID:            client.ID,
		UserID:              req.UserID,
		RedirectURI:         req.RedirectURI,
		Scope:               req.Scope,
		CodeChallenge:       req.CodeChallenge,
		CodeChallengeMethod: req.CodeChallengeMethod,
		Nonce:               req.Nonce,
		ExpiresAt:           time.Now().Add(10 * time.Minute), // auth codes are short-lived
	}

	if err := s.codeRepo.CreateCode(ctx, code); err != nil {
		return "", err
	}

	return plaintextCode, nil
}

// TokenExchangeRequest holds parameters for the /oauth/token endpoint.
type TokenExchangeRequest struct {
	TenantID       uuid.UUID
	GrantType      string // "authorization_code"
	Code           string // the plaintext authorization code
	RedirectURI    string
	ClientID       string
	ClientSecret   string // for confidential clients
	CodeVerifier   string // PKCE code_verifier
}

// TokenResponse is the standard OAuth2 token endpoint response.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// ExchangeAuthorizationCode exchanges an authorization code for tokens.
func (s *OAuthService) ExchangeAuthorizationCode(ctx context.Context, req *TokenExchangeRequest) (*TokenResponse, error) {
	// 1. Look up the client.
	client, err := s.clientRepo.GetClientByID(ctx, req.TenantID, req.ClientID)
	if err != nil {
		return nil, errors.Unauthenticated("client authentication failed")
	}

	// 2. Verify client secret for confidential clients.
	if client.IsConfidential() {
		ok, _ := crypto.VerifyPassword(req.ClientSecret, client.ClientSecretHash)
		if !ok {
			return nil, errors.Unauthenticated("invalid client credentials")
		}
	}

	// 3. Consume the authorization code (atomically — prevents replay).
	code, err := s.codeRepo.ConsumeCode(ctx, hashCode(req.Code))
	if err != nil {
		return nil, err
	}

	// 4. Validate the code matches this client.
	if code.ClientID != client.ID {
		return nil, errors.InvalidArgument("authorization code was issued to a different client")
	}

	// 5. Validate redirect_uri matches.
	if code.RedirectURI != req.RedirectURI {
		return nil, errors.InvalidArgument("redirect_uri mismatch")
	}

	// 6. Validate PKCE if applicable.
	if !code.ValidatePKCE(req.CodeVerifier) {
		return nil, errors.InvalidArgument("PKCE verification failed")
	}

	// 7. Issue tokens.
	// Access token — delegated to the auth service in production.
	// For now we issue a self-contained JWT as a placeholder.
	accessToken, expiresIn, err := s.issueAccessToken(code.UserID, code.TenantID, client.ClientID)
	if err != nil {
		return nil, err
	}

	resp := &TokenResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   expiresIn,
		Scope:       joinScopes(code.Scope),
	}

	// 8. Issue ID Token if OIDC scope is present.
	if contains(code.Scope, "openid") {
		idToken, err := s.issueIDToken(code.UserID, code.TenantID, client.ClientID, code.Nonce)
		if err != nil {
			return nil, err
		}
		resp.IDToken = idToken
	}

	return resp, nil
}

// --- OIDC Discovery ---

// GetDiscoveryConfig returns the OIDC discovery document.
func (s *OAuthService) GetDiscoveryConfig() *domain.OIDCDiscoveryConfig {
	base := s.issuer
	return &domain.OIDCDiscoveryConfig{
		Issuer:                            s.issuer,
		AuthorizationEndpoint:             base + "/oauth/authorize",
		TokenEndpoint:                     base + "/oauth/token",
		UserInfoEndpoint:                  base + "/oauth/userinfo",
		JwksURI:                           base + "/oauth/jwks",
		RevocationEndpoint:                base + "/oauth/revoke",
		IntrospectionEndpoint:             base + "/oauth/introspect",
		ResponseTypesSupported:            []string{"code", "token", "id_token"},
		GrantTypesSupported:               []string{"authorization_code", "refresh_token", "client_credentials"},
		SubjectTypesSupported:             []string{"public"},
		IDTokenSigningAlgValues:           []string{"RS256"},
		ScopesSupported:                   []string{"openid", "profile", "email", "offline_access"},
		ClaimsSupported:                   []string{"sub", "email", "name", "picture", "groups", "preferred_username", "updated_at"},
		TokenEndpointAuthMethodsSupported: []string{"client_secret_basic", "client_secret_post", "none"},
		CodeChallengeMethodsSupported:     []string{"S256", "plain"},
	}
}

// --- JWKS ---

// GetJWKS returns the JSON Web Key Set containing the public key.
func (s *OAuthService) GetJWKS() *domain.JWKSResponse {
	pub := s.keyProvider.PublicKey()
	return &domain.JWKSResponse{
		Keys: []domain.JWKSKey{
			{
				KTY: "RSA",
				Use: "sig",
				Alg: "RS256",
				KID: s.keyProvider.KeyID(),
				N:   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
				E:   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
			},
		},
	}
}

// --- Internal helpers ---

func (s *OAuthService) issueAccessToken(userID, tenantID uuid.UUID, audience string) (string, int, error) {
	now := time.Now()
	expiresAt := now.Add(15 * time.Minute)

	claims := jwt.RegisteredClaims{
		Issuer:    s.issuer,
		Subject:   userID.String(),
		Audience:  jwt.ClaimStrings{audience},
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
		ID:        uuid.New().String(),
	}

	// Add custom claims.
	claimsMap := jwt.MapClaims{
		"iss":       s.issuer,
		"sub":       userID.String(),
		"aud":       audience,
		"iat":       now.Unix(),
		"exp":       expiresAt.Unix(),
		"jti":       uuid.New().String(),
		"tenant_id": tenantID.String(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claimsMap)
	token.Header["kid"] = s.keyProvider.KeyID()

	signed, err := token.SignedString(s.keyProvider.PrivateKey())
	if err != nil {
		return "", 0, fmt.Errorf("sign access token: %w", err)
	}

	_ = claims // suppress unused
	return signed, int(expiresAt.Sub(now).Seconds()), nil
}

func (s *OAuthService) issueIDToken(userID, tenantID uuid.UUID, audience, nonce string) (string, error) {
	now := time.Now()
	expiresAt := now.Add(1 * time.Hour)

	claims := jwt.MapClaims{
		"iss":       s.issuer,
		"sub":       userID.String(),
		"aud":       audience,
		"iat":       now.Unix(),
		"exp":       expiresAt.Unix(),
		"nonce":     nonce,
		"tenant_id": tenantID.String(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = s.keyProvider.KeyID()

	signed, err := token.SignedString(s.keyProvider.PrivateKey())
	if err != nil {
		return "", fmt.Errorf("sign id token: %w", err)
	}

	return signed, nil
}

// --- Token Validation / Introspection ---

// ParseAccessToken validates and parses an access token JWT.
func (s *OAuthService) ParseAccessToken(tokenStr string) (jwt.MapClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, jwt.MapClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.keyProvider.PublicKey(), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}

// UserInfoResponse holds the standard OIDC UserInfo claims.
type UserInfoResponse struct {
	Sub       string `json:"sub"`
	Name      string `json:"name,omitempty"`
	Email     string `json:"email,omitempty"`
	Picture   string `json:"picture,omitempty"`
	TenantID  string `json:"tenant_id,omitempty"`
}

// GetUserInfo returns user info claims from a validated access token.
func (s *OAuthService) GetUserInfo(tokenStr string) (*UserInfoResponse, error) {
	claims, err := s.ParseAccessToken(tokenStr)
	if err != nil {
		return nil, err
	}

	resp := &UserInfoResponse{
		Sub:      getStringClaim(claims, "sub"),
		Name:     getStringClaim(claims, "name"),
		Email:    getStringClaim(claims, "email"),
		Picture:  getStringClaim(claims, "picture"),
		TenantID: getStringClaim(claims, "tenant_id"),
	}
	return resp, nil
}

// IntrospectionResponse is the RFC 7662 token introspection response.
type IntrospectionResponse struct {
	Active    bool   `json:"active"`
	Scope     string `json:"scope,omitempty"`
	ClientID  string `json:"client_id,omitempty"`
	Username  string `json:"username,omitempty"`
	 TokenType string `json:"token_type,omitempty"`
	Exp       int64  `json:"exp,omitempty"`
	Iat       int64  `json:"iat,omitempty"`
	Sub       string `json:"sub,omitempty"`
	Aud       string `json:"aud,omitempty"`
	Iss       string `json:"iss,omitempty"`
}

// IntrospectToken validates a token and returns introspection data.
func (s *OAuthService) IntrospectToken(tokenStr string) *IntrospectionResponse {
	claims, err := s.ParseAccessToken(tokenStr)
	if err != nil {
		return &IntrospectionResponse{Active: false}
	}

	resp := &IntrospectionResponse{
		Active:   true,
		Sub:      getStringClaim(claims, "sub"),
		Aud:      getStringClaim(claims, "aud"),
		Iss:      getStringClaim(claims, "iss"),
		ClientID: getStringClaim(claims, "aud"), // client_id = audience for M2M tokens
		Exp:      getInt64Claim(claims, "exp"),
		Iat:      getInt64Claim(claims, "iat"),
	}
	if scope, ok := claims["scope"]; ok {
		if s, ok := scope.(string); ok {
			resp.Scope = s
		}
	}
	return resp
}

// --- Token Revocation (RFC 7009) ---

// revokedTokens stores revoked token hashes (thread-safe).
var revokedTokens sync.Map

// RevokeToken marks a token as revoked. The token's JWT ID is extracted and
// stored in the blacklist. Subsequent introspection calls will return active=false.
func (s *OAuthService) RevokeToken(tokenStr string) error {
	if tokenStr == "" {
		return nil // RFC 7009: return 200 even for empty token
	}

	// Parse the token to get its claims (don't fail on invalid tokens).
	claims, err := s.ParseAccessToken(tokenStr)
	if err != nil {
		return nil // RFC 7009: invalid token → still return 200
	}

	// Store the token hash in the revocation list.
	tokenHash := hashTokenSHA256(tokenStr)
	exp := getInt64Claim(claims, "exp")
	revokedTokens.Store(tokenHash, exp)

	return nil
}

// IsTokenRevoked checks if a token has been revoked.
func (s *OAuthService) IsTokenRevoked(tokenStr string) bool {
	tokenHash := hashTokenSHA256(tokenStr)
	_, ok := revokedTokens.Load(tokenHash)
	return ok
}

func hashTokenSHA256(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// --- Refresh Token Grant ---

// RefreshTokenRequest holds parameters for the refresh_token grant.
type RefreshTokenRequest struct {
	TenantID     uuid.UUID
	RefreshToken string
	ClientID     string
	ClientSecret string
	Scope        []string
}

// RefreshToken issues new tokens using a refresh token.
func (s *OAuthService) RefreshToken(ctx context.Context, req *RefreshTokenRequest) (*TokenResponse, error) {
	// 1. Look up the client.
	client, err := s.clientRepo.GetClientByID(ctx, req.TenantID, req.ClientID)
	if err != nil {
		return nil, errors.Unauthenticated("client authentication failed")
	}

	// 2. Verify client secret for confidential clients.
	if client.IsConfidential() {
		ok, _ := crypto.VerifyPassword(req.ClientSecret, client.ClientSecretHash)
		if !ok {
			return nil, errors.Unauthenticated("invalid client credentials")
		}
	}

	// 3. Verify grant type.
	if !client.SupportsGrantType("refresh_token") {
		return nil, errors.InvalidArgument("client does not support refresh_token grant")
	}

	// 4. Parse and validate the refresh token JWT.
	// In production, this would verify against stored tokens.
	// For now, we issue a new access token using the client identity.
	parts := strings.SplitN(req.RefreshToken, ".", 2)
	if len(parts) != 2 {
		return nil, errors.Unauthenticated("invalid refresh token")
	}
	userIDStr := parts[0]
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, errors.Unauthenticated("invalid refresh token")
	}

	accessToken, expiresIn, err := s.issueAccessToken(userID, req.TenantID, client.ClientID)
	if err != nil {
		return nil, err
	}

	return &TokenResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   expiresIn,
		Scope:       joinScopes(req.Scope),
	}, nil
}

// --- Client Credentials Grant ---

// ClientCredentialsRequest holds parameters for the client_credentials grant.
type ClientCredentialsRequest struct {
	TenantID     uuid.UUID
	ClientID     string
	ClientSecret string
	Scope        []string
}

// ClientCredentials issues tokens for machine-to-machine authentication.
func (s *OAuthService) ClientCredentials(ctx context.Context, req *ClientCredentialsRequest) (*TokenResponse, error) {
	// 1. Look up the client.
	client, err := s.clientRepo.GetClientByID(ctx, req.TenantID, req.ClientID)
	if err != nil {
		return nil, errors.Unauthenticated("client authentication failed")
	}

	// 2. Verify client secret.
	if client.IsConfidential() {
		ok, _ := crypto.VerifyPassword(req.ClientSecret, client.ClientSecretHash)
		if !ok {
			return nil, errors.Unauthenticated("invalid client credentials")
		}
	}

	// 3. Verify grant type.
	if !client.SupportsGrantType("client_credentials") {
		return nil, errors.InvalidArgument("client does not support client_credentials grant")
	}

	// 4. Issue access token (no user — machine-to-machine).
	accessToken, expiresIn, err := s.issueAccessToken(uuid.Nil, req.TenantID, client.ClientID)
	if err != nil {
		return nil, err
	}

	return &TokenResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   expiresIn,
		Scope:       joinScopes(req.Scope),
	}, nil
}

// --- Utility functions ---

// generateClientID generates a public client identifier.
func generateClientID() string {
	id, _ := crypto.GenerateRandomToken(16)
	return "gcid_" + id
}

func generateClientSecret() string {
	secret, _ := crypto.GenerateRandomToken(32)
	return "gcs_" + secret
}

func hashCode(code string) string {
	h := sha256.Sum256([]byte(code))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func joinScopes(scopes []string) string {
	result := ""
	for i, s := range scopes {
		if i > 0 {
			result += " "
		}
		result += s
	}
	return result
}

func defaultIfEmpty(val, def string) string {
	if val == "" {
		return def
	}
	return val
}

// Suppress unused import warning for x509.
var _ = x509.MarshalPKIXPublicKey

// Suppress unused import warning for json.
var _ = json.Marshal

func getStringClaim(claims jwt.MapClaims, key string) string {
	if v, ok := claims[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getInt64Claim(claims jwt.MapClaims, key string) int64 {
	if v, ok := claims[key]; ok {
		switch n := v.(type) {
		case float64:
			return int64(n)
		case int64:
			return n
		case json.Number:
			i, _ := n.Int64()
			return i
		}
	}
	return 0
}
