package middleware

import (
	"sync"
	"time"
)

// JTIReplayTracker prevents JWT replay attacks by tracking used jti values.
// Uses an in-memory map with expiry cleanup. In production, replace with Redis SETNX.
type JTIReplayTracker struct {
	mu      sync.Mutex
	seen    map[string]time.Time // jti -> expiry
	maxAge  time.Duration
}

// NewJTIReplayTracker creates a tracker with the given max token lifetime.
func NewJTIReplayTracker(maxAge time.Duration) *JTIReplayTracker {
	t := &JTIReplayTracker{
		seen:   make(map[string]time.Time),
		maxAge: maxAge,
	}
	go t.cleanupLoop()
	return t
}

// IsReplayed returns true if the jti has already been seen.
// If not seen, marks it as seen with the token's expiry time.
// Returns true (replayed) if jti is empty — empty jti is invalid.
func (t *JTIReplayTracker) IsReplayed(jti string, expiresAt time.Time) bool {
	if jti == "" {
		return true // empty jti = invalid, treat as replayed
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if exp, ok := t.seen[jti]; ok {
		// Already seen and not expired
		if time.Now().Before(exp) {
			return true // replayed
		}
		// Expired — remove and allow re-use
		delete(t.seen, jti)
	}

	// Mark as seen
	t.seen[jti] = expiresAt
	return false
}

// cleanupLoop periodically removes expired entries.
func (t *JTIReplayTracker) cleanupLoop() {
	ticker := time.NewTicker(t.maxAge / 2)
	defer ticker.Stop()
	for range ticker.C {
		t.mu.Lock()
		now := time.Now()
		for jti, exp := range t.seen {
			if now.After(exp) {
				delete(t.seen, jti)
			}
		}
		t.mu.Unlock()
	}
}
