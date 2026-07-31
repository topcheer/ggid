// Package domain defines the core entities for the OAuth/OIDC Service.
package domain

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ClientType determines whether a client can keep a secret.
type ClientType string

const (
	ClientTypeConfidential ClientType = "confidential"
	ClientTypePublic       ClientType = "public"
)

// IsValid returns true if the client type is recognised.
func (t ClientType) IsValid() bool {
	return t == ClientTypeConfidential || t == ClientTypePublic
}

// OAuthClient represents an application registered to use OAuth2/OIDC.
type OAuthClient struct {
	ID                      uuid.UUID      `json:"id"`
	TenantID                uuid.UUID      `json:"tenant_id"`
	ClientID                string         `json:"client_id"`                    // public identifier
	ClientSecretHash        string         `json:"client_secret_hash,omitempty"` // Argon2id hash; empty for public clients
	Name                    string         `json:"name"`
	Description             string         `json:"description,omitempty"`
	Type                    ClientType     `json:"type"`
	GrantTypes              []string       `json:"grant_types"`
	ResponseTypes           []string       `json:"response_types"`
	RedirectURIs            []string       `json:"redirect_uris"`
	PostLogoutRedirectURIs  []string       `json:"post_logout_redirect_uris,omitempty"`
	Scopes                  []string       `json:"scopes"`
	TokenEndpointAuthMethod string         `json:"token_endpoint_auth_method"`
	Metadata                map[string]any `json:"metadata,omitempty"`
	RequirePKCE             bool           `json:"require_pkce"`           // enforce PKCE for this client
	AuthMethods             []string       `json:"auth_methods,omitempty"` // allowed auth: password, passkey, sms_otp, email_otp
	Enabled                 bool           `json:"enabled"`
	CreatedAt               time.Time      `json:"created_at"`
	UpdatedAt               time.Time      `json:"updated_at"`
}

// IsConfidential returns true for confidential clients.
func (c *OAuthClient) IsConfidential() bool { return c.Type == ClientTypeConfidential }

// IsPublic returns true for public clients.
func (c *OAuthClient) IsPublic() bool { return c.Type == ClientTypePublic }

// RequiresPKCE returns true if PKCE should be enforced (public clients or RequirePKCE flag).
func (c *OAuthClient) RequiresPKCE() bool { return c.RequirePKCE || c.IsPublic() }

// GetAuthMethods returns the allowed authentication methods.
// Defaults to ["password"] if not configured (backward compatible).
func (c *OAuthClient) GetAuthMethods() []string {
	if len(c.AuthMethods) == 0 {
		return []string{"password"}
	}
	return c.AuthMethods
}

// FAPI2_0 returns true if the client is configured for FAPI 2.0 profile.
// The value is stored in client Metadata["fapi_2_0"].
func (c *OAuthClient) FAPI2_0() bool {
	if c.Metadata == nil {
		return false
	}
	v, ok := c.Metadata["fapi_2_0"].(bool)
	return ok && v
}

// SetFAPI2_0 sets the FAPI 2.0 flag in client Metadata.
func (c *OAuthClient) SetFAPI2_0(enabled bool) {
	if c.Metadata == nil {
		c.Metadata = make(map[string]any)
	}
	c.Metadata["fapi_2_0"] = enabled
}

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
	// FamilyID groups tokens created through rotation (RFC 6749 §10.4).
	// All tokens in a family descend from one initial grant; reuse of any
	// rotated token revokes the whole family.
	FamilyID string
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
	CodeChallengeMethod string // "S256" (OAuth 2.1 — plain deprecated)
	Nonce               string
	ExpiresAt           time.Time
	Used                bool
	CreatedAt           time.Time
	// NIST 800-63B AAL/AMR support
	AMR          []string  // Authentication Method References
	ACR          string    // Authentication Context Class Reference (AAL1/AAL2/AAL3)
	AuthTime     time.Time // When authentication occurred
	RequestedACR string    // acr_values from /authorize
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
	// RFC 7636 §4.1: code_verifier = 43*128unreserved — length 43..128,
	// characters [A-Z] [a-z] [0-9] "-" "." "_" "~".
	if len(verifier) < 43 || len(verifier) > 128 {
		return false
	}
	for i := 0; i < len(verifier); i++ {
		ch := verifier[i]
		if !(ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' ||
			ch == '-' || ch == '.' || ch == '_' || ch == '~') {
			return false
		}
	}
	switch c.CodeChallengeMethod {
	case "", "S256":
		// OAuth 2.1: S256 is the only supported method. Empty defaults to S256.
		h := sha256.Sum256([]byte(verifier))
		encoded := base64.RawURLEncoding.EncodeToString(h[:])
		// Use constant-time comparison to prevent timing side-channel.
		return subtle.ConstantTimeCompare([]byte(encoded), []byte(c.CodeChallenge)) == 1
	default:
		// "plain" and any other methods are not supported per OAuth 2.1
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
	Issuer                             string   `json:"issuer"`
	AuthorizationEndpoint              string   `json:"authorization_endpoint"`
	TokenEndpoint                      string   `json:"token_endpoint"`
	UserInfoEndpoint                   string   `json:"userinfo_endpoint"`
	JwksURI                            string   `json:"jwks_uri"`
	RevocationEndpoint                 string   `json:"revocation_endpoint"`
	IntrospectionEndpoint              string   `json:"introspection_endpoint"`
	ResponseTypesSupported             []string `json:"response_types_supported"`
	GrantTypesSupported                []string `json:"grant_types_supported"`
	SubjectTypesSupported              []string `json:"subject_types_supported"`
	IDTokenSigningAlgValues            []string `json:"id_token_signing_alg_values_supported"`
	ScopesSupported                    []string `json:"scopes_supported"`
	ClaimsSupported                    []string `json:"claims_supported"`
	TokenEndpointAuthMethodsSupported  []string `json:"token_endpoint_auth_methods_supported"`
	CodeChallengeMethodsSupported      []string `json:"code_challenge_methods_supported"`
	CheckSessionIFrame                 string   `json:"check_session_iframe,omitempty"`
	BackchannelLogoutSupported         bool     `json:"backchannel_logout_supported"`
	FrontchannelLogoutSupported        bool     `json:"frontchannel_logout_supported,omitempty"`
	EndSessionEndpoint                 string   `json:"end_session_endpoint,omitempty"`
	DeviceAuthorizationEndpoint        string   `json:"device_authorization_endpoint,omitempty"`
	RegistrationEndpoint               string   `json:"registration_endpoint,omitempty"`
	PushedAuthorizationRequestEndpoint string   `json:"pushed_authorization_request_endpoint,omitempty"`
	BackchannelAuthenticationEndpoint  string   `json:"backchannel_authentication_endpoint,omitempty"`
}

// JWKSKey represents a single key in a JWKS response.
type JWKSKey struct {
	KTY string `json:"kty"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	KID string `json:"kid"`
	N   string `json:"n,omitempty"`
	E   string `json:"e,omitempty"`
	X   string `json:"x,omitempty"`
	Y   string `json:"y,omitempty"`
	Crv string `json:"crv,omitempty"`
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
