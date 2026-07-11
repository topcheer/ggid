package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

// SessionTimeoutConfig holds session timeout parameters.
type SessionTimeoutConfig struct {
	AbsoluteTimeout time.Duration // Maximum session duration from creation
	IdleTimeout     time.Duration // Maximum inactivity period
}

// DefaultSessionTimeoutConfig returns production-safe defaults.
func DefaultSessionTimeoutConfig() SessionTimeoutConfig {
	return SessionTimeoutConfig{
		AbsoluteTimeout: 8 * time.Hour,
		IdleTimeout:     30 * time.Minute,
	}
}

// SessionTimeoutMiddleware enforces absolute and idle session timeouts.
// This is wired after JWTAuth and SessionMiddleware in the chain.
//
// It checks:
//   - Absolute timeout: if the session's created_at + absolute_timeout has passed,
//     the session is rejected.
//   - Idle timeout: if the last-activity timestamp in Redis is older than
//     idle_timeout, the session is rejected.
//
// On each successful check, the last-activity timestamp is refreshed.
func (sm *SessionManager) SessionTimeoutMiddleware(cfg SessionTimeoutConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip public paths.
			if isPublicPath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			// Extract session ID from context.
			sessionID, ok := r.Context().Value(SessionIDKey).(string)
			if !ok || sessionID == "" {
				// No session — let JWT auth handle it.
				next.ServeHTTP(w, r)
				return
			}

			// If no Redis, fail open (infrastructure issue).
			if sm.rdb == nil {
				next.ServeHTTP(w, r)
				return
			}

			ctx := r.Context()
			activityKey := fmt.Sprintf("ggid:session_activity:%s", sessionID)
			createdKey := fmt.Sprintf("ggid:session_created:%s", sessionID)

			// Check absolute timeout.
			if cfg.AbsoluteTimeout > 0 {
				createdStr, err := sm.rdb.Get(ctx, createdKey).Result()
				if err == nil {
					createdAt, err := time.Parse(time.RFC3339, createdStr)
					if err == nil && time.Since(createdAt) > cfg.AbsoluteTimeout {
						// Session exceeded absolute timeout — revoke.
						sm.MarkSessionRevoked(ctx, sessionID)
						writeSessionTimeoutError(w, "session expired (absolute timeout)")
						return
					}
				}
			}

			// Check idle timeout.
			if cfg.IdleTimeout > 0 {
				lastActiveStr, err := sm.rdb.Get(ctx, activityKey).Result()
				if err == nil {
					lastActive, err := time.Parse(time.RFC3339, lastActiveStr)
					if err == nil && time.Since(lastActive) > cfg.IdleTimeout {
						// Session idle timeout exceeded — revoke.
						sm.MarkSessionRevoked(ctx, sessionID)
						writeSessionTimeoutError(w, "session expired (idle timeout)")
						return
					}
				}
				// Refresh last-activity timestamp.
				now := time.Now().Format(time.RFC3339)
				sm.rdb.Set(ctx, activityKey, now, cfg.IdleTimeout)
			}

			// Session is valid — continue.
			next.ServeHTTP(w, r)
		})
	}
}

// RecordSessionCreation stores the session creation time for absolute timeout checks.
// Called when a new session is created.
func (sm *SessionManager) RecordSessionCreation(ctx context.Context, sessionID string, cfg SessionTimeoutConfig) {
	if sm.rdb == nil {
		return
	}
	createdKey := fmt.Sprintf("ggid:session_created:%s", sessionID)
	now := time.Now().Format(time.RFC3339)
	sm.rdb.Set(ctx, createdKey, now, cfg.AbsoluteTimeout)
}

// writeSessionTimeoutError writes a 401 with a session timeout message.
func writeSessionTimeoutError(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]string{
		"error":   "session_timeout",
		"message": msg,
	})
}

// CheckSessionTimeoutRedis is the standalone check function that mirrors
// AuthService.CheckSessionTimeout but operates at the middleware level
// using Redis directly. This wires the previously dead code into the
// gateway middleware chain.
func CheckSessionTimeoutRedis(ctx context.Context, rdb *redis.Client, sessionID string, cfg SessionTimeoutConfig) error {
	if rdb == nil {
		return nil // fail open
	}

	if cfg.AbsoluteTimeout > 0 {
		createdKey := fmt.Sprintf("ggid:session_created:%s", sessionID)
		createdStr, err := rdb.Get(ctx, createdKey).Result()
		if err == nil {
			createdAt, err := time.Parse(time.RFC3339, createdStr)
			if err == nil && time.Since(createdAt) > cfg.AbsoluteTimeout {
				return ErrSessionTimeoutAbsolute
			}
		}
	}

	if cfg.IdleTimeout > 0 {
		activityKey := fmt.Sprintf("ggid:session_activity:%s", sessionID)
		lastActiveStr, err := rdb.Get(ctx, activityKey).Result()
		if err == nil {
			lastActive, err := time.Parse(time.RFC3339, lastActiveStr)
			if err == nil && time.Since(lastActive) > cfg.IdleTimeout {
				return ErrSessionTimeoutIdle
			}
		}
		// Refresh activity.
		now := time.Now().Format(time.RFC3339)
		rdb.Set(ctx, activityKey, now, cfg.IdleTimeout)
	}

	return nil
}

// Session timeout errors.
var (
	ErrSessionTimeoutAbsolute = fmt.Errorf("session exceeded absolute timeout")
	ErrSessionTimeoutIdle     = fmt.Errorf("session exceeded idle timeout")
)
