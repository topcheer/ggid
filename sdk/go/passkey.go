package ggid

import (
	"context"
	"fmt"
)

// PasskeyBeginRegistration initiates WebAuthn/Passkey registration.
// Returns the server challenge (credential creation options).
//
// Usage:
//
//	opts, _ := client.PasskeyBeginRegistration(ctx, token, "my-passkey")
//	// Send opts to browser, call navigator.credentials.create()
func (c *Client) PasskeyBeginRegistration(ctx context.Context, accessToken, deviceName string) (map[string]any, error) {
	var result map[string]any
	err := c.post(ctx, "/api/v1/auth/mfa/enroll", map[string]string{
		"type": "webauthn",
		"name": deviceName,
	}, &result)
	if err != nil {
		return nil, fmt.Errorf("passkey begin registration: %w", err)
	}
	return result, nil
}

// PasskeyFinishRegistration completes WebAuthn/Passkey registration.
// The attestationResponse is the result from navigator.credentials.create().
func (c *Client) PasskeyFinishRegistration(ctx context.Context, accessToken, deviceID, attestationResponse string) error {
	return c.post(ctx, "/api/v1/auth/mfa/verify", map[string]string{
		"device_id": deviceID,
		"code": attestationResponse,
	}, nil)
}

// PasskeyBeginLogin initiates WebAuthn/Passkey authentication.
// Returns the server challenge (credential request options).
func (c *Client) PasskeyBeginLogin(ctx context.Context, username string) (map[string]any, error) {
	var result map[string]any
	err := c.post(ctx, "/api/v1/auth/webauthn/login/begin", map[string]string{
		"username": username,
	}, &result)
	if err != nil {
		return nil, fmt.Errorf("passkey begin login: %w", err)
	}
	return result, nil
}

// PasskeyFinishLogin completes WebAuthn/Passkey authentication.
// The assertionResponse is the result from navigator.credentials.get().
func (c *Client) PasskeyFinishLogin(ctx context.Context, assertionResponse string) (map[string]any, error) {
	var result map[string]any
	err := c.post(ctx, "/api/v1/auth/webauthn/login/finish", map[string]string{
		"assertion": assertionResponse,
	}, &result)
	if err != nil {
		return nil, fmt.Errorf("passkey finish login: %w", err)
	}
	return result, nil
}
