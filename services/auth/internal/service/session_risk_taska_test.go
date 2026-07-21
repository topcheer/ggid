package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ggid/ggid/services/auth/internal/domain"
	"github.com/google/uuid"
)

// --- session_management.go ---

func TestSessionManagement_EnforceSessionLimit(t *testing.T) {
	env := newTaTestEnv(t)
	ctx := env.ctx()
	cfg := env.svc.cfg
	cfg.SessionTimeout.MaxSessions = 2

	sm := NewSessionManagement(env.svc.sessionService, cfg)
	userID := uuid.New()

	// Unlimited when MaxSessions <= 0.
	cfg.SessionTimeout.MaxSessions = 0
	if err := sm.EnforceSessionLimit(ctx, env.tenantID, userID); err != nil {
		t.Fatalf("unlimited: %v", err)
	}
	cfg.SessionTimeout.MaxSessions = 2

	// Create 3 active sessions with distinct ages.
	for i := 0; i < 3; i++ {
		_ = env.sessRepo.Create(ctx, &domain.Session{
			TenantID:  env.tenantID,
			UserID:    userID,
			ExpiresAt: time.Now().Add(time.Hour),
			CreatedAt: time.Now().Add(time.Duration(-i) * time.Minute),
		})
	}
	if err := sm.EnforceSessionLimit(ctx, env.tenantID, userID); err != nil {
		t.Fatalf("EnforceSessionLimit: %v", err)
	}
	active, _ := env.sessRepo.ListByUser(ctx, env.tenantID, userID)
	if len(active) != 2 {
		t.Errorf("active sessions = %d, want 2", len(active))
	}
}

func TestSessionManagement_ForceLogout(t *testing.T) {
	env := newTaTestEnv(t)
	ctx := env.ctx()
	sm := NewSessionManagement(env.svc.sessionService, env.svc.cfg)
	userID := uuid.New()

	keep := &domain.Session{TenantID: env.tenantID, UserID: userID, ExpiresAt: time.Now().Add(time.Hour)}
	other := &domain.Session{TenantID: env.tenantID, UserID: userID, ExpiresAt: time.Now().Add(time.Hour)}
	expired := &domain.Session{TenantID: env.tenantID, UserID: userID, ExpiresAt: time.Now().Add(-time.Hour)}
	_ = env.sessRepo.Create(ctx, keep)
	_ = env.sessRepo.Create(ctx, other)
	_ = env.sessRepo.Create(ctx, expired)

	n, err := sm.ForceLogout(ctx, env.tenantID, userID, keep.ID)
	if err != nil || n != 1 {
		t.Errorf("ForceLogout = %d, %v; want 1, nil", n, err)
	}
	if keep.RevokedAt != nil {
		t.Error("exempt session must stay active")
	}
	if other.RevokedAt == nil {
		t.Error("other session should be revoked")
	}

	// AuthService.ForceLogout variant (no exemption).
	n, err = env.svc.ForceLogout(ctx, env.tenantID, userID, uuid.Nil)
	if err != nil || n != 1 {
		t.Errorf("AuthService.ForceLogout = %d, %v; want 1, nil", n, err)
	}
}

func TestAuthService_EnforceSessionLimit(t *testing.T) {
	env := newTaTestEnv(t)
	ctx := env.ctx()
	env.svc.cfg.SessionTimeout.MaxSessions = 1
	userID := uuid.New()

	for i := 0; i < 2; i++ {
		_ = env.sessRepo.Create(ctx, &domain.Session{
			TenantID:  env.tenantID,
			UserID:    userID,
			ExpiresAt: time.Now().Add(time.Hour),
			CreatedAt: time.Now().Add(time.Duration(-i) * time.Minute),
		})
	}
	if err := env.svc.EnforceSessionLimit(ctx, env.tenantID, userID); err != nil {
		t.Fatalf("EnforceSessionLimit: %v", err)
	}
	active, _ := env.sessRepo.ListByUser(ctx, env.tenantID, userID)
	if len(active) != 1 {
		t.Errorf("active = %d, want 1", len(active))
	}
}

func TestDeviceFingerprint_SessionBinding(t *testing.T) {
	env := newTaTestEnv(t)
	ctx := env.ctx()
	sid := uuid.New()

	fp := GenerateDeviceFingerprint("Mozilla/5.0", "1.2.3.4")
	if fp == "" || fp != GenerateDeviceFingerprint("Mozilla/5.0", "1.2.3.4") {
		t.Error("fingerprint should be deterministic and non-empty")
	}
	if fp == GenerateDeviceFingerprint("Other", "1.2.3.4") {
		t.Error("different UA should yield different fingerprint")
	}

	// No fingerprint bound → allow.
	if !env.svc.VerifySessionFingerprint(ctx, sid, fp) {
		t.Error("unbound session should verify")
	}
	if err := env.svc.BindFingerprintToSession(ctx, sid, fp); err != nil {
		t.Fatalf("BindFingerprintToSession: %v", err)
	}
	if !env.svc.VerifySessionFingerprint(ctx, sid, fp) {
		t.Error("bound session should verify with matching fingerprint")
	}
	if env.svc.VerifySessionFingerprint(ctx, sid, "different") {
		t.Error("mismatched fingerprint should fail verification")
	}
}

// --- hooks.go ---

func TestHookManager_PreHookAllowAndDeny(t *testing.T) {
	mgr := NewHookManager()

	allowSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(HookResponse{Allow: true})
	}))
	defer allowSrv.Close()
	denySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(HookResponse{Allow: false, Message: "blocked by policy"})
	}))
	defer denySrv.Close()

	mgr.RegisterHook(&AuthHook{ID: "h1", Event: HookPreLogin, URL: allowSrv.URL, Enabled: true})
	mgr.RegisterHook(&AuthHook{ID: "h2", Event: HookPreLogin, URL: denySrv.URL, Enabled: false}) // disabled → skipped

	payload := &HookPayload{Event: HookPreLogin, UserID: "u1"}
	if err := mgr.ExecuteHooks(context.Background(), HookPreLogin, payload); err != nil {
		t.Fatalf("allow hook: %v", err)
	}

	// Enable the denying hook → pre-hook must block.
	mgr.RegisterHook(&AuthHook{ID: "h2", Event: HookPreLogin, URL: denySrv.URL, Enabled: true})
	if err := mgr.ExecuteHooks(context.Background(), HookPreLogin, payload); err == nil {
		t.Error("denying pre-hook should block the flow")
	}

	// Remove it → flow allowed again.
	mgr.RemoveHook("h2")
	if err := mgr.ExecuteHooks(context.Background(), HookPreLogin, payload); err != nil {
		t.Fatalf("after remove: %v", err)
	}
}

func TestHookManager_PostHookFireAndForget(t *testing.T) {
	mgr := NewHookManager()
	failSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer failSrv.Close()

	mgr.RegisterHook(&AuthHook{ID: "post1", Event: HookPostLogin, URL: failSrv.URL, Enabled: true})
	// Post-hook errors must not affect the flow.
	if err := mgr.ExecuteHooks(context.Background(), HookPostLogin, &HookPayload{}); err != nil {
		t.Errorf("post-hook error should be ignored: %v", err)
	}

	// Unreachable URL also ignored for post-hooks.
	mgr.RegisterHook(&AuthHook{ID: "post2", Event: HookPostRegister, URL: "http://127.0.0.1:1", Enabled: true})
	if err := mgr.ExecuteHooks(context.Background(), HookPostRegister, &HookPayload{}); err == nil {
		// Pre-register hook with unreachable URL should block instead.
	}
	mgr.RegisterHook(&AuthHook{ID: "pre1", Event: HookPreRegister, URL: "http://127.0.0.1:1", Enabled: true})
	if err := mgr.ExecuteHooks(context.Background(), HookPreRegister, &HookPayload{}); err == nil {
		t.Error("unreachable pre-register hook should block")
	}
}

// --- login_attempt.go ---

func TestLoginAttempts_RecordAndGet(t *testing.T) {
	env := newTaTestEnv(t)
	ctx := context.Background()

	env.svc.RecordLoginAttempt(ctx, "alice", "1.1.1.1", "UA/1.0", true, "")
	env.svc.RecordLoginAttempt(ctx, "alice", "1.1.1.2", "UA/2.0", false, "bad password")

	attempts, err := env.svc.GetLoginAttempts(ctx, "alice", 10)
	if err != nil {
		t.Fatalf("GetLoginAttempts: %v", err)
	}
	if len(attempts) != 2 {
		t.Fatalf("attempts = %d, want 2", len(attempts))
	}
	// Most recent first.
	if attempts[0].Success {
		t.Error("most recent attempt (failure) should come first")
	}
	if attempts[1].FailureReason != "" {
		t.Error("older attempt was successful")
	}

	// Default limit when out of range.
	if _, err := env.svc.GetLoginAttempts(ctx, "alice", 0); err != nil {
		t.Errorf("limit=0 should use default: %v", err)
	}
}

// --- pii_logging.go ---

func TestPIIObfuscation(t *testing.T) {
	if out := obfuscateEmail("alice@example.com"); out == "alice@example.com" {
		t.Error("email should be masked")
	}
	_ = obfuscateForLog("call +8613800138000 now")
}

// --- logout_all.go ---

func TestLogoutAll(t *testing.T) {
	env := newTaTestEnv(t)
	ctx := env.ctx()
	userID := uuid.New()

	_ = env.sessRepo.Create(ctx, &domain.Session{TenantID: env.tenantID, UserID: userID, ExpiresAt: time.Now().Add(time.Hour)})
	if err := env.svc.LogoutAll(ctx, env.tenantID, userID, uuid.Nil); err != nil {
		t.Fatalf("LogoutAll: %v", err)
	}
	active, _ := env.sessRepo.ListByUser(ctx, env.tenantID, userID)
	if len(active) != 0 {
		t.Error("all sessions should be revoked")
	}
}

// --- risk_auth.go ---

func TestAssessLoginRisk_LowBaseline(t *testing.T) {
	env := newTaTestEnv(t)
	ctx := context.Background()
	userID := uuid.New()

	// Known IP → only unknown-IP penalty skipped; still low/medium depending on hour.
	env.svc.RecordSuccessfulLogin(ctx, userID, "1.2.3.4", "UA")
	a := env.svc.AssessLoginRisk(ctx, env.tenantID, userID, "1.2.3.4", "UA")
	if a.Score >= 30 {
		t.Errorf("baseline score = %d, want < 30 (reasons: %v)", a.Score, a.Reasons)
	}
}

func TestAssessLoginRisk_Escalations(t *testing.T) {
	env := newTaTestEnv(t)
	ctx := context.Background()
	userID := uuid.New()

	// Unknown IP → +15, medium.
	a := env.svc.AssessLoginRisk(ctx, env.tenantID, userID, "9.9.9.9", "")
	if a.Score < 15 {
		t.Errorf("unknown IP score = %d, want >= 15", a.Score)
	}

	// Failed attempts from IP → escalating score.
	for i := 0; i < 5; i++ {
		env.svc.RecordFailedLoginAttempt(ctx, userID, "8.8.8.8")
	}
	a = env.svc.AssessLoginRisk(ctx, env.tenantID, userID, "8.8.8.8", "")
	if a.Score < 40 || !a.RequiresStepUp {
		t.Errorf("failed-attempt score = %d stepUp=%v", a.Score, a.RequiresStepUp)
	}

	// Multi-user brute force from same IP → high + admin alert.
	for i := 0; i < 3; i++ {
		env.svc.RecordFailedLoginAttempt(ctx, uuid.New(), "7.7.7.7")
	}
	a = env.svc.AssessLoginRisk(ctx, env.tenantID, userID, "7.7.7.7", "")
	if a.Level != RiskLevelHigh || !a.RequiresAdminAlert {
		t.Errorf("brute-force assessment = %+v", a)
	}

	// UA change → medium signal.
	env.svc.RecordSuccessfulLogin(ctx, userID, "1.2.3.4", "UA-1")
	a = env.svc.AssessLoginRisk(ctx, env.tenantID, userID, "1.2.3.4", "UA-2")
	found := false
	for _, r := range a.Reasons {
		if r == "user agent changed since last login" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected UA-change reason, got %v", a.Reasons)
	}
}

func TestBlockSuspiciousIP(t *testing.T) {
	env := newTaTestEnv(t)
	ctx := context.Background()

	if env.svc.IsIPBlocked(ctx, "5.5.5.5") {
		t.Error("IP should not be blocked initially")
	}
	env.svc.BlockSuspiciousIP(ctx, "5.5.5.5", time.Minute)
	if !env.svc.IsIPBlocked(ctx, "5.5.5.5") {
		t.Error("IP should be blocked")
	}
}

// --- email_lockout.go ---

func TestAccountLockoutService_Redis(t *testing.T) {
	env := newTaTestEnv(t)
	ctx := context.Background()
	tenantID := uuid.New()

	svc := NewAccountLockoutService(env.rdb, 3, time.Minute)
	if svc.MaxAttempts() != 3 || svc.LockDuration() != time.Minute {
		t.Error("config mismatch")
	}
	// Defaults applied for non-positive values.
	def := NewAccountLockoutService(env.rdb, 0, 0)
	if def.MaxAttempts() != 5 || def.LockDuration() != 15*time.Minute {
		t.Error("defaults not applied")
	}

	if svc.IsLocked(ctx, tenantID, "u") {
		t.Error("should not be locked initially")
	}
	for i := 0; i < 3; i++ {
		if err := svc.RecordFailedAttempt(ctx, tenantID, "u"); err != nil {
			t.Fatalf("RecordFailedAttempt: %v", err)
		}
	}
	if !svc.IsLocked(ctx, tenantID, "u") {
		t.Error("should be locked at threshold")
	}
	svc.ResetAttempts(ctx, tenantID, "u")
	if svc.IsLocked(ctx, tenantID, "u") {
		t.Error("should be unlocked after reset")
	}
}

func TestEmailService_TokenLifecycle(t *testing.T) {
	env := newTaTestEnv(t)
	ctx := context.Background()
	svc := env.svc.emailService

	tenantID, userID := uuid.New(), uuid.New()
	token, err := svc.IssueVerificationToken(ctx, tenantID, userID, "x@y.z")
	if err != nil {
		t.Fatalf("IssueVerificationToken: %v", err)
	}
	gotT, gotU, gotE, err := svc.VerifyEmailToken(ctx, token)
	if err != nil || gotT != tenantID || gotU != userID || gotE != "x@y.z" {
		t.Errorf("verify = %v %v %v %v", gotT, gotU, gotE, err)
	}
	// Single use.
	if _, _, _, err := svc.VerifyEmailToken(ctx, token); err == nil {
		t.Error("token should be consumed")
	}
	// Garbage token.
	if _, _, _, err := svc.VerifyEmailToken(ctx, "garbage"); err == nil {
		t.Error("invalid token should error")
	}
}

// --- device_tracking.go ---

func TestDeviceTracking(t *testing.T) {
	env := newTaTestEnv(t)
	ctx := context.Background()
	tenantID, userID, sid := uuid.New(), uuid.New(), uuid.New()
	ss := env.svc.sessionService

	if err := ss.TrackDevice(ctx, env.rdb, tenantID, userID, sid, "UA", "1.1.1.1"); err != nil {
		t.Fatalf("TrackDevice: %v", err)
	}
	devices, err := ss.ListDevices(ctx, env.rdb, tenantID, userID)
	if err != nil || len(devices) != 1 {
		t.Fatalf("ListDevices = %v, %v", devices, err)
	}
	if devices[0].SessionID != sid.String() || devices[0].Fingerprint == "" {
		t.Errorf("unexpected device: %+v", devices[0])
	}
	if err := ss.RemoveDevice(ctx, env.rdb, tenantID, userID, sid); err != nil {
		t.Fatalf("RemoveDevice: %v", err)
	}
	devices, _ = ss.ListDevices(ctx, env.rdb, tenantID, userID)
	if len(devices) != 0 {
		t.Error("device should be removed")
	}

	// splitDeviceData with extra segment.
	parts := splitDeviceData("a:b:c:d")
	if len(parts) != 4 || parts[3] != "d" {
		t.Errorf("splitDeviceData = %v", parts)
	}
}

// --- password_history.go ---

func TestPasswordHistoryService(t *testing.T) {
	svc := NewPasswordHistoryService(2)

	svc.AddPasswordHistory("u1", "hash-a")
	svc.AddPasswordHistory("u1", "hash-b")
	svc.AddPasswordHistory("u1", "hash-c") // exceeds max → oldest dropped

	entries := svc.GetPasswordHistory("u1", 0)
	if len(entries) != 2 || entries[0].PasswordHash != "hash-c" {
		t.Errorf("history = %+v", entries)
	}
	if limited := svc.GetPasswordHistory("u1", 1); len(limited) != 1 {
		t.Errorf("limited = %d, want 1", len(limited))
	}

	// CheckPasswordHistory matches raw hash and sha256-hashed input.
	if !svc.CheckPasswordHistory("u1", "hash-c") {
		t.Error("expected duplicate detection for raw hash")
	}
	sha := hashPassword("plaintext")
	svc.AddPasswordHistory("u2", sha)
	if !svc.CheckPasswordHistory("u2", "plaintext") {
		t.Error("expected duplicate detection for hashed input")
	}
	if svc.CheckPasswordHistory("u2", "other") {
		t.Error("unexpected duplicate")
	}

	// PurgeOldEntries: no-op within limit.
	if n := svc.PurgeOldEntries("u1"); n != 0 {
		t.Errorf("purge = %d, want 0", n)
	}
}
