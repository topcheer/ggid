package service

import (
	"context"
	"testing"

	"github.com/ggid/ggid/services/auth/internal/conf"
	"github.com/google/uuid"
)

// Cover GetPasswordPolicy and GetPasswordService accessors
func TestCov8_GetPasswordPolicyAndService(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)

	// GetPasswordPolicy should return the config policy
	p := svc.GetPasswordPolicy()
	if p.MinLength != conf.Default().Password.MinLength {
		t.Errorf("expected default min length %d, got %d", conf.Default().Password.MinLength, p.MinLength)
	}

	// GetPasswordService should return the service
	ps := svc.GetPasswordService()
	if ps == nil {
		t.Fatal("expected non-nil password service")
	}
}

// Cover RevokeSession error path
func TestCov8_RevokeSession_Error(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	ctx := context.Background()

	err := svc.RevokeSession(ctx, uuid.New())
	// With mock repo, should not error but let's exercise it
	_ = err
}

// Cover ACR level mapping
func TestCov8_ACRLevel(t *testing.T) {
	tests := []struct {
		acr  string
		want int
	}{
		{"urn:mace:incommon:iap:gold", 2},
		{"urn:mace:incommon:iap:silver", 1},
		{"1", 1},
		{"2", 2},
		{"unknown", 0},
		{"", 0},
	}

	for _, tc := range tests {
		got := acrLevel(tc.acr)
		if got != tc.want {
			t.Errorf("acrLevel(%q) = %d, want %d", tc.acr, got, tc.want)
		}
	}
}

// Cover ACRStepUpCheck — satisfied case (current >= required)
func TestCov8_ACRStepUpCheck_Satisfied(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	ctx := context.Background()
	userID := uuid.New()

	satisfied, challenge, err := svc.ACRStepUpCheck(ctx, userID, "urn:mace:incommon:iap:gold", "urn:mace:incommon:iap:silver")
	if err != nil {
		t.Fatalf("ACRStepUpCheck: %v", err)
	}
	if !satisfied {
		t.Error("expected satisfied when current ACR >= required")
	}
	if challenge != nil {
		t.Error("expected nil challenge when satisfied")
	}
}

// Cover ACRStepUpCheck — needs step-up (current < required)
func TestCov8_ACRStepUpCheck_NeedsStepUp(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	ctx, tid := tCtxTenant()
	_ = tid
	userID := uuid.New()

	satisfied, challenge, err := svc.ACRStepUpCheck(ctx, userID, "0", "urn:mace:incommon:iap:gold")
	if err != nil {
		t.Fatalf("ACRStepUpCheck: %v", err)
	}
	if satisfied {
		t.Error("expected not satisfied when current ACR < required")
	}
	if challenge == nil {
		t.Error("expected non-nil challenge when step-up needed")
	}
}

// Cover PasswordService.StrengthScore via domain
func TestCov8_PasswordStrength(t *testing.T) {
	svc, _, _, _ := tNewAuthSvc(t)
	ps := svc.GetPasswordService()
	if ps == nil {
		t.Fatal("nil password service")
	}
	// Weak password
	if err := ps.Validate("weak"); err == nil {
		t.Error("expected error for weak password")
	}
	// Strong password
	if err := ps.Validate("StrongP@ssw0rd!"); err != nil {
		t.Errorf("expected nil for strong password, got %v", err)
	}
}

// Cover RecordFailedLoginAnomaly remaining attempt counting
func TestCov8_RecordFailedLogin_Remaining(t *testing.T) {
	svc := newAuthSvcWithRedis(t)
	ctx := context.Background()

	// First failure: remaining should be 4
	result, err := svc.RecordFailedLoginAnomaly(ctx, "cov8user")
	if err != nil {
		t.Fatalf("RecordFailedLoginAnomaly: %v", err)
	}
	if result.Remaining != 4 {
		t.Errorf("expected remaining=4, got %d", result.Remaining)
	}

	// Second failure: remaining should be 3
	result, _ = svc.RecordFailedLoginAnomaly(ctx, "cov8user")
	if result.Remaining != 3 {
		t.Errorf("expected remaining=3, got %d", result.Remaining)
	}
}

// Cover AssessLoginAnomaly with first-time user (no known IPs/devices)
func TestCov8_AssessLoginAnomaly_FirstTime(t *testing.T) {
	svc := newAuthSvcWithRedis(t)
	ctx := context.Background()

	result, err := svc.AssessLoginAnomaly(ctx, "firstuser", "firstuid", "1.2.3.4", "fp1", 40.7, -74.0)
	if err != nil {
		t.Fatalf("AssessLoginAnomaly: %v", err)
	}
	if result.Locked {
		t.Error("expected not locked for first-time user")
	}
	if result.GeoAnomaly {
		t.Error("expected no geo anomaly for first-time user")
	}
	if result.NewDevice {
		t.Error("expected no new device for first-time user")
	}
}

// Cover AssessLoginAnomaly with no coordinates (lat=lon=0)
func TestCov8_AssessLoginAnomaly_NoCoords(t *testing.T) {
	svc := newAuthSvcWithRedis(t)
	ctx := context.Background()

	result, err := svc.AssessLoginAnomaly(ctx, "nocorords", "nocorords-uid", "1.2.3.4", "fp1", 0, 0)
	if err != nil {
		t.Fatalf("AssessLoginAnomaly: %v", err)
	}
	if result.RequireNotify {
		t.Error("expected no notify when no anomaly")
	}
}
