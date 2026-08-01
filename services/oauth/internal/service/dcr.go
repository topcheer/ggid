package service

// Dynamic Client Registration (RFC 7591) methods.
// Extracted from oauth_service.go.

import (
	"context"
	"strings"
	"time"

	"github.com/ggid/ggid/services/oauth/internal/domain"
	pkgcrypto "github.com/ggid/ggid/pkg/crypto"
	"github.com/ggid/ggid/pkg/errors"
	"github.com/ggid/ggid/pkg/tenant"
	"github.com/google/uuid"
)
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
	// SECURITY: filter requested scopes — DCR is a self-service endpoint.
	// Only allow standard OIDC scopes, never admin/system scopes.
	// Use case-insensitive check to prevent bypass via "Platform:admin" etc.
	requestedScopes := strings.Fields(req.Scope)
	safeScopes := []string{}
	for _, sc := range requestedScopes {
		lowerScope := strings.ToLower(sc)
		switch lowerScope {
		case "openid", "profile", "email", "offline_access", "address", "phone":
			safeScopes = append(safeScopes, sc)
		default:
			// Block admin, platform, system, tenant prefixed scopes (case-insensitive)
			if !strings.HasPrefix(lowerScope, "admin") && !strings.HasPrefix(lowerScope, "platform") &&
				!strings.HasPrefix(lowerScope, "system") && !strings.HasPrefix(lowerScope, "tenant") {
				safeScopes = append(safeScopes, sc)
			}
		}
	}
	scopes := safeScopes

	// SECURITY: Restrict DCR grant types to safe subset — prevent password grant abuse.
	safeGrants := map[string]bool{
		"authorization_code": true, "refresh_token": true, "client_credentials": true,
	}
	var filteredGrants []string
	for _, g := range req.GrantTypes {
		if safeGrants[g] {
			filteredGrants = append(filteredGrants, g)
		}
	}
	if len(filteredGrants) == 0 {
		filteredGrants = []string{"authorization_code", "refresh_token"}
	}

	client := &domain.OAuthClient{
		ID:                      uuid.New(),
		TenantID:                tc.TenantID,
		ClientID:                clientID,
		Name:                    defaultIfEmpty(req.ClientName, "Dynamic Client"),
		Type:                    domain.ClientTypeConfidential,
		GrantTypes:              filteredGrants,
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
		hash, err := pkgcrypto.HashPassword(plaintextSecret)
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
		RedirectURIs:            req.RedirectURIs,
		GrantTypes:              req.GrantTypes,
		ResponseTypes:           req.ResponseTypes,
		TokenEndpointAuthMethod: req.TokenEndpointAuthMethod,
		Scope:                   req.Scope,
	}, nil
}

// --- Token Exchange (RFC 8693) ---

// TokenExchangeRequestRFC8693 implements RFC 8693 token exchange parameters.
type TokenExchangeRequestRFC8693 struct {
	TenantID           uuid.UUID
	ClientID           string
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
//
// Deprecated: this legacy entry point previously performed no client
