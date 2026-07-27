package repository

import (
	"database/sql"
	"encoding/json"
	"net/netip"
	"testing"
	"time"

	"github.com/ggid/ggid/services/identity/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// mockScanner implements pgx.Row for testing scan functions.
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
	if val == nil {
		return
	}
	switch d := dest.(type) {
	case *uuid.UUID:
		*d = val.(uuid.UUID)
	case **uuid.UUID:
		switch v := val.(type) {
		case uuid.UUID:
			*d = &v
		case *uuid.UUID:
			*d = v
		}
	case *string:
		switch v := val.(type) {
		case string:
			*d = v
		case []byte:
			*d = string(v)
		}
	case **string:
		switch v := val.(type) {
		case string:
			*d = &v
		case *string:
			*d = v
		}
	case *bool:
		*d = val.(bool)
	case *int:
		*d = val.(int)
	case *time.Time:
		*d = val.(time.Time)
	case **time.Time:
		switch v := val.(type) {
		case time.Time:
			*d = &v
		case *time.Time:
			*d = v
		}
	case *sql.NullTime:
		*d = val.(sql.NullTime)
	case *sql.NullString:
		*d = val.(sql.NullString)
	case *[]byte:
		*d = val.([]byte)
	default:
		// Fallback: try direct assignment via type assertion to *any
		if d, ok := dest.(*any); ok {
			*d = val
		}
	}
}

var pgxErrNoRows = pgx.ErrNoRows

// TestScanUser_Success tests scanUser with all fields populated.
func TestScanUser_Success(t *testing.T) {
	userID := uuid.New()
	tenantID := uuid.New()
	primaryEmailID := uuid.New()
	now := time.Now()
	lastLoginAt := now.Add(-1 * time.Hour)
	deletedAt := now.Add(-24 * time.Hour)
	lastLoginIP := "192.168.1.1"

	addr, _ := netip.ParseAddr(lastLoginIP)

	s := mockScanner{vals: []any{
		userID,                 // id
		tenantID,               // tenant_id
		"testuser",             // username
		"test@example.com",     // email
		"+1234567890",          // phone
		"active",               // status (string)
		true,                   // email_verified
		false,                  // phone_verified
		primaryEmailID,        // primary_email_id (value, not pointer)
		"Test User",            // display_name
		"https://avatar.url",   // avatar_url
		"en_US",                // locale
		"UTC",                  // timezone
		&lastLoginAt,          // last_login_at
		&lastLoginIP,          // last_login_ip
		"passwordhash",        // password_hash
		now,                   // created_at
		now,                    // updated_at
		&deletedAt,             // deleted_at
	}}

	user, err := scanUser(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user == nil {
		t.Fatal("expected non-nil user")
	}

	if user.ID != userID {
		t.Errorf("ID mismatch: got %v, want %v", user.ID, userID)
	}
	if user.TenantID != tenantID {
		t.Errorf("TenantID mismatch: got %v, want %v", user.TenantID, tenantID)
	}
	if user.Username != "testuser" {
		t.Errorf("Username mismatch: got %s, want testuser", user.Username)
	}
	if user.Email != "test@example.com" {
		t.Errorf("Email mismatch: got %s, want test@example.com", user.Email)
	}
	if user.Phone != "+1234567890" {
		t.Errorf("Phone mismatch: got %s, want +1234567890", user.Phone)
	}
	if user.Status != domain.UserStatusActive {
		t.Errorf("Status mismatch: got %v, want active", user.Status)
	}
	if !user.EmailVerified {
		t.Error("expected EmailVerified=true")
	}
	if user.PhoneVerified {
		t.Error("expected PhoneVerified=false")
	}
	if user.PrimaryEmailID == nil || *user.PrimaryEmailID != primaryEmailID {
		t.Errorf("PrimaryEmailID mismatch: got %v, want %v", user.PrimaryEmailID, &primaryEmailID)
	}
	if user.DisplayName != "Test User" {
		t.Errorf("DisplayName mismatch: got %s, want Test User", user.DisplayName)
	}
	if user.AvatarURL != "https://avatar.url" {
		t.Errorf("AvatarURL mismatch: got %s, want https://avatar.url", user.AvatarURL)
	}
	if user.Locale != "en_US" {
		t.Errorf("Locale mismatch: got %s, want en_US", user.Locale)
	}
	if user.Timezone != "UTC" {
		t.Errorf("Timezone mismatch: got %s, want UTC", user.Timezone)
	}
	if user.LastLoginAt == nil {
		t.Fatal("expected non-nil LastLoginAt")
	}
	if !user.LastLoginAt.Equal(lastLoginAt) {
		t.Errorf("LastLoginAt mismatch: got %v, want %v", user.LastLoginAt, lastLoginAt)
	}
	if user.LastLoginIP == nil || *user.LastLoginIP != addr {
		t.Errorf("LastLoginIP mismatch: got %v, want %v", user.LastLoginIP, addr)
	}
	if user.PasswordHash != "passwordhash" {
		t.Errorf("PasswordHash mismatch: got %s, want passwordhash", user.PasswordHash)
	}
	if !user.CreatedAt.Equal(now) {
		t.Errorf("CreatedAt mismatch")
	}
	if !user.UpdatedAt.Equal(now) {
		t.Errorf("UpdatedAt mismatch")
	}
	if user.DeletedAt == nil {
		t.Fatal("expected non-nil DeletedAt")
	}
	if !user.DeletedAt.Equal(deletedAt) {
		t.Errorf("DeletedAt mismatch: got %v, want %v", user.DeletedAt, deletedAt)
	}
}

// TestScanUser_WithNullValues tests scanUser with NULL values for optional fields.
func TestScanUser_WithNullValues(t *testing.T) {
	userID := uuid.New()
	tenantID := uuid.New()
	now := time.Now()

	s := mockScanner{vals: []any{
		userID,            // id
		tenantID,          // tenant_id
		"testuser",        // username
		"test@example.com", // email
		sql.NullString{},  // phone (NULL)
		"active",          // status (string)
		false,             // email_verified
		false,             // phone_verified
		(*uuid.UUID)(nil), // primary_email_id (NULL)
		sql.NullString{},  // display_name (NULL)
		sql.NullString{},  // avatar_url (NULL)
		sql.NullString{},  // locale (NULL)
		sql.NullString{},  // timezone (NULL)
		(*time.Time)(nil), // last_login_at (NULL)
		(*string)(nil),    // last_login_ip (NULL)
		"hash",            // password_hash
		now,               // created_at
		now,               // updated_at
		(*time.Time)(nil), // deleted_at (NULL)
	}}

	user, err := scanUser(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user == nil {
		t.Fatal("expected non-nil user")
	}

	if user.Phone != "" {
		t.Errorf("expected empty Phone for NULL value, got %s", user.Phone)
	}
	if user.DisplayName != "" {
		t.Errorf("expected empty DisplayName for NULL value, got %s", user.DisplayName)
	}
	if user.AvatarURL != "" {
		t.Errorf("expected empty AvatarURL for NULL value, got %s", user.AvatarURL)
	}
	if user.Locale != "" {
		t.Errorf("expected empty Locale for NULL value, got %s", user.Locale)
	}
	if user.Timezone != "" {
		t.Errorf("expected empty Timezone for NULL value, got %s", user.Timezone)
	}
	if user.LastLoginAt != nil {
		t.Errorf("expected nil LastLoginAt for NULL value, got %v", user.LastLoginAt)
	}
	if user.LastLoginIP != nil {
		t.Errorf("expected nil LastLoginIP for NULL value, got %v", user.LastLoginIP)
	}
	if user.PrimaryEmailID != nil {
		t.Errorf("expected nil PrimaryEmailID for NULL value, got %v", user.PrimaryEmailID)
	}
	if user.DeletedAt != nil {
		t.Errorf("expected nil DeletedAt for NULL value, got %v", user.DeletedAt)
	}
}

// TestScanUser_NoRows tests scanUser with pgx.ErrNoRows.
func TestScanUser_NoRows(t *testing.T) {
	s := mockScanner{err: pgxErrNoRows}
	user, err := scanUser(s)
	if err != pgxErrNoRows {
		t.Fatalf("expected pgx.ErrNoRows error, got: %v", err)
	}
	if user != nil {
		t.Fatal("expected nil user for no rows")
	}
}

// TestScanUser_WithInvalidLastLoginIP tests scanUser with an invalid IP address.
func TestScanUser_WithInvalidLastLoginIP(t *testing.T) {
	userID := uuid.New()
	tenantID := uuid.New()
	now := time.Now()
	invalidIP := "not-an-ip-address"

	s := mockScanner{vals: []any{
		userID, tenantID, "testuser", "test@example.com", "+1234567890",
		"active", true, false, (*uuid.UUID)(nil), "Test User", "https://avatar.url",
		"en_US", "UTC", (*time.Time)(nil), &invalidIP, "hash", now, now, (*time.Time)(nil),
	}}

	user, err := scanUser(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user == nil {
		t.Fatal("expected non-nil user")
	}
	// Invalid IP should result in nil LastLoginIP
	if user.LastLoginIP != nil {
		t.Errorf("expected nil LastLoginIP for invalid IP address, got %v", user.LastLoginIP)
	}
}

// TestScanUserEmail_Success tests scanUserEmail with all fields populated.
func TestScanUserEmail_Success(t *testing.T) {
	emailID := uuid.New()
	userID := uuid.New()
	now := time.Now()
	verifiedAt := now.Add(-1 * time.Hour)

	s := mockScanner{vals: []any{
		emailID,            // id
		userID,             // user_id
		"test@example.com", // email
		true,               // is_primary
		&verifiedAt,        // verified_at
		now,                // created_at
	}}

	email, err := scanUserEmail(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if email == nil {
		t.Fatal("expected non-nil email")
	}

	if email.ID != emailID {
		t.Errorf("ID mismatch: got %v, want %v", email.ID, emailID)
	}
	if email.UserID != userID {
		t.Errorf("UserID mismatch: got %v, want %v", email.UserID, userID)
	}
	if email.Email != "test@example.com" {
		t.Errorf("Email mismatch: got %s, want test@example.com", email.Email)
	}
	if !email.IsPrimary {
		t.Error("expected IsPrimary=true")
	}
	if email.VerifiedAt == nil {
		t.Fatal("expected non-nil VerifiedAt")
	}
	if !email.VerifiedAt.Equal(verifiedAt) {
		t.Errorf("VerifiedAt mismatch: got %v, want %v", email.VerifiedAt, verifiedAt)
	}
	if !email.CreatedAt.Equal(now) {
		t.Errorf("CreatedAt mismatch")
	}
}

// TestScanUserEmail_WithNullVerifiedAt tests scanUserEmail with NULL verified_at.
func TestScanUserEmail_WithNullVerifiedAt(t *testing.T) {
	emailID := uuid.New()
	userID := uuid.New()
	now := time.Now()

	s := mockScanner{vals: []any{
		emailID,            // id
		userID,             // user_id
		"test@example.com", // email
		false,              // is_primary
		(*time.Time)(nil),  // verified_at (NULL)
		now,                // created_at
	}}

	email, err := scanUserEmail(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if email == nil {
		t.Fatal("expected non-nil email")
	}

	if email.VerifiedAt != nil {
		t.Errorf("expected nil VerifiedAt for NULL value, got %v", email.VerifiedAt)
	}
	if email.IsPrimary {
		t.Error("expected IsPrimary=false")
	}
}

// TestScanUserEmail_NoRows tests scanUserEmail with pgx.ErrNoRows.
func TestScanUserEmail_NoRows(t *testing.T) {
	s := mockScanner{err: pgxErrNoRows}
	email, err := scanUserEmail(s)
	if err != pgxErrNoRows {
		t.Fatalf("expected pgx.ErrNoRows error, got: %v", err)
	}
	if email != nil {
		t.Fatal("expected nil email for no rows")
	}
}

// TestScanExternalIdentity_Success tests scanExternalIdentity with all fields populated.
func TestScanExternalIdentity_Success(t *testing.T) {
	id := uuid.New()
	userID := uuid.New()
	now := time.Now()
	metadata := map[string]any{"sub": "12345", "provider": "google"}
	metaJSON, _ := json.Marshal(metadata)

	s := mockScanner{vals: []any{
		id,                  // id
		userID,              // user_id
		"google",            // provider
		"google-user-123",   // external_id
		metaJSON,            // metadata (JSON as []byte)
		now,                 // linked_at
	}}

	ei, err := scanExternalIdentity(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ei == nil {
		t.Fatal("expected non-nil external identity")
	}

	if ei.ID != id {
		t.Errorf("ID mismatch: got %v, want %v", ei.ID, id)
	}
	if ei.UserID != userID {
		t.Errorf("UserID mismatch: got %v, want %v", ei.UserID, userID)
	}
	if ei.Provider != "google" {
		t.Errorf("Provider mismatch: got %s, want google", ei.Provider)
	}
	if ei.ExternalID != "google-user-123" {
		t.Errorf("ExternalID mismatch: got %s, want google-user-123", ei.ExternalID)
	}
	if len(ei.Metadata) == 0 {
		t.Fatal("expected non-empty Metadata")
	}
	if ei.Metadata["sub"] != "12345" {
		t.Errorf("Metadata sub mismatch: got %v, want 12345", ei.Metadata["sub"])
	}
	if ei.Metadata["provider"] != "google" {
		t.Errorf("Metadata provider mismatch: got %v, want google", ei.Metadata["provider"])
	}
	if !ei.LinkedAt.Equal(now) {
		t.Errorf("LinkedAt mismatch")
	}
}

// TestScanExternalIdentity_WithEmptyMetadata tests scanExternalIdentity with empty metadata.
func TestScanExternalIdentity_WithEmptyMetadata(t *testing.T) {
	id := uuid.New()
	userID := uuid.New()
	now := time.Now()

	s := mockScanner{vals: []any{
		id,                // id
		userID,            // user_id
		"github",          // provider
		"github-user-456", // external_id
		[]byte{},          // metadata (empty)
		now,               // linked_at
	}}

	ei, err := scanExternalIdentity(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ei == nil {
		t.Fatal("expected non-nil external identity")
	}

	if len(ei.Metadata) != 0 {
		t.Errorf("expected empty Metadata, got %v", ei.Metadata)
	}
}

// TestScanExternalIdentity_WithNullMetadata tests scanExternalIdentity with NULL metadata.
func TestScanExternalIdentity_WithNullMetadata(t *testing.T) {
	id := uuid.New()
	userID := uuid.New()
	now := time.Now()

	s := mockScanner{vals: []any{
		id,                // id
		userID,            // user_id
		"azure",           // provider
		"azure-user-789",  // external_id
		nil,               // metadata (NULL)
		now,               // linked_at
	}}

	ei, err := scanExternalIdentity(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ei == nil {
		t.Fatal("expected non-nil external identity")
	}

	if len(ei.Metadata) != 0 {
		t.Errorf("expected empty Metadata for NULL value, got %v", ei.Metadata)
	}
}

// TestScanExternalIdentity_NoRows tests scanExternalIdentity with pgx.ErrNoRows.
func TestScanExternalIdentity_NoRows(t *testing.T) {
	s := mockScanner{err: pgxErrNoRows}
	ei, err := scanExternalIdentity(s)
	if err != pgxErrNoRows {
		t.Fatalf("expected pgx.ErrNoRows error, got: %v", err)
	}
	if ei != nil {
		t.Fatal("expected nil external identity for no rows")
	}
}

// TestScanExternalIdentity_InvalidJSON tests scanExternalIdentity with invalid JSON metadata.
func TestScanExternalIdentity_InvalidJSON(t *testing.T) {
	id := uuid.New()
	userID := uuid.New()
	now := time.Now()

	s := mockScanner{vals: []any{
		id,               // id
		userID,           // user_id
		"test",           // provider
		"test-id",        // external_id
		[]byte("{bad}"),  // metadata (invalid JSON)
		now,              // linked_at
	}}

	ei, err := scanExternalIdentity(s)
	if err == nil {
		t.Fatal("expected error for invalid JSON metadata")
	}
	if ei != nil {
		t.Error("expected nil external identity for invalid JSON")
	}
}