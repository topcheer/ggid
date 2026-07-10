// Package domain defines the core entities for the OAuth/OIDC Service.
package domain

import (
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ClientType determines whether a client can keep a secret.
type ClientType string

const (
	ClientTypeConfidential ClientType = "confidential"
	ClientTypePublic      ClientType = "public"
)

// IsValid returns true if the client type is recognised.
func (t ClientType) IsValid() bool {
	return t == ClientTypeConfidential || t == ClientTypePublic
}

// OAuthClient represents an application registered to use OAuth2/OIDC.
type OAuthClient struct {
	ID                      uuid.UUID
	TenantID                uuid.UUID
	ClientID                string // public identifier
	ClientSecretHash        string // Argon2id hash; empty for public clients
	Name                    string
	Type                    ClientType
	GrantTypes              []string
	ResponseTypes           []string
	RedirectURIs            []string
	Scopes                  []string
	TokenEndpointAuthMethod string
	Metadata                map[string]any
	RequirePKCE             bool // enforce PKCE for this client
	Enabled                 bool
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

// IsConfidential returns true for confidential clients.
func (c *OAuthClient) IsConfidential() bool { return c.Type == ClientTypeConfidential }

// IsPublic returns true for public clients.
func (c *OAuthClient) IsPublic() bool { return c.Type == ClientTypePublic }

// RequiresPKCE returns true if PKCE should be enforced (public clients or RequirePKCE flag).
func (c *OAuthClient) RequiresPKCE() bool { return c.RequirePKCE || c.IsPublic() }

// SupportsGrantType checks if the client allows the given grant type.
func (c *OAuthClient) SupportsGrantType(gt string) bool {
	for _, g := range c.GrantTypes {
		if g == gt {
			return true
		}
	}
	return false
}

// ValidateRedirectURI checks if the given redirect URI is registered.
func (c *OAuthClient) ValidateRedirectURI(uri string) bool {
	for _, r := range c.RedirectURIs {
		if r == uri {
			return true
		}
	}
	return false
}

// MetadataJSON returns metadata as a json.RawMessage suitable for pgx.
func (c *OAuthClient) MetadataJSON() json.RawMessage {
	if c.Metadata == nil {
		return json.RawMessage("{}")
	}
	b, _ := json.Marshal(c.Metadata)
	return b
}

// RefreshTokenRecord tracks an issued refresh token for rotation and reuse detection.
type RefreshTokenRecord struct {
	ID        uuid.UUID
	TenantID  uuid.UUID
	ClientID  uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	Scope     []string
	ExpiresAt time.Time
	Revoked   bool
	Used      bool
	CreatedAt time.Time
}

// AuthorizationCode represents a short-lived OAuth2 authorization code.
type AuthorizationCode struct {
	ID                  uuid.UUID
	TenantID            uuid.UUID
	CodeHash            string
	ClientID            uuid.UUID
	UserID              uuid.UUID
	RedirectURI         string
	Scope               []string
	CodeChallenge       string // PKCE
	CodeChallengeMethod string // "plain" or "S256"
	Nonce               string
	ExpiresAt           time.Time
	Used                bool
	CreatedAt           time.Time
}

// IsExpired returns true if the authorization code has expired.
func (c *AuthorizationCode) IsExpired() bool {
	return time.Now().After(c.ExpiresAt)
}

// ValidatePKCE checks the provided verifier against the stored challenge.
func (c *AuthorizationCode) ValidatePKCE(verifier string) bool {
	if c.CodeChallenge == "" {
		return true // PKCE not required for this code
	}
	if verifier == "" {
		return false
	}
	switch c.CodeChallengeMethod {
	case "plain", "":
		return verifier == c.CodeChallenge
	case "S256":
		h := sha256.Sum256([]byte(verifier))
		encoded := base64.RawURLEncoding.EncodeToString(h[:])
		return encoded == c.CodeChallenge
	default:
		return false
	}
}

// IDTokenClaims holds the claims for an OIDC ID Token.
// The token itself is a JWT signed with RS256; this struct is for audit storage.
type IDTokenRecord struct {
	ID        uuid.UUID
	JTI       string
	UserID    uuid.UUID
	ClientID  uuid.UUID
	TenantID  uuid.UUID
	Scope     []string
	Claims    map[string]any
	ExpiresAt time.Time
	IssuedAt  time.Time
}

// OIDCDiscoveryConfig is the /.well-known/openid-configuration response.
type OIDCDiscoveryConfig struct {
	Issuer                            string   `json:"issuer"`
	AuthorizationEndpoint             string   `json:"authorization_endpoint"`
	TokenEndpoint                     string   `json:"token_endpoint"`
	UserInfoEndpoint                  string   `json:"userinfo_endpoint"`
	JwksURI                           string   `json:"jwks_uri"`
	RevocationEndpoint                string   `json:"revocation_endpoint"`
	IntrospectionEndpoint             string   `json:"introspection_endpoint"`
	ResponseTypesSupported            []string `json:"response_types_supported"`
	GrantTypesSupported               []string `json:"grant_types_supported"`
	SubjectTypesSupported             []string `json:"subject_types_supported"`
	IDTokenSigningAlgValues           []string `json:"id_token_signing_alg_values_supported"`
	ScopesSupported                   []string `json:"scopes_supported"`
	ClaimsSupported                   []string `json:"claims_supported"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported"`
	CodeChallengeMethodsSupported     []string `json:"code_challenge_methods_supported"`
	CheckSessionIFrame                string   `json:"check_session_iframe,omitempty"`
	BackchannelLogoutSupported        bool     `json:"backchannel_logout_supported"`
	EndSessionEndpoint                string   `json:"end_session_endpoint,omitempty"`
}

// JWKSKey represents a single key in a JWKS response.
type JWKSKey struct {
	KTY string `json:"kty"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	KID string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// JWKSResponse is the /oauth/jwks response.
type JWKSResponse struct {
	Keys []JWKSKey `json:"keys"`
}

// IDTokenIssuer is implemented by the auth service's token service.
// The OAuth service delegates JWT signing to avoid duplicating key management.
type IDTokenIssuer interface {
	IssueIDToken(claims IDTokenClaims) (string, error)
}

// IDTokenClaims holds the standard and custom claims for an OIDC ID Token.
type IDTokenClaims struct {
	Issuer    string
	Subject   string
	Audience  string
	Nonce     string
	ExpiresAt time.Time
	IssuedAt  time.Time
	Extra     map[string]any
}

// KeyProvider supplies the RSA keys for JWT signing and JWKS.
type KeyProvider interface {
	PublicKey() *rsa.PublicKey
	PrivateKey() *rsa.PrivateKey
	KeyID() string
}
