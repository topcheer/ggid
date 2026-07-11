package service

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/ggid/ggid/pkg/tenant"
	"github.com/google/uuid"
)

func newBackupCtx() context.Context {
	tc := &tenant.Context{TenantID: uuid.New()}
	return tenant.WithContext(context.Background(), tc)
}

func TestBackupCodes_Generate(t *testing.T) {
	svc := NewBackupCodeService(NewInMemBackupCodeRepo())
	userID := uuid.New()
	ctx := newBackupCtx()

	codes, err := svc.GenerateBackupCodes(ctx, userID)
	if err != nil {
		t.Fatalf("GenerateBackupCodes: %v", err)
	}
	if len(codes) != 10 {
		t.Errorf("expected 10 codes, got %d", len(codes))
	}
	for i, c := range codes {
		if len(c) != 9 { // XXXX-XXXX
			t.Errorf("code[%d] = %q, expected format XXXX-XXXX", i, c)
		}
		if !strings.Contains(c, "-") {
			t.Errorf("code[%d] missing dash: %q", i, c)
		}
	}
}

func TestBackupCodes_Verify(t *testing.T) {
	svc := NewBackupCodeService(NewInMemBackupCodeRepo())
	userID := uuid.New()
	ctx := newBackupCtx()

	codes, _ := svc.GenerateBackupCodes(ctx, userID)

	tc, _ := tenant.FromContext(ctx)
	if err := svc.VerifyBackupCode(ctx, tc.TenantID, userID, codes[0]); err != nil {
		t.Errorf("VerifyBackupCode valid code: %v", err)
	}
}

func TestBackupCodes_SingleUse(t *testing.T) {
	svc := NewBackupCodeService(NewInMemBackupCodeRepo())
	userID := uuid.New()
	ctx := newBackupCtx()

	codes, _ := svc.GenerateBackupCodes(ctx, userID)
	tc, _ := tenant.FromContext(ctx)

	// First use succeeds.
	if err := svc.VerifyBackupCode(ctx, tc.TenantID, userID, codes[0]); err != nil {
		t.Fatalf("first use: %v", err)
	}
	// Second use of same code fails.
	if err := svc.VerifyBackupCode(ctx, tc.TenantID, userID, codes[0]); err != ErrInvalidBackupCode {
		t.Errorf("expected ErrInvalidBackupCode on reuse, got %v", err)
	}
}

func TestBackupCodes_InvalidCode(t *testing.T) {
	svc := NewBackupCodeService(NewInMemBackupCodeRepo())
	userID := uuid.New()
	ctx := newBackupCtx()

	_, _ = svc.GenerateBackupCodes(ctx, userID)
	tc, _ := tenant.FromContext(ctx)

	if err := svc.VerifyBackupCode(ctx, tc.TenantID, userID, "INVALID-CODE"); err != ErrInvalidBackupCode {
		t.Errorf("expected ErrInvalidBackupCode, got %v", err)
	}
}

func TestBackupCodes_Remaining(t *testing.T) {
	svc := NewBackupCodeService(NewInMemBackupCodeRepo())
	userID := uuid.New()
	ctx := newBackupCtx()

	codes, _ := svc.GenerateBackupCodes(ctx, userID)
	tc, _ := tenant.FromContext(ctx)

	remaining, _ := svc.RemainingBackupCodes(ctx, tc.TenantID, userID)
	if remaining != 10 {
		t.Errorf("expected 10 remaining, got %d", remaining)
	}

	_ = svc.VerifyBackupCode(ctx, tc.TenantID, userID, codes[0])
	_ = svc.VerifyBackupCode(ctx, tc.TenantID, userID, codes[1])

	remaining, _ = svc.RemainingBackupCodes(ctx, tc.TenantID, userID)
	if remaining != 8 {
		t.Errorf("expected 8 remaining, got %d", remaining)
	}
}

func TestBackupCodes_Regenerate(t *testing.T) {
	svc := NewBackupCodeService(NewInMemBackupCodeRepo())
	userID := uuid.New()
	ctx := newBackupCtx()

	codes1, _ := svc.GenerateBackupCodes(ctx, userID)
	codes2, _ := svc.GenerateBackupCodes(ctx, userID)

	tc, _ := tenant.FromContext(ctx)

	// Old codes should be invalidated.
	if err := svc.VerifyBackupCode(ctx, tc.TenantID, userID, codes1[0]); err != ErrInvalidBackupCode {
		t.Errorf("old code should be invalid after regenerate, got %v", err)
	}
	// New codes should work.
	if err := svc.VerifyBackupCode(ctx, tc.TenantID, userID, codes2[0]); err != nil {
		t.Errorf("new code should be valid: %v", err)
	}
}

func TestBackupCodes_UniqueFormat(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		code := generateBackupCode()
		if seen[code] {
			t.Fatalf("duplicate code generated: %s", code)
		}
		seen[code] = true
		if len(code) != 9 {
			t.Errorf("code %q has wrong length", code)
		}
	}
}

func TestBackupCodes_NoAmbiguousChars(t *testing.T) {
	ambiguous := "O0Il1"
	for i := 0; i < 100; i++ {
		code := generateBackupCode()
		for _, ch := range ambiguous {
			if strings.ContainsRune(code, ch) {
				t.Errorf("code %q contains ambiguous char %q", code, ch)
			}
		}
	}
}

func TestBackupCodes_NoTenantContext(t *testing.T) {
	svc := NewBackupCodeService(NewInMemBackupCodeRepo())
	_, err := svc.GenerateBackupCodes(context.Background(), uuid.New())
	if err == nil {
		t.Error("expected error without tenant context")
	}
	if !strings.Contains(err.Error(), "tenant") {
		t.Errorf("expected tenant error, got %v", err)
	}
}

// Suppress unused import guard.
var _ = fmt.Sprintf
