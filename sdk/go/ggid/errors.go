package ggid

import (
	"errors"
	"fmt"
	"net/http"
)

// APIError represents an error returned by the GGID API.
// It implements the error interface and supports errors.Is/As.
type APIError struct {
	StatusCode int
	Code       string
	Message    string
}

func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("ggid: %s (status %d): %s", e.Code, e.StatusCode, e.Message)
	}
	return fmt.Sprintf("ggid: API error (status %d): %s", e.StatusCode, e.Message)
}

// NewAPIError creates an APIError from an HTTP status code and message.
func NewAPIError(statusCode int, message string) *APIError {
	return &APIError{
		StatusCode: statusCode,
		Code:       statusToCode(statusCode),
		Message:    message,
	}
}

// Common sentinel errors for errors.Is() support.
var (
	// ErrUnauthorized is returned when the API responds with 401.
	ErrUnauthorized = errors.New("ggid: unauthorized")
	// ErrForbidden is returned when the API responds with 403.
	ErrForbidden = errors.New("ggid: forbidden")
	// ErrNotFound is returned when the API responds with 404.
	ErrNotFound = errors.New("ggid: not found")
	// ErrConflict is returned when the API responds with 409.
	ErrConflict = errors.New("ggid: conflict")
	// ErrRateLimited is returned when the API responds with 429.
	ErrRateLimited = errors.New("ggid: rate limited")
	// ErrInternalServerError is returned when the API responds with 5xx.
	ErrInternalServerError = errors.New("ggid: internal server error")
	// ErrBadRequest is returned when the API responds with 400.
	ErrBadRequest = errors.New("ggid: bad request")
)

// Is enables errors.Is(err, ErrUnauthorized) to match APIError with 401.
func (e *APIError) Is(target error) bool {
	switch target {
	case ErrBadRequest:
		return e.StatusCode == http.StatusBadRequest
	case ErrUnauthorized:
		return e.StatusCode == http.StatusUnauthorized
	case ErrForbidden:
		return e.StatusCode == http.StatusForbidden
	case ErrNotFound:
		return e.StatusCode == http.StatusNotFound
	case ErrConflict:
		return e.StatusCode == http.StatusConflict
	case ErrRateLimited:
		return e.StatusCode == http.StatusTooManyRequests
	case ErrInternalServerError:
		return e.StatusCode >= 500 && e.StatusCode < 600
	}
	return false
}

// As enables errors.As(err, &apiErr) to extract the APIError.
func (e *APIError) As(target interface{}) bool {
	if t, ok := target.(**APIError); ok {
		*t = e
		return true
	}
	return false
}

// IsRetryable returns true for errors that may succeed on retry.
func (e *APIError) IsRetryable() bool {
	return e.StatusCode >= 500 || e.StatusCode == http.StatusTooManyRequests
}

// statusToCode converts an HTTP status code to a machine-readable error code.
func statusToCode(statusCode int) string {
	switch statusCode {
	case http.StatusBadRequest:
		return "bad_request"
	case http.StatusUnauthorized:
		return "unauthorized"
	case http.StatusForbidden:
		return "forbidden"
	case http.StatusNotFound:
		return "not_found"
	case http.StatusConflict:
		return "conflict"
	case http.StatusTooManyRequests:
		return "rate_limited"
	default:
		if statusCode >= 500 {
			return "server_error"
		}
		return "unknown"
	}
}
