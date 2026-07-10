package service

import (
	"context"
	"testing"
)

func TestCheckGeoAnomaly_NoHistory(t *testing.T) {
	// No known IPs — not an anomaly (first login).
	if CheckGeoAnomaly(40.7, -74.0, nil) {
		t.Error("expected false for empty known IPs")
	}
	if CheckGeoAnomaly(40.7, -74.0, map[string]string{}) {
		t.Error("expected false for empty map")
	}
}

func TestCheckGeoAnomaly_WithinRange(t *testing.T) {
	known := map[string]string{
		"1.2.3.4": "40.7,-74.0", // NYC
	}
	// Same location → not anomaly.
	if CheckGeoAnomaly(40.71, -74.01, known) {
		t.Error("expected false for same city")
	}
}

func TestCheckGeoAnomaly_FarAway(t *testing.T) {
	known := map[string]string{
		"1.2.3.4": "40.7,-74.0",  // NYC
		"5.6.7.8": "41.8,-87.6",  // Chicago
	}
	// Login from LA (34.0, -118.2) — > 500km from both.
	if !CheckGeoAnomaly(34.0, -118.2, known) {
		t.Error("expected true for far-away login")
	}
}

func TestCheckGeoAnomaly_InvalidCoords(t *testing.T) {
	known := map[string]string{
		"1.2.3.4": "invalid",
	}
	// Invalid coords are skipped; with no valid entries, should return false.
	if CheckGeoAnomaly(40.7, -74.0, known) {
		t.Error("expected false when all known coords are invalid")
	}
}

func TestCheckNewDevice_Unknown(t *testing.T) {
	known := []string{"fp1", "fp2", "fp3"}
	if !CheckNewDevice("fp4", known) {
		t.Error("expected true for unknown device")
	}
}

func TestCheckNewDevice_Known(t *testing.T) {
	known := []string{"fp1", "fp2", "fp3"}
	if CheckNewDevice("fp2", known) {
		t.Error("expected false for known device")
	}
}

func TestCheckNewDevice_EmptyList(t *testing.T) {
	// No known devices — everything is new.
	if !CheckNewDevice("fp1", nil) {
		t.Error("expected true for empty device list")
	}
	if !CheckNewDevice("fp1", []string{}) {
		t.Error("expected true for empty device list")
	}
}

func TestHaversineDistance(t *testing.T) {
	// NYC to LA ≈ 3936 km.
	dist := haversineDistance(40.7128, -74.0060, 34.0522, -118.2437)
	if dist < 3900 || dist > 4000 {
		t.Errorf("NYC-LA distance = %.0f km, expected ~3936", dist)
	}

	// Same point → 0.
	dist = haversineDistance(40.7, -74.0, 40.7, -74.0)
	if dist != 0 {
		t.Errorf("same point distance = %f, expected 0", dist)
	}
}

func TestRecordFailedLogin_Lockout(t *testing.T) {
	svc := newAuthSvcWithRedis(t)
	ctx := context.Background()
	username := "testuser_lockout"

	// 4 failures should not lock.
	for i := 0; i < 4; i++ {
		result, err := svc.RecordFailedLoginAnomaly(ctx, username)
		if err != nil {
			t.Fatalf("RecordFailedLogin #%d: %v", i+1, err)
		}
		if result.Locked {
			t.Fatalf("locked after only %d attempts", i+1)
		}
		if result.Remaining != 4-i {
			t.Errorf("attempt %d: remaining = %d, want %d", i+1, result.Remaining, 4-i)
		}
	}

	// 5th failure should lock.
	result, err := svc.RecordFailedLoginAnomaly(ctx, username)
	if err != nil {
		t.Fatalf("5th RecordFailedLogin: %v", err)
	}
	if !result.Locked {
		t.Fatal("expected lock after 5th failure")
	}

	// User should be locked.
	if !svc.IsLoginLocked(ctx, username) {
		t.Error("expected user to be locked")
	}
}

func TestClearFailedLogins(t *testing.T) {
	svc := newAuthSvcWithRedis(t)
	ctx := context.Background()
	username := "testuser_clear"

	// Record 3 failures.
	for i := 0; i < 3; i++ {
		_, _ = svc.RecordFailedLoginAnomaly(ctx, username)
	}

	// Clear.
	svc.ClearFailedLogins(ctx, username)

	// Not locked, and counter is reset.
	if svc.IsLoginLocked(ctx, username) {
		t.Error("expected not locked after clear")
	}

	// Next failure should start from remaining=4.
	result, _ := svc.RecordFailedLoginAnomaly(ctx, username)
	if result.Remaining != 4 {
		t.Errorf("after clear, remaining = %d, want 4", result.Remaining)
	}
}

func TestIsLoginLocked_NotLocked(t *testing.T) {
	svc := newAuthSvcWithRedis(t)
	ctx := context.Background()
	if svc.IsLoginLocked(ctx, "neverlocked") {
		t.Error("expected false for user never locked")
	}
}

func TestRecordKnownDevice(t *testing.T) {
	svc := newAuthSvcWithRedis(t)
	ctx := context.Background()
	userID := "user-devices-1"

	if err := svc.RecordKnownDevice(ctx, userID, "fp1"); err != nil {
		t.Fatalf("RecordKnownDevice: %v", err)
	}
	if err := svc.RecordKnownDevice(ctx, userID, "fp2"); err != nil {
		t.Fatalf("RecordKnownDevice: %v", err)
	}

	devices, err := svc.GetKnownDevices(ctx, userID)
	if err != nil {
		t.Fatalf("GetKnownDevices: %v", err)
	}
	if len(devices) != 2 {
		t.Errorf("expected 2 devices, got %d", len(devices))
	}
}

func TestGetKnownDevices_Empty(t *testing.T) {
	svc := newAuthSvcWithRedis(t)
	ctx := context.Background()
	devices, err := svc.GetKnownDevices(ctx, "user-no-devices")
	if err != nil {
		t.Fatalf("GetKnownDevices: %v", err)
	}
	if len(devices) != 0 {
		t.Errorf("expected 0 devices, got %d", len(devices))
	}
}

func TestAssessLoginAnomaly_NoAnomaly(t *testing.T) {
	svc := newAuthSvcWithRedis(t)
	ctx := context.Background()
	userID := "user-normal"
	username := "normaluser"

	// Record a known IP and device first.
	_ = svc.RecordKnownDevice(ctx, userID, "known-fp")
	knownIPsKey := "ggid:anomaly:ips:" + userID
	_ = svc.rateLimiter.rdb.HSet(ctx, knownIPsKey, "1.2.3.4", "40.7,-74.0").Err()

	result, err := svc.AssessLoginAnomaly(ctx, username, userID, "1.2.3.4", "known-fp", 40.7, -74.0)
	if err != nil {
		t.Fatalf("AssessLoginAnomaly: %v", err)
	}
	if result.Locked {
		t.Error("expected not locked")
	}
	if result.GeoAnomaly {
		t.Error("expected no geo anomaly")
	}
	if result.NewDevice {
		t.Error("expected no new device")
	}
	if result.RequireNotify {
		t.Error("expected no notification")
	}
}

func TestAssessLoginAnomaly_GeoAnomaly(t *testing.T) {
	svc := newAuthSvcWithRedis(t)
	ctx := context.Background()
	userID := "user-geo"
	username := "geouser"

	_ = svc.RecordKnownDevice(ctx, userID, "known-fp")
	knownIPsKey := "ggid:anomaly:ips:" + userID
	_ = svc.rateLimiter.rdb.HSet(ctx, knownIPsKey, "1.2.3.4", "40.7,-74.0").Err() // NYC

	// Login from LA.
	result, err := svc.AssessLoginAnomaly(ctx, username, userID, "5.6.7.8", "known-fp", 34.0, -118.2)
	if err != nil {
		t.Fatalf("AssessLoginAnomaly: %v", err)
	}
	if !result.GeoAnomaly {
		t.Error("expected geo anomaly")
	}
	if !result.RequireNotify {
		t.Error("expected notification required")
	}
}

func TestAssessLoginAnomaly_NewDevice(t *testing.T) {
	svc := newAuthSvcWithRedis(t)
	ctx := context.Background()
	userID := "user-dev"
	username := "devuser"

	_ = svc.RecordKnownDevice(ctx, userID, "known-fp")
	knownIPsKey := "ggid:anomaly:ips:" + userID
	_ = svc.rateLimiter.rdb.HSet(ctx, knownIPsKey, "1.2.3.4", "40.7,-74.0").Err()

	result, err := svc.AssessLoginAnomaly(ctx, username, userID, "1.2.3.4", "unknown-fp", 40.7, -74.0)
	if err != nil {
		t.Fatalf("AssessLoginAnomaly: %v", err)
	}
	if !result.NewDevice {
		t.Error("expected new device anomaly")
	}
	if !result.RequireNotify {
		t.Error("expected notification required")
	}
}

func TestAssessLoginAnomaly_Locked(t *testing.T) {
	svc := newAuthSvcWithRedis(t)
	ctx := context.Background()
	username := "lockeduser"

	// Lock the user.
	for i := 0; i < 5; i++ {
		_, _ = svc.RecordFailedLoginAnomaly(ctx, username)
	}

	result, err := svc.AssessLoginAnomaly(ctx, username, "lockeduser", "1.2.3.4", "fp", 0, 0)
	if err != nil {
		t.Fatalf("AssessLoginAnomaly: %v", err)
	}
	if !result.Locked {
		t.Error("expected locked")
	}
}

// newAuthSvcWithRedis creates an AuthService backed by miniredis for testing.
func newAuthSvcWithRedis(t *testing.T) *AuthService {
	t.Helper()
	rdb := tRedis(t)
	rl := &RateLimiter{rdb: rdb}
	return &AuthService{
		rateLimiter: rl,
	}
}
