package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// HTTPProviderConfig defines a fully custom HTTP webhook for SMS/Email sending.
// Users can configure any HTTP endpoint with templated request/response matching.
type HTTPProviderConfig struct {
	// URL is the target endpoint (e.g. https://api.custom-sms.com/send)
	URL string `json:"url"`
	// Method: POST (default), GET, PUT
	Method string `json:"method,omitempty"`
	// Headers: custom HTTP headers (e.g. Authorization, Content-Type)
	Headers map[string]string `json:"headers,omitempty"`
	// BodyTemplate: request body template with {{variables}}.
	// For SMS: {{phone}}, {{message}}
	// For Email: {{to}}, {{subject}}, {{body}}, {{from}}
	BodyTemplate string `json:"body_template,omitempty"`
	// ContentType: defaults to application/json
	ContentType string `json:"content_type,omitempty"`
	// Timeout seconds (default 10)
	TimeoutSec int `json:"timeout_sec,omitempty"`
	// SuccessCondition: how to determine if the send succeeded.
	SuccessCondition SuccessCondition `json:"success_condition"`
}

// SuccessCondition defines the logic to evaluate whether an HTTP response
// indicates success or failure.
type SuccessCondition struct {
	// ExpectedStatusCodes: HTTP status codes that indicate success (default [200, 201, 202])
	ExpectedStatusCodes []int `json:"expected_status_codes,omitempty"`
	// ResponseBodySuccessPath: JSON path in response body to check.
	// If set, the value at this path must be truthy (true / "ok" / "success" / 0-success).
	// e.g. "status" → checks response.body.status == "success"
	// e.g. "result.code" → checks nested field
	ResponseBodySuccessPath string `json:"response_body_success_path,omitempty"`
	// ResponseBodySuccessValue: the expected value at the path above.
	// If empty, any truthy value counts as success.
	ResponseBodySuccessValue string `json:"response_body_success_value,omitempty"`
}

// renderTemplate replaces {{variables}} in a template string with actual values.
func renderVars(template string, vars map[string]string) string {
	result := template
	for k, v := range vars {
		result = strings.ReplaceAll(result, "{{"+k+"}}", v)
	}
	return result
}

// executeHTTPProvider sends a request to a custom HTTP endpoint and evaluates
// the response against the success condition.
func executeHTTPProvider(cfg HTTPProviderConfig, vars map[string]string) error {
	return ExecuteHTTPProvider(cfg, vars)
}

// ExecuteHTTPProvider is the exported version for cross-package use.
func ExecuteHTTPProvider(cfg HTTPProviderConfig, vars map[string]string) error {
	// Apply defaults
	method := cfg.Method
	if method == "" {
		method = "POST"
	}
	contentType := cfg.ContentType
	if contentType == "" {
		contentType = "application/json"
	}
	timeout := cfg.TimeoutSec
	if timeout == 0 {
		timeout = 10
	}

	// Build request body from template
	var bodyReader io.Reader
	body := renderVars(cfg.BodyTemplate, vars)
	if body != "" {
		bodyReader = bytes.NewReader([]byte(body))
	}

	// Build request
	req, err := http.NewRequest(method, cfg.URL, bodyReader)
	if err != nil {
		return fmt.Errorf("build HTTP provider request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)
	for k, v := range cfg.Headers {
		req.Header.Set(k, renderVars(v, vars))
	}

	// Send
	client := &http.Client{Timeout: time.Duration(timeout) * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP provider request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	// Check status code
	if len(cfg.SuccessCondition.ExpectedStatusCodes) == 0 {
		// Default: 2xx is success
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("HTTP provider returned status %d: %s", resp.StatusCode, string(respBody))
		}
	} else {
		matched := false
		for _, code := range cfg.SuccessCondition.ExpectedStatusCodes {
			if resp.StatusCode == code {
				matched = true
				break
			}
		}
		if !matched {
			return fmt.Errorf("HTTP provider returned status %d (expected %v): %s",
				resp.StatusCode, cfg.SuccessCondition.ExpectedStatusCodes, string(respBody))
		}
	}

	// Check response body success path if configured
	if cfg.SuccessCondition.ResponseBodySuccessPath != "" {
		if !checkResponseBodySuccess(respBody, cfg.SuccessCondition) {
			return fmt.Errorf("HTTP provider response indicates failure: %s", string(respBody))
		}
	}

	return nil
}

// checkResponseBodySuccess navigates a JSON response body to find the value
// at the given path and checks if it matches the expected success value.
func checkResponseBodySuccess(body []byte, cond SuccessCondition) bool {
	var data interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		// Non-JSON response — skip body check, rely on status code only
		return true
	}

	// Navigate to the path
	current := data
	parts := strings.Split(cond.ResponseBodySuccessPath, ".")
	for _, part := range parts {
		m, ok := current.(map[string]interface{})
		if !ok {
			return false
		}
		current, ok = m[part]
		if !ok {
			return false
		}
	}

	// Check value
	if cond.ResponseBodySuccessValue != "" {
		strVal := fmt.Sprintf("%v", current)
		return strings.EqualFold(strVal, cond.ResponseBodySuccessValue)
	}

	// No expected value — check truthiness
	switch v := current.(type) {
	case bool:
		return v
	case string:
		lower := strings.ToLower(v)
		return lower == "ok" || lower == "success" || lower == "true" || lower == "sent"
	case float64:
		return v == 0 // some APIs use 0 for success
	case nil:
		return false
	default:
		return true
	}
}

// ExecuteEmailHTTPProvider sends an email via a custom HTTP endpoint.
// Variables available in templates: {{to}}, {{subject}}, {{body}}, {{from}}
func ExecuteEmailHTTPProvider(cfg HTTPProviderConfig, to, subject, body, from string) error {
	vars := map[string]string{
		"to":      to,
		"subject": subject,
		"body":    body,
		"from":    from,
	}
	return ExecuteHTTPProvider(cfg, vars)
}
