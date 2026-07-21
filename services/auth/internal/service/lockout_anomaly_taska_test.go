package service

import (
	"context"
	"errors"
	"testing"
	"time"
)

// --- account_lockout.go ---

func taLockoutConfig() LockoutConfig {
	return LockoutConfig{
		MaxAttempts:      3,
		WindowMinutes:    5,
		LockoutDuration:  time.Minute,
		CaptchaThreshold: 2,
	}
}

func TestLoginLockout_RecordAndCheck(t *testing.T) {
	svc := NewLoginLockoutService(taLockoutConfig())

	// Unknown user → unlocked.
	if d, s := svc.CheckLockout("ghost"); d != LockoutUnlocked || s != nil {
		t.Errorf("unknown user: %v %v", d, s)
	}

	// First attempt: below captcha threshold.
	if n := svc.RecordFailedAttempt("u1", "1.1.1.1"); n != 1 {
		t.Errorf("attempts = %d, want 1", n)
	}
	if d, _ := svc.CheckLockout("u1"); d != LockoutUnlocked {
		t.Errorf("after 1 attempt: %v, want unlocked", d)
	}

	// Second attempt: captcha threshold.
	svc.RecordFailedAttempt("u1", "1.1.1.1")
	if d, s := svc.CheckLockout("u1"); d != LockoutCaptchaRequired || s == nil {
		t.Errorf("after 2 attempts: %v, want captcha_required", d)
	}

	// Third attempt: locked.
	svc.RecordFailedAttempt("u1", "1.1.1.1")
	if d, s := svc.CheckLockout("u1"); d != LockoutLocked || !s.Locked {
		t.Errorf("after 3 attempts: %v, want locked", d)
	}
}

func TestLoginLockout_Expiry(t *testing.T) {
	svc := NewLoginLockoutService(LockoutConfig{MaxAttempts: 1, LockoutDuration: -time.Second})
	svc.RecordFailedAttempt("u1", "ip")
	// LockoutDuration in the past → immediately expired.
	if d, _ := svc.CheckLockout("u1"); d != LockoutUnlocked {
		t.Errorf("expired lock: %v, want unlocked", d)
	}
	if n := svc.AutoUnlockExpired(); n != 1 {
		t.Errorf("AutoUnlockExpired = %d, want 1", n)
	}
}

func TestLoginLockout_ManualLockUnlockReset(t *testing.T) {
	svc := NewLoginLockoutService(taLockoutConfig())

	svc.LockUser("u2", time.Hour, "admin action", "admin-1")
	d, s := svc.CheckLockout("u2")
	if d != LockoutLocked || s.Reason != "admin action" || s.LockedBy != "admin-1" {
		t.Errorf("manual lock: %v %+v", d, s)
	}

	svc.UnlockUser("u2", "admin-1")
	if d, s := svc.CheckLockout("u2"); d != LockoutUnlocked || s.Locked {
		t.Errorf("after unlock: %v", d)
	}

	// Reset attempts.
	svc.RecordFailedAttempt("u3", "ip")
	svc.ResetAttempts("u3")
	if _, s := svc.CheckLockout("u3"); s.Attempts != 0 {
		t.Errorf("attempts = %d, want 0", s.Attempts)
	}
	// Reset on unknown user is a no-op.
	svc.ResetAttempts("ghost")
	svc.UnlockUser("ghost", "admin")
}

// --- login_security.go ---

func TestLoginSecurity_IPLists(t *testing.T) {
	svc := NewLoginSecurityService(LoginSecurityConfig{
		MaxAttempts: 10,
		IPBlocklist: []string{"6.6.6.6"},
		IPAllowlist: []string{"1.1.1.1"},
	})
	if d, why := svc.CheckLoginPolicy("u", "6.6.6.6", 0, false); d != LoginDeny || why != "IP blocked" {
		t.Errorf("blocklist: %v %q", d, why)
	}
	if d, why := svc.CheckLoginPolicy("u", "2.2.2.2", 0, false); d != LoginDeny {
		t.Errorf("not in allowlist: %v %q", d, why)
	}
	if d, _ := svc.CheckLoginPolicy("u", "1.1.1.1", 0, false); d != LoginAllow {
		t.Errorf("allowlisted: %v, want allow", d)
	}
}

func TestLoginSecurity_LockAndCaptcha(t *testing.T) {
	// Captcha path (captcha threshold checked before max attempts).
	svcCaptcha := NewLoginSecurityService(LoginSecurityConfig{
		MaxAttempts:          10,
		LockoutDuration:      time.Hour,
		CaptchaAfterAttempts: 3,
	})
	if d, _ := svcCaptcha.CheckLoginPolicy("u1", "ip", 3, false); d != LoginCaptcha {
		t.Errorf("3 attempts: %v, want captcha", d)
	}

	// Lock path (no captcha threshold configured).
	svc := NewLoginSecurityService(LoginSecurityConfig{
		MaxAttempts:     10,
		LockoutDuration: time.Hour,
	})
	if d, _ := svc.CheckLoginPolicy("u1", "ip", 10, false); d != LoginDeny {
		t.Errorf("10 attempts: %v, want deny", d)
	}
	if !svc.IsAccountLocked("u1") {
		t.Error("u1 should be locked")
	}
	// Locked user is denied.
	if d, why := svc.CheckLoginPolicy("u1", "ip", 0, false); d != LoginDeny || why != "account locked" {
		t.Errorf("locked: %v %q", d, why)
	}
	// Unlock.
	svc.UnlockAccount("u1")
	if svc.IsAccountLocked("u1") {
		t.Error("u1 should be unlocked")
	}
}

func TestLoginSecurity_AdminMFA(t *testing.T) {
	svc := NewLoginSecurityService(LoginSecurityConfig{MaxAttempts: 100, EnforceMFAForAdmin: true})
	if d, _ := svc.CheckLoginPolicy("admin", "ip", 0, true); d != LoginStepUp {
		t.Errorf("admin: %v, want step_up", d)
	}
	if d, _ := svc.CheckLoginPolicy("user", "ip", 0, false); d != LoginAllow {
		t.Errorf("user: %v, want allow", d)
	}
}

func TestLoginSecurity_AnomalyCaptcha(t *testing.T) {
	svc := NewLoginSecurityService(LoginSecurityConfig{MaxAttempts: 10, LockoutDuration: time.Minute})
	// 5 <= attempts < max → anomalous pattern → captcha.
	if d, why := svc.CheckLoginPolicy("u", "ip", 5, false); d != LoginCaptcha || why == "" {
		t.Errorf("anomaly: %v %q", d, why)
	}
}

func TestLoginSecurity_ExpiredLock(t *testing.T) {
	svc := NewLoginSecurityService(LoginSecurityConfig{MaxAttempts: 100})
	svc.LockAccount("u", -time.Second) // already expired
	if svc.IsAccountLocked("u") {
		t.Error("expired lock should report unlocked")
	}
	if d, _ := svc.CheckLoginPolicy("u", "ip", 0, false); d != LoginAllow {
		t.Errorf("expired lock: %v, want allow", d)
	}
}

// --- ratelimit_service.go ---

func TestRateLimiter_CheckAndIncrement(t *testing.T) {
	env := newTaTestEnv(t)
	rl := env.svc.rateLimiter
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		if err := rl.CheckAndIncrement(ctx, "k1", 3); err != nil {
			t.Fatalf("request %d: %v", i, err)
		}
	}
	if err := rl.CheckAndIncrement(ctx, "k1", 3); !errors.Is(err, ErrRateLimited) {
		t.Errorf("4th request: %v, want ErrRateLimited", err)
	}
	// Different key is independent.
	if err := rl.CheckAndIncrement(ctx, "k2", 3); err != nil {
		t.Errorf("k2: %v", err)
	}
}

// --- anomaly_detection.go ---

func TestRecordFailedLoginAnomaly_Lockout(t *testing.T) {
	env := newTaTestEnv(t)
	ctx := context.Background()

	for i := 0; i < anomalyLockoutThreshold-1; i++ {
		res, err := env.svc.RecordFailedLoginAnomaly(ctx, "victim")
		if err != nil {
			t.Fatalf("attempt %d: %v", i, err)
		}
		if res.Locked {
			t.Fatalf("locked too early at attempt %d", i)
		}
	}
	res, err := env.svc.RecordFailedLoginAnomaly(ctx, "victim")
	if err != nil {
		t.Fatalf("final attempt: %v", err)
	}
	if !res.Locked || res.LockReason == "" {
		t.Errorf("expected lockout, got %+v", res)
	}
	if !env.svc.IsLoginLocked(ctx, "victim") {
		t.Error("IsLoginLocked should be true")
	}

	// Clear on success.
	env.svc.ClearFailedLogins(ctx, "victim")
	if env.svc.IsLoginLocked(ctx, "victim") {
		t.Error("lock should be cleared")
	}
}

func TestCheckGeoAnomaly(t *testing.T) {
	// No history → no anomaly.
	if CheckGeoAnomaly(31.2, 121.5, nil) {
		t.Error("no history should not be anomalous")
	}
	known := map[string]string{"1.1.1.1": "31.20,121.50"}
	// Same city → not anomalous.
	if CheckGeoAnomaly(31.2, 121.5, known) {
		t.Error("same location should not be anomalous")
	}
	// Far away (New York vs Shanghai) → anomalous.
	if !CheckGeoAnomaly(40.7, -74.0, known) {
		t.Error("distant location should be anomalous")
	}
	// Invalid coords → not anomalous.
	if CheckGeoAnomaly(40.7, -74.0, map[string]string{"1.1.1.1": "garbage"}) {
		t.Error("invalid coords should not be anomalous")
	}
}

func TestCheckNewDevice(t *testing.T) {
	if CheckNewDevice("fp1", []string{"fp1", "fp2"}) {
		t.Error("known device should not be new")
	}
	if !CheckNewDevice("fp3", []string{"fp1", "fp2"}) {
		t.Error("unknown device should be new")
	}
}

func TestKnownDevices_Redis(t *testing.T) {
	env := newTaTestEnv(t)
	ctx := context.Background()

	if err := env.svc.RecordKnownDevice(ctx, "u1", "fp-a"); err != nil {
		t.Fatalf("RecordKnownDevice: %v", err)
	}
	devices, err := env.svc.GetKnownDevices(ctx, "u1")
	if err != nil || len(devices) != 1 || devices[0] != "fp-a" {
		t.Errorf("GetKnownDevices = %v, %v", devices, err)
	}
}

func TestAssessLoginAnomaly(t *testing.T) {
	env := newTaTestEnv(t)
	ctx := context.Background()

	// Locked user short-circuits.
	env.svc.ClearFailedLogins(ctx, "locked-user")
	for i := 0; i < anomalyLockoutThreshold; i++ {
		_, _ = env.svc.RecordFailedLoginAnomaly(ctx, "locked-user")
	}
	res, err := env.svc.AssessLoginAnomaly(ctx, "locked-user", "u1", "1.1.1.1", "fp", 0, 0)
	if err != nil || !res.Locked {
		t.Errorf("locked assessment = %+v, %v", res, err)
	}

	// Geo + device anomaly for a fresh user with history.
	uid := "u2"
	env.rdb.HSet(ctx, "ggid:anomaly:ips:"+uid, "1.1.1.1", "31.20,121.50")
	_ = env.svc.RecordKnownDevice(ctx, uid, "old-fp")
	res, err = env.svc.AssessLoginAnomaly(ctx, "user2", uid, "2.2.2.2", "new-fp", 40.7, -74.0)
	if err != nil {
		t.Fatalf("AssessLoginAnomaly: %v", err)
	}
	if !res.GeoAnomaly || !res.NewDevice || !res.RequireNotify {
		t.Errorf("expected geo+device anomaly, got %+v", res)
	}

	// Known location + known device → clean.
	res, err = env.svc.AssessLoginAnomaly(ctx, "user2", uid, "1.1.1.1", "old-fp", 31.2, 121.5)
	if err != nil {
		t.Fatalf("AssessLoginAnomaly: %v", err)
	}
	if res.GeoAnomaly || res.NewDevice || res.Locked {
		t.Errorf("expected clean result, got %+v", res)
	}
}

func TestHaversineDistance(t *testing.T) {
	// Shanghai → New York ≈ 11800 km.
	d := haversineDistance(31.2, 121.5, 40.7, -74.0)
	if d < 11000 || d > 12500 {
		t.Errorf("distance = %.0f km, out of expected range", d)
	}
	// Same point → 0.
	if d := haversineDistance(1, 1, 1, 1); d != 0 {
		t.Errorf("same point distance = %v, want 0", d)
	}
}
