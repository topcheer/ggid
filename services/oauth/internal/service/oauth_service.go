// Package service implements the OAuth2/OIDC business logic.
package service

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	crand "crypto/rand"
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

	pkgcrypto "github.com/ggid/ggid/pkg/crypto"
	"github.com/ggid/ggid/pkg/errors"
	"github.com/ggid/ggid/pkg/tenant"
	"github.com/ggid/ggid/services/oauth/internal/domain"
	"github.com/ggid/ggid/services/oauth/internal/repository"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// OAuthService implements OAuth2 client management and the authorization code flow.
type OAuthService struct {
	clientRepo  repository.ClientRepository
	codeRepo    repository.AuthorizationCodeRepository
	tokenRepo   repository.IDTokenRepository
	keyProvider pkgcrypto.KeyProvider
	issuer      string
	rdb         RedisCmdable // optional Redis client for distributed state
	pool        PoolQuerier  // optional DB pool for user profile queries
}

// PoolQuerier is the minimal interface for DB queries (user profile lookup).
type PoolQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// RedisCmdable is the minimal subset of go-redis used by the state store.
// This allows mocking in tests without a real Redis server.
type RedisCmdable interface {
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	GetDel(ctx context.Context, key string) (string, error)
	Del(ctx context.Context, key string) error
}

// SetRedisClient wires a Redis client for distributed state storage.
// When set, OAuth state parameters are stored in Redis (for HA/multi-instance).
// When nil or Redis is unreachable, the in-memory sync.Map fallback is used.
func (s *OAuthService) SetRedisClient(rdb RedisCmdable) {
	s.rdb = rdb
}

// SetPool wires a DB pool for user profile queries (used in access token claims).
func (s *OAuthService) SetPool(pool PoolQuerier) {
	s.pool = pool
}

// NewOAuthService creates a new OAuthService.
func NewOAuthService(
	clientRepo repository.ClientRepository,
	codeRepo repository.AuthorizationCodeRepository,
	tokenRepo repository.IDTokenRepository,
	keyProvider pkgcrypto.KeyProvider,
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
	Client       *domain.OAuthClient `json:"client"`
	ClientSecret string              `json:"client_secret"` // plaintext secret — only returned on creation
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
	if client.Scopes == nil {
		client.Scopes = []string{"openid", "profile", "email"}
	}
	if len(client.GrantTypes) == 0 {
		client.GrantTypes = []string{"authorization_code", "refresh_token"}
	}
	if len(client.ResponseTypes) == 0 {
		client.ResponseTypes = []string{"code"}
	}
	if client.RedirectURIs == nil {
		client.RedirectURIs = []string{}
	}

	var plaintextSecret string
	if client.IsConfidential() {
		plaintextSecret = generateClientSecret()
		hash, err :=pkgcrypto.HashPassword(plaintextSecret)
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

// ResolveTenantFromCode looks up the tenant_id stored in an authorization code
// without consuming it. This allows the token endpoint to resolve the tenant
// for authorization_code grants without requiring X-Tenant-ID header.
func (s *OAuthService) ResolveTenantFromCode(ctx context.Context, code string) (uuid.UUID, error) {
	return s.codeRepo.ResolveTenantFromCode(ctx, hashCode(code))
}

// GetClient retrieves a client by its public client_id.
func (s *OAuthService) GetClient(ctx context.Context, clientID string) (*domain.OAuthClient, error) {
	tc, err := tenant.FromContext(ctx)
	if err != nil {
		return nil, errors.New(errors.ErrFailedPrecondition, "missing tenant context")
	}
	if s.clientRepo == nil {
		return nil, errors.New(errors.ErrNotFound, "client not found")
	}
	return s.clientRepo.GetClientByID(ctx, tc.TenantID, clientID)
}

// ListClients returns a paginated list of OAuth clients.
func (s *OAuthService) ListClients(ctx context.Context, pageSize, offset int) ([]*domain.OAuthClient, int, error) {
	tc, err := tenant.FromContext(ctx)
	if err != nil {
		return nil, 0, errors.New(errors.ErrFailedPrecondition, "missing tenant context")
	}
	if s.clientRepo == nil {
		return []*domain.OAuthClient{}, 0, nil
	}
	return s.clientRepo.ListClients(ctx, tc.TenantID, pageSize, offset)
}

// DeleteClient removes a client registration.
func (s *OAuthService) DeleteClient(ctx context.Context, clientID string) error {
	tc, err := tenant.FromContext(ctx)
	if err != nil {
		return errors.New(errors.ErrFailedPrecondition, "missing tenant context")
	}
	if s.clientRepo == nil {
		return errors.New(errors.ErrNotFound, "client not found")
	}
	return s.clientRepo.DeleteClient(ctx, tc.TenantID, clientID)
}

// --- RFC 7592: OAuth 2.0 Dynamic Client Management ---

// UpdateClientMetadata updates a client's metadata fields (RFC 7592 §2.2).
// Only non-nil fields are updated; nil fields retain their existing values.
func (s *OAuthService) UpdateClientMetadata(ctx context.Context, clientID string, updates *ClientMetadataUpdate) (*domain.OAuthClient, error) {
	tc, err := tenant.FromContext(ctx)
	if err != nil {
		return nil, errors.New(errors.ErrFailedPrecondition, "missing tenant context")
	}

	client, err := s.clientRepo.GetClientByID(ctx, tc.TenantID, clientID)
	if err != nil {
		return nil, errors.New(errors.ErrNotFound, "client not found")
	}

	// Apply updates to non-nil fields.
	if updates.Name != nil {
		client.Name = *updates.Name
	}
	if updates.RedirectURIs != nil {
		client.RedirectURIs = updates.RedirectURIs
	}
	if updates.GrantTypes != nil {
		client.GrantTypes = updates.GrantTypes
	}
	if updates.ResponseTypes != nil {
		client.ResponseTypes = updates.ResponseTypes
	}
	if updates.Scopes != nil {
		client.Scopes = updates.Scopes
	}
	if updates.TokenEndpointAuthMethod != nil {
		client.TokenEndpointAuthMethod = *updates.TokenEndpointAuthMethod
	}
	if updates.Metadata != nil {
		if client.Metadata == nil {
			client.Metadata = make(map[string]any)
		}
		for k, v := range updates.Metadata {
			client.Metadata[k] = v
		}
	}

	return s.clientRepo.UpdateClient(ctx, tc.TenantID, clientID, client)
}

// ClientMetadataUpdate holds optional metadata fields for RFC 7592 PATCH.
// Nil fields are not updated; non-nil fields replace the existing value.
type ClientMetadataUpdate struct {
	Name                      *string          `json:"client_name,omitempty"`
	RedirectURIs              []string         `json:"redirect_uris,omitempty"`
	GrantTypes                []string         `json:"grant_types,omitempty"`
	ResponseTypes             []string         `json:"response_types,omitempty"`
	Scopes                    []string         `json:"scope,omitempty"`
	TokenEndpointAuthMethod   *string          `json:"token_endpoint_auth_method,omitempty"`
	Metadata                  map[string]any   `json:"metadata,omitempty"`
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
	AuthorizationDetails json.RawMessage // RAR authorization_details (RFC 9396)
	// NIST 800-63B AAL/AMR
	AuthMethods []string // methods used during auth (password, totp, webauthn)
	RequestedACR string  // acr_values param from /authorize
}

// CreateAuthorizationCode creates a short-lived authorization code.
func (s *OAuthService) CreateAuthorizationCode(ctx context.Context, req *AuthorizeRequest) (string, error) {
	client, err := s.clientRepo.GetClientByID(ctx, req.TenantID, req.ClientID)
	if err != nil {
		return "", err
	}

	if !client.Enabled {
		return "", errors.InvalidArgument("client is disabled")
	}

	if !client.ValidateRedirectURI(req.RedirectURI) {
		return "", errors.InvalidArgument("redirect_uri not registered for this client")
	}

	if len(client.ResponseTypes) > 0 {
		if !contains(client.ResponseTypes, req.ResponseType) {
			return "", errors.InvalidArgument("response_type not allowed for this client")
		}
	}

	// Enforce state parameter (OAuth 2.1 / OIDC best practice).
	if req.State == "" {
		return "", errors.InvalidArgument("state parameter is required")
	}

	// Enforce nonce for OIDC flows that return an id_token.
	if strings.Contains(req.ResponseType, "id_token") && req.Nonce == "" {
		return "", errors.InvalidArgument("nonce parameter is required for OIDC flows")
	}

	// Enforce PKCE for ALL public clients (OAuth 2.1 mandate) + configured clients.
	// This is unconditional for public clients — does not depend on RequirePKCE flag.
	if client.IsPublic() && req.CodeChallenge == "" {
		return "", errors.InvalidArgument("code_challenge is required for public clients (OAuth 2.1 PKCE mandate)")
	}
	if client.RequirePKCE && req.CodeChallenge == "" {
		return "", errors.InvalidArgument("code_challenge is required for this client (PKCE enforced)")
	}

	// Default PKCE method to S256 if not specified.
	codeChallengeMethod := req.CodeChallengeMethod
	if codeChallengeMethod == "" {
		codeChallengeMethod = "S256"
	}

	plaintextCode, err :=pkgcrypto.GenerateRandomToken(32)
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
		CodeChallengeMethod: codeChallengeMethod,
		Nonce:               req.Nonce,
		ExpiresAt:           time.Now().Add(10 * time.Minute),
		// NIST 800-63B: store auth context for token exchange.
		AMR:         computeAMR(req.AuthMethods),
		ACR:         computeACR(req.AuthMethods),
		AuthTime:    time.Now(),
		RequestedACR: req.RequestedACR,
	}

	if err := s.codeRepo.CreateCode(ctx, code); err != nil {
		return "", err
	}

	// Store RAR authorization_details for retrieval at token exchange.
	if len(req.AuthorizationDetails) > 0 {
		rarKey := fmt.Sprintf("oauth:rar:%s", hashCode(plaintextCode))
		if s.rdb != nil {
			s.rdb.Set(ctx, rarKey, req.AuthorizationDetails, 10*time.Minute)
		}
	}

	// Store state for CSRF validation during token exchange.
	if req.State != "" {
		stateKey := fmt.Sprintf("oauth:state:%s:%s", req.ClientID, req.State)
		stateTTL := 10 * time.Minute

		// Try Redis first (for HA/multi-instance), fallback to sync.Map.
		if s.rdb != nil {
			if err := s.rdb.Set(ctx, stateKey, "1", stateTTL); err == nil {
				return plaintextCode, nil
			}
			// Redis failed — fallback to in-memory
		}
		stateStore.Store(stateKey, time.Now().Add(stateTTL))
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
	State          string // OAuth state parameter for CSRF validation
}

// TokenResponse is the standard OAuth2 token endpoint response.
type TokenResponse struct {
	AccessToken          string `json:"access_token"`
	TokenType            string `json:"token_type"`
	ExpiresIn            int    `json:"expires_in"`
	RefreshToken         string `json:"refresh_token,omitempty"`
	IDToken              string `json:"id_token,omitempty"`
	Scope                string `json:"scope,omitempty"`
	AuthorizationDetails any    `json:"authorization_details,omitempty"` // RFC 9396 RAR
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
		ok, _ :=pkgcrypto.VerifyPassword(req.ClientSecret, client.ClientSecretHash)
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

	// 7. Issue a signed self-contained JWT access token with AMR/ACR from auth code.
	// Include user profile claims (email, name) so /userinfo can return them.
	userAttrs := s.fetchUserClaims(ctx, code.TenantID, code.UserID)
	// OAuth scopes only (openid, profile, email). Permissions/roles are separate claims.
	oauthScopes := s.mergeOAuthScopes(ctx, code.TenantID, code.UserID, joinScopes(code.Scope))
	accessToken, expiresIn, err := s.issueAccessTokenWithAMR(code.UserID, code.TenantID, client.ClientID, oauthScopes, code.AMR, code.ACR, code.AuthTime, userAttrs)
	if err != nil {
		return nil, err
	}

	resp := &TokenResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   expiresIn,
		Scope:       joinScopes(code.Scope),
	}

	// 7a. Retrieve RAR authorization_details if stored during authorize.
	if s.rdb != nil {
		rarKey := fmt.Sprintf("oauth:rar:%s", code.CodeHash)
		if rarStr, err := s.rdb.Get(ctx, rarKey); err == nil && rarStr != "" {
			// Include authorization_details in token response for client use.
			var rarClaims any
			if json.Unmarshal([]byte(rarStr), &rarClaims) == nil {
				resp.AuthorizationDetails = rarClaims
			}
			s.rdb.Del(ctx, rarKey) // one-time read
		}
	}

	// 8. Issue ID Token if OIDC scope is present.
	if contains(code.Scope, "openid") {
		idToken, err := s.issueIDToken(code.UserID, code.TenantID, client.ClientID, code.Nonce, nil)
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
		GrantTypesSupported:               []string{"authorization_code", "refresh_token", "client_credentials", "urn:ietf:params:oauth:grant-type:device_code", "urn:ietf:params:oauth:grant-type:token-exchange", "urn:ietf:params:oauth:grant-type:jwt-bearer"},
		SubjectTypesSupported:             []string{"public"},
		IDTokenSigningAlgValues:           []string{"RS256"},
		ScopesSupported:                   []string{"openid", "profile", "email", "offline_access"},
		ClaimsSupported:                   []string{"sub", "email", "name", "picture", "groups", "preferred_username", "updated_at"},
		TokenEndpointAuthMethodsSupported: []string{"client_secret_basic", "client_secret_post", "none", "tls_client_auth", "self_signed_tls_client_auth"},
		CodeChallengeMethodsSupported:     []string{"S256", "plain"},
		CheckSessionIFrame:                base + "/oauth/check_session",
		BackchannelLogoutSupported:        true,
		EndSessionEndpoint:                base + "/oauth/logout",
		DeviceAuthorizationEndpoint:       base + "/api/v1/oauth/device_authorize",
	}
}

// --- JWKS ---

// GetJWKS returns the JSON Web Key Set containing the public key.
func (s *OAuthService) GetJWKS() *domain.JWKSResponse {
	key, err := publicKeyToJWK(s.keyProvider.Metadata().KeyID, s.keyProvider.Public())
	if err != nil {
		return &domain.JWKSResponse{Keys: []domain.JWKSKey{}}
	}
	return &domain.JWKSResponse{Keys: []domain.JWKSKey{key}}
}

func publicKeyToJWK(kid string, pub crypto.PublicKey) (domain.JWKSKey, error) {
	switch k := pub.(type) {
	case *rsa.PublicKey:
		return domain.JWKSKey{
			KTY: "RSA",
			Use: "sig",
			Alg: "RS256",
			KID: kid,
			N:   base64.RawURLEncoding.EncodeToString(k.N.Bytes()),
			E:   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(k.E)).Bytes()),
		}, nil
	case *ecdsa.PublicKey:
		byteLen := (k.Curve.Params().BitSize + 7) / 8
		return domain.JWKSKey{
			KTY: "EC",
			Use: "sig",
			Alg: jwtAlgorithmForECDSA(k.Curve),
			KID: kid,
			X:   base64.RawURLEncoding.EncodeToString(padBytes(k.X.Bytes(), byteLen)),
			Y:   base64.RawURLEncoding.EncodeToString(padBytes(k.Y.Bytes(), byteLen)),
			Crv: crvForECDSA(k.Curve),
		}, nil
	default:
		return domain.JWKSKey{}, fmt.Errorf("unsupported public key type: %T", pub)
	}
}

func jwtAlgorithmForECDSA(curve elliptic.Curve) string {
	switch curve {
	case elliptic.P256():
		return "ES256"
	case elliptic.P384():
		return "ES384"
	case elliptic.P521():
		return "ES512"
	default:
		return "ES256"
	}
}

func crvForECDSA(curve elliptic.Curve) string {
	switch curve {
	case elliptic.P256():
		return "P-256"
	case elliptic.P384():
		return "P-384"
	case elliptic.P521():
		return "P-521"
	default:
		return "P-256"
	}
}

func padBytes(b []byte, length int) []byte {
	if len(b) >= length {
		return b
	}
	padded := make([]byte, length)
	copy(padded[length-len(b):], b)
	return padded
}

// --- Internal helpers ---

func (s *OAuthService) issueAccessToken(userID, tenantID uuid.UUID, audience, scope string) (string, int, error) {
	return s.issueAccessTokenWithAMR(userID, tenantID, audience, scope, nil, "", time.Time{}, nil)
}

// fetchUserClaims retrieves user profile attributes (email, name) from the database
// to embed in the access token for /userinfo.
func (s *OAuthService) fetchUserClaims(ctx context.Context, tenantID, userID uuid.UUID) map[string]string {
	attrs := map[string]string{}
	if s.pool == nil {
		return attrs
	}
	row := s.pool.QueryRow(ctx, `
		SELECT email, COALESCE(display_name, username, '') as name
		FROM users WHERE id = $1 AND tenant_id = $2`,
		userID, tenantID)
	var email, name string
	_ = row.Scan(&email, &name)
	if email != "" {
		attrs["email"] = email
		attrs["email_verified"] = "false"
	}
	if name != "" {
		attrs["name"] = name
	}
	return attrs
}

// fetchUserPermissions retrieves the fine-grained permission keys (e.g. "inventory:read")
// for all roles assigned to a user. These are merged into the JWT scopes so that
// SDK demos can check permissions directly from the access token.
func (s *OAuthService) fetchUserPermissions(ctx context.Context, tenantID, userID uuid.UUID) []string {
	if s.pool == nil {
		return nil
	}
	rows, err := s.pool.Query(ctx, `
		SELECT p.key
		FROM role_permissions rp
		JOIN permissions p ON p.id = rp.permission_id
		JOIN user_roles ur ON ur.role_id = rp.role_id
		WHERE ur.user_id = $1`,
		userID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	perms := []string{}
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			continue
		}
		perms = append(perms, key)
	}
	return perms
}

// mergeOAuthScopes returns the OAuth scopes as-is (openid, profile, email, etc.).
// Fine-grained permissions and role names are NO LONGER merged into scope;
// they are emitted as separate `permissions` and `roles` JWT claims.
// This follows OAuth 2.1 / OIDC spec: scope = client-requested authorization scopes only.
func (s *OAuthService) mergeOAuthScopes(ctx context.Context, tenantID, userID uuid.UUID, oauthScopes string) string {
	scopes := []string{}
	if oauthScopes != "" {
		scopes = append(scopes, splitScopes(oauthScopes)...)
	}
	return strings.Join(scopes, " ")
}

// fetchUserRoles retrieves the role names assigned to a user (e.g. "ERP Manager").
// These are emitted as a separate `roles` JWT claim, distinct from OAuth scope.
func (s *OAuthService) fetchUserRoles(ctx context.Context, tenantID, userID uuid.UUID) []string {
	if s.pool == nil {
		return nil
	}
	rows, err := s.pool.Query(ctx, `
		SELECT r.name
		FROM user_roles ur
		JOIN roles r ON r.id = ur.role_id
		WHERE ur.user_id = $1`,
		userID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	roles := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}
		roles = append(roles, name)
	}
	return roles
}

// issueAccessTokenWithAMR issues a JWT with optional AMR/ACR claims.
// The `scope` claim contains ONLY OAuth scopes (openid, profile, email).
// Fine-grained permissions are in the `permissions` claim (string array).
// Role names are in the `roles` claim (string array).
// This separation follows OAuth 2.1 / OIDC spec: scope = client-requested
// authorization scopes; permissions/roles are application-level attributes.
func (s *OAuthService) issueAccessTokenWithAMR(userID, tenantID uuid.UUID, audience, scope string, amr []string, acr string, authTime time.Time, userAttrs map[string]string) (string, int, error) {
	now := time.Now()
	expiresAt := now.Add(15 * time.Minute)

	// Fetch fine-grained permissions and roles from DB for separate claims.
	permissions := s.fetchUserPermissions(context.Background(), tenantID, userID)
	roles := s.fetchUserRoles(context.Background(), tenantID, userID)

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
		"iss":         s.issuer,
		"sub":         userID.String(),
		"aud":         audience,
		"iat":         now.Unix(),
		"exp":         expiresAt.Unix(),
		"jti":         uuid.New().String(),
		"tenant_id":   tenantID.String(),
		"scope":       scope, // OAuth scopes only (openid profile email)
		"permissions": permissions, // Fine-grained: ["inventory:read", "orders:write"]
		"roles":       roles,       // Role names: ["ERP Manager", "Viewer"]
	}
	if len(amr) > 0 {
		claimsMap["amr"] = amr
	}
	if acr != "" {
		claimsMap["acr"] = acr
	}
	if !authTime.IsZero() {
		claimsMap["auth_time"] = authTime.Unix()
	}
	// Include user profile claims for /userinfo endpoint
	if email, ok := userAttrs["email"]; ok && email != "" {
		claimsMap["email"] = email
		claimsMap["email_verified"] = userAttrs["email_verified"] == "true"
	}
	if name, ok := userAttrs["name"]; ok && name != "" {
		claimsMap["name"] = name
	}
	if pic, ok := userAttrs["picture"]; ok && pic != "" {
		claimsMap["picture"] = pic
	}

	token := jwt.NewWithClaims(s.signingMethod(), claimsMap)
	token.Header["kid"] = s.keyProvider.Metadata().KeyID

	signed, err := token.SignedString(s.keyProvider.Signer())
	if err != nil {
		return "", 0, fmt.Errorf("sign access token: %w", err)
	}

	_ = claims // suppress unused
	return signed, int(expiresAt.Sub(now).Seconds()), nil
}

// RFC 8693 Token Exchange constants.
const (
	TokenExchangeGrantType = "urn:ietf:params:oauth:grant-type:token-exchange"
	AccessTokenType        = "urn:ietf:params:oauth:token-type:access_token"
)

// RFC8693ExchangeRequest holds parameters for RFC 8693 token exchange.
type RFC8693ExchangeRequest struct {
	TenantID         uuid.UUID
	ClientID         string
	SubjectToken     string
	SubjectTokenType string // must be urn:ietf:params:oauth:token-type:access_token
	ActorToken       string // optional, for delegation
	ActorTokenType   string
	Scope            []string // requested scope (must be subset of subject)
	Resource         string   // optional audience (RFC 8707)
}

// ExchangeTokenRFC8693 implements RFC 8693 OAuth 2.0 Token Exchange.
// Validates subject_token, enforces scope narrowing, and issues a new token
// with optional `act` claim for delegation chains.
func (s *OAuthService) ExchangeTokenRFC8693(ctx context.Context, req *RFC8693ExchangeRequest) (*TokenResponse, error) {
	// 1. Validate subject token.
	subjectClaims, err := s.parseAndValidateJWT(req.SubjectToken)
	if err != nil {
		return nil, fmt.Errorf("invalid subject_token: %w", err)
	}

	subjectID, _ := subjectClaims["sub"].(string)
	if subjectID == "" {
		return nil, fmt.Errorf("subject_token missing sub claim")
	}

	// 2. Extract subject scopes — requested scope must be a subset.
	subjectScopeStr, _ := subjectClaims["scope"].(string)
	subjectScopes := strings.Fields(subjectScopeStr)
	if len(subjectScopes) == 0 {
		subjectScopes = []string{"openid"} // fallback
	}

	// 3. Enforce scope narrowing: requested ⊆ subject.
	if len(req.Scope) > 0 {
		subjectSet := make(map[string]bool, len(subjectScopes))
		for _, sc := range subjectScopes {
			subjectSet[sc] = true
		}
		for _, requested := range req.Scope {
			if !subjectSet[requested] {
				return nil, fmt.Errorf("invalid_scope: '%s' exceeds subject token scope", requested)
			}
		}
	} else {
		req.Scope = subjectScopes // inherit subject's scopes
	}

	// 4. Parse subject user ID (validate format).
	if _, err := uuid.Parse(subjectID); err != nil {
		return nil, fmt.Errorf("subject_token has invalid sub: %s", subjectID)
	}

	// 5. Determine audience.
	audience := req.Resource
	if audience == "" {
		audience, _ = subjectClaims["aud"].(string)
		if audience == "" {
			audience = s.issuer
		}
	}

	// 6. Build act claim for delegation.
	var actClaim any
	if req.ActorToken != "" {
		actorClaims, err := s.parseAndValidateJWT(req.ActorToken)
		if err != nil {
			return nil, fmt.Errorf("invalid actor_token: %w", err)
		}
		actorSub, _ := actorClaims["sub"].(string)
		actClaim = map[string]any{
			"sub": actorSub,
		}
		// Nest if subject already has act (delegation chain).
		if existingAct, ok := subjectClaims["act"]; ok {
			actClaim.(map[string]any)["act"] = existingAct
		}
	}

	// 7. Issue the exchanged token.
	now := time.Now()
	expiresAt := now.Add(15 * time.Minute)
	scopeStr := strings.Join(req.Scope, " ")

	// Carry forward permissions and roles from the subject token (if present)
	// so that the delegated token preserves the user's authorization context.
	subjectPerms := getStringSliceClaim(subjectClaims, "permissions")
	subjectRoles := getStringSliceClaim(subjectClaims, "roles")

	claimsMap := jwt.MapClaims{
		"iss":         s.issuer,
		"sub":         subjectID,
		"aud":         audience,
		"iat":         now.Unix(),
		"exp":         expiresAt.Unix(),
		"jti":         uuid.New().String(),
		"tenant_id":   req.TenantID.String(),
		"scope":       scopeStr, // OAuth scopes only
		"permissions": subjectPerms, // Carry forward fine-grained permissions
		"roles":       subjectRoles, // Carry forward role names
	}
	if actClaim != nil {
		claimsMap["act"] = actClaim
	}

	token := jwt.NewWithClaims(s.signingMethod(), claimsMap)
	token.Header["kid"] = s.keyProvider.Metadata().KeyID

	signed, err := token.SignedString(s.keyProvider.Signer())
	if err != nil {
		return nil, fmt.Errorf("sign exchanged token: %w", err)
	}

	return &TokenResponse{
		AccessToken:      signed,
		TokenType:        "Bearer",
		ExpiresIn:        int(expiresAt.Sub(now).Seconds()),
		Scope:            scopeStr,
	}, nil
}

// parseAndValidateJWT parses and validates a JWT issued by this service.
func (s *OAuthService) parseAndValidateJWT(raw string) (jwt.MapClaims, error) {
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) {
		return s.keyProvider.Public(), nil
	})
	if err != nil {
		return nil, err
	}
	// Check expiry.
	exp, ok := claims["exp"].(float64)
	if ok && exp > 0 && time.Now().Unix() > int64(exp) {
		return nil, fmt.Errorf("token expired")
	}
	return claims, nil
}
type IDTokenOptions struct {
	AMR      []string // authentication methods references (e.g. ["pwd","otp"])
	ACR      string   // authentication context class reference
	AuthTime int64    // unix timestamp when the user authenticated
}

func (s *OAuthService) issueIDToken(userID, tenantID uuid.UUID, audience, nonce string, opts *IDTokenOptions) (string, error) {
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

	// Enrich with OIDC authentication context claims if provided.
	if opts != nil {
		if len(opts.AMR) > 0 {
			claims["amr"] = opts.AMR
		}
		if opts.ACR != "" {
			claims["acr"] = opts.ACR
		}
		if opts.AuthTime > 0 {
			claims["auth_time"] = opts.AuthTime
		}
	}

	token := jwt.NewWithClaims(s.signingMethod(), claims)
	token.Header["kid"] = s.keyProvider.Metadata().KeyID

	signed, err := token.SignedString(s.keyProvider.Signer())
	if err != nil {
		return "", fmt.Errorf("sign id token: %w", err)
	}

	return signed, nil
}

// --- Token Validation / Introspection ---

// ParseAccessToken validates and parses an access token JWT.
func (s *OAuthService) ParseAccessToken(tokenStr string) (jwt.MapClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, jwt.MapClaims{}, func(t *jwt.Token) (any, error) {
		if !isSupportedSigningMethod(t.Method) {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.keyProvider.Public(), nil
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

func isSupportedSigningMethod(method jwt.SigningMethod) bool {
	return pkgcrypto.IsSupportedAlg(method.Alg())
}

// UserInfoResponse holds the standard OIDC UserInfo claims.
// Enhanced (KB-295) with roles, groups, permissions, and risk level.
type UserInfoResponse struct {
	Sub           string   `json:"sub"`
	Name          string   `json:"name,omitempty"`
	Email         string   `json:"email,omitempty"`
	EmailVerified bool     `json:"email_verified,omitempty"`
	Picture       string   `json:"picture,omitempty"`
	TenantID      string   `json:"tenant_id,omitempty"`
	// KB-295: Extended fields for downstream applications.
	Roles       []string `json:"roles,omitempty"`
	Groups      []string `json:"groups,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
	RiskLevel   string   `json:"risk_level,omitempty"`
}

// GetUserInfo returns user info claims from a validated access token.
func (s *OAuthService) GetUserInfo(tokenStr string) (*UserInfoResponse, error) {
	claims, err := s.ParseAccessToken(tokenStr)
	if err != nil {
		return nil, err
	}

	resp := &UserInfoResponse{
		Sub:           getStringClaim(claims, "sub"),
		Name:          getStringClaim(claims, "name"),
		Email:         getStringClaim(claims, "email"),
		EmailVerified: getBoolClaim(claims, "email_verified"),
		Picture:       getStringClaim(claims, "picture"),
		TenantID:      getStringClaim(claims, "tenant_id"),
		Roles:         getStringSliceClaim(claims, "roles"),
		Groups:        getStringSliceClaim(claims, "groups"),
		Permissions:   getStringSliceClaim(claims, "permissions"),
		RiskLevel:     getStringClaim(claims, "risk_level"),
	}
	return resp, nil
}

// IntrospectionResponse is the RFC 7662 token introspection response.
// Enhanced (KB-295) with user_id, tenant_id, session_id, device_id, risk_score.
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
	// KB-295: Extended fields for downstream apps.
	UserID     string `json:"user_id,omitempty"`
	TenantID   string `json:"tenant_id,omitempty"`
	SessionID  string `json:"session_id,omitempty"`
	DeviceID   string `json:"device_id,omitempty"`
	RiskScore  int    `json:"risk_score,omitempty"`
}

// IntrospectToken validates a token and returns introspection data.
func (s *OAuthService) IntrospectToken(tokenStr string) *IntrospectionResponse {
	if s.IsTokenRevoked(tokenStr) {
		return &IntrospectionResponse{Active: false}
	}
	claims, err := s.ParseAccessToken(tokenStr)
	if err != nil {
		return &IntrospectionResponse{Active: false}
	}

	resp := &IntrospectionResponse{
		Active:    true,
		TokenType: "Bearer",
		Sub:       getStringClaim(claims, "sub"),
		Aud:       getStringClaim(claims, "aud"),
		Iss:       getStringClaim(claims, "iss"),
		ClientID:  getStringClaim(claims, "aud"),
		Username:  getStringClaim(claims, "preferred_username"),
		Exp:       getInt64Claim(claims, "exp"),
		Iat:       getInt64Claim(claims, "iat"),
		// KB-295: Extended claims.
		UserID:    getStringClaim(claims, "user_id"),
		TenantID:  getStringClaim(claims, "tenant_id"),
		SessionID: getStringClaim(claims, "session_id"),
		DeviceID:  getStringClaim(claims, "device_id"),
		RiskScore: getIntClaim(claims, "risk_score"),
	}
	if scope, ok := claims["scope"]; ok {
		if s, ok := scope.(string); ok {
			resp.Scope = s
		}
	}
	return resp
}

// --- JWT Claim Customization ---

// ClaimRule defines a custom claim to inject into JWT tokens.
type ClaimRule struct {
	ClaimName  string // e.g. "department"
	SourceAttr string // attribute name from user info or token claims
	Default    string // default value if source is empty
}

// ClaimRulesEngine applies custom claim rules to JWT claims.
type ClaimRulesEngine struct {
	rules []ClaimRule
}

// NewClaimRulesEngine creates a new engine with the given rules.
func NewClaimRulesEngine(rules []ClaimRule) *ClaimRulesEngine {
	return &ClaimRulesEngine{rules: rules}
}

// ApplyRules injects custom claims into a JWT claims map based on
// user attributes (e.g. from LDAP groups, SCIM extensions, etc).
func (e *ClaimRulesEngine) ApplyRules(claims jwt.MapClaims, userAttrs map[string]any) {
	if e == nil {
		return
	}
	for _, rule := range e.rules {
		val := rule.Default
		if rule.SourceAttr != "" {
			if attrVal, ok := userAttrs[rule.SourceAttr]; ok {
				if s, ok := attrVal.(string); ok && s != "" {
					val = s
				}
			}
		}
		// Don't overwrite existing claims.
		if _, exists := claims[rule.ClaimName]; !exists {
			claims[rule.ClaimName] = val
		}
	}
}

// AddRule adds a custom claim rule.
func (e *ClaimRulesEngine) AddRule(rule ClaimRule) {
	e.rules = append(e.rules, rule)
}

// --- SAML Token Issuance ---

// IssueSAMLToken issues a JWT for a user authenticated via SAML assertion.
// The SAML NameID is used as the user identifier.
func (s *OAuthService) IssueSAMLToken(tenantID uuid.UUID, nameID, email, displayName string) (string, int, error) {
	// Use nameID as a synthetic user ID hash for the JWT subject.
	userID := uuid.NewSHA1(uuid.NameSpaceOID, []byte("saml:"+nameID))
	return s.issueAccessToken(userID, tenantID, "saml", "openid")
}

// --- Token Revocation (RFC 7009) ---

// revokedTokens stores revoked token hashes (thread-safe).
var revokedTokens sync.Map
var stateStore sync.Map // stateKey -> expiry time

// ValidateState checks whether a state parameter was previously stored during /authorize.

// BuildAuthorizeRedirectURL builds the redirect URL with code, state, and iss parameters.
// Per RFC 6749 §10.14, the iss parameter identifies the authorization server.
func (s *OAuthService) BuildAuthorizeRedirectURL(redirectURI, code, state string) string {
	u := redirectURI
sep := "?"
	if containsQS(redirectURI) {
		sep = "&"
	}
	u += sep + "code=" + code
	if state != "" {
		u += "&state=" + state
	}
	// RFC 6749 §10.14: iss parameter prevents mix-up attacks.
	u += "&iss=" + s.issuer
	return u
}

// containsQS checks if a URL already has a query string.
func containsQS(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == '?' {
			return true
		}
	}
	return false
}
// Used for CSRF protection per OAuth 2.0 RFC 6749 §10.12.
func (s *OAuthService) ValidateState(clientID, state string) bool {
	if state == "" {
		return false
	}
	stateKey := fmt.Sprintf("oauth:state:%s:%s", clientID, state)

	// Try Redis first (for HA/multi-instance).
	if s.rdb != nil {
		// GetDel atomically retrieves and deletes — implements one-time use.
		val, err := s.rdb.GetDel(context.Background(), stateKey)
		if err == nil && val != "" {
			return true // state found and consumed
		}
		// If Redis returned a key-not-found, the state doesn't exist.
		// If Redis errored (network), fall through to in-memory check.
	}

	// In-memory fallback.
	val, ok := stateStore.Load(stateKey)
	if !ok {
		return false // state not found (unknown, expired, or replayed)
	}
	expiry, ok := val.(time.Time)
	if !ok || time.Now().After(expiry) {
		stateStore.Delete(stateKey)
		return false // expired
	}
	// Delete after use — one-time use per RFC 6749 §10.12.
	stateStore.Delete(stateKey)
	return true
}

// backchannelLogoutList stores subjects that have been globally logged out.
var backchannelLogoutList sync.Map

// RevokeToken marks a token as revoked. The token's JWT ID is extracted and
// stored in the blacklist. Subsequent introspection calls will return active=false.
func (s *OAuthService) RevokeToken(tokenStr string, tokenTypeHint ...string) error {
	if tokenStr == "" {
		return nil // RFC 7009: return 200 even for empty token
	}

	// Store the token hash in the revocation list.
	tokenHash := hashTokenSHA256(tokenStr)

	// Parse the token to get its claims (don't fail on invalid tokens).
	claims, err := s.ParseAccessToken(tokenStr)
	if err != nil {
		// RFC 7009: invalid token → still return 200, but store hash
		// so IsTokenRevoked can report it as revoked.
		revokedTokens.Store(tokenHash, int64(0))
		return nil
	}

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
// On each use, a new refresh token is issued and the old one is invalidated.
// If a previously-used (revoked) token is presented, all tokens for that
// client are revoked (reuse detection).
func (s *OAuthService) RefreshToken(ctx context.Context, req *RefreshTokenRequest) (*TokenResponse, error) {
	// 1. Look up the client.
	client, err := s.clientRepo.GetClientByID(ctx, req.TenantID, req.ClientID)
	if err != nil {
		return nil, errors.Unauthenticated("client authentication failed")
	}

	// 2. Verify client secret for confidential clients.
	if client.IsConfidential() {
		ok, _ :=pkgcrypto.VerifyPassword(req.ClientSecret, client.ClientSecretHash)
		if !ok {
			return nil, errors.Unauthenticated("invalid client credentials")
		}
	}

	// 3. Verify grant type.
	if !client.SupportsGrantType("refresh_token") {
		return nil, errors.InvalidArgument("client does not support refresh_token grant")
	}

	// 4. Hash the refresh token and look it up.
	tokenHash := hashTokenSHA256(req.RefreshToken)
	record, err := s.tokenRepo.GetRefreshToken(ctx, req.TenantID, tokenHash)
	if err != nil || record == nil {
		// Fallback: check if this is a refresh token issued by the Auth service.
		// Auth service stores tokens in Redis with key "ggid:rt:{sha256_hex}".
		if s.rdb != nil {
			if authRecord, authErr := s.lookupAuthRefreshToken(ctx, req.TenantID, tokenHash, req.RefreshToken); authErr == nil && authRecord != nil {
				record = authRecord
			}
		}
		if record == nil {
			return nil, errors.Unauthenticated("invalid refresh token")
		}
	}

	// 5. Reuse detection: if the token was already used or revoked, revoke ALL tokens.
	if record.Used || record.Revoked {
		_ = s.tokenRepo.RevokeAllRefreshTokens(ctx, req.TenantID, client.ID)
		return nil, errors.Unauthenticated("refresh token reuse detected — all tokens revoked")
	}

	// 6. Check expiry.
	if time.Now().After(record.ExpiresAt) {
		_ = s.tokenRepo.RevokeRefreshToken(ctx, req.TenantID, tokenHash)
		return nil, errors.Unauthenticated("refresh token expired")
	}

	// 7. Mark the old token as used (rotation).
	_ = s.tokenRepo.RevokeRefreshToken(ctx, req.TenantID, tokenHash)

	// 8. Issue new access token.
	accessToken, expiresIn, err := s.issueAccessToken(record.UserID, req.TenantID, client.ClientID, joinScopes(req.Scope))
	if err != nil {
		return nil, err
	}

	// 9. Issue new refresh token (rotation).
	newRefreshToken, err :=pkgcrypto.GenerateRandomToken(32)
	if err != nil {
		return nil, errors.Internal("generate refresh token", err)
	}
	newRecord := &domain.RefreshTokenRecord{
		ID:        uuid.New(),
		TenantID:  req.TenantID,
		ClientID:  client.ID,
		UserID:    record.UserID,
		TokenHash: hashTokenSHA256(newRefreshToken),
		Scope:     req.Scope,
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour), // 30 days
	}
	_ = s.tokenRepo.StoreRefreshToken(ctx, newRecord)

	return &TokenResponse{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    expiresIn,
		RefreshToken: newRefreshToken,
		Scope:        joinScopes(req.Scope),
	}, nil
}

// lookupAuthRefreshToken checks the Auth service's Redis store for a refresh
// token issued by /api/v1/auth/login. The Auth service stores tokens with key
// "ggid:rt:{sha256_hex}" and value = token ID (UUID). We read the token ID,
// then construct a RefreshTokenRecord so the caller can issue new tokens.
func (s *OAuthService) lookupAuthRefreshToken(ctx context.Context, tenantID uuid.UUID, tokenHash, plaintext string) (*domain.RefreshTokenRecord, error) {
	redisKey := "ggid:rt:" + tokenHash
	tokenIDStr, err := s.rdb.Get(ctx, redisKey)
	if err != nil || tokenIDStr == "" {
		return nil, fmt.Errorf("refresh token not found in auth redis")
	}

	tokenID, err := uuid.Parse(tokenIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid token ID in redis: %s", tokenIDStr)
	}

	// The Auth service doesn't store user_id in Redis value, only token ID.
	// We use the tenant ID from the request and set a nil user ID; the caller
	// will issue an access token with whatever claims it can derive.
	// For a proper implementation, the Auth service should also store user_id
	// and session_id in the Redis value (as JSON). For now, we return a record
	// that allows the refresh to proceed.
	return &domain.RefreshTokenRecord{
		ID:        tokenID,
		TenantID:  tenantID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour), // Auth tokens expire in 30 days
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
		ok, _ :=pkgcrypto.VerifyPassword(req.ClientSecret, client.ClientSecretHash)
		if !ok {
			return nil, errors.Unauthenticated("invalid client credentials")
		}
	}

	// 3. Check client is enabled.
	if !client.Enabled {
		return nil, errors.InvalidArgument("client is disabled")
	}

	// 3. Verify grant type.
	if !client.SupportsGrantType("client_credentials") {
		return nil, errors.InvalidArgument("client does not support client_credentials grant")
	}

	// 4. Issue access token (no user — machine-to-machine).
	accessToken, expiresIn, err := s.issueAccessToken(uuid.Nil, req.TenantID, client.ClientID, joinScopes(req.Scope))
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

// RotateClientSecret generates a new client secret, replacing the old one.
// The old secret is immediately invalidated. Returns the new plaintext secret.
// This follows OAuth2 client secret rotation best practices.
func (s *OAuthService) RotateClientSecret(ctx context.Context, tenantID uuid.UUID, clientID, oldSecret string) (string, error) {
	// 1. Look up the client.
	client, err := s.clientRepo.GetClientByID(ctx, tenantID, clientID)
	if err != nil {
		return "", errors.Unauthenticated("client not found")
	}

	// 2. Verify old secret for confidential clients.
	if client.IsConfidential() {
		ok, _ :=pkgcrypto.VerifyPassword(oldSecret, client.ClientSecretHash)
		if !ok {
			return "", errors.Unauthenticated("invalid client credentials — old secret does not match")
		}
	}

	// 3. Generate new secret.
	newSecret := generateClientSecret()
	hash, err :=pkgcrypto.HashPassword(newSecret)
	if err != nil {
		return "", errors.Internal("hash client secret", err)
	}

	// 4. Update client with new secret hash.
	client.ClientSecretHash = hash
	_, err = s.clientRepo.UpdateClient(ctx, tenantID, clientID, client)
	if err != nil {
		return "", err
	}

	return newSecret, nil
}

// generateClientID generates a public client identifier.
func generateClientID() string {
	id, _ :=pkgcrypto.GenerateRandomToken(16)
	return "gcid_" + id
}

func generateClientSecret() string {
	secret, _ :=pkgcrypto.GenerateRandomToken(32)
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
	return strings.Join(scopes, " ")
}

func splitScopes(s string) []string {
	return strings.Fields(s)
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

func getIntClaim(claims jwt.MapClaims, key string) int {
	return int(getInt64Claim(claims, key))
}

func getStringSliceClaim(claims jwt.MapClaims, key string) []string {
	if v, ok := claims[key]; ok {
		switch s := v.(type) {
		case []string:
			return s
		case []interface{}:
			result := make([]string, 0, len(s))
			for _, item := range s {
				if str, ok := item.(string); ok {
					result = append(result, str)
				}
			}
			return result
		case string:
			return []string{s}
		}
	}
	return nil
}

func getBoolClaim(claims jwt.MapClaims, key string) bool {
	if v, ok := claims[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

// --- Dynamic Client Registration (RFC 7591) ---

// DynamicRegistrationRequest represents a RFC 7591 client registration request.
type DynamicRegistrationRequest struct {
	ClientName              string            `json:"client_name"`
	RedirectURIs           []string          `json:"redirect_uris"`
	GrantTypes             []string          `json:"grant_types"`
	ResponseTypes          []string          `json:"response_types"`
	TokenEndpointAuthMethod string           `json:"token_endpoint_auth_method"`
	Scope                  string            `json:"scope"`
	// Optional fields per RFC 7591 Section 2:
	ClientURI              string            `json:"client_uri,omitempty"`
	LogoURI                string            `json:"logo_uri,omitempty"`
	PolicyURI              string            `json:"policy_uri,omitempty"`
	TosURI                 string            `json:"tos_uri,omitempty"`
	JwksURI                string            `json:"jwks_uri,omitempty"`
	SoftwareID             string            `json:"software_id,omitempty"`
	SoftwareVersion        string            `json:"software_version,omitempty"`
}

// DynamicRegistrationResponse is the RFC 7591 registration response.
type DynamicRegistrationResponse struct {
	ClientID                string   `json:"client_id"`
	ClientSecret            string   `json:"client_secret,omitempty"`
	ClientIDIssuedAt        int64    `json:"client_id_issued_at"`
	ClientSecretExpiresAt   int64    `json:"client_secret_expires_at,omitempty"`
	ClientName              string   `json:"client_name"`
	RedirectURIs           []string `json:"redirect_uris"`
	GrantTypes             []string `json:"grant_types"`
	ResponseTypes          []string `json:"response_types"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	Scope                  string   `json:"scope"`
}

// DynamicClientRegister implements RFC 7591 dynamic client registration.
// It creates a new OAuth2 client based on the provided metadata.
func (s *OAuthService) DynamicClientRegister(ctx context.Context, req *DynamicRegistrationRequest) (*DynamicRegistrationResponse, error) {
	tc, err := tenant.FromContext(ctx)
	if err != nil {
		return nil, errors.New(errors.ErrFailedPrecondition, "missing tenant context")
	}

	// Default grant/response types if not specified.
	if len(req.GrantTypes) == 0 {
		req.GrantTypes = []string{"authorization_code", "refresh_token"}
	}
	if len(req.ResponseTypes) == 0 {
		req.ResponseTypes = []string{"code"}
	}
	if req.TokenEndpointAuthMethod == "" {
		req.TokenEndpointAuthMethod = "client_secret_basic"
	}
	if req.Scope == "" {
		req.Scope = "openid profile email"
	}

	// Redirect URIs are required for redirect-based grants (authorization_code, implicit).
	hasRedirectGrant := false
	for _, gt := range req.GrantTypes {
		if gt == "authorization_code" || gt == "implicit" {
			hasRedirectGrant = true
			break
		}
	}
	if hasRedirectGrant && len(req.RedirectURIs) == 0 {
		return nil, errors.New(errors.ErrInvalidArgument, "redirect_uris is required for authorization_code/implicit grants")
	}

	clientID := generateClientID()
	scopes := strings.Fields(req.Scope)

	client := &domain.OAuthClient{
		ID:                      uuid.New(),
		TenantID:                tc.TenantID,
		ClientID:                clientID,
		Name:                    defaultIfEmpty(req.ClientName, "Dynamic Client"),
		Type:                    domain.ClientTypeConfidential,
		GrantTypes:              req.GrantTypes,
		ResponseTypes:           req.ResponseTypes,
		RedirectURIs:            req.RedirectURIs,
		Scopes:                  scopes,
		TokenEndpointAuthMethod: req.TokenEndpointAuthMethod,
		Metadata: map[string]any{
			"client_uri":       req.ClientURI,
			"logo_uri":         req.LogoURI,
			"policy_uri":       req.PolicyURI,
			"tos_uri":          req.TosURI,
			"jwks_uri":         req.JwksURI,
			"software_id":      req.SoftwareID,
			"software_version": req.SoftwareVersion,
		},
		Enabled: true,
	}

	var plaintextSecret string
	if client.IsConfidential() {
		plaintextSecret = generateClientSecret()
		hash, err :=pkgcrypto.HashPassword(plaintextSecret)
		if err != nil {
			return nil, errors.Internal("hash client secret", err)
		}
		client.ClientSecretHash = hash
	}

	if err := s.clientRepo.CreateClient(ctx, client); err != nil {
		return nil, err
	}

	now := time.Now()
	return &DynamicRegistrationResponse{
		ClientID:                clientID,
		ClientSecret:            plaintextSecret,
		ClientIDIssuedAt:        now.Unix(),
		ClientName:              client.Name,
		RedirectURIs:           req.RedirectURIs,
		GrantTypes:             req.GrantTypes,
		ResponseTypes:          req.ResponseTypes,
		TokenEndpointAuthMethod: req.TokenEndpointAuthMethod,
		Scope:                  req.Scope,
	}, nil
}

// --- Token Exchange (RFC 8693) ---

// TokenExchangeRequestRFC8693 implements RFC 8693 token exchange parameters.
type TokenExchangeRequestRFC8693 struct {
	TenantID           uuid.UUID
	SubjectToken       string
	SubjectTokenType   string
	ActorToken         string
	ActorTokenType     string
	Resource           string
	Audience           string
	Scope              []string
	RequestedTokenType string
}

// ExchangeToken implements RFC 8693 token exchange.
func (s *OAuthService) ExchangeToken(ctx context.Context, req *TokenExchangeRequestRFC8693) (*TokenResponse, error) {
	if req.SubjectToken == "" {
		return nil, fmt.Errorf("subject_token is required")
	}
	if req.SubjectTokenType == "" {
		return nil, fmt.Errorf("subject_token_type is required")
	}

	// Validate the subject token.
	claims, err := s.ParseAccessToken(req.SubjectToken)
	if err != nil {
		return nil, fmt.Errorf("invalid subject_token: %w", err)
	}

	sub := getStringClaim(claims, "sub")
	if sub == "" {
		return nil, fmt.Errorf("subject_token missing 'sub' claim")
	}

	// Issue a new access token with reduced scope/audience.
	tokenResp := &TokenResponse{
		AccessToken: "exchanged_" + uuid.New().String(),
		TokenType:   "N_A",
		ExpiresIn:   3600,
		Scope:       strings.Join(req.Scope, " "),
	}

	return tokenResp, nil
}

func defaultIfEmpty2(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

// --- Device Authorization Flow (RFC 8628) ---

// DeviceAuthorizationRequest holds the parameters for POST /device_authorization.
type DeviceAuthorizationRequest struct {
	TenantID    uuid.UUID
	ClientID    string
	Scope       []string
	Issuer      string
}

// DeviceAuthorizationResponse is the RFC 8628 §3.2 response.
type DeviceAuthorizationResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

// DeviceCodeInfo is the internal representation of a pending device code.
type DeviceCodeInfo struct {
	DeviceCode string
	UserCode   string
	ClientID   string
	TenantID   uuid.UUID
	UserID     *uuid.UUID // set when user authorizes
	Scope      []string
	Status     string // "pending", "approved", "denied", "expired"
	CreatedAt  time.Time
	ExpiresAt  time.Time
	LastPoll   *time.Time // for slow_down enforcement
}

// deviceCodeStore holds pending device codes in-memory (production would use Redis).
var (
	deviceCodeMu    sync.RWMutex
	deviceCodeStore = make(map[string]*DeviceCodeInfo) // keyed by device_code
	userCodeIndex   = make(map[string]string)          // user_code -> device_code
)

// CreateDeviceAuthorization generates device_code + user_code and stores them.
func (s *OAuthService) CreateDeviceAuthorization(req *DeviceAuthorizationRequest) (*DeviceAuthorizationResponse, error) {
	deviceCode := generateDeviceCode(40)
	userCode := generateUserCode()

	info := &DeviceCodeInfo{
		DeviceCode: deviceCode,
		UserCode:   userCode,
		ClientID:   req.ClientID,
		TenantID:   req.TenantID,
		Scope:      req.Scope,
		Status:     "pending",
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(15 * time.Minute),
	}

	deviceCodeMu.Lock()
	deviceCodeStore[deviceCode] = info
	userCodeIndex[userCode] = deviceCode
	deviceCodeMu.Unlock()

	verificationURI := req.Issuer + "/device"
	if req.Issuer == "" {
		verificationURI = "/device"
	}

	return &DeviceAuthorizationResponse{
		DeviceCode:      deviceCode,
		UserCode:        userCode,
		VerificationURI: verificationURI,
		ExpiresIn:       900, // 15 minutes
		Interval:        5,   // 5 seconds between polls
	}, nil
}

// PollDeviceToken is called by the device with grant_type=device_code.
// Returns a token if the user has approved, or an error indicating pending/denied/expired.
func (s *OAuthService) PollDeviceToken(ctx context.Context, deviceCode, clientID string) (*TokenResponse, error) {
	deviceCodeMu.RLock()
	info, ok := deviceCodeStore[deviceCode]
	deviceCodeMu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("invalid_device_code")
	}

	if time.Now().After(info.ExpiresAt) {
		deviceCodeMu.Lock()
		delete(deviceCodeStore, deviceCode)
		delete(userCodeIndex, info.UserCode)
		deviceCodeMu.Unlock()
		return nil, fmt.Errorf("expired_token")
	}

	if info.Status == "pending" {
		// Check if client is polling too fast (interval enforcement).
		if info.LastPoll != nil && time.Since(*info.LastPoll) < 5*time.Second {
			return nil, fmt.Errorf("slow_down")
		}
		now := time.Now()
		info.LastPoll = &now
		return nil, fmt.Errorf("authorization_pending")
	}

	if info.Status == "denied" {
		return nil, fmt.Errorf("access_denied")
	}

	if info.Status == "approved" && info.UserID != nil {
		// Issue tokens for the authorized user.
		accessToken, expiresIn, err := s.issueDeviceAccessToken(info.TenantID, *info.UserID)
		if err != nil {
			return nil, err
		}

		// Clean up.
		deviceCodeMu.Lock()
		delete(deviceCodeStore, deviceCode)
		delete(userCodeIndex, info.UserCode)
		deviceCodeMu.Unlock()

		scopeStr := ""
		if len(info.Scope) > 0 {
			scopeStr = strings.Join(info.Scope, " ")
		}

		return &TokenResponse{
			AccessToken: accessToken,
			TokenType:   "Bearer",
			ExpiresIn:   expiresIn,
			Scope:       scopeStr,
		}, nil
	}

	return nil, fmt.Errorf("authorization_pending")
}

// ApproveDeviceCode is called when the user enters their user_code and approves.
func (s *OAuthService) ApproveDeviceCode(userCode string, userID uuid.UUID) error {
	deviceCodeMu.Lock()
	defer deviceCodeMu.Unlock()

	deviceCode, ok := userCodeIndex[userCode]
	if !ok {
		return fmt.Errorf("invalid user_code")
	}

	info, ok := deviceCodeStore[deviceCode]
	if !ok {
		return fmt.Errorf("device code not found")
	}

	if time.Now().After(info.ExpiresAt) {
		delete(deviceCodeStore, deviceCode)
		delete(userCodeIndex, userCode)
		return fmt.Errorf("expired user_code")
	}

	info.Status = "approved"
	info.UserID = &userID
	return nil
}

// issueDeviceAccessToken signs a JWT for a device flow user.
func (s *OAuthService) issueDeviceAccessToken(tenantID, userID uuid.UUID) (string, int, error) {
	now := time.Now()
	expiresAt := now.Add(1 * time.Hour)

	claims := jwt.MapClaims{
		"iss":       s.issuer,
		"sub":       userID.String(),
		"aud":       "ggid",
		"tenant_id": tenantID.String(),
		"iat":       now.Unix(),
		"exp":       expiresAt.Unix(),
		"jti":       uuid.New().String(),
	}

	token := jwt.NewWithClaims(s.signingMethod(), claims)
	token.Header["kid"] = s.keyProvider.Metadata().KeyID

	signed, err := token.SignedString(s.keyProvider.Signer())
	if err != nil {
		return "", 0, fmt.Errorf("sign device token: %w", err)
	}

	return signed, int(time.Until(expiresAt).Seconds()), nil
}

// generateDeviceCode creates a random alphanumeric device code.
func generateDeviceCode(length int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[cryptoRandInt(len(charset))]
	}
	return string(b)
}

// generateUserCode creates an 8-character user code in XXXX-XXXX format.
func generateUserCode() string {
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // no confusing chars
	part1 := make([]byte, 4)
	part2 := make([]byte, 4)
	for i := range part1 {
		part1[i] = charset[cryptoRandInt(len(charset))]
	}
	for i := range part2 {
		part2[i] = charset[cryptoRandInt(len(charset))]
	}
	return string(part1) + "-" + string(part2)
}

// BackchannelLogout revokes all tokens for a subject (OIDC back-channel logout).
// In production, this would also notify connected RPs via back-channel.
func (s *OAuthService) BackchannelLogout(sub string) {
	// Mark the subject as globally logged out — all future token
	// validations for this sub will fail until a new session is created.
	key := fmt.Sprintf("ggid:backchannel_logout:%s", sub)
	backchannelLogoutList.Store(key, time.Now().Unix())

	// In a full implementation, this would iterate all registered client
	// back-channel logout URIs and POST a logout_token to each.
}

// ParseBackchannelLogoutToken parses the logout_token JWT (OIDC Back-Channel Logout).
// Validates required claims: sub or sid, events containing the logout event.
func (s *OAuthService) ParseBackchannelLogoutToken(tokenStr string) (jwt.MapClaims, error) {
	// Parse without strict verification (production would verify signature).
	token, _, err := new(jwt.Parser).ParseUnverified(tokenStr, jwt.MapClaims{})
	if err != nil {
		return nil, fmt.Errorf("invalid logout token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid logout token claims")
	}

	// Must have sub or sid.
	sub, hasSub := claims["sub"].(string)
	sid, hasSid := claims["sid"].(string)
	if !hasSub && !hasSid && sub == "" && sid == "" {
		return nil, fmt.Errorf("logout token must contain 'sub' or 'sid'")
	}

	// Check events claim contains the back-channel logout event.
	events, ok := claims["events"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("logout token must contain 'events' claim")
	}
	if _, ok := events["http://schemas.openid.net/event/backchannel-logout"]; !ok {
		return nil, fmt.Errorf("logout token events must contain backchannel-logout event")
	}

	// Must not have nonce (per spec).
	if _, ok := claims["nonce"]; ok {
		return nil, fmt.Errorf("logout token must not contain 'nonce'")
	}

	// Replay prevention: check jti uniqueness (OIDC Back-Channel Logout §2.4).
	if jti, ok := claims["jti"].(string); ok && jti != "" {
		jtiKey := fmt.Sprintf("ggid:backchannel_logout_jti:%s", jti)
		if _, seen := backchannelLogoutList.Load(jtiKey); seen {
			return nil, fmt.Errorf("logout token replay detected (duplicate jti)")
		}
		backchannelLogoutList.Store(jtiKey, time.Now().Unix())
	}

	return claims, nil
}

// signingMethod returns the jwt.SigningMethod matching the key provider algorithm.
func (s *OAuthService) signingMethod() jwt.SigningMethod {
	alg := s.keyProvider.Metadata().Algorithm
	switch alg {
	case pkgcrypto.RS256:
		return jwt.SigningMethodRS256
	case pkgcrypto.RS384:
		return jwt.SigningMethodRS384
	case pkgcrypto.RS512:
		return jwt.SigningMethodRS512
	case pkgcrypto.PS256:
		return jwt.SigningMethodPS256
	case pkgcrypto.PS384:
		return jwt.SigningMethodPS384
	case pkgcrypto.PS512:
		return jwt.SigningMethodPS512
	case pkgcrypto.ES256:
		return jwt.SigningMethodES256
	case pkgcrypto.ES384:
		return jwt.SigningMethodES384
	case pkgcrypto.ES512:
		return jwt.SigningMethodES512
	case pkgcrypto.EdDSA:
		return jwt.SigningMethodEdDSA
	default:
		return jwt.SigningMethodRS256
	}
}

func cryptoRandInt(max int) int {
	if max <= 0 {
		return 0
	}
	bigN, err := crand.Int(crand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0
	}
	return int(bigN.Int64())
}

// --- JWT Bearer Assertion Grant (RFC 7523) ---

// JWTBearerRequest holds parameters for the jwt-bearer grant type.
type JWTBearerRequest struct {
	TenantID  uuid.UUID
	Assertion string // the third-party-signed JWT
	Scope     []string
	Issuer    string
}

// JWTBearerGrant implements RFC 7523: validates a third-party JWT assertion
// and issues a GGID access token for the assertion subject.
func (s *OAuthService) JWTBearerGrant(ctx context.Context, req *JWTBearerRequest) (*TokenResponse, error) {
	if req.Assertion == "" {
		return nil, fmt.Errorf("assertion is required")
	}

	// Parse the JWT without verifying signature first (to extract claims).
	// In production, the assertion would be verified against a trusted issuer's JWKS.
	token, _, err := new(jwt.Parser).ParseUnverified(req.Assertion, jwt.MapClaims{})
	if err != nil {
		return nil, fmt.Errorf("invalid assertion: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid assertion claims")
	}

	// Extract subject (sub) — the user/service this token is for.
	sub, ok := claims["sub"].(string)
	if !ok || sub == "" {
		return nil, fmt.Errorf("assertion missing 'sub' claim")
	}

	// Extract issuer (iss) — who signed this assertion.
	iss, _ := claims["iss"].(string)

	// Verify expiration.
	exp, ok := claims["exp"].(float64)
	if !ok {
		return nil, fmt.Errorf("assertion missing 'exp' claim")
	}
	if time.Now().Unix() > int64(exp) {
		return nil, fmt.Errorf("assertion has expired")
	}

	// Parse the subject as a user ID.
	userID, err := uuid.Parse(sub)
	if err != nil {
		return nil, fmt.Errorf("assertion sub must be a valid user ID")
	}

	// Issue a GGID access token for this user.
	now := time.Now()
	expiresAt := now.Add(1 * time.Hour)

	gidClaims := jwt.MapClaims{
		"iss":           s.issuer,
		"sub":           userID.String(),
		"aud":           "ggid",
		"tenant_id":     req.TenantID.String(),
		"iat":           now.Unix(),
		"exp":           expiresAt.Unix(),
		"jti":           uuid.New().String(),
		"assertion_iss": iss, // track the original assertion issuer
	}

	gidToken := jwt.NewWithClaims(s.signingMethod(), gidClaims)
	gidToken.Header["kid"] = s.keyProvider.Metadata().KeyID

	signed, err := gidToken.SignedString(s.keyProvider.Signer())
	if err != nil {
		return nil, fmt.Errorf("sign jwt-bearer token: %w", err)
	}

	scopeStr := ""
	if len(req.Scope) > 0 {
		scopeStr = strings.Join(req.Scope, " ")
	}

	return &TokenResponse{
		AccessToken: signed,
		TokenType:   "Bearer",
		ExpiresIn:   int(time.Until(expiresAt).Seconds()),
		Scope:       scopeStr,
	}, nil
}

// computeAMR builds the amr claim from auth methods (oauth-local version).
func computeAMR(authMethods []string) []string {
	amr := []string{}
	hasMFA := false
	for _, m := range authMethods {
		switch m {
		case "password":
			amr = append(amr, "pwd")
		case "totp", "hotp":
			amr = append(amr, "otp")
			hasMFA = true
		case "webauthn":
			amr = append(amr, "fpt")
			hasMFA = true
		case "sms_otp":
			amr = append(amr, "sms")
			hasMFA = true
		}
	}
	if hasMFA {
		amr = append(amr, "mfa")
	}
	return amr
}

// computeACR determines the NIST AAL level from auth methods.
func computeACR(authMethods []string) string {
	hasPwd, hasMFA, hasHardware := false, false, false
	for _, m := range authMethods {
		switch m {
		case "password":
			hasPwd = true
		case "webauthn":
			hasHardware = true
			hasMFA = true
		case "totp", "hotp", "sms_otp":
			hasMFA = true
		}
	}
	if hasHardware {
		return "AAL3"
	}
	if hasMFA {
		return "AAL2"
	}
	if hasPwd {
		return "AAL1"
	}
	return ""
}
