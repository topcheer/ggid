package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/ggid/ggid/pkg/sysconfig"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// SessionManager validates sessions against Redis.
type SessionManager struct {
	rdb   *redis.Client
	store sysconfig.Store
}

// NewSessionManager creates a session validator backed by Redis.
func NewSessionManager(rdb *redis.Client) *SessionManager {
	return &SessionManager{rdb: rdb}
}

// SetSysconfigStore injects the system config store for hot-reloadable session timeouts.
func (sm *SessionManager) SetSysconfigStore(store sysconfig.Store) {
	sm.store = store
}

// SessionMiddleware extracts session_id from JWT claims or X-Session-ID header,
// validates it against Redis, and rejects if the session is revoked or expired.
func (sm *SessionManager) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip session validation for public paths
		if isPublicPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Extract session ID from context (set by JWTAuth) or header
		sessionID, _ := r.Context().Value(SessionIDKey).(string)
		if sessionID == "" {
			sessionID = r.Header.Get("X-Session-ID")
		}

		// If no session ID, allow through (JWT already validated by JWTAuth)
		if sessionID == "" || sm.rdb == nil {
			next.ServeHTTP(w, r)
			return
		}

		ctx := r.Context()
		key := sessionKey(sessionID)

		// Check if session is revoked
		val, err := sm.rdb.Get(ctx, key).Result()
		if err == redis.Nil {
			// Session not in Redis — could be revoked or expired
			writeSessionError(w, "session revoked or expired")
			return
		}
		if err != nil {
			// Redis error — fail open (don't block on infra issues)
			next.ServeHTTP(w, r)
			return
		}

		// Session is valid — store in context for downstream
		_ = val // session metadata, not needed here
		r = r.WithContext(context.WithValue(ctx, SessionValidKey, true))
		next.ServeHTTP(w, r)
	})
}

// IsSessionRevoked checks Redis if a session ID is still valid.
func (sm *SessionManager) IsSessionRevoked(ctx context.Context, sessionID string) bool {
	if sm.rdb == nil {
		return false
	}
	_, err := sm.rdb.Get(ctx, sessionKey(sessionID)).Result()
	return err == redis.Nil
}

// MarkSessionRevoked removes a session from Redis.
func (sm *SessionManager) MarkSessionRevoked(ctx context.Context, sessionID string) error {
	if sm.rdb == nil {
		return nil
	}
	return sm.rdb.Del(ctx, sessionKey(sessionID)).Err()
}

// --- Session List/Revoke Handlers ---

// SessionListHandler returns active sessions for the current user.
// Uses the sessions set stored in Redis under sessions:user:{userID}.
func (sm *SessionManager) SessionListHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := UserIDFromRequest(r)
		if !ok {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		if sm.rdb == nil {
			writeJSONError(w, http.StatusServiceUnavailable, "session store unavailable")
			return
		}

		ctx := r.Context()
		key := "sessions:user:" + userID.String()
		members, err := sm.rdb.SMembers(ctx, key).Result()
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to query sessions")
			return
		}

		sessions := make([]map[string]any, 0, len(members))
		for _, sid := range members {
			data, err := sm.rdb.Get(ctx, sessionKey(sid)).Result()
			if err != nil {
				continue // skip expired/revoked
			}
			var meta map[string]any
			if json.Unmarshal([]byte(data), &meta) == nil {
				sessions = append(sessions, meta)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"sessions": sessions,
			"total":    len(sessions),
		})
	})
}

// SessionRevokeHandler revokes a specific session by ID.
func (sm *SessionManager) SessionRevokeHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		userID, ok := UserIDFromRequest(r)
		if !ok {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		// Extract session ID from path: /api/v1/sessions/{id}
		sessionID := r.URL.Path[len("/api/v1/sessions/"):]
		if _, err := uuid.Parse(sessionID); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid session ID")
			return
		}

		if sm.rdb == nil {
			writeJSONError(w, http.StatusServiceUnavailable, "session store unavailable")
			return
		}

		ctx := r.Context()
		// Verify the session belongs to this user
		userKey := "sessions:user:" + userID.String()
		isMember, err := sm.rdb.SIsMember(ctx, userKey, sessionID).Result()
		if err != nil || !isMember {
			writeJSONError(w, http.StatusNotFound, "session not found")
			return
		}

		// Revoke: delete session + remove from user set
		pipe := sm.rdb.TxPipeline()
		pipe.Del(ctx, sessionKey(sessionID))
		pipe.SRem(ctx, userKey, sessionID)
		if _, err := pipe.Exec(ctx); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to revoke session")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "revoked"})
	})
}

// --- Helpers ---

func sessionKey(sessionID string) string {
	return "ggid:session:" + sessionID
}

func isPublicPath(path string) bool {
	for _, p := range publicPathPrefixes {
		if path == p || (len(path) > len(p) && path[:len(p)] == p) {
			return true
		}
	}
	return path == "/healthz" || path == "/.well-known/jwks.json"
}

var publicPathPrefixes = []string{
	"/api/v1/auth/login",
	"/api/v1/auth/register",
	"/api/v1/auth/refresh",
	"/api/v1/auth/password/forgot",
	"/api/v1/auth/password/reset",
	"/api/v1/auth/social/",
	"/oauth/",
	"/api/v1/oauth/register",
	"/saml/",
	"/.well-known/",
	"/docs",
	"/api-docs",
	"/login",
	"/register",
	"/forgot-password",
}

func writeSessionError(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func writeJSONError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// context keys
type sessionCtxKey string

var (
	SessionIDKey   sessionCtxKey = "session_id"
	SessionValidKey sessionCtxKey = "session_valid"
)

// SessionIDFromContext extracts the validated session ID.
func SessionIDFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(SessionIDKey).(string)
	return v, ok
}

// touchSessionTTL resets the TTL for a session in Redis.
func (sm *SessionManager) touchSessionTTL(ctx context.Context, sessionID string, ttl time.Duration) {
	if sm.rdb == nil {
		return
	}
	sm.rdb.Expire(ctx, sessionKey(sessionID), ttl)
}
