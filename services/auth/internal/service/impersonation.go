package service

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ImpersonationToken represents a delegated admin token for impersonation.
type ImpersonationToken struct {
	TokenID            uuid.UUID
	ImpersonatorID     uuid.UUID // admin who impersonates
	TargetUserID       uuid.UUID // user being impersonated
	TenantID           uuid.UUID
	Reason             string
	IssuedAt           time.Time
	ExpiresAt          time.Time
	Revoked            bool
}

var (
	impersonationMu    sync.RWMutex
	impersonationStore = make(map[uuid.UUID]*ImpersonationToken)
)

// IssueImpersonationToken creates a temporary token for admin to act as target user.
func IssueImpersonationToken(impersonatorID, targetUserID, tenantID uuid.UUID, reason string) (*ImpersonationToken, error) {
	if impersonatorID == uuid.Nil || targetUserID == uuid.Nil {
		return nil, fmt.Errorf("impersonator and target IDs required")
	}
	if impersonatorID == targetUserID {
		return nil, fmt.Errorf("cannot impersonate self")
	}
	if reason == "" {
		return nil, fmt.Errorf("reason is required for audit trail")
	}

	now := time.Now().UTC()
	t := &ImpersonationToken{
		TokenID:        uuid.New(),
		ImpersonatorID: impersonatorID,
		TargetUserID:   targetUserID,
		TenantID:       tenantID,
		Reason:         reason,
		IssuedAt:       now,
		ExpiresAt:      now.Add(15 * time.Minute),
	}

	impersonationMu.Lock()
	impersonationStore[t.TokenID] = t
	impersonationMu.Unlock()

	return t, nil
}

// GetImpersonationToken retrieves an impersonation token by ID.
func GetImpersonationToken(id uuid.UUID) (*ImpersonationToken, error) {
	impersonationMu.RLock()
	defer impersonationMu.RUnlock()
	t, ok := impersonationStore[id]
	if !ok {
		return nil, fmt.Errorf("impersonation token not found")
	}
	return t, nil
}

// ValidateImpersonationToken checks if a token is valid (not revoked, not expired).
func ValidateImpersonationToken(id uuid.UUID) (*ImpersonationToken, error) {
	impersonationMu.RLock()
	defer impersonationMu.RUnlock()
	t, ok := impersonationStore[id]
	if !ok {
		return nil, fmt.Errorf("token not found")
	}
	if t.Revoked {
		return nil, fmt.Errorf("token revoked")
	}
	if time.Now().UTC().After(t.ExpiresAt) {
		return nil, fmt.Errorf("token expired")
	}
	return t, nil
}

// RevokeImpersonationToken revokes an active impersonation token.
func RevokeImpersonationToken(id uuid.UUID) error {
	impersonationMu.Lock()
	defer impersonationMu.Unlock()
	t, ok := impersonationStore[id]
	if !ok {
		return fmt.Errorf("token not found")
	}
	t.Revoked = true
	return nil
}

// ListActiveImpersonations returns all active impersonation tokens for audit.
func ListActiveImpersonations() []*ImpersonationToken {
	impersonationMu.RLock()
	defer impersonationMu.RUnlock()
	var out []*ImpersonationToken
	for _, t := range impersonationStore {
		if !t.Revoked && time.Now().UTC().Before(t.ExpiresAt) {
			out = append(out, t)
		}
	}
	return out
}

// ResetImpersonationStore clears all tokens (for testing).
func ResetImpersonationStore() {
	impersonationMu.Lock()
	defer impersonationMu.Unlock()
	impersonationStore = make(map[uuid.UUID]*ImpersonationToken)
}

// --- Session Revocation ---

var (
	jtiBlocklistMu sync.RWMutex
	jtiBlocklist   = make(map[string]time.Time) // jti → revokedAt
)

// RevokeAllUserSessions blocks all JWTs for a user by adding their jtis to the blocklist.
func RevokeAllUserSessions(jtis []string) {
	jtiBlocklistMu.Lock()
	defer jtiBlocklistMu.Unlock()
	now := time.Now().UTC()
	for _, jti := range jtis {
		jtiBlocklist[jti] = now
	}
}

// IsJTIRevoked checks if a JWT's jti has been revoked.
func IsJTIRevoked(jti string) bool {
	jtiBlocklistMu.RLock()
	defer jtiBlocklistMu.RUnlock()
	_, revoked := jtiBlocklist[jti]
	return revoked
}

// ResetJTIMocklist clears the blocklist (for testing).
func ResetJTIBlocklist() {
	jtiBlocklistMu.Lock()
	defer jtiBlocklistMu.Unlock()
	jtiBlocklist = make(map[string]time.Time)
}

// --- JWT Expiry Notification ---

// ExpiryNotification represents a notification to refresh a token before expiry.
type ExpiryNotification struct {
	UserID    uuid.UUID
	TokenID   string
	ExpiresAt time.Time
	NotifiedAt time.Time
	Message   string
}

var (
	expiryNotifMu  sync.RWMutex
	expiryNotifs   = make(map[uuid.UUID]*ExpiryNotification)
	expiryChannels = make(map[uuid.UUID]chan *ExpiryNotification)
)

// RegisterExpiryChannel creates an SSE channel for JWT expiry notifications.
func RegisterExpiryChannel(userID uuid.UUID) chan *ExpiryNotification {
	expiryNotifMu.Lock()
	defer expiryNotifMu.Unlock()
	ch := make(chan *ExpiryNotification, 1)
	expiryChannels[userID] = ch
	return ch
}

// ScheduleExpiryNotification queues a notification 5 minutes before token expiry.
func ScheduleExpiryNotification(userID uuid.UUID, tokenID string, expiresAt time.Time) {
	expiryNotifMu.Lock()
	defer expiryNotifMu.Unlock()

	notif := &ExpiryNotification{
		UserID:     userID,
		TokenID:    tokenID,
		ExpiresAt:  expiresAt,
		NotifiedAt: time.Now().UTC(),
		Message:    "Your session expires in 5 minutes. Please refresh.",
	}
	expiryNotifs[userID] = notif

	if ch, ok := expiryChannels[userID]; ok {
		select {
		case ch <- notif:
		default: // channel full, skip
		}
	}
}

// GetExpiryNotification returns the last notification for a user.
func GetExpiryNotification(userID uuid.UUID) *ExpiryNotification {
	expiryNotifMu.RLock()
	defer expiryNotifMu.RUnlock()
	return expiryNotifs[userID]
}

// ResetExpiryNotifs clears all notifications (for testing).
func ResetExpiryNotifs() {
	expiryNotifMu.Lock()
	defer expiryNotifMu.Unlock()
	expiryNotifs = make(map[uuid.UUID]*ExpiryNotification)
	for _, ch := range expiryChannels {
		close(ch)
	}
	expiryChannels = make(map[uuid.UUID]chan *ExpiryNotification)
}
