package service

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/ggid/ggid/services/oauth/internal/domain"
)

// RotatingKeyProvider implements domain.KeyProvider with support for graceful
// key rotation. The old key remains valid for a configurable grace period
// after a new key is generated, allowing in-flight tokens to be verified.
type RotatingKeyProvider struct {
	mu       sync.RWMutex
	current  *rsa.PrivateKey
	currentID string
	previous *rsa.PrivateKey
	previousID string
	rotatedAt time.Time
	gracePeriod time.Duration
}

// NewRotatingKeyProvider creates a new key provider with an initial key.
func NewRotatingKeyProvider(initialKey *rsa.PrivateKey, gracePeriod time.Duration) *RotatingKeyProvider {
	if gracePeriod == 0 {
		gracePeriod = 24 * time.Hour
	}
	return &RotatingKeyProvider{
		current:    initialKey,
		currentID:  generateKeyID(initialKey),
		gracePeriod: gracePeriod,
	}
}

// PublicKey returns the current signing public key.
func (r *RotatingKeyProvider) PublicKey() *rsa.PublicKey {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return &r.current.PublicKey
}

// PrivateKey returns the current signing private key.
func (r *RotatingKeyProvider) PrivateKey() *rsa.PrivateKey {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.current
}

// KeyID returns the current key identifier.
func (r *RotatingKeyProvider) KeyID() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.currentID
}

// RotateKey generates a new signing key and demotes the current key to
// "previous" status. The previous key remains available for JWT verification
// during the grace period.
func (r *RotatingKeyProvider) RotateKey() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	newKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("generate new signing key: %w", err)
	}

	// Demote current to previous.
	r.previous = r.current
	r.previousID = r.currentID
	r.rotatedAt = time.Now()

	// Set new current.
	r.current = newKey
	r.currentID = generateKeyID(newKey)

	slog.Info("JWT signing key rotated",
		"new_kid", r.currentID,
		"previous_kid", r.previousID,
		"grace_period", r.gracePeriod.String(),
	)
	return nil
}

// PreviousPublicKey returns the previous key if within grace period, nil otherwise.
// Used for JWT verification of tokens signed before rotation.
func (r *RotatingKeyProvider) PreviousPublicKey() *rsa.PublicKey {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.previous == nil {
		return nil
	}
	if time.Since(r.rotatedAt) > r.gracePeriod {
		return nil
	}
	return &r.previous.PublicKey
}

// PreviousKeyID returns the previous key ID if within grace period.
func (r *RotatingKeyProvider) PreviousKeyID() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.previous == nil || time.Since(r.rotatedAt) > r.gracePeriod {
		return ""
	}
	return r.previousID
}

// IsGracePeriodExpired returns true if the previous key has aged out.
func (r *RotatingKeyProvider) IsGracePeriodExpired() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.previous != nil && time.Since(r.rotatedAt) > r.gracePeriod
}

// CleanupExpired removes the previous key if the grace period has elapsed.
func (r *RotatingKeyProvider) CleanupExpired() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.previous != nil && time.Since(r.rotatedAt) > r.gracePeriod {
	slog.Info("JWT previous key grace period expired, removing", "kid", r.previousID)
		r.previous = nil
		r.previousID = ""
	}
}

// StartRotationTicker starts a background goroutine that rotates the key at
// the given interval. Returns a stop function.
func (r *RotatingKeyProvider) StartRotationTicker(interval time.Duration) func() {
	ticker := time.NewTicker(interval)
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				if err := r.RotateKey(); err != nil {
					slog.Error("scheduled key rotation failed", "error", err)
				}
				r.CleanupExpired()
			case <-done:
				ticker.Stop()
				return
			}
		}
	}()
	return func() { close(done) }
}

// ResolveKeyByID returns the private key matching the given kid, checking both
// current and previous (within grace period).
func (r *RotatingKeyProvider) ResolveKeyByID(kid string) *rsa.PrivateKey {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if kid == r.currentID {
		return r.current
	}
	if kid == r.previousID && time.Since(r.rotatedAt) <= r.gracePeriod {
		return r.previous
	}
	return nil
}

// generateKeyID derives a stable key ID from the public key using SHA256.
func generateKeyID(key *rsa.PrivateKey) string {
	pubBytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return "fallback-kid"
	}
	h := sha256.Sum256(pubBytes)
	return hex.EncodeToString(h[:8])
}

// Compile-time interface check.
var _ domain.KeyProvider = (*RotatingKeyProvider)(nil)
