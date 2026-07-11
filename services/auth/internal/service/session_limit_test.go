package service

// Gap Regression Verification Test
// Verifies: Gap #5 — Concurrent Session Limits (was PARTIAL → now VERIFIED)
// Method: Unit tests for SessionManagement.EnforceSessionLimit covering:
//         within limit, over limit, unlimited config, empty sessions, expired filtering.
// Date: 2026-07-25

import (
	"context"
	"testing"
	"time"

	"github.com/ggid/ggid/services/auth/internal/conf"
	"github.com/ggid/ggid/services/auth/internal/domain"
	"github.com/google/uuid"
)

// ========== GAP #5: Concurrent Session Limits — Enforcement Verification ==========

// TestEnforceSessionLimit_WithinLimit verifies no revocation when under the limit.
func TestEnforceSessionLimit_WithinLimit(t *testing.T) {
	repo := newMockSessionRepo()
	ss := NewSessionService(repo)
	cfg := &conf.Config{}
	cfg.SessionTimeout.MaxSessions = 5

	sm := NewSessionManagement(ss, cfg)

	tenantID := uuid.New()
	userID := uuid.New()

	// Create 3 active sessions (under limit of 5)
	for i := 0; i < 3; i++ {
		s := &domain.Session{
			ID:        uuid.New(),
			TenantID:  tenantID,
			UserID:    userID,
			CreatedAt: time.Now().Add(-time.Duration(i) * time.Minute),
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}
		_ = repo.Create(context.Background(), s)
	}

	err := sm.EnforceSessionLimit(context.Background(), tenantID, userID)
	if err != nil {
		t.Fatalf("EnforceSessionLimit should not error: %v", err)
	}

	// All 3 sessions should still be active
	sessions, _ := ss.ListByUser(context.Background(), tenantID, userID)
	active := countActive(t, sessions)
	if active != 3 {
		t.Errorf("expected 3 active sessions, got %d", active)
	}
}

// TestEnforceSessionLimit_OverLimit verifies oldest sessions are revoked when over limit.
func TestEnforceSessionLimit_OverLimit(t *testing.T) {
	repo := newMockSessionRepo()
	ss := NewSessionService(repo)
	cfg := &conf.Config{}
	cfg.SessionTimeout.MaxSessions = 2

	sm := NewSessionManagement(ss, cfg)

	tenantID := uuid.New()
	userID := uuid.New()

	// Create 4 active sessions — oldest 2 should be revoked
	baseTime := time.Now()
	sessionsToCreate := []*domain.Session{
		{ID: uuid.New(), TenantID: tenantID, UserID: userID, CreatedAt: baseTime.Add(-4 * time.Minute), ExpiresAt: baseTime.Add(1 * time.Hour)},
		{ID: uuid.New(), TenantID: tenantID, UserID: userID, CreatedAt: baseTime.Add(-3 * time.Minute), ExpiresAt: baseTime.Add(1 * time.Hour)},
		{ID: uuid.New(), TenantID: tenantID, UserID: userID, CreatedAt: baseTime.Add(-2 * time.Minute), ExpiresAt: baseTime.Add(1 * time.Hour)},
		{ID: uuid.New(), TenantID: tenantID, UserID: userID, CreatedAt: baseTime.Add(-1 * time.Minute), ExpiresAt: baseTime.Add(1 * time.Hour)},
	}
	for _, s := range sessionsToCreate {
		_ = repo.Create(context.Background(), s)
	}

	err := sm.EnforceSessionLimit(context.Background(), tenantID, userID)
	if err != nil {
		t.Fatalf("EnforceSessionLimit should not error: %v", err)
	}

	sessions, _ := ss.ListByUser(context.Background(), tenantID, userID)
	active := countActive(t, sessions)
	if active != 2 {
		t.Errorf("expected 2 active sessions after enforcement, got %d", active)
	}

	// Verify the oldest 2 were revoked (sessionsToCreate[0] and [1])
	if sessionsToCreate[0].RevokedAt == nil {
		t.Error("oldest session should have been revoked")
	}
	if sessionsToCreate[1].RevokedAt == nil {
		t.Error("second-oldest session should have been revoked")
	}
	if sessionsToCreate[2].RevokedAt != nil {
		t.Error("third session should NOT have been revoked")
	}
	if sessionsToCreate[3].RevokedAt != nil {
		t.Error("newest session should NOT have been revoked")
	}
}

// TestEnforceSessionLimit_Unlimited verifies no revocation when MaxSessions <= 0.
func TestEnforceSessionLimit_Unlimited(t *testing.T) {
	repo := newMockSessionRepo()
	ss := NewSessionService(repo)
	cfg := &conf.Config{}
	cfg.SessionTimeout.MaxSessions = 0 // unlimited

	sm := NewSessionManagement(ss, cfg)

	tenantID := uuid.New()
	userID := uuid.New()

	for i := 0; i < 10; i++ {
		s := &domain.Session{
			ID:        uuid.New(),
			TenantID:  tenantID,
			UserID:    userID,
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}
		_ = repo.Create(context.Background(), s)
	}

	err := sm.EnforceSessionLimit(context.Background(), tenantID, userID)
	if err != nil {
		t.Fatalf("EnforceSessionLimit should not error with unlimited: %v", err)
	}

	sessions, _ := ss.ListByUser(context.Background(), tenantID, userID)
	active := countActive(t, sessions)
	if active != 10 {
		t.Errorf("expected all 10 sessions active with unlimited config, got %d", active)
	}
}

// TestEnforceSessionLimit_ExpiredNotCounted verifies expired sessions are not counted
// toward the active session count.
func TestEnforceSessionLimit_ExpiredNotCounted(t *testing.T) {
	repo := newMockSessionRepo()
	ss := NewSessionService(repo)
	cfg := &conf.Config{}
	cfg.SessionTimeout.MaxSessions = 3

	sm := NewSessionManagement(ss, cfg)

	tenantID := uuid.New()
	userID := uuid.New()

	// Create 3 active + 3 expired = 6 total, but only 3 active (within limit)
	now := time.Now()
	for i := 0; i < 3; i++ {
		s := &domain.Session{
			ID:        uuid.New(),
			TenantID:  tenantID,
			UserID:    userID,
			CreatedAt: now.Add(-time.Duration(i+1) * time.Hour),
			ExpiresAt: now.Add(1 * time.Hour), // active
		}
		_ = repo.Create(context.Background(), s)
	}
	for i := 0; i < 3; i++ {
		s := &domain.Session{
			ID:        uuid.New(),
			TenantID:  tenantID,
			UserID:    userID,
			CreatedAt: now.Add(-time.Duration(i+10) * time.Hour),
			ExpiresAt: now.Add(-1 * time.Hour), // expired
		}
		_ = repo.Create(context.Background(), s)
	}

	err := sm.EnforceSessionLimit(context.Background(), tenantID, userID)
	if err != nil {
		t.Fatalf("EnforceSessionLimit should not error: %v", err)
	}

	sessions, _ := ss.ListByUser(context.Background(), tenantID, userID)
	active := countActive(t, sessions)
	if active != 3 {
		t.Errorf("expected 3 active sessions (expired not counted), got %d", active)
	}
}

// TestEnforceSessionLimit_NoSessions verifies no error when user has zero sessions.
func TestEnforceSessionLimit_NoSessions(t *testing.T) {
	repo := newMockSessionRepo()
	ss := NewSessionService(repo)
	cfg := &conf.Config{}
	cfg.SessionTimeout.MaxSessions = 3

	sm := NewSessionManagement(ss, cfg)

	tenantID := uuid.New()
	userID := uuid.New()

	err := sm.EnforceSessionLimit(context.Background(), tenantID, userID)
	if err != nil {
		t.Fatalf("EnforceSessionLimit should not error with zero sessions: %v", err)
	}
}

// TestEnforceSessionLimit_AlreadyRevokedNotCounted verifies already-revoked sessions
// are excluded from the active count.
func TestEnforceSessionLimit_AlreadyRevokedNotCounted(t *testing.T) {
	repo := newMockSessionRepo()
	ss := NewSessionService(repo)
	cfg := &conf.Config{}
	cfg.SessionTimeout.MaxSessions = 2

	sm := NewSessionManagement(ss, cfg)

	tenantID := uuid.New()
	userID := uuid.New()

	// 2 active + 3 revoked = 5 total, but only 2 active (within limit)
	now := time.Now()
	for i := 0; i < 2; i++ {
		s := &domain.Session{
			ID:        uuid.New(),
			TenantID:  tenantID,
			UserID:    userID,
			CreatedAt: now.Add(-time.Duration(i+1) * time.Minute),
			ExpiresAt: now.Add(1 * time.Hour),
		}
		_ = repo.Create(context.Background(), s)
	}
	revokedAt := now
	for i := 0; i < 3; i++ {
		s := &domain.Session{
			ID:        uuid.New(),
			TenantID:  tenantID,
			UserID:    userID,
			CreatedAt: now.Add(-time.Duration(i+10) * time.Minute),
			ExpiresAt: now.Add(1 * time.Hour),
			RevokedAt: &revokedAt,
		}
		_ = repo.Create(context.Background(), s)
	}

	err := sm.EnforceSessionLimit(context.Background(), tenantID, userID)
	if err != nil {
		t.Fatalf("EnforceSessionLimit should not error: %v", err)
	}

	sessions, _ := ss.ListByUser(context.Background(), tenantID, userID)
	active := countActive(t, sessions)
	if active != 2 {
		t.Errorf("expected 2 active sessions (revoked not counted), got %d", active)
	}
}

// countActive counts non-revoked, non-expired sessions.
func countActive(t *testing.T, sessions []*domain.Session) int {
	t.Helper()
	count := 0
	now := time.Now()
	for _, s := range sessions {
		if s.RevokedAt == nil && s.ExpiresAt.After(now) {
			count++
		}
	}
	return count
}
