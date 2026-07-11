package service

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ggid/ggid/pkg/tenant"
	"github.com/google/uuid"
)

// memOTPStore is an in-memory OTP store for testing.
type memOTPStore struct {
	mu    sync.Mutex
	codes map[string]string
}

func newMemOTPStore() *memOTPStore {
	return &memOTPStore{codes: make(map[string]string)}
}

func (m *memOTPStore) SetOTP(_ context.Context, key, code string, _ time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.codes[key] = code
	return nil
}

func (m *memOTPStore) GetOTP(_ context.Context, key string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	code, ok := m.codes[key]
	if !ok {
		return "", fmt.Errorf("not found")
	}
	return code, nil
}

func (m *memOTPStore) DeleteOTP(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.codes, key)
	return nil
}

func testOTPCtx() context.Context {
	return tenant.WithContext(context.Background(), &tenant.Context{
		TenantID: uuid.New(),
	})
}

func TestEmailOTP_SendAndVerify(t *testing.T) {
	store := newMemOTPStore()
	svc := NewEmailOTPService(store)
	ctx := testOTPCtx()
	userID := uuid.New()

	code, err := svc.SendOTP(ctx, userID)
	if err != nil {
		t.Fatalf("SendOTP: %v", err)
	}
	if len(code) != 6 {
		t.Errorf("expected 6-digit code, got %q", code)
	}

	if err := svc.VerifyOTP(ctx, userID, code); err != nil {
		t.Errorf("VerifyOTP: %v", err)
	}
}

func TestEmailOTP_VerifyInvalid(t *testing.T) {
	store := newMemOTPStore()
	svc := NewEmailOTPService(store)
	ctx := testOTPCtx()
	userID := uuid.New()

	_, _ = svc.SendOTP(ctx, userID)
	if err := svc.VerifyOTP(ctx, userID, "000000"); err != ErrInvalidOTPCode {
		t.Errorf("expected ErrInvalidOTPCode, got %v", err)
	}
}

func TestEmailOTP_NotFound(t *testing.T) {
	store := newMemOTPStore()
	svc := NewEmailOTPService(store)
	ctx := testOTPCtx()

	if err := svc.VerifyOTP(ctx, uuid.New(), "123456"); err != ErrOTPNotFound {
		t.Errorf("expected ErrOTPNotFound, got %v", err)
	}
}

func TestEmailOTP_SingleUse(t *testing.T) {
	store := newMemOTPStore()
	svc := NewEmailOTPService(store)
	ctx := testOTPCtx()
	userID := uuid.New()

	code, _ := svc.SendOTP(ctx, userID)
	_ = svc.VerifyOTP(ctx, userID, code) // first use — success

	// Second use should fail — code is deleted.
	if err := svc.VerifyOTP(ctx, userID, code); err != ErrOTPNotFound {
		t.Errorf("expected ErrOTPNotFound on reuse, got %v", err)
	}
}

func TestEmailOTP_NoTenantContext(t *testing.T) {
	store := newMemOTPStore()
	svc := NewEmailOTPService(store)

	_, err := svc.SendOTP(context.Background(), uuid.New())
	if err == nil {
		t.Error("expected error without tenant context")
	}

	err = svc.VerifyOTP(context.Background(), uuid.New(), "123456")
	if err == nil {
		t.Error("expected error without tenant context")
	}
}

func TestEmailOTP_CodeFormat(t *testing.T) {
	store := newMemOTPStore()
	svc := NewEmailOTPService(store)
	ctx := testOTPCtx()

	for i := 0; i < 20; i++ {
		code, err := svc.SendOTP(ctx, uuid.New())
		if err != nil {
			t.Fatalf("SendOTP: %v", err)
		}
		if len(code) != 6 {
			t.Errorf("expected 6 digits, got %d", len(code))
		}
		for _, c := range code {
			if c < '0' || c > '9' {
				t.Errorf("expected digit, got %c", c)
			}
		}
	}
}
