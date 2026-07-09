// Package errors provides unified error types for GGID services.
// Errors are mapped to HTTP and gRPC status codes consistently.
package errors

import (
	"errors"
	"fmt"
)

// ErrorCode represents a categorized error type.
type ErrorCode string

const (
	ErrInternal          ErrorCode = "internal"
	ErrNotFound          ErrorCode = "not_found"
	ErrAlreadyExists     ErrorCode = "already_exists"
	ErrInvalidArgument   ErrorCode = "invalid_argument"
	ErrUnauthenticated   ErrorCode = "unauthenticated"
	ErrPermissionDenied  ErrorCode = "permission_denied"
	ErrResourceExhausted ErrorCode = "resource_exhausted"
	ErrFailedPrecondition ErrorCode = "failed_precondition"
)

// GGIDError is the canonical error type used across all services.
type GGIDError struct {
	Code    ErrorCode
	Message string
	Detail  string
	Cause   error
}

func (e *GGIDError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *GGIDError) Unwrap() error { return e.Cause }

// New creates a new GGIDError.
func New(code ErrorCode, message string) *GGIDError {
	return &GGIDError{Code: code, Message: message}
}

// Wrap wraps an existing error with additional context.
func Wrap(code ErrorCode, message string, cause error) *GGIDError {
	return &GGIDError{Code: code, Message: message, Cause: cause}
}

// As GGIDError extracts a GGIDError from a standard error.
func AsGGIDError(err error) (*GGIDError, bool) {
	var ge *GGIDError
	if errors.As(err, &ge) {
		return ge, true
	}
	return nil, false
}

// Common error constructors
func NotFound(resource, id string) *GGIDError {
	return New(ErrNotFound, fmt.Sprintf("%s not found: %s", resource, id))
}

func AlreadyExists(resource, id string) *GGIDError {
	return New(ErrAlreadyExists, fmt.Sprintf("%s already exists: %s", resource, id))
}

func InvalidArgument(msg string) *GGIDError {
	return New(ErrInvalidArgument, msg)
}

func Unauthenticated(msg string) *GGIDError {
	return New(ErrUnauthenticated, msg)
}

func PermissionDenied(msg string) *GGIDError {
	return New(ErrPermissionDenied, msg)
}

func Internal(msg string, cause error) *GGIDError {
	return Wrap(ErrInternal, msg, cause)
}
