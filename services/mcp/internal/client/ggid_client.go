// Package client provides an HTTP client wrapper for GGID Gateway API calls.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// Client wraps HTTP calls to the GGID Gateway.
type Client struct {
	baseURL  string
	token    string
	tenantID string
	http     *http.Client
}

// New creates a new GGID Gateway client. tenantID is resolved from the
// GGID_TENANT_ID env var or the JWT claim at runtime — never hardcoded.
func New(baseURL, token, tenantID string) *Client {
	if tenantID == "" {
		tenantID = os.Getenv("GGID_TENANT_ID")
	}
	return &Client{
		baseURL:  baseURL,
		token:    token,
		tenantID: tenantID,
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GatewayURL returns the base gateway URL.
func (c *Client) GatewayURL() string {
	return c.baseURL
}

// Get performs a GET request and unmarshals JSON response.
func (c *Client) Get(ctx context.Context, path string, result any) error {
	return c.do(ctx, http.MethodGet, path, nil, result)
}

// Post performs a POST request with a JSON body and unmarshals JSON response.
func (c *Client) Post(ctx context.Context, path string, body, result any) error {
	return c.do(ctx, http.MethodPost, path, body, result)
}

// Put performs a PUT request.
func (c *Client) Put(ctx context.Context, path string, body, result any) error {
	return c.do(ctx, http.MethodPut, path, body, result)
}

// Delete performs a DELETE request.
func (c *Client) Delete(ctx context.Context, path string, result any) error {
	return c.do(ctx, http.MethodDelete, path, nil, result)
}

func (c *Client) do(ctx context.Context, method, path string, body, result any) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", c.tenantID)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}
