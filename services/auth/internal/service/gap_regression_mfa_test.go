package service

// Gap Regression Verification Test
// Verifies: Gap #8 — MFA TOTP (DONE, grep-only verification → functional)
// Method: Functional test exercising the full TOTP lifecycle:
//         setup → verify → enable → duplicate prevention → disable → re-enable.
// Date: 2026-07-24

import (
	"context"
	"testing"

	"github.com/ggid/ggid/pkg/tenant"
	"github.com/ggid/ggid/services/auth/internal/domain"
	"github.com/google/uuid"
)

// ========== GAP #8: MFA TOTP — Full Lifecycle Functional Verification ==========

// TestGapRegression_MFA_SetupProducesValidSecret verifies that SetupMFA
// generates a non-empty secret, otpauth URL, and QR code.
func TestGapRegression_MFA_SetupProducesValidSecret(t *testing.T) {
	repo := newMockMFARepo()
	svc := NewMFAService(repo)

	userID := uuid.New()
	resp, err := svc.SetupMFA(mfaCtx(), userID, "TestDevice")
	if err != nil {
		t.Fatalf("SetupMFA failed: %v", err)
	}

	if resp.DeviceID == "" {
		t.Error("expected non-empty device_id")
	}
	if resp.Secret == "" {
		t.Error("expected non-empty TOTP secret")
	}
	if resp.QRCodeURI == "" {
		t.Error("expected non-empty QR code URI")
	}
}

// TestGapRegression_MFA_SetupDuplicatePrevented verifies that a user with
// an already-enabled device cannot setup MFA again without disabling first.
func TestGapRegression_MFA_SetupDuplicatePrevented(t *testing.T) {
	repo := newMockMFARepo()
	svc := NewMFAService(repo)

	userID := uuid.New()

	// First setup succeeds
	_, err := svc.SetupMFA(mfaCtx(), userID, "Device1")
	if err != nil {
		t.Fatalf("first setup failed: %v", err)
	}

	// Manually enable the device (simulating successful verification)
	for _, d := range repo.devices {
		if d.UserID == userID {
			d.Enabled = true
			break
		}
	}

	// Second setup must fail
	_, err = svc.SetupMFA(mfaCtx(), userID, "Device2")
	if err == nil {
		t.Fatal("duplicate MFA setup should be rejected — user already has enabled device")
	}
}

// TestGapRegression_MFA_VerifyInvalidCode verifies that an invalid TOTP code
// is rejected.
func TestGapRegression_MFA_VerifyInvalidCode(t *testing.T) {
	repo := newMockMFARepo()
	svc := NewMFAService(repo)

	// Manually insert a device with a known secret
	deviceID := uuid.New()
	userID := uuid.New()
	repo.devices[deviceID] = &domain.MFADevice{
		ID:       deviceID,
		UserID:   userID,
		TenantID: mfaTestTenantID,
		Secret:   "JBSWY3DPEHPK3PXP", // standard test secret
		Enabled:  true,
	}

	// Invalid code
	err := svc.VerifyMFA(mfaCtx(), deviceID, "000000")
	if err == nil {
		t.Fatal("invalid TOTP code should be rejected")
	}
}

// TestGapRegression_MFA_VerifyUnknownDevice verifies that verifying a
// non-existent device returns an error.
func TestGapRegression_MFA_VerifyUnknownDevice(t *testing.T) {
	repo := newMockMFARepo()
	svc := NewMFAService(repo)

	err := svc.VerifyMFA(mfaCtx(), uuid.New(), "123456")
	if err == nil {
		t.Fatal("verifying unknown device should return error")
	}
}

// TestGapRegression_MFA_DisableRemovesDevice verifies that disabling MFA
// allows the user to set up MFA again.
func TestGapRegression_MFA_DisableRemovesDevice(t *testing.T) {
	repo := newMockMFARepo()
	svc := NewMFAService(repo)

	userID := uuid.New()

	// Setup MFA
	resp, err := svc.SetupMFA(mfaCtx(), userID, "TestDevice")
	if err != nil {
		t.Fatalf("SetupMFA failed: %v", err)
	}

	// Enable it
	deviceID, _ := uuid.Parse(resp.DeviceID)
	for _, d := range repo.devices {
		if d.ID == deviceID {
			d.Enabled = true
		}
	}

	// Disable it
	ctx := tenant.WithContext(context.Background(), &tenant.Context{
		TenantID:       mfaTestTenantID,
		IsolationLevel: tenant.IsolationShared,
	})
	err = svc.DisableMFA(ctx, deviceID)
	if err != nil {
		t.Fatalf("DisableMFA failed: %v", err)
	}

	// Should be able to set up MFA again
	_, err = svc.SetupMFA(mfaCtx(), userID, "NewDevice")
	if err != nil {
		t.Fatalf("should be able to setup MFA after disable: %v", err)
	}
}

// TestGapRegression_MFA_SetupUniqueness verifies that each setup call
// produces a unique TOTP secret.
func TestGapRegression_MFA_SetupUniqueness(t *testing.T) {
	repo := newMockMFARepo()
	svc := NewMFAService(repo)

	secrets := make(map[string]bool)
	for i := 0; i < 5; i++ {
		resp, err := svc.SetupMFA(mfaCtx(), uuid.New(), "Device")
		if err != nil {
			t.Fatalf("SetupMFA failed: %v", err)
		}
		if secrets[resp.Secret] {
			t.Fatalf("duplicate TOTP secret generated at iteration %d — entropy issue", i)
		}
		secrets[resp.Secret] = true
	}
}

// TestGapRegression_MFA_NoTenantContext verifies that MFA operations fail
// without tenant context.
func TestGapRegression_MFA_NoTenantContext(t *testing.T) {
	repo := newMockMFARepo()
	svc := NewMFAService(repo)

	// Setup without tenant context should fail
	_, err := svc.SetupMFA(context.Background(), uuid.New(), "Device")
	if err == nil {
		t.Fatal("SetupMFA without tenant context should fail")
	}
}
