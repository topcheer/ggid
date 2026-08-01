package service

// Authorization code grant methods for OAuthService.
// Extracted from oauth_service.go.

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	pkgcrypto "github.com/ggid/ggid/pkg/crypto"
	"github.com/ggid/ggid/pkg/errors"
	"github.com/ggid/ggid/services/oauth/internal/domain"
	"github.com/google/uuid"
)
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

	plaintextCode, err := pkgcrypto.GenerateRandomToken(32)
	if err != nil {
		return "", errors.Internal("generate auth code", err)
	}

	// SECURITY: Intersect requested scopes with client's allowed scopes to prevent escalation.
	// If client has no scopes configured (empty), allow all requested (backward compat).
	if len(client.Scopes) > 0 {
		req.Scope = intersectScopes(req.Scope, client.Scopes)
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
		AMR:          computeAMR(req.AuthMethods),
		ACR:          computeACR(req.AuthMethods),
		AuthTime:     time.Now(),
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
	TenantID     uuid.UUID
	GrantType    string // "authorization_code"
	Code         string // the plaintext authorization code
	RedirectURI  string
	ClientID     string
	ClientSecret string // for confidential clients
	CodeVerifier string // PKCE code_verifier
	State        string // OAuth state parameter for CSRF validation
	Audience     string // optional RFC 8707/Auth0-style target audience for the access token
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
	//    RFC 6749 §2.3: If token_endpoint_auth_method is "none", skip secret.
	if client.IsConfidential() && client.TokenEndpointAuthMethod != "none" {
		ok, _ := pkgcrypto.VerifyPassword(req.ClientSecret, client.ClientSecretHash)
		if !ok {
			return nil, errors.Unauthenticated("invalid client credentials")
		}
	}

	// 3. Reject disabled clients — a client disabled after code issuance must
	//    not be able to exchange outstanding codes for tokens.
	if !client.Enabled {
		return nil, errors.InvalidArgument("client is disabled")
	}

	// 4. Consume the authorization code (atomically — prevents replay).
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
	audience := resolveAudience(req.Audience, client.ClientID)
	accessToken, expiresIn, err := s.issueAccessTokenWithAMR(code.UserID, code.TenantID, audience, oauthScopes, code.AMR, code.ACR, code.AuthTime, userAttrs)
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
	// Include AMR/ACR + at_hash + c_hash (OIDC Core §3.1.3.6) + auth_time.
	if contains(code.Scope, "openid") {
		accessTokenHash := sha256.Sum256([]byte(accessToken))
		atHash := base64.RawURLEncoding.EncodeToString(accessTokenHash[:16])
		// c_hash: hash of the authorization code
		codeHashBytes := sha256.Sum256([]byte(req.Code))
		cHash := base64.RawURLEncoding.EncodeToString(codeHashBytes[:16])
		idTokenOpts := &IDTokenOptions{
			AMR:      code.AMR,
			ACR:      code.ACR,
			AuthTime: code.AuthTime.Unix(),
			AtHash:   atHash,
			CHash:    cHash,
		}
		idToken, err := s.issueIDToken(code.UserID, code.TenantID, client.ClientID, code.Nonce, idTokenOpts)
		if err != nil {
			return nil, err
		}
		resp.IDToken = idToken
	}

	// 9. Issue a refresh token when the client requested offline_access and
	// is allowed the refresh_token grant. The token roots a new rotation
	// family (RFC 6749 §10.4) — without this, web clients can never reach
	// the refresh_token grant at all.
	if contains(code.Scope, "offline_access") && client.SupportsGrantType("refresh_token") {
		refreshPlain, err := s.issueRefreshTokenRecord(ctx, code.TenantID, client.ID, code.UserID, code.Scope, "")
		if err != nil {
			return nil, err
		}
		resp.RefreshToken = refreshPlain
	}

	return resp, nil
}

// issueRefreshTokenRecord creates and stores a refresh token record.
// When familyID is empty, the record roots a new rotation family at its
// own ID; otherwise it joins the given family.
func (s *OAuthService) issueRefreshTokenRecord(ctx context.Context, tenantID, clientID, userID uuid.UUID, scope []string, familyID string) (string, error) {
	refreshPlain, err := pkgcrypto.GenerateRandomToken(32)
	if err != nil {
		return "", errors.Internal("generate refresh token", err)
	}
	rec := &domain.RefreshTokenRecord{
		ID:        uuid.New(),
		TenantID:  tenantID,
		ClientID:  clientID,
		UserID:    userID,
		TokenHash: hashTokenSHA256(refreshPlain),
		Scope:     scope,
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour), // 30 days
		CreatedAt: time.Now(),
		FamilyID:  familyID,
	}
	if rec.FamilyID == "" {
		rec.FamilyID = rec.ID.String() // family root
	}
	if err := s.tokenRepo.StoreRefreshToken(ctx, rec); err != nil {
		return "", errors.Internal("store refresh token", err)
	}
	return refreshPlain, nil
}

// --- OIDC Discovery ---

