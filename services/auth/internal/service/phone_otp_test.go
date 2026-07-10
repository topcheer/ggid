package service

import (
	"context"
	"testing"
	"time"

	"github.com/ggid/ggid/services/auth/internal/conf"
	"github.com/ggid/ggid/services/auth/internal/domain"
	"github.com/google/uuid"
)

// mockCredentialRepoForExpiration is a minimal mock for password expiration tests.
type mockCredRepoExpiration struct {
	cred    *domain.Credential
	history []domain.CredentialHistoryEntry
}

func (m *mockCredRepoExpiration) FindByIDentifier(ctx context.Context, tenantID uuid.UUID, identifier string) (*domain.Credential, error) {
	return nil, nil
}
func (m *mockCredRepoExpiration) FindByUserID(ctx context.Context, tenantID, userID uuid.UUID) (*domain.Credential, error) {
	return m.cred, nil
}
func (m *mockCredRepoExpiration) Create(ctx context.Context, c *domain.Credential) error { return nil }
func (m *mockCredRepoExpiration) UpdateFailedAttempts(ctx context.Context, id uuid.UUID, attempts int, lockedUntil *time.Time) error {
	return nil
}
func (m *mockCredRepoExpiration) UpdateSecret(ctx context.Context, id uuid.UUID, secret string) error { return nil }
func (m *mockCredRepoExpiration) AddToHistory(ctx context.Context, tenantID, userID uuid.UUID, secret string) error {
	return nil
}
func (m *mockCredRepoExpiration) GetHistory(ctx context.Context, tenantID, userID uuid.UUID, limit int) ([]domain.CredentialHistoryEntry, error) {
	return m.history, nil
}

func TestCheckPasswordExpiration_NoPolicy(t *testing.T) {
	ps := &PasswordService{
		policy: conf.PasswordPolicy{MaxAgeDays: 0},
		credentialRepo: &mockCredRepoExpiration{},
	}
	tenantID := uuid.New()
	userID := uuid.New()
	if err := ps.CheckPasswordExpiration(context.Background(), tenantID, userID); err != nil {
		t.Fatalf("expected nil when MaxAgeDays=0, got %v", err)
	}
}

func TestCheckPasswordExpired_Fresh(t *testing.T) {
	ps := &PasswordService{
		policy: conf.PasswordPolicy{MaxAgeDays: 90},
		credentialRepo: &mockCredRepoExpiration{
			cred: &domain.Credential{
				ID:        uuid.New(),
				UpdatedAt: time.Now().Add(-10 * 24 * time.Hour), // 10 days ago
			},
		},
	}
	tenantID := uuid.New()
	userID := uuid.New()
	if err := ps.CheckPasswordExpiration(context.Background(), tenantID, userID); err != nil {
		t.Fatalf("expected nil for fresh password, got %v", err)
	}
}

func TestCheckPasswordExpired_Expired(t *testing.T) {
	ps := &PasswordService{
		policy: conf.PasswordPolicy{MaxAgeDays: 30},
		credentialRepo: &mockCredRepoExpiration{
			cred: &domain.Credential{
				ID:        uuid.New(),
				UpdatedAt: time.Now().Add(-60 * 24 * time.Hour), // 60 days ago, > 30 max
			},
		},
	}
	tenantID := uuid.New()
	userID := uuid.New()
	if err := ps.CheckPasswordExpiration(context.Background(), tenantID, userID); err != ErrPasswordExpired {
		t.Fatalf("expected ErrPasswordExpired, got %v", err)
	}
}

func TestMustChangePassword_True(t *testing.T) {
	ps := &PasswordService{
		policy: conf.PasswordPolicy{MaxAgeDays: 1},
		credentialRepo: &mockCredRepoExpiration{
			cred: &domain.Credential{
				ID:        uuid.New(),
				UpdatedAt: time.Now().Add(-10 * 24 * time.Hour), // 10 days, > 1 max
			},
		},
	}
	tenantID := uuid.New()
	userID := uuid.New()
	if !ps.MustChangePassword(context.Background(), tenantID, userID) {
		t.Fatal("expected MustChangePassword=true for expired password")
	}
}

func TestGenerateNumericOTP(t *testing.T) {
	otp, err := generateNumericOTP(6)
	if err != nil {
		t.Fatalf("generateNumericOTP failed: %v", err)
	}
	if len(otp) != 6 {
		t.Fatalf("expected 6-digit OTP, got %d chars: %s", len(otp), otp)
	}
	for _, c := range otp {
		if c < '0' || c > '9' {
			t.Fatalf("OTP contains non-digit: %c", c)
		}
	}
}

func TestGenerateNumericOTP_DifferentEachCall(t *testing.T) {
	otps := make(map[string]bool)
	for i := 0; i < 100; i++ {
		otp, err := generateNumericOTP(6)
		if err != nil {
			t.Fatal(err)
		}
		otps[otp] = true
	}
	// With 100 6-digit OTPs, we should see at least 90 unique values.
	if len(otps) < 90 {
		t.Fatalf("expected high OTP entropy, got only %d unique values out of 100", len(otps))
	}
}

func TestSplitColon(t *testing.T) {
	tests := []struct {
		input string
		n     int
		want  int
	}{
		{"a:b:c", 3, 3},
		{"a:b", 2, 2},
		{"a:b:c:d", 3, 3}, // n=3 means at most 3 parts
		{"abc", 3, 1},
		{"", 3, 1},
	}
	for _, tt := range tests {
		got := splitColon(tt.input, tt.n)
		if len(got) != tt.want {
			t.Errorf("splitColon(%q, %d) = %v (len=%d), want len=%d", tt.input, tt.n, got, len(got), tt.want)
		}
	}
}
