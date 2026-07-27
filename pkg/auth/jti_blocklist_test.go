package auth

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestBlocklist(t *testing.T) (*JTIBlocklist, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	t.Cleanup(mr.Close)

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { rdb.Close() })

	return NewJTIBlocklist(rdb), mr
}

func TestJTIBlocklist_RevokeAndCheck(t *testing.T) {
	bl, _ := newTestBlocklist(t)
	ctx := context.Background()

	jti := "test-jti-12345"
	exp := time.Now().Add(1 * time.Hour)

	// Before revoke — not blocked
	if bl.IsRevoked(ctx, jti) {
		t.Error("expected jti to not be revoked before Revoke call")
	}

	// Revoke it
	if err := bl.Revoke(ctx, jti, exp); err != nil {
		t.Fatalf("Revoke failed: %v", err)
	}

	// After revoke — blocked
	if !bl.IsRevoked(ctx, jti) {
		t.Error("expected jti to be revoked after Revoke call")
	}
}

func TestJTIBlocklist_RevokeAll(t *testing.T) {
	bl, _ := newTestBlocklist(t)
	ctx := context.Background()

	jtis := []string{"jti-1", "jti-2", "jti-3"}
	exp := time.Now().Add(1 * time.Hour)

	if err := bl.RevokeAll(ctx, jtis, exp); err != nil {
		t.Fatalf("RevokeAll failed: %v", err)
	}

	for _, jti := range jtis {
		if !bl.IsRevoked(ctx, jti) {
			t.Errorf("expected jti %s to be revoked", jti)
		}
	}
}

func TestJTIBlocklist_RevokeAll_Empty(t *testing.T) {
	bl, _ := newTestBlocklist(t)
	ctx := context.Background()

	err := bl.RevokeAll(ctx, nil, time.Now().Add(time.Hour))
	if err != nil {
		t.Errorf("expected nil error for empty jtis, got %v", err)
	}
}

func TestJTIBlocklist_IsRevoked_EmptyJTI(t *testing.T) {
	bl, _ := newTestBlocklist(t)
	ctx := context.Background()

	if bl.IsRevoked(ctx, "") {
		t.Error("expected empty jti to return false")
	}
}

func TestJTIBlocklist_IsRevoked_NonExistent(t *testing.T) {
	bl, _ := newTestBlocklist(t)
	ctx := context.Background()

	if bl.IsRevoked(ctx, "nonexistent-jti") {
		t.Error("expected non-existent jti to return false")
	}
}

func TestJTIBlocklist_NilRedis_NoOp(t *testing.T) {
	bl := NewJTIBlocklist(nil)
	ctx := context.Background()

	// All operations should be no-ops with nil Redis
	if err := bl.Revoke(ctx, "jti", time.Now().Add(time.Hour)); err != nil {
		t.Errorf("Revoke with nil redis should be no-op, got %v", err)
	}
	if err := bl.RevokeAll(ctx, []string{"jti"}, time.Now().Add(time.Hour)); err != nil {
		t.Errorf("RevokeAll with nil redis should be no-op, got %v", err)
	}
	if bl.IsRevoked(ctx, "jti") {
		t.Error("IsRevoked with nil redis should return false")
	}
	if err := bl.CleanupExpired(ctx); err != nil {
		t.Errorf("CleanupExpired with nil redis should be no-op, got %v", err)
	}
}

func TestJTIBlocklist_CleanupExpired(t *testing.T) {
	bl, mr := newTestBlocklist(t)
	ctx := context.Background()

	// Add expired jti (past time)
	expired := time.Now().Add(-1 * time.Hour)
	if err := bl.Revoke(ctx, "expired-jti", expired); err != nil {
		t.Fatalf("Revoke failed: %v", err)
	}

	// Add active jti (future time)
	active := time.Now().Add(1 * time.Hour)
	if err := bl.Revoke(ctx, "active-jti", active); err != nil {
		t.Fatalf("Revoke failed: %v", err)
	}

	// Cleanup expired entries
	if err := bl.CleanupExpired(ctx); err != nil {
		t.Fatalf("CleanupExpired failed: %v", err)
	}

	// Advance time in miniredis to trigger cleanup
	mr.FastForward(2 * time.Hour)

	// Run cleanup again with advanced time
	if err := bl.CleanupExpired(ctx); err != nil {
		t.Fatalf("CleanupExpired after fastforward failed: %v", err)
	}
}

func TestJTIBlocklist_DuplicateRevoke(t *testing.T) {
	bl, _ := newTestBlocklist(t)
	ctx := context.Background()

	jti := "dup-jti"
	exp := time.Now().Add(1 * time.Hour)

	// Revoke twice — should not error
	if err := bl.Revoke(ctx, jti, exp); err != nil {
		t.Fatalf("first Revoke failed: %v", err)
	}
	if err := bl.Revoke(ctx, jti, exp); err != nil {
		t.Fatalf("second Revoke failed: %v", err)
	}

	// Should still be revoked
	if !bl.IsRevoked(ctx, jti) {
		t.Error("expected jti to still be revoked after duplicate revoke")
	}
}
