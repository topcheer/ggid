package service

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/ggid/ggid/pkg/errors"
)

// --- OIDC RP-Initiated Logout ---

// RPInitiatedLogoutRequest holds parameters for the RP-initiated logout endpoint.
type RPInitiatedLogoutRequest struct {
	IDTokenHint            string // ID token previously issued to the RP
	ClientID               string // optional: identifies the RP
	PostLogoutRedirectURI  string // registered post-logout redirect target
	State                  string // opaque value echoed back in redirect
	SessionID              string // optional session identifier
}

// RPInitiatedLogoutResult indicates the result of the logout operation.
type RPInitiatedLogoutResult struct {
	RedirectURL string // URL to redirect the user-agent to (empty if no redirect)
	Subject     string // subject from the id_token_hint (for audit)
	Revoked     bool   // whether the session was revoked
}

// RPInitiatedLogout implements OIDC RP-Initiated Logout 1.0.
// Validates id_token_hint and post_logout_redirect_uri, revokes the session,
// and returns the redirect URL.
func (s *OAuthService) RPInitiatedLogout(req *RPInitiatedLogoutRequest) (*RPInitiatedLogoutResult, error) {
	result := &RPInitiatedLogoutResult{}

	// 1. Parse id_token_hint if provided to extract subject.
	if req.IDTokenHint != "" {
		claims, err := s.ParseAccessToken(req.IDTokenHint)
		if err == nil {
			result.Subject = getStringClaim(claims, "sub")
		}
	}

	// 2. Revoke the session (mark subject for backchannel logout).
	if result.Subject != "" {
		s.BackchannelLogout(result.Subject)
		result.Revoked = true
	}

	// 3. Validate post_logout_redirect_uri if provided.
	if req.PostLogoutRedirectURI != "" {
		// Validate that it's a proper URL.
		u, err := url.Parse(req.PostLogoutRedirectURI)
		if err != nil || u.Scheme == "" || u.Host == "" {
			return nil, errors.InvalidArgument("invalid post_logout_redirect_uri")
		}

		// SECURITY: Only allow HTTPS scheme (or localhost for development).
		if u.Scheme != "https" && u.Host != "localhost" && !strings.HasPrefix(u.Host, "localhost:") {
			return nil, errors.InvalidArgument("post_logout_redirect_uri must use HTTPS")
		}

		// SECURITY: validate against client's registered URIs.
		// Must have either id_token_hint or client_id to identify the client.
		clientID := ""
		if req.IDTokenHint != "" {
			claims, _ := s.ParseAccessToken(req.IDTokenHint)
			clientID, _ = claims["aud"].(string)
		}
		if clientID == "" {
			clientID = req.ClientID
		}
		if clientID == "" {
			return nil, errors.InvalidArgument("client_id or id_token_hint required for post_logout_redirect")
		}
		client, cerr := s.GetClient(context.Background(), clientID)
		if cerr != nil || client == nil {
			return nil, errors.InvalidArgument("client not found — cannot validate post_logout_redirect_uri")
		}
		// Check registered post_logout_redirect_uris; fall back to redirect_uris.
		registeredURIs := client.PostLogoutRedirectURIs
		if len(registeredURIs) == 0 {
			registeredURIs = client.RedirectURIs
		}
		matched := false
		for _, registered := range registeredURIs {
			if registered == req.PostLogoutRedirectURI {
				matched = true
				break
			}
		}
		if !matched {
			return nil, errors.InvalidArgument("post_logout_redirect_uri not registered for this client")
		}

		if req.State != "" {
			q := u.Query()
			q.Set("state", req.State)
			u.RawQuery = q.Encode()
		}
		result.RedirectURL = u.String()
	}

	return result, nil
}

// --- OIDC Back-Channel Logout (RFC 8417) endpoint logic ---

// BackchannelLogoutEndpoint handles POST /oauth/backchannel_logout.
// Accepts a logout_token JWT, validates it per RFC 8417, and revokes the session.
func (s *OAuthService) BackchannelLogoutEndpoint(logoutToken string) error {
	if logoutToken == "" {
		return fmt.Errorf("logout_token is required")
	}

	claims, err := s.ParseBackchannelLogoutToken(logoutToken)
	if err != nil {
		return err
	}

	// Extract subject and revoke session.
	sub := ""
	if v, ok := claims["sub"].(string); ok {
		sub = v
	}
	if sub == "" {
		if sid, ok := claims["sid"].(string); ok {
			sub = sid
		}
	}

	if sub != "" {
		s.BackchannelLogout(sub)
	}

	return nil
}
