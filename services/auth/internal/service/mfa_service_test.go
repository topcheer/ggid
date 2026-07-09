package service

import (
	"context"
	"testing"
	"time"

	"github.com/ggid/ggid/pkg/tenant"
	"github.com/ggid/ggid/services/auth/internal/domain"
	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"
)

// --- Mock MFA Repository ---

type mockMFARepo struct {
	devices map[uuid.UUID]*domain.MFADevice
}

func newMockMFARepo() *mockMFARepo {
	return &mockMFARepo{devices: make(map[uuid.UUID]*domain.MFADevice)}
}

func (m *mockMFARepo) CreateDevice(_ context.Context, device *domain.MFADevice) error {
	device.CreatedAt = time.Now()
	device.UpdatedAt = time.Now()
	m.devices[device.ID] = device
	return nil
}

func (m *mockMFARepo) GetDeviceByID(_ context.Context, _ uuid.UUID, id uuid.UUID) (*domain.MFADevice, error) {
	d, ok := m.devices[id]
	if !ok {
		return nil, errNotFound("mfa device not found")
	}
	return d, nil
}

func (m *mockMFARepo) ListDevicesByUser(_ context.Context, _ uuid.UUID, userID uuid.UUID) ([]*domain.MFADevice, error) {
	var result []*domain.MFADevice
	for _, d := range m.devices {
		if d.UserID == userID {
			result = append(result, d)
		}
	}
	return result, nil
}

func (m *mockMFARepo) GetEnabledDevice(_ context.Context, _ uuid.UUID, userID uuid.UUID) (*domain.MFADevice, error) {
	for _, d := range m.devices {
		if d.UserID == userID && d.Enabled {
			return d, nil
		}
	}
	return nil, nil
}

func (m *mockMFARepo) UpdateDevice(_ context.Context, device *domain.MFADevice) error {
	if existing, ok := m.devices[device.ID]; ok {
		existing.Enabled = device.Enabled
		existing.VerifiedAt = device.VerifiedAt
		existing.Name = device.Name
		return nil
	}
	return errNotFound("mfa device not found")
}

func (m *mockMFARepo) DeleteDevice(_ context.Context, _ uuid.UUID, id uuid.UUID) error {
	delete(m.devices, id)
	return nil
}

type simpleErr string

func (e simpleErr) Error() string { return string(e) }

func errNotFound(msg string) error { return simpleErr(msg) }

// --- Helpers ---

var mfaTestTenantID = uuid.MustParse("00000000-0000-0000-0000-000000000030")

func mfaCtx() context.Context {
	return tenant.WithContext(context.Background(), &tenant.Context{
		TenantID:       mfaTestTenantID,
		IsolationLevel: tenant.IsolationShared,
	})
}

// --- Tests ---

func TestMFAService_Setup_Success(t *testing.T) {
	repo := newMockMFARepo()
	svc := NewMFAService(repo)

	userID := uuid.New()
	resp, err := svc.SetupMFA(mfaCtx(), userID, "iPhone")
	if err != nil {
		t.Fatalf("SetupMFA failed: %v", err)
	}

	if resp.DeviceID == "" {
		t.Error("expected non-empty device_id")
	}
	if resp.Secret == "" {
		t.Error("expected non-empty secret")
	}
	if resp.QRCodeURI == "" {
		t.Error("expected non-empty qr_code_uri")
	}

	// Verify device was stored as disabled.
	deviceID, _ := uuid.Parse(resp.DeviceID)
	device := repo.devices[deviceID]
	if device == nil {
		t.Fatal("expected device to be stored")
	}
	if device.Enabled {
		t.Error("device should be disabled until verified")
	}
	if device.UserID != userID {
		t.Error("user ID mismatch")
	}
	if device.Name != "iPhone" {
		t.Error("device name mismatch")
	}
}

func TestMFAService_Setup_AlreadyEnabled(t *testing.T) {
	repo := newMockMFARepo()
	svc := NewMFAService(repo)

	userID := uuid.New()
	now := time.Now()

	// Pre-create an enabled device.
	existingID := uuid.New()
	repo.devices[existingID] = &domain.MFADevice{
		ID:        existingID,
		TenantID:  mfaTestTenantID,
		UserID:    userID,
		Secret:    "EXISTINGSECRET",
		Enabled:   true,
		VerifiedAt: &now,
	}

	_, err := svc.SetupMFA(mfaCtx(), userID, "new")
	if err == nil {
		t.Fatal("expected error when MFA already enabled")
	}
}

func TestMFAService_Setup_DefaultDeviceName(t *testing.T) {
	repo := newMockMFARepo()
	svc := NewMFAService(repo)

	resp, err := svc.SetupMFA(mfaCtx(), uuid.New(), "")
	if err != nil {
		t.Fatalf("SetupMFA failed: %v", err)
	}

	deviceID, _ := uuid.Parse(resp.DeviceID)
	device := repo.devices[deviceID]
	if device.Name != "default" {
		t.Errorf("expected device name 'default', got '%s'", device.Name)
	}
}

func TestMFAService_Verify_FirstTime_EnablesDevice(t *testing.T) {
	repo := newMockMFARepo()
	svc := NewMFAService(repo)

	// Setup device.
	resp, err := svc.SetupMFA(mfaCtx(), uuid.New(), "test")
	if err != nil {
		t.Fatalf("SetupMFA failed: %v", err)
	}

	// Generate valid TOTP code using the secret.
	now := time.Now()
	code, _ := totp.GenerateCode(resp.Secret, now)
	deviceID, _ := uuid.Parse(resp.DeviceID)

	// Verify the code.
	err = svc.VerifyMFA(mfaCtx(), deviceID, code)
	if err != nil {
		t.Fatalf("VerifyMFA failed: %v", err)
	}

	// Device should now be enabled.
	device := repo.devices[deviceID]
	if !device.Enabled {
		t.Error("expected device to be enabled after verification")
	}
	if device.VerifiedAt == nil {
		t.Error("expected verified_at to be set")
	}
}

func TestMFAService_Verify_InvalidCode(t *testing.T) {
	repo := newMockMFARepo()
	svc := NewMFAService(repo)

	resp, _ := svc.SetupMFA(mfaCtx(), uuid.New(), "test")
	deviceID, _ := uuid.Parse(resp.DeviceID)

	err := svc.VerifyMFA(mfaCtx(), deviceID, "000000")
	if err == nil {
		t.Fatal("expected error for invalid TOTP code")
	}
	if err != ErrInvalidMFACode {
		t.Errorf("expected ErrInvalidMFACode, got %v", err)
	}
}

func TestMFAService_Verify_DeviceNotFound(t *testing.T) {
	repo := newMockMFARepo()
	svc := NewMFAService(repo)

	err := svc.VerifyMFA(mfaCtx(), uuid.New(), "123456")
	if err == nil {
		t.Fatal("expected error for non-existent device")
	}
}

func TestMFAService_VerifyUserCode_Success(t *testing.T) {
	repo := newMockMFARepo()
	svc := NewMFAService(repo)

	userID := uuid.New()

	// Setup and verify a device.
	resp, _ := svc.SetupMFA(mfaCtx(), userID, "test")
	code, _ := totp.GenerateCode(resp.Secret, time.Now())
	deviceID, _ := uuid.Parse(resp.DeviceID)
	_ = svc.VerifyMFA(mfaCtx(), deviceID, code)

	// Now verify by user code.
	code2, _ := totp.GenerateCode(resp.Secret, time.Now())
	err := svc.VerifyUserCode(context.Background(), mfaTestTenantID, userID, code2)
	if err != nil {
		t.Fatalf("VerifyUserCode failed: %v", err)
	}
}

func TestMFAService_VerifyUserCode_NoDevice(t *testing.T) {
	repo := newMockMFARepo()
	svc := NewMFAService(repo)

	err := svc.VerifyUserCode(context.Background(), mfaTestTenantID, uuid.New(), "123456")
	if err == nil {
		t.Fatal("expected error when no MFA device is enabled")
	}
}

func TestMFAService_Disable_Success(t *testing.T) {
	repo := newMockMFARepo()
	svc := NewMFAService(repo)

	resp, _ := svc.SetupMFA(mfaCtx(), uuid.New(), "test")
	deviceID, _ := uuid.Parse(resp.DeviceID)

	err := svc.DisableMFA(mfaCtx(), deviceID)
	if err != nil {
		t.Fatalf("DisableMFA failed: %v", err)
	}

	if _, ok := repo.devices[deviceID]; ok {
		t.Error("expected device to be deleted")
	}
}

func TestMFAService_HasMFAEnabled(t *testing.T) {
	repo := newMockMFARepo()
	svc := NewMFAService(repo)

	userID := uuid.New()

	// No device yet.
	if svc.HasMFAEnabled(context.Background(), mfaTestTenantID, userID) {
		t.Error("expected HasMFAEnabled=false before setup")
	}

	// Setup and verify.
	resp, _ := svc.SetupMFA(mfaCtx(), userID, "test")
	code, _ := totp.GenerateCode(resp.Secret, time.Now())
	deviceID, _ := uuid.Parse(resp.DeviceID)
	_ = svc.VerifyMFA(mfaCtx(), deviceID, code)

	// Now should be enabled.
	if !svc.HasMFAEnabled(context.Background(), mfaTestTenantID, userID) {
		t.Error("expected HasMFAEnabled=true after verification")
	}
}

func TestMFAService_ListDevices(t *testing.T) {
	repo := newMockMFARepo()
	svc := NewMFAService(repo)

	userID := uuid.New()

	// Create two devices.
	_, _ = svc.SetupMFA(mfaCtx(), userID, "phone")
	_, _ = svc.SetupMFA(mfaCtx(), userID, "tablet")

	devices, err := svc.ListDevices(mfaCtx(), userID)
	if err != nil {
		t.Fatalf("ListDevices failed: %v", err)
	}

	if len(devices) != 2 {
		t.Errorf("expected 2 devices, got %d", len(devices))
	}
}

func TestMFAService_Setup_NoTenantContext(t *testing.T) {
	repo := newMockMFARepo()
	svc := NewMFAService(repo)

	_, err := svc.SetupMFA(context.Background(), uuid.New(), "test")
	if err == nil {
		t.Fatal("expected error without tenant context")
	}
}

func TestMFAChallenge_IsExpired(t *testing.T) {
	challenge := &domain.MFAChallenge{
		Token:     "test-token",
		ExpiresAt: time.Now().Add(-1 * time.Minute), // expired 1 minute ago
	}
	if !challenge.IsExpired() {
		t.Error("expected expired challenge to be expired")
	}

	challenge.ExpiresAt = time.Now().Add(5 * time.Minute)
	if challenge.IsExpired() {
		t.Error("expected non-expired challenge to not be expired")
	}
}
