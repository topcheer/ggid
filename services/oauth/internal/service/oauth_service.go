// Package service implements the OAuth2/OIDC business logic.
package service

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
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

// --- Utility functions ---

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
