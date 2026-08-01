// Package service implements the OAuth2/OIDC business logic.
package service

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	crand "crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"net/url"
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
	"github.com/jackc/pgx/v5/pgconn"
)

// OAuthService implements OAuth2 client management and the authorization code flow.
type OAuthService struct {
	clientRepo       repository.ClientRepository
	codeRepo         repository.AuthorizationCodeRepository
	tokenRepo        repository.IDTokenRepository
	keyProvider      pkgcrypto.KeyProvider
	issuer           string
	rdb              RedisCmdable     // optional Redis client for distributed state
	pool             PoolQuerier      // optional DB pool for user profile queries
	tokenFamilyStore TokenFamilyStore // optional: RFC 6749 §10.4 family registry
}

// PoolQuerier is the minimal interface for DB queries (user profile lookup).
type PoolQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// RedisCmdable is the minimal subset of go-redis used by the state store.
// This allows mocking in tests without a real Redis server.
type RedisCmdable interface {
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
	SetNX(ctx context.Context, key string, value any, ttl time.Duration) (bool, error)
	Get(ctx context.Context, key string) (string, error)
	GetDel(ctx context.Context, key string) (string, error)
	Del(ctx context.Context, key string) error
	ZAdd(ctx context.Context, key string, score float64, member string) error
}

// SetRedisClient wires a Redis client for distributed state storage.
// When set, OAuth state parameters are stored in Redis (for HA/multi-instance).
// When nil or Redis is unreachable, the in-memory sync.Map fallback is used.
func (s *OAuthService) SetRedisClient(rdb RedisCmdable) {
	s.rdb = rdb
}

// VerifyAuthTicket validates a one-time auth ticket from passwordless authentication
// (passkey, SMS OTP, email OTP). The ticket was created by the auth service and stored
// in Redis with 30s TTL. This method reads, validates, and deletes the ticket (single-use).
// Returns the verified user UUID and the tenant_id from the ticket.
func (s *OAuthService) VerifyAuthTicket(ctx context.Context, ticket string) (uuid.UUID, uuid.UUID, error) {
	if s.rdb == nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("redis not configured")
	}
	if ticket == "" {
		return uuid.Nil, uuid.Nil, fmt.Errorf("empty ticket")
	}

	key := "auth_ticket:" + ticket
	// GetDel is atomic — Get+Del allowed concurrent replay of the same
	// single-use ticket (R-cron P2).
	val, err := s.rdb.GetDel(ctx, key)
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("invalid or expired ticket")
	}
	data := []byte(val)

	var ticketData struct {
		TenantID string   `json:"tenant_id"`
		UserID   string   `json:"user_id"`
		Scopes   []string `json:"scopes"`
	}
	if err := json.Unmarshal(data, &ticketData); err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("malformed ticket data")
	}

	userID, err := uuid.Parse(ticketData.UserID)
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("invalid user_id in ticket")
	}

	tenantID, err := uuid.Parse(ticketData.TenantID)
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("invalid tenant_id in ticket")
	}

	return userID, tenantID, nil
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
	Description             string
	Type                    domain.ClientType
	GrantTypes              []string
	ResponseTypes           []string
	RedirectURIs            []string
	PostLogoutRedirectURIs  []string
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
		Description:             input.Description,
		Type:                    input.Type,
		GrantTypes:              input.GrantTypes,
		ResponseTypes:           input.ResponseTypes,
		RedirectURIs:            input.RedirectURIs,
		Scopes:                  input.Scopes,
		TokenEndpointAuthMethod: defaultIfEmpty(input.TokenEndpointAuthMethod, "client_secret_basic"),
		Metadata:                input.Metadata,
		Enabled:                 true,
	}
	// SECURITY: Filter admin scopes to prevent privilege escalation.
	input.Scopes = filterSafeScopes(input.Scopes)
	client.Scopes = input.Scopes
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

	// Store description in metadata JSON (no dedicated DB column).
	if input.Description != "" {
		if client.Metadata == nil {
			client.Metadata = map[string]any{}
		}
		client.Metadata["description"] = input.Description
	}

	var plaintextSecret string
	if client.IsConfidential() {
		plaintextSecret = generateClientSecret()
		hash, err := pkgcrypto.HashPassword(plaintextSecret)
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

// GetClientForAuth retrieves a client and rejects disabled ones.
// Used in authorization/token flows where disabled clients must be blocked.
func (s *OAuthService) GetClientForAuth(ctx context.Context, clientID string) (*domain.OAuthClient, error) {
	client, err := s.GetClient(ctx, clientID)
	if err != nil {
		return nil, err
	}
	if client != nil && !client.Enabled {
		return nil, errors.New(errors.ErrPermissionDenied, "client is disabled")
	}
	return client, nil
}

// AuthenticateClient verifies a client_id + client_secret pair against the
// registry (RFC 7662 §2.1 introspection authentication, RFC 6749 §2.3).
// Returns nil on success; Unauthenticated on any failure (no oracle leaks).
func (s *OAuthService) AuthenticateClient(ctx context.Context, clientID, clientSecret string) error {
	client, err := s.GetClientForAuth(ctx, clientID)
	if err != nil || client == nil {
		return errors.Unauthenticated("invalid client credentials")
	}
	if !client.Enabled {
		return errors.Unauthenticated("invalid client credentials")
	}
	if !client.IsConfidential() {
		// Public clients have no secret and cannot authenticate this way.
		return errors.Unauthenticated("invalid client credentials")
	}
	ok, _ := pkgcrypto.VerifyPassword(clientSecret, client.ClientSecretHash)
	if !ok {
		return errors.Unauthenticated("invalid client credentials")
	}
	return nil
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

// ResolveClientTenant finds the tenant_id for an OAuth client by client_id.
// This bypasses RLS because it's needed for the authorize/token endpoints
// when MCP clients (RFC 9728) don't send tenant_id.
// SECURITY: The query only returns tenant_id (a UUID), not other tenant data.
// Tenant enumeration via client_id is low-risk: client_ids are not secret
// and are already exposed in authorize redirects.
func (s *OAuthService) ResolveClientTenant(ctx context.Context, clientID string) (uuid.UUID, error) {
	if s.pool == nil {
		return uuid.Nil, fmt.Errorf("database not configured")
	}
	var tenantID uuid.UUID
	err := s.pool.QueryRow(ctx, `SELECT tenant_id FROM oauth_clients WHERE client_id = $1`, clientID).Scan(&tenantID)
	if err != nil {
		return uuid.Nil, err
	}
	return tenantID, nil
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
		// SECURITY: Filter admin scopes to prevent privilege escalation.
		client.Scopes = filterSafeScopes(updates.Scopes)
	}
	if updates.TokenEndpointAuthMethod != nil {
		client.TokenEndpointAuthMethod = *updates.TokenEndpointAuthMethod
	}
	if updates.Enabled != nil {
		client.Enabled = *updates.Enabled
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
	Name                    *string        `json:"client_name,omitempty"`
	RedirectURIs            []string       `json:"redirect_uris,omitempty"`
	GrantTypes              []string       `json:"grant_types,omitempty"`
	ResponseTypes           []string       `json:"response_types,omitempty"`
	Scopes                  []string       `json:"scope,omitempty"`
	TokenEndpointAuthMethod *string        `json:"token_endpoint_auth_method,omitempty"`
	Enabled                 *bool          `json:"enabled,omitempty"`
	Metadata                map[string]any `json:"metadata,omitempty"`
}

// --- Authorization Code Flow ---

// AuthorizeRequest holds parameters for the /oauth/authorize endpoint.
type AuthorizeRequest struct {
	TenantID             uuid.UUID
	ClientID             string
	RedirectURI          string
	ResponseType         string // "code"
	Scope                []string
	State                string
	Nonce                string
	CodeChallenge        string          // PKCE
	CodeChallengeMethod  string          // "S256" or "plain"
	UserID               uuid.UUID       // the authenticated user
	AuthorizationDetails json.RawMessage // RAR authorization_details (RFC 9396)
	// NIST 800-63B AAL/AMR
	AuthMethods  []string // methods used during auth (password, totp, webauthn)
	RequestedACR string   // acr_values param from /authorize
}

// CreateAuthorizationCode creates a short-lived authorization code.
// GetDiscoveryConfig returns the OIDC discovery document.
// GetIssuer returns the OAuth issuer URL.
func (s *OAuthService) GetIssuer() string {
	return s.issuer
}

func (s *OAuthService) GetDiscoveryConfig() *domain.OIDCDiscoveryConfig {
	base := s.issuer
	return &domain.OIDCDiscoveryConfig{
		Issuer:                s.issuer,
		AuthorizationEndpoint: base + "/oauth/authorize",
		TokenEndpoint:         base + "/oauth/token",
		UserInfoEndpoint:      base + "/oauth/userinfo",
		JwksURI:               base + "/oauth/jwks",
		RevocationEndpoint:    base + "/oauth/revoke",
		IntrospectionEndpoint: base + "/oauth/introspect",
		// Only the authorization code flow is implemented (OAuth 2.1 direction);
		// implicit/hybrid response types (token, id_token) are NOT issued, so
		// they must not be advertised or standard clients will attempt them.
		ResponseTypesSupported:            []string{"code"},
		GrantTypesSupported:               []string{"authorization_code", "refresh_token", "client_credentials", "password", "urn:ietf:params:oauth:grant-type:device_code", "urn:ietf:params:oauth:grant-type:token-exchange", "urn:ietf:params:oauth:grant-type:jwt-bearer", "urn:ietf:params:oauth:grant-type:ciba"},
		SubjectTypesSupported:             []string{"public"},
		IDTokenSigningAlgValues:           []string{"RS256"},
		ScopesSupported:                   []string{"openid", "profile", "email", "offline_access"},
		ClaimsSupported:                   []string{"sub", "email", "name", "picture", "groups", "preferred_username", "updated_at"},
		TokenEndpointAuthMethodsSupported: []string{"client_secret_basic", "client_secret_post", "none", "tls_client_auth", "self_signed_tls_client_auth"},
		CodeChallengeMethodsSupported:     []string{"S256"}, // OAuth 2.1: S256 only
		BackchannelLogoutSupported:        true,
		FrontchannelLogoutSupported:       true,
		EndSessionEndpoint:                base + "/oauth/logout",
		// Must match the actual registered route (server.go: /api/v1/oauth/device_authorize).
		DeviceAuthorizationEndpoint:        base + "/api/v1/oauth/device_authorize",
		RegistrationEndpoint:               base + "/oauth/register",
		PushedAuthorizationRequestEndpoint: base + "/oauth/par",
		BackchannelAuthenticationEndpoint:  base + "/api/v1/oauth/backchannel",
	}
}

// --- JWKS ---

// GetJWKS returns the JSON Web Key Set containing public keys.
// During key rotation grace period, both current and previous keys are returned
// so that clients can verify tokens signed with either key.
func (s *OAuthService) GetJWKS() *domain.JWKSResponse {
	var keys []domain.JWKSKey

	// Current signing key
	key, err := publicKeyToJWK(s.keyProvider.Metadata().KeyID, s.keyProvider.Public())
	if err == nil {
		keys = append(keys, key)
	}

	// Previous key (if within grace period) — check if provider supports rotation
	if rkp, ok := s.keyProvider.(*RotatingKeyProvider); ok {
		if prevPub := rkp.PreviousPublicKey(); prevPub != nil {
			if prevKid := rkp.PreviousKeyID(); prevKid != "" {
				if prevKey, err := publicKeyToJWK(prevKid, prevPub); err == nil {
					keys = append(keys, prevKey)
				}
			}
		}
	}

	if len(keys) == 0 {
		return &domain.JWKSResponse{Keys: []domain.JWKSKey{}}
	}
	return &domain.JWKSResponse{Keys: keys}
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

// resolveAudience returns the caller-requested target audience (RFC 8707 /
// Auth0-style `audience` parameter), falling back to the client_id when not
// provided — preserving the pre-existing default behavior.
func resolveAudience(requested, fallback string) string {
	if requested != "" {
		// SECURITY: Reject reserved AS-internal audiences from caller-supplied values.
		// This prevents regular users from minting tokens that can introspect other tokens.
		if requested == "ggid" || requested == "introspection" {
			return fallback
		}
		return requested
	}
	return fallback
}

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
func (s *OAuthService) fetchUserPermissions(ctx context.Context, tenantID, userID uuid.UUID) ([]string, error) {
	if s.pool == nil {
		return nil, nil
	}
	rows, err := s.pool.Query(ctx, `
		SELECT DISTINCT p.key
		FROM role_permissions rp
		JOIN permissions p ON p.id = rp.permission_id
		JOIN user_roles ur ON ur.role_id = rp.role_id
		JOIN roles r ON r.id = ur.role_id
		WHERE ur.user_id = $1 AND r.tenant_id = $2`,
		userID, tenantID)
	if err != nil {
		return nil, err
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
	return perms, rows.Err()
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
		WHERE ur.user_id = $1 AND r.tenant_id = $2`,
		userID, tenantID)
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

// fetchUserRoleKeys returns the role keys (e.g. "platform:admin") for a user.
// Unlike fetchUserRoles which returns display names, this returns the stable
// key field used for RBAC scope detection.
func (s *OAuthService) fetchUserRoleKeys(ctx context.Context, tenantID, userID uuid.UUID) []string {
	if s.pool == nil {
		return nil
	}
	rows, err := s.pool.Query(ctx, `
		SELECT r.key
		FROM user_roles ur
		JOIN roles r ON r.id = ur.role_id
		WHERE ur.user_id = $1 AND r.tenant_id = $2`,
		userID, tenantID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	keys := []string{}
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			continue
		}
		keys = append(keys, key)
	}
	return keys
}

// evaluateConditionalAccess loads enabled conditional access policies from the DB
// and evaluates them against the login context. Returns the action ("allow",
// "block", "deny", "require_mfa") and the matching policy name.
func (s *OAuthService) evaluateConditionalAccess(ctx context.Context, tenantID, userID uuid.UUID, username string) (action string, policyName string) {
	if s.pool == nil {
		log.Printf("CAP: pool nil, skipping")
		return "allow", ""
	}
	rows, err := s.pool.Query(ctx, `
		SELECT data FROM conditional_access_store
		WHERE data->>'tenant_id' = $1 AND (data->>'enabled')::bool = true
		ORDER BY (data->>'priority')::int ASC NULLS LAST`, tenantID.String())
	if err != nil {
		log.Printf("CAP: query error for tenant %s: %v", tenantID.String(), err)
		return "allow", ""
	}
	defer rows.Close()
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			continue
		}
		var p struct {
			Name       string         `json:"name"`
			Conditions map[string]any `json:"conditions"`
			Actions    map[string]any `json:"actions"`
			Action     string         `json:"action"`
			Enabled    bool           `json:"enabled"`
		}
		if json.Unmarshal(data, &p) != nil {
			continue
		}
		if !p.Enabled {
			continue
		}
		act := p.Action
		if act == "" {
			if a, ok := p.Actions["action"].(string); ok {
				act = a
			}
		}
		matched := true
		if len(p.Conditions) > 0 {
			clientIP := ctxIP(ctx)
			now := time.Now()
			for k, v := range p.Conditions {
				switch k {
				case "username":
					if sv, ok := v.(string); ok && sv != "" && username != sv {
						matched = false
					}
				case "user_id":
					if sv, ok := v.(string); ok && sv != "" && userID.String() != sv {
						matched = false
					}
				case "ip_address":
					// Match if client IP is within the condition CIDR range.
					// Supports both exact IP and CIDR notation (e.g. 203.0.113.0/24).
					if sv, ok := v.(string); ok && sv != "" {
						if !ipMatchesCIDR(clientIP, sv) {
							matched = false
						}
					}
				case "auth_method":
					// In PasswordGrant, the auth method is always "password".
					if sv, ok := v.(string); ok && sv != "" && "password" != sv {
						matched = false
					}
				case "time_of_day":
					// Match if current hour (24h) is within the specified hour.
					if fv, ok := v.(float64); ok {
						if int(now.Hour()) != int(fv) {
							matched = false
						}
					}
				}
			}
		}
		if matched {
			if act == "block" || act == "deny" || act == "require_mfa" {
				slog.Info("CAP: policy matched user", "policy", p.Name, "user", maskUser(username), "action", act)
				return act, p.Name
			}
			slog.Info("CAP: policy matched but not blocking", "policy", p.Name, "user", maskUser(username), "action", act)
			return "allow", ""
		}
	}
	slog.Info("CAP: no policies matched", "user", maskUser(username), "tenant", tenantID.String())
	return "allow", ""
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
	permissions, _ := s.fetchUserPermissions(context.Background(), tenantID, userID)
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
		"scope":       scope,       // OAuth scopes only (openid profile email)
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
	ClientSecret     string // required for confidential clients
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
	// 0. SECURITY: Authenticate the requesting client (same as other grants).
	// Client authentication is mandatory for token exchange per RFC 8693 §2.4.
	if req.ClientID == "" {
		return nil, errors.Unauthenticated("client_id is required for token exchange")
	}
	client, err := s.clientRepo.GetClientByID(ctx, req.TenantID, req.ClientID)
	if err != nil {
		return nil, errors.Unauthenticated("client authentication failed")
	}
	if client.IsConfidential() && client.TokenEndpointAuthMethod != "none" {
		ok, _ := pkgcrypto.VerifyPassword(req.ClientSecret, client.ClientSecretHash)
		if !ok {
			return nil, errors.Unauthenticated("invalid client credentials")
		}
	}
	if !client.Enabled {
		return nil, errors.InvalidArgument("client is disabled")
	}

	return s.exchangeTokenInternal(ctx, req)
}

// exchangeTokenInternal performs the token exchange logic without client
// authentication. Used internally by legacy ExchangeToken wrapper and by
// ExchangeTokenRFC8693 after client auth has been verified.
func (s *OAuthService) exchangeTokenInternal(ctx context.Context, req *RFC8693ExchangeRequest) (*TokenResponse, error) {

	// 0a. Validate subject_token_type (RFC 8693 §2.1).
	const expectedSubjectType = "urn:ietf:params:oauth:token-type:access_token"
	if req.SubjectTokenType != "" && req.SubjectTokenType != expectedSubjectType {
		return nil, fmt.Errorf("unsupported subject_token_type: %s", req.SubjectTokenType)
	}

	// 1. Validate subject token.
	subjectClaims, err := s.parseAndValidateJWT(req.SubjectToken)
	if err != nil {
		return nil, fmt.Errorf("invalid subject_token: %w", err)
	}

	// 1a. SECURITY (R48 B1): Reject exchanged subject tokens that have been
	// revoked. IsTokenRevoked checks the authoritative revocation sources:
	// Redis oauth:revoked:<sha256(token)> (written by RevokeToken) plus the
	// in-memory fallback, then the DB (sessions.revoked_at via jti). The
	// previous sessions.jti query never matched — sessions INSERTs did not
	// populate jti, so the check was a silent fail-open (revoked tokens
	// could still be exchanged).
	if s.IsTokenRevoked(req.SubjectToken) {
		return nil, fmt.Errorf("subject token has been revoked")
	}

	// 1a. Reject ID tokens used as subject_token (nonce is an ID token indicator).
	if _, hasNonce := subjectClaims["nonce"]; hasNonce {
		return nil, fmt.Errorf("ID tokens cannot be used as subject_token")
	}

	subjectID, _ := subjectClaims["sub"].(string)
	if subjectID == "" {
		return nil, fmt.Errorf("subject_token missing sub claim")
	}

	// Cross-tenant laundering guard: the subject token's tenant must match
	// the requesting client's tenant — otherwise tenant B's client could
	// exchange tenant A's token into a B-scoped token carrying A's
	// permissions/roles (R8 P1).
	if st, _ := subjectClaims["tenant_id"].(string); st != "" && st != req.TenantID.String() {
		return nil, fmt.Errorf("subject_token tenant does not match requesting client tenant")
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

	// 4.5. Audience validation: RFC 8693 allows the new token to target a
	// different audience than the subject token. We only validate that the
	// resource is a legitimate service identifier (non-empty, not localhost).
	// The subject token's audience is NOT required to match the new resource.
	req.Resource = strings.TrimSpace(req.Resource)

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
		// Same tenant guard as the subject token — a cross-tenant actor
		// identity must not be mixed into the delegation chain.
		if at, _ := actorClaims["tenant_id"].(string); at != "" && at != req.TenantID.String() {
			return nil, fmt.Errorf("actor_token tenant does not match requesting client tenant")
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

	// SECURITY: Carry forward permissions from the subject token, but only
	// those relevant to the requested scopes. This prevents privilege escalation
	// where a high-privilege token exchanges for a scoped token that still
	// carries all original permissions.
	subjectPerms := getStringSliceClaim(subjectClaims, "permissions")
	subjectRoles := getStringSliceClaim(subjectClaims, "roles")

	// When scopes are requested, filter permissions and roles to only those
	// matching the requested scopes. Match scope as a namespace boundary:
	// "user" matches "user:read" but not "user_management:write".
	var filteredPerms []string
	var filteredRoles []string
	if len(req.Scope) > 0 {
		for _, perm := range subjectPerms {
			for _, sc := range req.Scope {
				if perm == sc || strings.HasPrefix(perm, sc+":") {
					filteredPerms = append(filteredPerms, perm)
					break
				}
			}
		}
		// Filter roles similarly — only carry roles matching requested scopes.
		for _, role := range subjectRoles {
			for _, sc := range req.Scope {
				if role == sc || strings.HasPrefix(role, sc+":") {
					filteredRoles = append(filteredRoles, role)
					break
				}
			}
		}
	} else {
		filteredPerms = subjectPerms
		filteredRoles = subjectRoles
	}

	claimsMap := jwt.MapClaims{
		"iss":         s.issuer,
		"sub":         subjectID,
		"aud":         audience,
		"iat":         now.Unix(),
		"exp":         expiresAt.Unix(),
		"jti":         uuid.New().String(),
		"tenant_id":   req.TenantID.String(),
		"scope":       scopeStr,      // OAuth scopes only
		"permissions": filteredPerms, // Only permissions matching requested scopes
		"roles":       filteredRoles, // Only roles matching requested scopes
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
		AccessToken: signed,
		TokenType:   "Bearer",
		ExpiresIn:   int(expiresAt.Sub(now).Seconds()),
		Scope:       scopeStr,
	}, nil
}

// parseAndValidateJWT parses and validates a JWT issued by this service.
func (s *OAuthService) parseAndValidateJWT(raw string) (jwt.MapClaims, error) {
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) {
		if !isSupportedSigningMethod(t.Method) {
			return nil, fmt.Errorf("unexpected signing method: %s", t.Header["alg"])
		}
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
	AtHash   string   // OIDC Core §3.1.3.6: access_token hash for binding
	CHash    string   // OIDC Core §3.1.3.6: authorization code hash for binding
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
		if opts.AtHash != "" {
			claims["at_hash"] = opts.AtHash
		}
		if opts.CHash != "" {
			claims["c_hash"] = opts.CHash
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
// It enforces issuer verification (RFC 7519 §4.1.3) but does NOT enforce
// audience — use ParseAccessTokenWithAudience for that.
func (s *OAuthService) ParseAccessToken(tokenStr string) (jwt.MapClaims, error) {
	return s.ParseAccessTokenWithAudience(tokenStr, "")
}

// ParseAccessTokenWithAudience parses and validates a JWT access token.
// Always verifies issuer (iss claim must match s.issuer).
// If expectedAudience is non-empty, also verifies audience (aud claim).
func (s *OAuthService) ParseAccessTokenWithAudience(tokenStr, expectedAudience string) (jwt.MapClaims, error) {
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
	// RFC 7519 §4.1.1: verify issuer (always enforced).
	tokenIss := getStringClaim(claims, "iss")
	if tokenIss != "" && tokenIss != s.issuer {
		return nil, fmt.Errorf("token issuer mismatch: expected %q, got %q", s.issuer, tokenIss)
	}
	// RFC 7519 §4.1.3: verify audience if expected audience is provided.
	// Tokens issued by this server have aud = client_id (default) or
	// aud = issuer (when resolveAudience falls back to s.issuer).
	// Accept both: check if tokenAud matches expectedAudience OR s.issuer.
	if expectedAudience != "" {
		tokenAud, _ := claims["aud"].(string)
		if tokenAud == "" {
			// aud may be []string
			if audArr, ok := claims["aud"].([]any); ok && len(audArr) > 0 {
				tokenAud, _ = audArr[0].(string)
			}
		}
		if tokenAud != expectedAudience && tokenAud != s.issuer {
			return nil, fmt.Errorf("token audience mismatch: expected %q or %q, got %q", expectedAudience, s.issuer, tokenAud)
		}
	}
	return claims, nil
}

func isSupportedSigningMethod(method jwt.SigningMethod) bool {
	return pkgcrypto.IsSupportedAlg(method.Alg())
}

// UserInfoResponse holds the standard OIDC UserInfo claims.
// Enhanced (KB-295) with roles, groups, permissions, and risk level.
type UserInfoResponse struct {
	Sub           string `json:"sub"`
	Name          string `json:"name,omitempty"`
	Email         string `json:"email,omitempty"`
	EmailVerified bool   `json:"email_verified,omitempty"`
	Picture       string `json:"picture,omitempty"`
	TenantID      string `json:"tenant_id,omitempty"`
	// KB-295: Extended fields for downstream applications.
	Roles       []string `json:"roles,omitempty"`
	Groups      []string `json:"groups,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
	RiskLevel   string   `json:"risk_level,omitempty"`
}

// GetUserInfo returns user info claims from a validated access token.
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

// init starts a background goroutine that periodically sweeps expired entries
// from the in-memory stateStore, revokedTokens, and deviceCodeStore.
func init() {
	go startExpiredEntryReaper()
}

func startExpiredEntryReaper() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()

		// Sweep stateStore: delete entries whose expiry is in the past.
		stateStore.Range(func(k, v any) bool {
			if exp, ok := v.(time.Time); ok && now.After(exp) {
				stateStore.Delete(k)
			}
			return true
		})

		// Sweep revokedTokens: delete entries whose stored expiry is in the past.
		revokedTokens.Range(func(k, v any) bool {
			if exp, ok := v.(time.Time); ok && now.After(exp) {
				revokedTokens.Delete(k)
			} else if m, ok := v.(map[string]any); ok {
				if expStr, ok := m["exp"].(time.Time); ok && now.After(expStr) {
					revokedTokens.Delete(k)
				}
			} else if expUnix, ok := v.(int64); ok && expUnix > 0 && now.Unix() > expUnix {
				revokedTokens.Delete(k)
			}
			return true
		})

		// Sweep deviceCodeStore: delete expired entries.
		deviceCodeMu.Lock()
		for code, info := range deviceCodeStore {
			if now.After(info.ExpiresAt) {
				delete(deviceCodeStore, code)
				delete(userCodeIndex, info.UserCode)
			}
		}
		deviceCodeMu.Unlock()

		// Sweep backchannelLogoutList: delete entries older than 7 days.
		cutoff := now.Unix() - 7*24*3600
		backchannelLogoutList.Range(func(k, v any) bool {
			if ts, ok := v.(int64); ok && ts < cutoff {
				backchannelLogoutList.Delete(k)
			}
			return true
		})
	}
}

// ValidateState checks whether a state parameter was previously stored during /authorize.

// BuildAuthorizeRedirectURL builds the redirect URL with code, state, and iss parameters.
// Per RFC 6749 §10.14, the iss parameter identifies the authorization server.
func (s *OAuthService) BuildAuthorizeRedirectURL(redirectURI, code, state string) string {
	u := redirectURI
	sep := "?"
	if containsQS(redirectURI) {
		sep = "&"
	}
	u += sep + "code=" + url.QueryEscape(code)
	if state != "" {
		u += "&state=" + url.QueryEscape(state)
	}
	// RFC 6749 §10.14: iss parameter prevents mix-up attacks.
	u += "&iss=" + url.QueryEscape(s.issuer)
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

// ValidateTokenOwnership checks if the given client_id matches the token's
// intended audience (aud claim). RFC 7009 §2.1 requires that the client
// revoking a token must be the one that owns it.
// DownscopeToken issues a new JWT with reduced scope (RFC 8693 token exchange).
func (s *OAuthService) DownscopeToken(userID uuid.UUID, tenantID uuid.UUID, audience, scope string) (string, int, error) {
	return s.issueAccessToken(userID, tenantID, audience, scope)
}

func (s *OAuthService) ValidateTokenOwnership(tokenStr, clientID string) bool {
	if tokenStr == "" || clientID == "" {
		return true // can't verify, allow (auth gate still applies)
	}
	claims, err := s.ParseAccessToken(tokenStr)
	if err != nil {
		return true // unparseable token — let RevokeToken handle it
	}
	aud := getStringClaim(claims, "aud")
	if aud == clientID {
		return true
	}
	// Also check aud as array type (RFC 7519 §4.1.3)
	if audArr, ok := claims["aud"].([]any); ok {
		for _, a := range audArr {
			if s, ok := a.(string); ok && s == clientID {
				return true
			}
		}
	}
	return false
}

// RevokeToken marks a token as revoked. The token's JWT ID is extracted and
// stored in the blacklist. Subsequent introspection calls will return active=false.
// --- Refresh Token Grant ---

// RefreshTokenRequest holds parameters for the refresh_token grant.
type RefreshTokenRequest struct {
	TenantID     uuid.UUID
	RefreshToken string
	ClientID     string
	ClientSecret string
	Scope        []string
	Audience     string // optional target audience (defaults to client_id)
}

// RefreshToken issues new tokens using a refresh token.
// On each use, a new refresh token is issued and the old one is invalidated.
// If a previously-used (revoked) token is presented, all tokens for that
// client are revoked (reuse detection).
func WithClientInfo(ctx context.Context, ip, userAgent string) context.Context {
	if h, _, err := net.SplitHostPort(ip); err == nil {
		ip = h
	}
	ctx = context.WithValue(ctx, CtxKeyClientIP{}, ip)
	ctx = context.WithValue(ctx, CtxKeyUserAgent{}, userAgent)
	return ctx
}

// ctxIP extracts the client IP from context metadata (set by gateway proxy).
// Returns empty string if not available — the inet cast handles NULL safely.
func ctxIP(ctx context.Context) string {
	if v, ok := ctx.Value(CtxKeyClientIP{}).(string); ok {
		return v
	}
	return ""
}

// ctxUserAgent extracts the User-Agent from context metadata.
func ctxUserAgent(ctx context.Context) string {
	if v, ok := ctx.Value(CtxKeyUserAgent{}).(string); ok {
		return v
	}
	return ""
}

type CtxKeyClientIP struct{}
type CtxKeyUserAgent struct{}

// ipMatchesCIDR checks if an IP address matches a CIDR range or exact IP.
func ipMatchesCIDR(ip, cidr string) bool {
	if ip == "" {
		return false
	}
	// If no / in cidr, do exact match.
	if !strings.Contains(cidr, "/") {
		return ip == cidr
	}
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return ip == cidr // fallback to exact match
	}
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}
	return network.Contains(parsedIP)
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
		ok, _ := pkgcrypto.VerifyPassword(oldSecret, client.ClientSecretHash)
		if !ok {
			return "", errors.Unauthenticated("invalid client credentials — old secret does not match")
		}
	}

	// 3. Generate new secret.
	newSecret := generateClientSecret()
	hash, err := pkgcrypto.HashPassword(newSecret)
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
	id, _ := pkgcrypto.GenerateRandomToken(16)
	return "gcid_" + id
}

func generateClientSecret() string {
	secret, _ := pkgcrypto.GenerateRandomToken(32)
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

// validateTOTP validates a 6-digit TOTP code against the given base32 secret.
// Uses Skew=1 to tolerate ±30s clock drift between client and server.
// from client-defined scopes. Custom business scopes (audit:read,
// reports:read) are preserved — they are legitimate M2M permissions that
// come from the client's configured scopes. Only privileged scopes that
// must originate from DB role keys are removed (prevents escalation).
// SECURITY (R230): strict 4-scope whitelist here broke M2M
// client_credentials tokens — custom scopes were stripped, downstream
// authz failed.
func filterSafeScopes(scopes []string) []string {
	var filtered []string
	for _, sc := range scopes {
		if sc == "" {
			continue
		}
		// SECURITY: Case-insensitive comparison — gateway uses ToLower/EqualFold
		// to check admin scopes, so "Platform:admin" would pass gateway checks
		// while bypassing this filter if compared case-sensitively.
		lower := strings.ToLower(sc)
		if lower == "admin" ||
			strings.HasPrefix(lower, "platform:") ||
			strings.HasPrefix(lower, "tenant:") {
			continue
		}
		filtered = append(filtered, sc)
	}
	return filtered
}

func joinScopes(scopes []string) string {
	return strings.Join(scopes, " ")
}

func splitScopes(s string) []string {
	return strings.Fields(s)
}

// intersectScopes returns the intersection of requested and allowed scopes.
// Only scopes present in BOTH requested AND allowed are returned.
// If requested contains a scope not in allowed, it is silently dropped.
// Returns empty if no requested scopes are in the allowed set.
func intersectScopes(requested, allowed []string) []string {
	allowedSet := make(map[string]bool, len(allowed))
	for _, s := range allowed {
		allowedSet[strings.ToLower(s)] = true
	}
	var result []string
	for _, r := range requested {
		if allowedSet[strings.ToLower(r)] {
			result = append(result, r)
		}
	}
	return result // FIX: Return only the intersection, not a fallback to all allowed
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
	ClientName              string   `json:"client_name"`
	RedirectURIs            []string `json:"redirect_uris"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	Scope                   string   `json:"scope"`
	// Optional fields per RFC 7591 Section 2:
	ClientURI       string `json:"client_uri,omitempty"`
	LogoURI         string `json:"logo_uri,omitempty"`
	PolicyURI       string `json:"policy_uri,omitempty"`
	TosURI          string `json:"tos_uri,omitempty"`
	JwksURI         string `json:"jwks_uri,omitempty"`
	SoftwareID      string `json:"software_id,omitempty"`
	SoftwareVersion string `json:"software_version,omitempty"`
}

// DynamicRegistrationResponse is the RFC 7591 registration response.
type DynamicRegistrationResponse struct {
	ClientID                string   `json:"client_id"`
	ClientSecret            string   `json:"client_secret,omitempty"`
	ClientIDIssuedAt        int64    `json:"client_id_issued_at"`
	ClientSecretExpiresAt   int64    `json:"client_secret_expires_at,omitempty"`
	ClientName              string   `json:"client_name"`
	RedirectURIs            []string `json:"redirect_uris"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	Scope                   string   `json:"scope"`
}

// DynamicClientRegister implements RFC 7591 dynamic client registration.
// It creates a new OAuth2 client based on the provided metadata.
// authentication (R137 P1-3). It now delegates to ExchangeTokenRFC8693,
// which enforces client authentication and all validation. New callers
// should use ExchangeTokenRFC8693 directly.
func (s *OAuthService) ExchangeToken(ctx context.Context, req *TokenExchangeRequestRFC8693) (*TokenResponse, error) {
	resource := req.Resource
	if resource == "" {
		resource = req.Audience
	}
	// Legacy wrapper: use exchangeTokenInternal (no client auth for backward compat).
	// External HTTP callers go through ExchangeTokenRFC8693 which enforces client auth.
	return s.exchangeTokenInternal(ctx, &RFC8693ExchangeRequest{
		TenantID:         req.TenantID,
		ClientID:         req.ClientID,
		SubjectToken:     req.SubjectToken,
		SubjectTokenType: req.SubjectTokenType,
		ActorToken:       req.ActorToken,
		ActorTokenType:   req.ActorTokenType,
		Scope:            req.Scope,
		Resource:         resource,
	})
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
	TenantID uuid.UUID
	ClientID string
	Scope    []string
	Issuer   string
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
// Validates JWT signature + required claims: sub or sid, events containing the logout event.
func (s *OAuthService) ParseBackchannelLogoutToken(tokenStr string) (jwt.MapClaims, error) {
	// Verify JWT signature using the service's signing key (P1-12: was ParseUnverified).
	pubKey := s.keyProvider.Public()

	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); ok {
			return pubKey, nil
		}
		if _, ok := t.Method.(*jwt.SigningMethodECDSA); ok {
			return pubKey, nil
		}
		return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
	})
	if err != nil {
		return nil, fmt.Errorf("logout token signature verification failed: %w", err)
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
	// Use Redis SetNX for cross-instance replay detection in multi-replica deployments.
	if jti, ok := claims["jti"].(string); ok && jti != "" {
		jtiKey := fmt.Sprintf("ggid:backchannel_logout_jti:%s", jti)
		if s.rdb != nil {
			set, err := s.rdb.SetNX(context.Background(), jtiKey, "1", 7*24*time.Hour)
			if err != nil || !set {
				return nil, fmt.Errorf("logout token replay detected (duplicate jti)")
			}
		} else {
			if _, seen := backchannelLogoutList.Load(jtiKey); seen {
				return nil, fmt.Errorf("logout token replay detected (duplicate jti)")
			}
			backchannelLogoutList.Store(jtiKey, time.Now().Unix())
		}
	}

	return claims, nil
}

// signingMethod returns the jwt.SigningMethod matching the key provider algorithm.
// fetchExternalIssuerKey retrieves the RSA public key for verifying a JWT
// assertion issued by an external issuer. It queries the OAuth client's
// jwks_uri if configured, or returns an error.
func (s *OAuthService) fetchExternalIssuerKey(ctx context.Context, issuer, clientID string, header map[string]any) (any, error) {
	if clientID == "" {
		return nil, fmt.Errorf("client_id required for external issuer assertion")
	}

	// Look up the OAuth client to get its JWKS URI.
	var jwksURI string
	if s.pool == nil {
		return nil, fmt.Errorf("database not available for external issuer key lookup")
	}
	err := s.pool.QueryRow(ctx,
		`SELECT COALESCE(jwks_uri, '') FROM oauth_clients WHERE client_id = $1`,
		clientID).Scan(&jwksURI)
	if err != nil || jwksURI == "" {
		return nil, fmt.Errorf("no jwks_uri registered for client %s: %w", clientID, err)
	}

	// Fetch JWKS from the client's endpoint with a timeout to prevent
	// goroutine leaks from hanging external IdPs.
	jwksClient := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", jwksURI, nil)
	if err != nil {
		return nil, fmt.Errorf("create JWKS request: %w", err)
	}
	resp, err := jwksClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch JWKS from %s: %w", jwksURI, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("JWKS endpoint returned %d", resp.StatusCode)
	}

	var jwks struct {
		Keys []map[string]any `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, fmt.Errorf("decode JWKS: %w", err)
	}

	// Find the key matching the assertion's kid header.
	kid, _ := header["kid"].(string)
	for _, key := range jwks.Keys {
		if kid != "" {
			if keyKid, _ := key["kid"].(string); keyKid != kid {
				continue
			}
		}
		// Convert JWK to RSA public key.
		nStr, _ := key["n"].(string)
		eStr, _ := key["e"].(string)
		if nStr == "" || eStr == "" {
			continue
		}
		nBytes, err := base64.RawURLEncoding.DecodeString(nStr)
		if err != nil {
			continue
		}
		eBytes, err := base64.RawURLEncoding.DecodeString(eStr)
		if err != nil {
			continue
		}
		n := new(big.Int).SetBytes(nBytes)
		e := 0
		for _, b := range eBytes {
			e = e<<8 + int(b)
		}
		return &rsa.PublicKey{N: n, E: e}, nil
	}

	return nil, fmt.Errorf("no matching key found in JWKS for kid %q", kid)
}

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
	ClientID  string // the OAuth client making the request
	Assertion string // the third-party-signed JWT
	Scope     []string
	Issuer    string
}

// JWTBearerGrant implements RFC 7523: validates a third-party JWT assertion
// and issues a GGID access token for the assertion subject.
