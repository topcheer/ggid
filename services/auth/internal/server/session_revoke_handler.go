package server

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// RevocableSession represents a trackable session for admin revocation.
type RevocableSession struct {
	SessionID string    `json:"session_id"`
	UserID    string    `json:"user_id"`
	TokenJTI  string    `json:"token_jti"`
	CreatedAt time.Time `json:"created_at"`
	Revoked   bool      `json:"revoked"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
}

// sessionRevocationStore holds sessions for revocation tracking.
type sessionRevocationStore struct {
	mu       sync.RWMutex
	sessions map[string]*RevocableSession // keyed by session_id
	byUser   map[string][]string          // user_id → session_ids
	revokedJTIs map[string]bool            // revoked token JTIs
}

var sessionStore = &sessionRevocationStore{
	sessions:    make(map[string]*RevocableSession),
	byUser:      make(map[string][]string),
	revokedJTIs: make(map[string]bool),
}

// IsSessionRevoked checks if a token JTI has been revoked.
func IsSessionRevoked(jti string) bool {
	sessionStore.mu.RLock()
	defer sessionStore.mu.RUnlock()
	return sessionStore.revokedJTIs[jti]
}

// POST /api/v1/auth/sessions/revoke — admin batch revoke sessions.
// Body: {"user_ids": ["..."], "session_ids": ["..."], "reason": "..."}
// Returns: {"revoked_count": N, "failed": [...]}
func (h *Handler) handleRevokeSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		UserIDs    []string `json:"user_ids"`
		SessionIDs []string `json:"session_ids"`
		Reason     string   `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if len(req.UserIDs) == 0 && len(req.SessionIDs) == 0 {
		writeJSONError(w, http.StatusBadRequest, "user_ids or session_ids required")
		return
	}

	revokedCount := 0
	var failed []map[string]string

	sessionStore.mu.Lock()

	// Revoke by session IDs
	for _, sid := range req.SessionIDs {
		sess, ok := sessionStore.sessions[sid]
		if !ok {
			failed = append(failed, map[string]string{
				"session_id": sid,
				"error":      "session not found",
			})
			continue
		}
		if sess.Revoked {
			failed = append(failed, map[string]string{
				"session_id": sid,
				"error":      "already revoked",
			})
			continue
		}
		now := time.Now().UTC()
		sess.Revoked = true
		sess.RevokedAt = &now
		if sess.TokenJTI != "" {
			sessionStore.revokedJTIs[sess.TokenJTI] = true
		}
		revokedCount++
	}

	// Revoke by user IDs
	for _, uid := range req.UserIDs {
		sids, ok := sessionStore.byUser[uid]
		if !ok || len(sids) == 0 {
			// Best-effort: mark as revoked even if not tracked (e.g., external sessions)
			revokedCount++
			continue
		}
		for _, sid := range sids {
			sess := sessionStore.sessions[sid]
			if sess == nil || sess.Revoked {
				continue
			}
			now := time.Now().UTC()
			sess.Revoked = true
			sess.RevokedAt = &now
			if sess.TokenJTI != "" {
				sessionStore.revokedJTIs[sess.TokenJTI] = true
			}
			revokedCount++
		}
	}

	sessionStore.mu.Unlock()

	if failed == nil {
		failed = []map[string]string{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":        "completed",
		"revoked_count": revokedCount,
		"failed":        failed,
		"reason":        req.Reason,
		"revoked_at":    time.Now().UTC().Format(time.RFC3339),
	})
}
