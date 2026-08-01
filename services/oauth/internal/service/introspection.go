package service

// Token introspection (RFC 7662) and UserInfo (OIDC) methods.
// Extracted from oauth_service.go.

import (
	"fmt"
	"strings"
	"github.com/golang-jwt/jwt/v5"
)
func (s *OAuthService) GetUserInfo(tokenStr string) (*UserInfoResponse, error) {
	// SECURITY: Check revocation before serving claims.
	if s.IsTokenRevoked(tokenStr) {
		return nil, fmt.Errorf("token has been revoked")
	}
	claims, err := s.ParseAccessToken(tokenStr)
	if err != nil {
		return nil, err
	}

	// Parse scopes from the token to enforce OIDC §5.4 scope-based claims.
	tokenScope := getStringClaim(claims, "scope")
	scopeSet := make(map[string]bool)
	for _, sc := range strings.Fields(tokenScope) {
		scopeSet[sc] = true
	}

	resp := &UserInfoResponse{
		Sub: getStringClaim(claims, "sub"),
	}

	// If no scope claim, return all claims (backward compatibility for
	// tokens without explicit scope, and internal service tokens).
	noScopeFilter := tokenScope == ""

	// Profile scope: name, picture (OIDC §5.4)
	if noScopeFilter || scopeSet["profile"] {
		resp.Name = getStringClaim(claims, "name")
		resp.Picture = getStringClaim(claims, "picture")
	}

	// Email scope: email, email_verified (OIDC §5.4)
	if noScopeFilter || scopeSet["email"] {
		resp.Email = getStringClaim(claims, "email")
		resp.EmailVerified = getBoolClaim(claims, "email_verified")
	}

	// Tenant + roles always available to the token holder
	resp.TenantID = getStringClaim(claims, "tenant_id")
	resp.Roles = getStringSliceClaim(claims, "roles")
	resp.Groups = getStringSliceClaim(claims, "groups")
	resp.Permissions = getStringSliceClaim(claims, "permissions")
	resp.RiskLevel = getStringClaim(claims, "risk_level")
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
	UserID      string   `json:"user_id,omitempty"`
	TenantID    string   `json:"tenant_id,omitempty"`
	SessionID   string   `json:"session_id,omitempty"`
	DeviceID    string   `json:"device_id,omitempty"`
	RiskScore   int      `json:"risk_score,omitempty"`
	Roles       []string `json:"roles,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
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

	// RFC 7662 §2.2: client_id should be the issuing client (azp), not audience.
	introspectClientID := getStringClaim(claims, "azp")
	if introspectClientID == "" {
		introspectClientID = getStringClaim(claims, "client_id")
	}
	if introspectClientID == "" {
		introspectClientID = getStringClaim(claims, "aud") // backward fallback
	}

	resp := &IntrospectionResponse{
		Active:    true,
		TokenType: "Bearer",
		Sub:       getStringClaim(claims, "sub"),
		Aud:       getStringClaim(claims, "aud"),
		Iss:       getStringClaim(claims, "iss"),
		ClientID:  introspectClientID,
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
	resp.Roles = getStringSliceClaim(claims, "roles")
	resp.Permissions = getStringSliceClaim(claims, "permissions")
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

