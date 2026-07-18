package tap

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestGenerateTAPCode(t *testing.T) {
	code := generateTAPCode()
	if len(code) != 8 {
		t.Fatalf("expected 8-digit code, got %d chars: %s", len(code), code)
	}
	for _, c := range code {
		if c < '0' || c > '9' {
			t.Fatalf("code should be digits only, got: %s", code)
		}
	}
	// Two codes should differ (randomness).
	code2 := generateTAPCode()
	if code == code2 {
		t.Fatal("two random codes should differ")
	}
}

func TestHashCode(t *testing.T) {
	h1 := hashCode("12345678")
	h2 := hashCode("12345678")
	if h1 != h2 {
		t.Fatal("same code should produce same hash")
	}
	if h1 == hashCode("87654321") {
		t.Fatal("different codes should produce different hashes")
	}
	if len(h1) != 64 {
		t.Fatalf("expected 64-char hex hash, got %d", len(h1))
	}
}

func TestEngine_Issue_NilPool(t *testing.T) {
	engine := NewEngine(nil)
	ctx := context.Background()

	code, record, err := engine.Issue(ctx, "user-1", "admin-1", "passkey re-enroll", 15*time.Minute)
	if err != nil {
		t.Fatalf("Issue failed: %v", err)
	}
	if len(code) != 8 {
		t.Fatalf("expected 8-digit code, got %s", code)
	}
	if record.UserID != "user-1" {
		t.Fatalf("expected user-1, got %s", record.UserID)
	}
	if record.Reason != "passkey re-enroll" {
		t.Fatalf("expected reason, got %s", record.Reason)
	}
	if !record.ExpiresAt.After(record.CreatedAt) {
		t.Fatal("expires_at should be after created_at")
	}
}

func TestEngine_Verify_NilPool(t *testing.T) {
	engine := NewEngine(nil)
	_, err := engine.Verify(context.Background(), "12345678")
	if err == nil {
		t.Fatal("nil pool should not verify")
	}
	if !strings.Contains(err.Error(), "no database") {
		t.Fatalf("expected database error, got: %v", err)
	}
}

func TestEngine_ListUserTAPs_NilPool(t *testing.T) {
	engine := NewEngine(nil)
	records, err := engine.ListUserTAPs(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if records != nil {
		t.Fatal("nil pool should return nil")
	}
}

func TestEngine_EnsureSchema_NilPool(t *testing.T) {
	engine := NewEngine(nil)
	if err := engine.EnsureSchema(context.Background()); err != nil {
		t.Fatalf("nil pool should not error: %v", err)
	}
}

func TestEngine_Issue_DefaultTTL(t *testing.T) {
	engine := NewEngine(nil)
	_, record, _ := engine.Issue(context.Background(), "u1", "a1", "", 0)
	if record.ExpiresAt.Sub(record.CreatedAt) < 14*time.Minute || record.ExpiresAt.Sub(record.CreatedAt) > 16*time.Minute {
		t.Fatalf("expected ~15min TTL, got %v", record.ExpiresAt.Sub(record.CreatedAt))
	}
}
