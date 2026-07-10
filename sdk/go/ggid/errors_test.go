package ggid

import (
	"errors"
	"net/http"
	"testing"
)

func TestAPIError_Error(t *testing.T) {
	tests := []struct {
		name   string
		err    *APIError
		expect string
	}{
		{
			name:   "with code",
			err:    NewAPIError(401, "invalid credentials"),
			expect: "ggid: unauthorized (status 401): invalid credentials",
		},
		{
			name:   "without code (unknown)",
			err:    NewAPIError(418, "I'm a teapot"),
			expect: "ggid: unknown (status 418): I'm a teapot",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expect {
				t.Errorf("Error() = %q, want %q", got, tt.expect)
			}
		})
	}
}

func TestAPIError_Is(t *testing.T) {
	tests := []struct {
		name     string
		err      *APIError
		target   error
		expected bool
	}{
		{"400 is ErrBadRequest", NewAPIError(400, "bad"), ErrBadRequest, true},
		{"401 is ErrUnauthorized", NewAPIError(401, "unauth"), ErrUnauthorized, true},
		{"403 is ErrForbidden", NewAPIError(403, "forbidden"), ErrForbidden, true},
		{"404 is ErrNotFound", NewAPIError(404, "not found"), ErrNotFound, true},
		{"409 is ErrConflict", NewAPIError(409, "conflict"), ErrConflict, true},
		{"429 is ErrRateLimited", NewAPIError(429, "rate"), ErrRateLimited, true},
		{"500 is ErrInternalServerError", NewAPIError(500, "server"), ErrInternalServerError, true},
		{"502 is ErrInternalServerError", NewAPIError(502, "bad gateway"), ErrInternalServerError, true},
		{"503 is ErrInternalServerError", NewAPIError(503, "unavailable"), ErrInternalServerError, true},
		{"401 is NOT ErrNotFound", NewAPIError(401, "unauth"), ErrNotFound, false},
		{"404 is NOT ErrUnauthorized", NewAPIError(404, "not found"), ErrUnauthorized, false},
		{"200 is NOT ErrBadRequest", NewAPIError(200, "ok"), ErrBadRequest, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := errors.Is(tt.err, tt.target); got != tt.expected {
				t.Errorf("errors.Is() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAPIError_As(t *testing.T) {
	err := NewAPIError(404, "user not found")
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatal("errors.As should return true")
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", apiErr.StatusCode)
	}
	if apiErr.Code != "not_found" {
		t.Errorf("Code = %q, want %q", apiErr.Code, "not_found")
	}
}

func TestAPIError_IsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		expected bool
	}{
		{"400 not retryable", 400, false},
		{"401 not retryable", 401, false},
		{"404 not retryable", 404, false},
		{"429 retryable", 429, true},
		{"500 retryable", 500, true},
		{"502 retryable", 502, true},
		{"503 retryable", 503, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewAPIError(tt.status, "test")
			if got := err.IsRetryable(); got != tt.expected {
				t.Errorf("IsRetryable() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestStatusToCode(t *testing.T) {
	tests := []struct {
		status   int
		expected string
	}{
		{http.StatusBadRequest, "bad_request"},
		{http.StatusUnauthorized, "unauthorized"},
		{http.StatusForbidden, "forbidden"},
		{http.StatusNotFound, "not_found"},
		{http.StatusConflict, "conflict"},
		{http.StatusTooManyRequests, "rate_limited"},
		{http.StatusInternalServerError, "server_error"},
		{http.StatusBadGateway, "server_error"},
		{http.StatusServiceUnavailable, "server_error"},
		{418, "unknown"},
		{200, "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := statusToCode(tt.status); got != tt.expected {
				t.Errorf("statusToCode(%d) = %q, want %q", tt.status, got, tt.expected)
			}
		})
	}
}

func TestNewAPIError(t *testing.T) {
	err := NewAPIError(409, "email already exists")
	if err.StatusCode != 409 {
		t.Errorf("StatusCode = %d, want 409", err.StatusCode)
	}
	if err.Code != "conflict" {
		t.Errorf("Code = %q, want %q", err.Code, "conflict")
	}
	if err.Message != "email already exists" {
		t.Errorf("Message = %q, want %q", err.Message, "email already exists")
	}
}

// Test that APIError works with errors.Is in a chain
func TestAPIError_ChainedIs(t *testing.T) {
	wrapped := errors.Join(ErrUnauthorized, NewAPIError(401, "token expired"))
	if !errors.Is(wrapped, ErrUnauthorized) {
		t.Error("chained errors.Is should match ErrUnauthorized")
	}
}
