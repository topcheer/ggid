package middleware

import (
	"context"
	"sync"
	"time"
)

// APIKeyRotationConfig controls the grace period for rotated API keys.
type APIKeyRotationConfig struct {
	// GracePeriod is how long an expired key remains valid after its
	// expiration time. This allows zero-downtime rotation.
	GracePeriod time.Duration
}

// DefaultRotationConfig returns the default rotation config with a 7-day grace period.
func DefaultRotationConfig() *APIKeyRotationConfig {
	return &APIKeyRotationConfig{
		GracePeriod: 7 * 24 * time.Hour, // 7 days
	}
}

// RotatableAPIKeyValidator wraps a MemoryAPIKeyValidator with rotation support.
// Keys that have expired but are still within the grace period are accepted
// and marked with a rotation warning header.
type RotatableAPIKeyValidator struct {
	mu       sync.RWMutex
	keys     map[string]*rotatableKeyEntry
	gracePd  time.Duration
}

type rotatableKeyEntry struct {
	tenantID   string
	userID     string
	scopes     []string
	active     bool
	expires    time.Time
	rotated    bool
	replacedBy string // new key that replaced this one
}

// NewRotatableAPIKeyValidator creates a validator with the given grace period.
func NewRotatableAPIKeyValidator(gracePeriod time.Duration) *RotatableAPIKeyValidator {
	if gracePeriod <= 0 {
		gracePeriod = 7 * 24 * time.Hour
	}
	return &RotatableAPIKeyValidator{
		keys:    make(map[string]*rotatableKeyEntry),
		gracePd: gracePeriod,
	}
}

// AddKey registers a new API key.
func (v *RotatableAPIKeyValidator) AddKey(key, tenantID, userID string, scopes []string, expires time.Time) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.keys[key] = &rotatableKeyEntry{
		tenantID: tenantID,
		userID:   userID,
		scopes:   scopes,
		active:   true,
		expires:  expires,
	}
}

// RotateKey marks an old key as rotated and registers its replacement.
// The old key remains valid for the grace period.
func (v *RotatableAPIKeyValidator) RotateKey(oldKey, newKey, tenantID, userID string, scopes []string, expires time.Time) {
	v.mu.Lock()
	defer v.mu.Unlock()

	// Mark old key as rotated
	if entry, ok := v.keys[oldKey]; ok {
		entry.rotated = true
		entry.replacedBy = newKey
		entry.active = false
	}

	// Add new key
	v.keys[newKey] = &rotatableKeyEntry{
		tenantID: tenantID,
		userID:   userID,
		scopes:   scopes,
		active:   true,
		expires:  expires,
	}
}

// Validate checks the key against active keys, then falls back to
// rotated keys within the grace period.
func (v *RotatableAPIKeyValidator) Validate(_ context.Context, key string) (tenantID, userID string, scopes []string, err error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	now := time.Now()
	entry, ok := v.keys[key]
	if !ok {
		return "", "", nil, errInvalidAPIKey
	}

	// Active key — check normal expiry
	if entry.active {
		if now.After(entry.expires) {
			return "", "", nil, errInvalidAPIKey
		}
		return entry.tenantID, entry.userID, entry.scopes, nil
	}

	// Rotated/expired key — check grace period
	if entry.rotated {
		graceExpiry := entry.expires.Add(v.gracePd)
		if now.After(graceExpiry) {
			return "", "", nil, errInvalidAPIKey
		}
		// Still valid during grace period
		return entry.tenantID, entry.userID, entry.scopes, nil
	}

	return "", "", nil, errInvalidAPIKey
}

// IsRotated returns true if the key has been rotated (superseded by a new key).
func (v *RotatableAPIKeyValidator) IsRotated(key string) bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	entry, ok := v.keys[key]
	return ok && entry.rotated
}

// ReplacementKey returns the new key that replaced the given old key.
func (v *RotatableAPIKeyValidator) ReplacementKey(key string) string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	entry, ok := v.keys[key]
	if !ok {
		return ""
	}
	return entry.replacedBy
}
