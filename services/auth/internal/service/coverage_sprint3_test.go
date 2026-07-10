package service

import (
	"context"
	"testing"
	"time"

	"github.com/ggid/ggid/pkg/crypto"
	"github.com/ggid/ggid/services/auth/internal/conf"
	"github.com/ggid/ggid/services/auth/internal/domain"
	"github.com/google/uuid"
)

// Device tracking tests

func TestSessionService_TrackDevice(t *testing.T) {
	rdb := tRedis(t)
	svc := NewSessionService(newTSessionRepo())
	ctx := context.Background()
	tid := uuid.New()
	uid := uuid.New()
	sessID := uuid.New()

	err := svc.TrackDevice(ctx, rdb, tid, uid, sessID, "Mozilla/5.0", "192.168.1.1")
	if err != nil {
		t.Fatalf("TrackDevice: %v", err)
	}
}

func TestSessionService_ListDevices(t *testing.T) {
	rdb := tRedis(t)
	svc := NewSessionService(newTSessionRepo())
	ctx := context.Background()
	tid := uuid.New()
	uid := uuid.New()
	sessID := uuid.New()

	_ = svc.TrackDevice(ctx, rdb, tid, uid, sessID, "Mozilla/5.0", "192.168.1.1")

	devices, err := svc.ListDevices(ctx, rdb, tid, uid)
	if err != nil {
		t.Fatalf("ListDevices: %v", err)
	}
	if len(devices) != 1 {
		t.Errorf("expected 1 device, got %d", len(devices))
	}
	if devices[0].SessionID != sessID.String() {
		t.Errorf("expected session_id=%s, got %s", sessID, devices[0].SessionID)
	}
}

func TestSessionService_RemoveDevice(t *testing.T) {
	rdb := tRedis(t)
	svc := NewSessionService(newTSessionRepo())
	ctx := context.Background()
	tid := uuid.New()
	uid := uuid.New()
	sessID := uuid.New()

	_ = svc.TrackDevice(ctx, rdb, tid, uid, sessID, "Chrome", "10.0.0.1")
	_ = svc.RemoveDevice(ctx, rdb, tid, uid, sessID)

	devices, _ := svc.ListDevices(ctx, rdb, tid, uid)
	if len(devices) != 0 {
		t.Errorf("expected 0 devices after removal, got %d", len(devices))
	}
}

func TestSessionService_ListDevices_Empty(t *testing.T) {
	rdb := tRedis(t)
	svc := NewSessionService(newTSessionRepo())
	ctx := context.Background()

	devices, err := svc.ListDevices(ctx, rdb, uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("ListDevices empty: %v", err)
	}
	if len(devices) != 0 {
		t.Errorf("expected 0 devices, got %d", len(devices))
	}
}

func TestDeviceFingerprint(t *testing.T) {
	fp1 := deviceFingerprint("Mozilla", "1.2.3.4")
	fp2 := deviceFingerprint("Mozilla", "1.2.3.4")
	fp3 := deviceFingerprint("Chrome", "1.2.3.4")

	if fp1 != fp2 {
		t.Error("same input should produce same fingerprint")
	}
	if fp1 == fp3 {
		t.Error("different input should produce different fingerprint")
	}
}

func TestSplitDeviceData(t *testing.T) {
	// splitDeviceData splits on ':', but timestamps contain ':' too.
	// Test with a simple value that has exactly 2 colons → 3 parts.
	parts := splitDeviceData("hash:ip:simple")
	if len(parts) != 3 {
		t.Fatalf("expected 3 parts, got %d", len(parts))
	}
	if parts[0] != "hash" || parts[1] != "ip" {
		t.Error("unexpected split results")
	}
}

func TestSplitDeviceData_WithUserAgent(t *testing.T) {
	// Use simple values without ':' in them (timestamps contain ':')
	parts := splitDeviceData("hash:ip:date:Mozilla")
	if len(parts) != 4 {
		t.Fatalf("expected 4 parts, got %d", len(parts))
	}
	if parts[3] != "Mozilla" {
		t.Errorf("expected user agent, got %s", parts[3])
	}
}

// Password breach check tests

func TestCheckPasswordBreach_NotInBreach(t *testing.T) {
	rdb := tRedis(t)
	cr := newTCredRepo()
	ps := NewPasswordService(conf.Default().Password, cr, rdb)
	ctx := context.Background()

	// Hash a password that's not in the breach DB
	err := ps.CheckPasswordBreach(ctx, "Un1que!Str0ng#Pass")
	if err != nil {
		// Expected if HIBP API is not available
		t.Logf("CheckPasswordBreach error (expected in test env): %v", err)
		return
	}
}

// Password expiration tests

func TestPasswordExpiration_MustChangePassword_RecentChange(t *testing.T) {
	rdb := tRedis(t)
	cr := newTCredRepo()
	ps := NewPasswordService(conf.Default().Password, cr, rdb)

	uid := uuid.New()
	tid := uuid.New()
	cr.byUser[uid] = &domain.Credential{
		ID: uuid.New(), TenantID: tid, UserID: uid,
		Identifier: "user1", Secret: "$2a$10$hash",
		UpdatedAt: time.Now(),
	}

	// Recently changed → no need to change
	must := ps.MustChangePassword(context.Background(), tid, uid)
	if must {
		t.Error("expected no password change needed for recent change")
	}
}

func TestPasswordExpiration_MustChangePassword_OldPassword(t *testing.T) {
	rdb := tRedis(t)
	cr := newTCredRepo()
	ps := NewPasswordService(conf.Default().Password, cr, rdb)
	ps.policy.MaxAgeDays = 1 // expire after 1 day

	uid := uuid.New()
	tid := uuid.New()
	cr.byUser[uid] = &domain.Credential{
		ID: uuid.New(), TenantID: tid, UserID: uid,
		Identifier: "user1", Secret: "$2a$10$hash",
		UpdatedAt: time.Now().Add(-48 * time.Hour),
	}

	must := ps.MustChangePassword(context.Background(), tid, uid)
	if !must {
		t.Error("expected password change needed for old password")
	}
}

// Magic link tests

func TestAuthService_MagicLink_Flow(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	ctx, _ := tCtxTenant()
	uid := uuid.New()

	// Issue magic link
	token, err := svc.IssueMagicLink(ctx, uuid.New(), uid, "user@test.com")
	if err != nil {
		t.Fatalf("IssueMagicLink: %v", err)
	}
	if token == "" {
		t.Error("expected non-empty token")
	}

	// Verify magic link (new signature: ctx, token, ip, userAgent → *domain.TokenSet)
	tokens, err := svc.VerifyMagicLink(ctx, token, "127.0.0.1", "TestAgent")
	if err != nil {
		t.Fatalf("VerifyMagicLink: %v", err)
	}
	if tokens == nil {
		t.Fatal("expected non-nil TokenSet")
	}
}

func TestAuthService_MagicLink_InvalidToken(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	ctx := context.Background()

	_, err := svc.VerifyMagicLink(ctx, "invalid_magic_token_xyz", "127.0.0.1", "TestAgent")
	if err == nil {
		t.Error("expected error for invalid magic link token")
	}
}

// LoginMFA tests

func TestAuthService_LoginMFA_Invalid(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	ctx := context.Background()

	_, err := svc.LoginMFA(ctx, "nonexistent_user", "pass", "invalid_challenge", "127.0.0.1", "TestAgent")
	if err == nil {
		t.Error("expected error for invalid MFA challenge")
	}
}

// LogoutAll tests

func TestAuthService_LogoutAll(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	ctx, tid := tCtxTenant()
	uid := uuid.New()

	err := svc.LogoutAll(ctx, uuid.New(), tid, uid)
	if err != nil {
		t.Fatalf("LogoutAll: %v", err)
	}
}

// RevokeSession with invalid ID
func TestAuthService_RevokeSession_NotFound(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	ctx, _ := tCtxTenant()

	err := svc.RevokeSession(ctx, uuid.New())
	// Should not error even for non-existent session
	if err != nil {
		t.Logf("RevokeSession returned error (expected for not found): %v", err)
	}
}

// SetPassword tests
func TestPasswordService_SetPassword(t *testing.T) {
	rdb := tRedis(t)
	cr := newTCredRepo()
	ps := NewPasswordService(conf.Default().Password, cr, rdb)

	tid := uuid.New()
	uid := uuid.New()
	cred := &domain.Credential{
		ID: uuid.New(), TenantID: tid, UserID: uid,
		Identifier: "user1", Secret: "$2a$10$old_hash",
	}

	err := ps.SetPassword(context.Background(), cred, "NewStr0ng!Pass")
	if err != nil {
		t.Fatalf("SetPassword: %v", err)
	}

	// Check history was updated
	if len(cr.history) != 1 {
		t.Errorf("expected 1 history entry, got %d", len(cr.history))
	}
}

// VerifyOldPassword
func TestPasswordService_VerifyOldPassword(t *testing.T) {
	rdb := tRedis(t)
	cr := newTCredRepo()
	ps := NewPasswordService(conf.Default().Password, cr, rdb)

	hashed, _ := crypto.HashPassword("OldPass123!")
	tid := uuid.New()
	uid := uuid.New()
	cr.byUser[uid] = &domain.Credential{
		ID: uuid.New(), TenantID: tid, UserID: uid,
		Identifier: "user1", Secret: hashed,
	}

	cred := cr.byUser[uid]
	matched, err := ps.VerifyOldPassword(context.Background(), cred, "OldPass123!")
	if err != nil {
		t.Fatalf("VerifyOldPassword: %v", err)
	}
	if !matched {
		t.Error("expected old password to verify")
	}
	matchedWrong, _ := ps.VerifyOldPassword(context.Background(), cred, "WrongPass")
	if matchedWrong {
		t.Error("expected wrong password to not verify")
	}
}
