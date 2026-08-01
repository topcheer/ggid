package service

// Client credentials grant methods for OAuthService.
// Extracted from oauth_service.go.

import (
	"fmt"
	"context"
	"time"

	"github.com/ggid/ggid/pkg/errors"
	"github.com/google/uuid"
	pkgcrypto "github.com/ggid/ggid/pkg/crypto"
	"github.com/golang-jwt/jwt/v5"
)
func (s *OAuthService) ClientCredentials(ctx context.Context, req *ClientCredentialsRequest) (*TokenResponse, error) {
	// 1. Look up the client.
	client, err := s.clientRepo.GetClientByID(ctx, req.TenantID, req.ClientID)
	if err != nil {
		return nil, errors.Unauthenticated("client authentication failed")
	}

	// 2. SECURITY: client_credentials requires confidential clients (RFC 6749 §4.4.1).
	if !client.IsConfidential() {
		return nil, errors.Unauthenticated("client_credentials grant requires a confidential client")
	}
	ok, _ := pkgcrypto.VerifyPassword(req.ClientSecret, client.ClientSecretHash)
	if !ok {
		return nil, errors.Unauthenticated("invalid client credentials")
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
	// For M2M, the client's configured scopes serve as the permissions claim.
	// Security: requested scopes must be intersected with client's allowed
	// scopes to prevent scope escalation (e.g. client requests "platform:admin"
	// when only "audit:read" is configured).
	clientPermissions := client.Scopes
	finalScopes := client.Scopes
	if len(req.Scope) > 0 {
		finalScopes = intersectScopes(req.Scope, client.Scopes)
		clientPermissions = finalScopes
	}
	accessToken, expiresIn, err := s.issueClientAccessToken(req.TenantID, resolveAudience(req.Audience, client.ClientID), client.ClientID, joinScopes(finalScopes), clientPermissions)
	if err != nil {
		return nil, err
	}

	return &TokenResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   expiresIn,
		Scope:       joinScopes(finalScopes), // FIX: Return granted scopes, not requested
	}, nil
}

// issueClientAccessToken issues a JWT for M2M (client_credentials) flows.
// Unlike issueAccessToken, this does NOT query user_roles — instead it uses
// the client's configured permissions directly.
func (s *OAuthService) issueClientAccessToken(tenantID uuid.UUID, audience, clientID, scope string, permissions []string) (string, int, error) {
	now := time.Now()
	expiresAt := now.Add(15 * time.Minute)

	claimsMap := jwt.MapClaims{
		"iss":         s.issuer,
		"sub":         uuid.Nil.String(),
		"aud":         audience,
		"iat":         now.Unix(),
		"exp":         expiresAt.Unix(),
		"jti":         uuid.New().String(),
		"tenant_id":   tenantID.String(),
		"scope":       scope,
		"permissions": permissions,
		"roles":       []string{}, // M2M clients have no roles
	}

	token := jwt.NewWithClaims(s.signingMethod(), claimsMap)
	token.Header["kid"] = s.keyProvider.Metadata().KeyID

	signed, err := token.SignedString(s.keyProvider.Signer())
	if err != nil {
		return "", 0, fmt.Errorf("sign access token: %w", err)
	}
	return signed, int(expiresAt.Sub(now).Seconds()), nil
}

// PasswordGrantRequest holds parameters for the password grant (RFC 6749 §4.3).
type PasswordGrantRequest struct {
	TenantID     uuid.UUID
	Username     string
	Password     string
	ClientID     string
	ClientSecret string // required for confidential clients (RFC 6749 §4.3.2)
	Scope        []string
	Audience     string // optional target audience (defaults to client_id)
	MFACode      string // optional TOTP code for users with MFA enrolled
	BackupCode   string // optional single-use backup code
}

// PasswordGrant authenticates a user with username/password and issues tokens.
// This is the unified token issuance path after issuer unification — the OAuth
// service is the sole token issuer.
