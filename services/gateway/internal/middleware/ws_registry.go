package middleware

import (
	"sync"
	"time"
)

// WSSession represents an active WebSocket connection in the registry.
type WSSession struct {
	ID         string
	TenantID   string
	UserID     string
	StartedAt  time.Time
	RemoteAddr string
	// OnMessage is called when a broadcast targets this session.
	// The implementation writes the message to the underlying WS connection.
	OnMessage func(msg []byte)
}

// WSSessionRegistry maintains active WebSocket connections keyed by session ID,
// with indexes for tenant-wide and user-targeted broadcast.
type WSSessionRegistry struct {
	mu         sync.RWMutex
	sessions   map[string]*WSSession        // session ID → session
	byTenant   map[string]map[string]bool   // tenant ID → set of session IDs
	byUser     map[string]map[string]bool   // user ID → set of session IDs
}

// NewWSSessionRegistry creates an empty registry.
func NewWSSessionRegistry() *WSSessionRegistry {
	return &WSSessionRegistry{
		sessions: make(map[string]*WSSession),
		byTenant: make(map[string]map[string]bool),
		byUser:   make(map[string]map[string]bool),
	}
}

// Register adds a WebSocket session to the registry.
func (r *WSSessionRegistry) Register(sess *WSSession) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.sessions[sess.ID] = sess

	if sess.TenantID != "" {
		if r.byTenant[sess.TenantID] == nil {
			r.byTenant[sess.TenantID] = make(map[string]bool)
		}
		r.byTenant[sess.TenantID][sess.ID] = true
	}

	if sess.UserID != "" {
		if r.byUser[sess.UserID] == nil {
			r.byUser[sess.UserID] = make(map[string]bool)
		}
		r.byUser[sess.UserID][sess.ID] = true
	}
}

// Unregister removes a session from the registry.
func (r *WSSessionRegistry) Unregister(sessionID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	sess, ok := r.sessions[sessionID]
	if !ok {
		return
	}

	delete(r.sessions, sessionID)

	if sess.TenantID != "" {
		if set, ok := r.byTenant[sess.TenantID]; ok {
			delete(set, sessionID)
			if len(set) == 0 {
				delete(r.byTenant, sess.TenantID)
			}
		}
	}

	if sess.UserID != "" {
		if set, ok := r.byUser[sess.UserID]; ok {
			delete(set, sessionID)
			if len(set) == 0 {
				delete(r.byUser, sess.UserID)
			}
		}
	}
}

// Get returns a session by ID.
func (r *WSSessionRegistry) Get(sessionID string) (*WSSession, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	sess, ok := r.sessions[sessionID]
	return sess, ok
}

// Count returns the total number of active sessions.
func (r *WSSessionRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.sessions)
}

// CountByTenant returns the number of sessions for a given tenant.
func (r *WSSessionRegistry) CountByTenant(tenantID string) int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.byTenant[tenantID])
}

// CountByUser returns the number of sessions for a given user.
func (r *WSSessionRegistry) CountByUser(userID string) int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.byUser[userID])
}

// BroadcastToTenant sends a message to all sessions belonging to a tenant.
// Returns the number of sessions the message was delivered to.
func (r *WSSessionRegistry) BroadcastToTenant(tenantID string, msg []byte) int {
	r.mu.RLock()
	ids := r.byTenant[tenantID]
	// Copy IDs to avoid holding lock during delivery
	sessionIDs := make([]string, 0, len(ids))
	for id := range ids {
		sessionIDs = append(sessionIDs, id)
	}
	r.mu.RUnlock()

	delivered := 0
	for _, id := range sessionIDs {
		r.mu.RLock()
		sess, ok := r.sessions[id]
		r.mu.RUnlock()
		if ok && sess.OnMessage != nil {
			sess.OnMessage(msg)
			delivered++
		}
	}
	return delivered
}

// SendToUser sends a message to all sessions belonging to a specific user.
// Returns the number of sessions the message was delivered to.
func (r *WSSessionRegistry) SendToUser(userID string, msg []byte) int {
	r.mu.RLock()
	ids := r.byUser[userID]
	sessionIDs := make([]string, 0, len(ids))
	for id := range ids {
		sessionIDs = append(sessionIDs, id)
	}
	r.mu.RUnlock()

	delivered := 0
	for _, id := range sessionIDs {
		r.mu.RLock()
		sess, ok := r.sessions[id]
		r.mu.RUnlock()
		if ok && sess.OnMessage != nil {
			sess.OnMessage(msg)
			delivered++
		}
	}
	return delivered
}

// ListSessions returns metadata for all active sessions (without callbacks).
type WSSessionInfo struct {
	ID         string    `json:"id"`
	TenantID   string    `json:"tenant_id"`
	UserID     string    `json:"user_id"`
	StartedAt  time.Time `json:"started_at"`
	RemoteAddr string    `json:"remote_addr"`
}

// ListSessions returns info for all active sessions.
func (r *WSSessionRegistry) ListSessions() []WSSessionInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]WSSessionInfo, 0, len(r.sessions))
	for _, s := range r.sessions {
		result = append(result, WSSessionInfo{
			ID:         s.ID,
			TenantID:   s.TenantID,
			UserID:     s.UserID,
			StartedAt:  s.StartedAt,
			RemoteAddr: s.RemoteAddr,
		})
	}
	return result
}
