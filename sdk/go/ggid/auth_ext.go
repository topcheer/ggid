package ggid

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// --- Extended Auth Methods ---

// IntrospectToken queries the OAuth introspection endpoint for token metadata.
func (c *Client) IntrospectToken(ctx context.Context, token string) (*IntrospectionResult, error) {
	body := map[string]string{"token": token}
	resp, err := c.do(ctx, http.MethodPost, "/api/v1/oauth/introspect", body, "")
	if err != nil {
		return nil, err
	}
	var result IntrospectionResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("parse introspection result: %w", err)
	}
	return &result, nil
}

// Logout revokes the current access token and associated refresh token.
func (c *Client) Logout(ctx context.Context, accessToken string) error {
	_, err := c.do(ctx, http.MethodPost, "/api/v1/auth/logout", nil, accessToken)
	return err
}

// VerifyToken is already defined in api.go with JWT verifier support.

// Impersonate generates a delegated token for impersonating another user.
// Requires admin privileges. The impersonated token has an "impersonator" claim.
func (c *Client) Impersonate(ctx context.Context, adminToken, targetUserID, reason string) (*TokenSet, error) {
	body := map[string]string{
		"user_id": targetUserID,
		"reason":  reason,
	}
	resp, err := c.do(ctx, http.MethodPost, "/api/v1/auth/impersonate", body, adminToken)
	if err != nil {
		return nil, err
	}
	var tokens TokenSet
	if err := json.Unmarshal(resp, &tokens); err != nil {
		return nil, fmt.Errorf("parse impersonation tokens: %w", err)
	}
	return &tokens, nil
}

// RevokeImpersonation revokes an active impersonation session.
func (c *Client) RevokeImpersonation(ctx context.Context, adminToken, sessionID string) error {
	_, err := c.do(ctx, http.MethodPost, fmt.Sprintf("/api/v1/auth/impersonate/%s/revoke", sessionID), nil, adminToken)
	return err
}

// RevokeAllUserSessions revokes all active sessions for a user (admin operation).
func (c *Client) RevokeAllUserSessions(ctx context.Context, adminToken, userID string) error {
	_, err := c.do(ctx, http.MethodPost, fmt.Sprintf("/api/v1/users/%s/sessions/revoke", userID), nil, adminToken)
	return err
}

// CheckSoD checks if a user's role set violates any Separation of Duties rules.
func (c *Client) CheckSoD(ctx context.Context, token, userID string, roles []string) ([]SoDViolation, error) {
	body := map[string]any{
		"user_id": userID,
		"roles":   roles,
	}
	resp, err := c.do(ctx, http.MethodPost, "/api/v1/policies/sod/check", body, token)
	if err != nil {
		return nil, err
	}
	var violations []SoDViolation
	if err := json.Unmarshal(resp, &violations); err != nil {
		return nil, fmt.Errorf("parse SoD violations: %w", err)
	}
	return violations, nil
}

// --- Types ---

// IntrospectionResult represents the OAuth token introspection response (RFC 7662).
type IntrospectionResult struct {
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
	Jti       string `json:"jti,omitempty"`
}

// SoDViolation represents a Separation of Duties conflict.
type SoDViolation struct {
	UserID string   `json:"user_id"`
	Roles  []string `json:"roles"`
	Reason string   `json:"reason"`
}
