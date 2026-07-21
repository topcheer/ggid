// Package client provides the HTTP client for communicating with the GGID Gateway.
package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client is an authenticated HTTP client for the GGID API.
type Client struct {
	BaseURL    string
	TenantID   string
	Token      string
	HTTPClient *http.Client
}

// New creates a new API client.
func New(baseURL, tenantID, token string) *Client {
	return &Client{
		BaseURL:  strings.TrimRight(baseURL, "/"),
		TenantID: tenantID,
		Token:    token,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// APIError represents a structured error from the GGID API.
type APIError struct {
	StatusCode int
	Message    string
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Message)
}

// Do sends an HTTP request and returns the raw response body.
// It automatically attaches the Authorization header and tenant ID.
func (c *Client) Do(method, path string, body any) ([]byte, int, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.BaseURL+path, bodyReader)
	if err != nil {
		return nil, 0, fmt.Errorf("create request: %w", err)
	}

	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	if c.TenantID != "" {
		req.Header.Set("X-Tenant-ID", c.TenantID)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		msg := string(respBody)
		// Try to extract a structured error message.
		var errMap map[string]any
		if json.Unmarshal(respBody, &errMap) == nil {
			if e, ok := errMap["error"]; ok {
				if m, ok := e.(map[string]any); ok {
					if detail, ok := m["message"].(string); ok {
						msg = detail
					} else if detail, ok := m["detail"].(string); ok {
						msg = detail
					}
				} else if detail, ok := e.(string); ok {
					msg = detail
				}
			}
			if detail, ok := errMap["detail"].(string); ok && msg == string(respBody) {
				msg = detail
			}
			if detail, ok := errMap["message"].(string); ok && msg == string(respBody) {
				msg = detail
			}
		}
		return respBody, resp.StatusCode, &APIError{StatusCode: resp.StatusCode, Message: msg, Body: string(respBody)}
	}

	return respBody, resp.StatusCode, nil
}

// Get sends a GET request and unmarshals the JSON response.
func (c *Client) Get(path string, out any) error {
	body, _, err := c.Do(http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	if out != nil && len(body) > 0 {
		return json.Unmarshal(body, out)
	}
	return nil
}

// Post sends a POST request and unmarshals the JSON response.
func (c *Client) Post(path string, reqBody, out any) error {
	body, _, err := c.Do(http.MethodPost, path, reqBody)
	if err != nil {
		return err
	}
	if out != nil && len(body) > 0 {
		return json.Unmarshal(body, out)
	}
	return nil
}

// Put sends a PUT request and unmarshals the JSON response.
func (c *Client) Put(path string, reqBody, out any) error {
	body, _, err := c.Do(http.MethodPut, path, reqBody)
	if err != nil {
		return err
	}
	if out != nil && len(body) > 0 {
		return json.Unmarshal(body, out)
	}
	return nil
}

// Patch sends a PATCH request and unmarshals the JSON response.
func (c *Client) Patch(path string, reqBody, out any) error {
	body, _, err := c.Do(http.MethodPatch, path, reqBody)
	if err != nil {
		return err
	}
	if out != nil && len(body) > 0 {
		return json.Unmarshal(body, out)
	}
	return nil
}

// Delete sends a DELETE request.
func (c *Client) Delete(path string) error {
	_, _, err := c.Do(http.MethodDelete, path, nil)
	return err
}

// PostForm sends a form-encoded POST request (used for OAuth token endpoint).
func (c *Client) PostForm(path string, formData url.Values, out any) error {
	req, err := http.NewRequest(http.MethodPost, c.BaseURL+path, strings.NewReader(formData.Encode()))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	if c.TenantID != "" {
		req.Header.Set("X-Tenant-ID", c.TenantID)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return &APIError{StatusCode: resp.StatusCode, Message: string(respBody), Body: string(respBody)}
	}

	if out != nil && len(respBody) > 0 {
		return json.Unmarshal(respBody, out)
	}
	return nil
}
