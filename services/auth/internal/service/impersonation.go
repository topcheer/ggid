package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// ImpersonationToken represents a delegated admin token for impersonation.
type ImpersonationToken struct {
	TokenID        uuid.UUID `json:"token_id"`
	ImpersonatorID uuid.UUID `json:"impersonator_id"`
	TargetUserID   uuid.UUID `json:"target_user_id"`
	TenantID       uuid.UUID `json:"tenant_id"`
	Reason         string    `json:"reason"`
	IssuedAt       time.Time `json:"issued_at"`
	ExpiresAt      time.Time `json:"expires_at"`
	Revoked        bool      `json:"revoked"`
}

const (
	impersonationKeyPrefix = "ggid:imp_token:"
	// In-memory fallback when Redis is unavailable
)

var (
	impersonationMu    sync.RWMutex
	impersonationStore = make(map[uuid.UUID]*ImpersonationToken)
	impRedisClient     *redis.Client
)

// StartImpersonationCleanup starts a background goroutine that removes
// expired impersonation tokens from the in-memory store.
func StartImpersonationCleanup(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				now := time.Now().UTC()
				impersonationMu.Lock()
				for id, t := range impersonationStore {
					if now.After(t.ExpiresAt) {
						delete(impersonationStore, id)
					}
				}
				expiryNotifMu.Lock()
				for uid, n := range expiryNotifs {
					if now.After(n.ExpiresAt) {
						delete(expiryNotifs, uid)
					}
				}
				for uid, ch := range expiryChannels {
					select {
					case <-ch:
					default:
						close(ch)
						delete(expiryChannels, uid)
					}
				}
				expiryNotifMu.Unlock()
				impersonationMu.Unlock()
			case <-ctx.Done():
				return
			}
		}
	}()
}

// SetImpersonationRedis injects a Redis client for persistent storage.
// When set, impersonation tokens survive process restarts.
func SetImpersonationRedis(rdb *redis.Client) {
	impRedisClient = rdb
}

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

	// Persist to Redis with TTL = expiry
	if impRedisClient != nil {
		data, _ := json.Marshal(t)
		ttl := time.Until(t.ExpiresAt)
		if ttl > 0 {
			impRedisClient.Set(context.Background(), impersonationKeyPrefix+t.TokenID.String(), data, ttl)
		}
	}

	return t, nil
}

// GetImpersonationToken retrieves an impersonation token by ID.
func GetImpersonationToken(id uuid.UUID) (*ImpersonationToken, error) {
	// Try memory first
	impersonationMu.RLock()
	t, ok := impersonationStore[id]
	impersonationMu.RUnlock()
	if ok {
		return t, nil
	}
	// Try Redis (survives restart)
	if impRedisClient != nil {
		data, err := impRedisClient.Get(context.Background(), impersonationKeyPrefix+id.String()).Bytes()
		if err == nil {
			var rt ImpersonationToken
			if json.Unmarshal(data, &rt) == nil {
				// Cache in memory
				impersonationMu.Lock()
				impersonationStore[id] = &rt
				impersonationMu.Unlock()
				return &rt, nil
			}
		}
	}
	return nil, fmt.Errorf("impersonation token not found")
}

// ValidateImpersonationToken checks if a token is valid (not revoked, not expired).
func ValidateImpersonationToken(id uuid.UUID) (*ImpersonationToken, error) {
	t, err := GetImpersonationToken(id)
	if err != nil {
		return nil, err
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
	t, err := GetImpersonationToken(id)
	if err != nil {
		return err
	}
	impersonationMu.Lock()
	t.Revoked = true
	impersonationMu.Unlock()
	// Update Redis
	if impRedisClient != nil {
		data, _ := json.Marshal(t)
		ttl := time.Until(t.ExpiresAt)
		if ttl > 0 {
			impRedisClient.Set(context.Background(), impersonationKeyPrefix+id.String(), data, ttl)
		}
	}
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

// jtiTTL is how long a revoked JTI stays in the blocklist.
// JWTs expire after at most 1 hour, so entries older than that are stale.
const jtiTTL = 2 * time.Hour

func init() {
	// Periodic cleanup of expired JTI blocklist entries to prevent OOM.
	go func() {
		t := time.NewTicker(10 * time.Minute)
		defer t.Stop()
		for range t.C {
			cleanupJTIBlocklist()
		}
	}()
}

func cleanupJTIBlocklist() {
	cutoff := time.Now().UTC().Add(-jtiTTL)
	jtiBlocklistMu.Lock()
	defer jtiBlocklistMu.Unlock()
	for jti, revokedAt := range jtiBlocklist {
		if revokedAt.Before(cutoff) {
			delete(jtiBlocklist, jti)
		}
	}
}

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
	UserID     uuid.UUID
	TokenID    string
	ExpiresAt  time.Time
	NotifiedAt time.Time
	Message    string
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

func init() {
	// Periodic cleanup of expired expiry notifications and stale impersonation tokens.
	go func() {
		t := time.NewTicker(5 * time.Minute)
		defer t.Stop()
		for range t.C {
			cleanupExpiryNotifs()
			cleanupImpersonationStore()
		}
	}()
}

func cleanupExpiryNotifs() {
	cutoff := time.Now().UTC().Add(-30 * time.Minute)
	expiryNotifMu.Lock()
	defer expiryNotifMu.Unlock()
	for uid, n := range expiryNotifs {
		if n.ExpiresAt.Before(cutoff) {
			delete(expiryNotifs, uid)
		}
		if ch, ok := expiryChannels[uid]; ok {
			select {
			case <-ch: // drain stale channel
			default:
			}
		}
	}
}

func cleanupImpersonationStore() {
	now := time.Now().UTC()
	impersonationMu.Lock()
	defer impersonationMu.Unlock()
	for id, token := range impersonationStore {
		if now.After(token.ExpiresAt) {
			delete(impersonationStore, id)
		}
	}
}
