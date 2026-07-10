package service

import "errors"

// Service-level errors returned to callers.
var (
	ErrInvalidCredentials      = errors.New("invalid username or password")
	ErrAccountLocked          = errors.New("account is temporarily locked due to too many failed attempts")
	ErrRateLimited            = errors.New("rate limit exceeded")
	ErrSessionNotFound        = errors.New("session not found")
	ErrSessionExpired         = errors.New("session has expired")
	ErrCredentialAlreadyExists = errors.New("credential already exists")
)
