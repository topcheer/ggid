package repository

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/ggid/ggid/services/auth/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// mockScanner implements rowScanner for testing scan functions.
type mockScanner struct {
	err  error
	vals []any
}

func (m mockScanner) Scan(dest ...any) error {
	if m.err != nil {
		return m.err
	}
	for i, v := range m.vals {
		if i >= len(dest) {
			break
		}
		assign(dest[i], v)
	}
	return nil
}

// assign sets a value into a destination pointer, handling common DB types.
func assign(dest any, val any) {
	switch d := dest.(type) {
	case *uuid.UUID:
		*d = val.(uuid.UUID)
	case *string:
		switch v := val.(type) {
		case string:
			*d = v
		case []byte:
			*d = string(v)
		}
	case *[]byte:
		switch v := val.(type) {
		case []byte:
			*d = v
		case string:
			*d = []byte(v)
		}
	case *bool:
		*d = val.(bool)
	case *int:
		*d = val.(int)
	case *time.Time:
		*d = val.(time.Time)
	case *sql.NullTime:
		*d = val.(sql.NullTime)
	case *sql.NullString:
		*d = val.(sql.NullString)
	case *domain.CredentialType:
		*d = val.(domain.CredentialType)
	case *[]string:
		*d = val.([]string)
	default:
		// Fallback: try direct assignment via type assertion to *any
		if d, ok := dest.(*any); ok {
			*d = val
		}
	}
}

func TestScanCredential_Success(t *testing.T) {
	credID := uuid.New()
	tenantID := uuid.New()
	userID := uuid.New()
	now := time.Now()
	metadata := map[string]any{"source": "test"}

	metaJSON, _ := json.Marshal(metadata)

	s := mockScanner{vals: []any{
		credID, tenantID, userID, domain.CredentialType("password"),
		"testuser", "$argon2id$hash", metaJSON,
		true, 0, sql.NullTime{}, now, now, sql.NullTime{},
	}}

	cred, err := scanCredential(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cred == nil {
		t.Fatal("expected non-nil credential")
	}
	if cred.ID != credID {
		t.Errorf("ID mismatch: got %v, want %v", cred.ID, credID)
	}
	if cred.TenantID != tenantID {
		t.Errorf("TenantID mismatch: got %v, want %v", cred.TenantID, tenantID)
	}
	if cred.UserID != userID {
		t.Errorf("UserID mismatch: got %v, want %v", cred.UserID, userID)
	}
	if cred.Identifier != "testuser" {
		t.Errorf("Identifier mismatch: got %s, want testuser", cred.Identifier)
	}
	if cred.Secret != "$argon2id$hash" {
		t.Errorf("Secret mismatch")
	}
	if !cred.Enabled {
		t.Error("expected Enabled=true")
	}
	if cred.FailedAttempts != 0 {
		t.Errorf("FailedAttempts mismatch: got %d, want 0", cred.FailedAttempts)
	}
	if len(cred.Metadata) == 0 || cred.Metadata["source"] != "test" {
		t.Errorf("Metadata not parsed: %v", cred.Metadata)
	}
	if cred.LockedUntil != nil {
		t.Error("expected nil LockedUntil for null value")
	}
	if cred.LastUsedAt != nil {
		t.Error("expected nil LastUsedAt for null value")
	}
}

func TestScanCredential_WithLockAndLastUsed(t *testing.T) {
	lockTime := time.Now().Add(30 * time.Minute)
	lastUsed := time.Now().Add(-5 * time.Minute)

	s := mockScanner{vals: []any{
		uuid.New(), uuid.New(), uuid.New(), domain.CredentialType("password"),
		"user", "hash", []byte(nil),
		false, 3, sql.NullTime{Time: lockTime, Valid: true}, time.Now(), time.Now(),
		sql.NullTime{Time: lastUsed, Valid: true},
	}}

	cred, err := scanCredential(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cred == nil {
		t.Fatal("expected non-nil credential")
	}
	if cred.LockedUntil == nil {
		t.Fatal("expected non-nil LockedUntil")
	}
	if !cred.LockedUntil.Equal(lockTime) {
		t.Errorf("LockedUntil mismatch: got %v, want %v", *cred.LockedUntil, lockTime)
	}
	if cred.LastUsedAt == nil {
		t.Fatal("expected non-nil LastUsedAt")
	}
	if !cred.LastUsedAt.Equal(lastUsed) {
		t.Errorf("LastUsedAt mismatch: got %v, want %v", *cred.LastUsedAt, lastUsed)
	}
	if cred.Enabled {
		t.Error("expected Enabled=false")
	}
}

func TestScanCredential_NoRows(t *testing.T) {
	s := mockScanner{err: pgxErrNoRows}
	cred, err := scanCredential(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cred != nil {
		t.Fatal("expected nil credential for no rows")
	}
}

var pgxErrNoRows = pgx.ErrNoRows

func TestScanSession_Success(t *testing.T) {
	sessionID := uuid.New()
	tenantID := uuid.New()
	userID := uuid.New()
	now := time.Now()
	expiresAt := now.Add(24 * time.Hour)
	deviceInfo, _ := json.Marshal(map[string]any{"browser": "Chrome"})
	metadata, _ := json.Marshal(map[string]any{"mfa": "totp"})

	s := mockScanner{vals: []any{
		sessionID, tenantID, userID, "tokenhash123", deviceInfo,
		"127.0.0.1", "Mozilla/5.0", expiresAt,
		sql.NullTime{}, now, metadata,
	}}

	session, err := scanSession(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session == nil {
		t.Fatal("expected non-nil session")
	}
	if session.ID != sessionID {
		t.Errorf("ID mismatch")
	}
	if session.TokenHash != "tokenhash123" {
		t.Errorf("TokenHash mismatch: got %s", session.TokenHash)
	}
	if session.IPAddress != "127.0.0.1" {
		t.Errorf("IPAddress mismatch: got %s", session.IPAddress)
	}
	if session.RevokedAt != nil {
		t.Error("expected nil RevokedAt")
	}
	if session.DeviceInfo["browser"] != "Chrome" {
		t.Errorf("DeviceInfo not parsed: %v", session.DeviceInfo)
	}
	if session.Metadata["mfa"] != "totp" {
		t.Errorf("Metadata not parsed: %v", session.Metadata)
	}
}

func TestScanSession_Revoked(t *testing.T) {
	revokeTime := time.Now().Add(-1 * time.Hour)

	s := mockScanner{vals: []any{
		uuid.New(), uuid.New(), uuid.New(), "hash", []byte(nil),
		"", "", time.Now().Add(-1 * time.Hour),
		sql.NullTime{Time: revokeTime, Valid: true}, time.Now(), []byte(nil),
	}}

	session, err := scanSession(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session == nil {
		t.Fatal("expected non-nil session")
	}
	if session.RevokedAt == nil {
		t.Fatal("expected non-nil RevokedAt")
	}
	if !session.RevokedAt.Equal(revokeTime) {
		t.Errorf("RevokedAt mismatch")
	}
}

func TestScanRefreshToken_Success(t *testing.T) {
	tokenID := uuid.New()
	tenantID := uuid.New()
	userID := uuid.New()
	sessionID := uuid.New()
	clientID := uuid.New()
	now := time.Now()
	expiresAt := now.Add(7 * 24 * time.Hour)

	s := mockScanner{vals: []any{
		tokenID, tenantID, userID, sessionID,
		sql.NullString{String: clientID.String(), Valid: true},
		"tokenhash", []string{"openid", "profile"},
		expiresAt, sql.NullString{}, sql.NullTime{}, now,
	}}

	token, err := scanRefreshToken(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token == nil {
		t.Fatal("expected non-nil token")
	}
	if token.ID != tokenID {
		t.Errorf("ID mismatch")
	}
	if token.ClientID == nil || *token.ClientID != clientID {
		t.Errorf("ClientID mismatch: got %v", token.ClientID)
	}
	if len(token.Scope) != 2 || token.Scope[0] != "openid" || token.Scope[1] != "profile" {
		t.Errorf("Scope mismatch: got %v", token.Scope)
	}
	if token.RotatedFrom != nil {
		t.Error("expected nil RotatedFrom")
	}
	if token.RevokedAt != nil {
		t.Error("expected nil RevokedAt")
	}
}

func TestScanRefreshToken_WithRotatedFrom(t *testing.T) {
	rotatedFrom := uuid.New()

	s := mockScanner{vals: []any{
		uuid.New(), uuid.New(), uuid.New(), uuid.New(),
		sql.NullString{},
		"hash", []string{"openid"},
		time.Now().Add(time.Hour),
		sql.NullString{String: rotatedFrom.String(), Valid: true},
		sql.NullTime{}, time.Now(),
	}}

	token, err := scanRefreshToken(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token == nil {
		t.Fatal("expected non-nil token")
	}
	if token.ClientID != nil {
		t.Error("expected nil ClientID")
	}
	if token.RotatedFrom == nil || *token.RotatedFrom != rotatedFrom {
		t.Errorf("RotatedFrom mismatch: got %v", token.RotatedFrom)
	}
}

func TestScanRefreshToken_Revoked(t *testing.T) {
	revokeTime := time.Now().Add(-1 * time.Hour)

	s := mockScanner{vals: []any{
		uuid.New(), uuid.New(), uuid.New(), uuid.New(),
		sql.NullString{}, "hash", []string{"openid"},
		time.Now().Add(time.Hour), sql.NullString{},
		sql.NullTime{Time: revokeTime, Valid: true}, time.Now(),
	}}

	token, err := scanRefreshToken(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token.RevokedAt == nil {
		t.Fatal("expected non-nil RevokedAt")
	}
	if !token.RevokedAt.Equal(revokeTime) {
		t.Errorf("RevokedAt mismatch")
	}
}

// Domain logic tests
func TestCredential_IsLocked(t *testing.T) {
	tests := []struct {
		name        string
		lockedUntil *time.Time
		want        bool
	}{
		{"nil lock", nil, false},
		{"past lock", ptr(time.Now().Add(-1 * time.Hour)), false},
		{"future lock", ptr(time.Now().Add(30 * time.Minute)), true},
		{"just now", ptr(time.Now().Add(-1 * time.Second)), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &domain.Credential{LockedUntil: tt.lockedUntil}
			if got := c.IsLocked(); got != tt.want {
				t.Errorf("IsLocked() = %v, want %v", got, tt.want)
			}
		})
	}
}

func ptr[T any](v T) *T { return &v }
