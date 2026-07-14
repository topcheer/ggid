package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// TokenResponse represents the OAuth token endpoint response.
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// GGIDClient manages client_credentials token acquisition and caching.
type GGIDClient struct {
	ggidURL    string
	tenantID   string
	clientID   string
	clientSecret string
	httpClient *http.Client

	mu          sync.Mutex
	cachedToken string
	expiresAt   time.Time
}

// NewGGIDClient creates a new client_credentials client.
func NewGGIDClient(ggidURL, tenantID, clientID, clientSecret string) *GGIDClient {
	return &GGIDClient{
		ggidURL:      ggidURL,
		tenantID:     tenantID,
		clientID:     clientID,
		clientSecret: clientSecret,
		httpClient:   &http.Client{Timeout: 15 * time.Second},
	}
}

// GetToken returns a valid access token, refreshing via client_credentials if expired.
func (c *GGIDClient) GetToken() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Return cached token if still valid (with 30s buffer)
	if c.cachedToken != "" && time.Now().Before(c.expiresAt.Add(-30*time.Second)) {
		return c.cachedToken, nil
	}

	// Request new token via client_credentials grant
	form := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {c.clientID},
		"client_secret": {c.clientSecret},
	}

	req, err := http.NewRequest("POST", c.ggidURL+"/api/v1/oauth/token", bytes.NewBufferString(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Tenant-ID", c.tenantID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("parse token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("token response missing access_token")
	}

	c.cachedToken = tokenResp.AccessToken
	c.expiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	return c.cachedToken, nil
}
