package service

import (
	"context"
	"testing"
	"time"

	ggidauth "github.com/ggid/ggid/pkg/auth"
	"github.com/ggid/ggid/services/auth/internal/domain"
	"github.com/google/uuid"
)

// --- session_revocation.go ---

func TestSessionRevocationService(t *testing.T) {
	svc := NewSessionRevocationService()

	// Register sessions for user and tenant indexes.
	svc.RegisterSession("s1", "u1", "t1")
	svc.RegisterSession("s2", "u1", "t1")
	svc.RegisterSession("s3", "u2", "t1")

	// Single revoke.
	rec := svc.RevokeSession("s1", "manual")
	if rec.SessionID != "s1" || rec.Reason != "manual" {
		t.Errorf("record = %+v", rec)
	}
	if got := svc.GetRevocationStatus("s1"); got == nil {
		t.Error("s1 should be revoked")
	}
	if got := svc.GetRevocationStatus("nope"); got != nil {
		t.Error("unknown session should have no record")
	}

	// Revoke all for user.
	if n := svc.RevokeAllSessions("u1", "bulk"); n != 2 {
		t.Errorf("RevokeAllSessions = %d, want 2", n)
	}

	// Revoke by tenant (s1, s2, s3 all in t1).
	if n := svc.RevokeByTenant("t1", "tenant-lockdown"); n != 3 {
		t.Errorf("RevokeByTenant = %d, want 3", n)
	}

	// Cleanup: nothing expired yet.
	if n := svc.CleanupExpired(); n != 0 {
		t.Errorf("CleanupExpired = %d, want 0", n)
	}
	// Force expiry.
	svc.mu.Lock()
	for _, r := range svc.revocations {
		r.ExpiresAt = time.Now().Add(-time.Hour)
	}
	svc.mu.Unlock()
	if n := svc.CleanupExpired(); n == 0 {
		t.Error("expected expired records to be cleaned")
	}
}

// --- rotation_scheduler.go ---

func TestRotationScheduler(t *testing.T) {
	rs := NewRotationScheduler()

	sched := rs.ScheduleRotation("cred-1", RotationPolicy{IntervalDays: 30, AutoRotate: true})
	if sched.CredentialID != "cred-1" || sched.NextDue.Before(time.Now()) {
		t.Errorf("schedule = %+v", sched)
	}

	// Not due yet.
	if due := rs.CheckDueRotations(); len(due) != 0 {
		t.Errorf("due = %v, want empty", due)
	}

	// Force overdue.
	rs.mu.Lock()
	rs.schedules["cred-1"].NextDue = time.Now().Add(-48 * time.Hour)
	rs.mu.Unlock()
	due := rs.CheckDueRotations()
	if len(due) != 1 || due[0].CredentialID != "cred-1" || due[0].DaysOverdue < 2 {
		t.Errorf("due = %+v", due)
	}

	// Execute rotation.
	res := rs.ExecuteRotation("cred-1")
	if !res.Success {
		t.Error("rotation should succeed")
	}
	if due := rs.CheckDueRotations(); len(due) != 0 {
		t.Error("no longer due after rotation")
	}

	// Unknown credential.
	if res := rs.ExecuteRotation("ghost"); res.Success {
		t.Error("unknown credential rotation should fail")
	}
}

// --- device_binding.go ---

func TestDeviceBindingService(t *testing.T) {
	svc := NewDeviceBindingService()

	d, err := svc.BindDevice("u1", "laptop", "fp-1", "macos")
	if err != nil {
		t.Fatalf("BindDevice: %v", err)
	}
	if d.TrustScore != 50 || d.DeviceID == "" {
		t.Errorf("binding = %+v", d)
	}

	// Duplicate fingerprint for same user → error.
	if _, err := svc.BindDevice("u1", "laptop2", "fp-1", "macos"); err == nil {
		t.Error("duplicate fingerprint should fail")
	}
	// Same fingerprint for a different user is allowed.
	if _, err := svc.BindDevice("u2", "laptop", "fp-1", "macos"); err != nil {
		t.Errorf("different user same fingerprint: %v", err)
	}

	// List.
	if list := svc.ListBoundDevices("u1"); len(list) != 1 {
		t.Errorf("list = %d, want 1", len(list))
	}

	// Verify updates trust score.
	got, ok := svc.VerifyDeviceBinding("u1", "fp-1")
	if !ok || got.TrustScore != 55 {
		t.Errorf("verify = %+v %v", got, ok)
	}
	if _, ok := svc.VerifyDeviceBinding("u1", "fp-x"); ok {
		t.Error("unknown fingerprint should not verify")
	}

	// Unbind.
	if err := svc.UnbindDevice(d.DeviceID); err != nil {
		t.Fatalf("UnbindDevice: %v", err)
	}
	if err := svc.UnbindDevice(d.DeviceID); err == nil {
		t.Error("double unbind should fail")
	}
	if list := svc.ListBoundDevices("u1"); len(list) != 0 {
		t.Error("device should be unbound")
	}
}

// --- session_revocation_manager.go ---

func TestSessionRevocationManager_RevokeUser(t *testing.T) {
	env := newTaTestEnv(t)
	ctx := context.Background()
	tenantID, userID := uuid.New(), uuid.New()

	// Seed active JTIs.
	sessRepo := env.sessRepo
	sessRepo.sessions[uuid.New()] = &domain.Session{
		ID: uuid.New(), TenantID: tenantID, UserID: userID,
		JTI: "jti-1", TokenExp: time.Now().Add(time.Hour), ExpiresAt: time.Now().Add(time.Hour),
	}

	mgr := NewSessionRevocationManager(
		sessRepo, newTaRefreshRepo(), ggidauth.NewJTIBlocklist(env.rdb), env.rdb, nil,
	)

	// ListActiveJTIForUser in our mock returns nil — seed via a custom repo is
	// unnecessary; RevokeUser tolerates empty JTI lists.
	res, err := mgr.RevokeUser(ctx, tenantID, userID, "test revocation")
	if err != nil {
		t.Fatalf("RevokeUser: %v", err)
	}
	if res.RefreshRevoked != 1 {
		t.Errorf("RefreshRevoked = %d, want 1", res.RefreshRevoked)
	}
}

// jtiListingRepo wraps taSessionRepo to return active JTIs.
type jtiListingRepo struct {
	*taSessionRepo
	jtis []domain.SessionJTI
}

func (r *jtiListingRepo) ListActiveJTIForUser(_ context.Context, _, _ uuid.UUID) ([]domain.SessionJTI, error) {
	return r.jtis, nil
}

func TestSessionRevocationManager_BlocklistsJTIs(t *testing.T) {
	env := newTaTestEnv(t)
	ctx := context.Background()
	tenantID, userID := uuid.New(), uuid.New()

	repo := &jtiListingRepo{
		taSessionRepo: newTaSessionRepo(),
		jtis: []domain.SessionJTI{
			{SessionID: uuid.New(), JTI: "jti-block-me", TokenExp: time.Now().Add(time.Hour)},
			{SessionID: uuid.New(), JTI: "", TokenExp: time.Time{}},           // skipped: empty JTI
			{SessionID: uuid.New(), JTI: "jti-expired", TokenExp: time.Time{}}, // fallback TTL
		},
	}
	blocklist := ggidauth.NewJTIBlocklist(env.rdb)
	mgr := NewSessionRevocationManager(repo, newTaRefreshRepo(), blocklist, env.rdb, nil)

	res, err := mgr.RevokeUser(ctx, tenantID, userID, "cae test")
	if err != nil {
		t.Fatalf("RevokeUser: %v", err)
	}
	if res.JTIsBlocked != 2 || res.SessionsRevoked != 3 {
		t.Errorf("result = %+v", res)
	}
	// Verify JTIs are actually blocked in Redis.
	if !blocklist.IsRevoked(ctx, "jti-block-me") {
		t.Error("jti-block-me should be revoked")
	}
}
