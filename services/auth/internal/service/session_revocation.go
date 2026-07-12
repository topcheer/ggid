package service

import (
	"sync"
	"time"
)

type RevocationRecord struct {
	SessionID  string    `json:"session_id"`
	UserID     string    `json:"user_id"`
	TenantID   string    `json:"tenant_id"`
	Reason     string    `json:"reason"`
	RevokedAt  time.Time `json:"revoked_at"`
	ExpiresAt  time.Time `json:"expires_at"` // when the revocation record itself can be cleaned up
}

type SessionRevocationService struct {
	mu         sync.RWMutex
	revocations map[string]*RevocationRecord // sessionID -> record
	byUser     map[string][]string            // userID -> []sessionID
	byTenant   map[string][]string            // tenantID -> []sessionID
}

func NewSessionRevocationService() *SessionRevocationService {
	return &SessionRevocationService{
		revocations: make(map[string]*RevocationRecord),
		byUser:      make(map[string][]string),
		byTenant:    make(map[string][]string),
	}
}

func (s *SessionRevocationService) RevokeSession(sessionID, reason string) *RevocationRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec := &RevocationRecord{
		SessionID: sessionID,
		Reason:    reason,
		RevokedAt: time.Now(),
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour), // keep revocation record for 7 days
	}
	s.revocations[sessionID] = rec
	return rec
}

func (s *SessionRevocationService) RevokeAllSessions(userID, reason string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	sessionIDs := s.byUser[userID]
	count := 0
	for _, sid := range sessionIDs {
		rec := &RevocationRecord{
			SessionID: sid,
			UserID:    userID,
			Reason:    reason,
			RevokedAt: time.Now(),
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		}
		s.revocations[sid] = rec
		count++
	}
	return count
}

func (s *SessionRevocationService) RevokeByTenant(tenantID, reason string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	sessionIDs := s.byTenant[tenantID]
	count := 0
	for _, sid := range sessionIDs {
		rec := &RevocationRecord{
			SessionID: sid,
			TenantID:  tenantID,
			Reason:    reason,
			RevokedAt: time.Now(),
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		}
		s.revocations[sid] = rec
		count++
	}
	return count
}

func (s *SessionRevocationService) GetRevocationStatus(sessionID string) *RevocationRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.revocations[sessionID]
}

func (s *SessionRevocationService) RegisterSession(sessionID, userID, tenantID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byUser[userID] = append(s.byUser[userID], sessionID)
	s.byTenant[tenantID] = append(s.byTenant[tenantID], sessionID)
}

func (s *SessionRevocationService) CleanupExpired() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	count := 0
	for sid, rec := range s.revocations {
		if now.After(rec.ExpiresAt) {
			delete(s.revocations, sid)
			count++
		}
	}
	return count
}