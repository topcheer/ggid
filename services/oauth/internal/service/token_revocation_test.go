package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestTokenRevocation_RevokeToken(t *testing.T) {
	svc := NewTokenRevocationService()
	tokenID := "token-123"
	expires := time.Now().Add(1 * time.Hour)

	err := svc.RevokeToken(context.Background(), tokenID, "user_logout", expires)
	if err != nil {
		t.Fatalf("RevokeToken: %v", err)
	}

	if !svc.IsRevoked(context.Background(), tokenID) {
		t.Error("token should be revoked")
	}

	status, err := svc.GetRevocationStatus(context.Background(), tokenID)
	if err != nil {
		t.Fatalf("GetRevocationStatus: %v", err)
	}
	if !status.Revoked {
		t.Error("status should show revoked")
	}
	if status.Reason != "user_logout" {
		t.Errorf("expected reason 'user_logout', got '%s'", status.Reason)
	}
}

func TestTokenRevocation_RevokeToken_EmptyID(t *testing.T) {
	svc := NewTokenRevocationService()
	err := svc.RevokeToken(context.Background(), "", "test", time.Now().Add(time.Hour))
	if err == nil {
		t.Error("should error on empty tokenID")
	}
}

func TestTokenRevocation_NotRevoked(t *testing.T) {
	svc := NewTokenRevocationService()
	status, err := svc.GetRevocationStatus(context.Background(), "unknown-token")
	if err != nil {
		t.Fatalf("GetRevocationStatus: %v", err)
	}
	if status.Revoked {
		t.Error("unknown token should not be revoked")
	}
}

func TestTokenRevocation_CascadeRevoke(t *testing.T) {
	svc := NewTokenRevocationService()
	userID := uuid.New()
	expires := time.Now().Add(1 * time.Hour)
	tokenIDs := map[string]string{
		"access":  "access-123",
		"refresh": "refresh-456",
		"session": "session-789",
	}

	err := svc.CascadeRevoke(context.Background(), userID, tokenIDs, "security_incident", expires)
	if err != nil {
		t.Fatalf("CascadeRevoke: %v", err)
	}

	for _, tokenID := range tokenIDs {
		if !svc.IsRevoked(context.Background(), tokenID) {
			t.Errorf("token %s should be revoked", tokenID)
		}
	}
}

func TestTokenRevocation_CascadeRevoke_NilUser(t *testing.T) {
	svc := NewTokenRevocationService()
	err := svc.CascadeRevoke(context.Background(), uuid.Nil, map[string]string{"access": "x"}, "test", time.Now())
	if err == nil {
		t.Error("should error on nil userID")
	}
}

func TestTokenRevocation_CleanupExpired(t *testing.T) {
	svc := NewTokenRevocationService()
	// Add an expired token.
	svc.RevokeToken(context.Background(), "expired", "test", time.Now().Add(-1*time.Hour))
	// Add a valid token.
	svc.RevokeToken(context.Background(), "valid", "test", time.Now().Add(1*time.Hour))

	removed := svc.CleanupExpired()
	if removed != 1 {
		t.Errorf("expected 1 expired entry removed, got %d", removed)
	}
	if !svc.IsRevoked(context.Background(), "valid") {
		t.Error("valid token should still be revoked")
	}
}

func TestTokenRevocation_RevokeByClient_EmptyClient(t *testing.T) {
	svc := NewTokenRevocationService()
	_, err := svc.RevokeByClient(context.Background(), "", time.Now().Add(time.Hour))
	if err == nil {
		t.Error("should error on empty clientID")
	}
}

func TestTokenRevocation_RevokeByUser_NilUser(t *testing.T) {
	svc := NewTokenRevocationService()
	_, err := svc.RevokeByUser(context.Background(), uuid.Nil, time.Now().Add(time.Hour))
	if err == nil {
		t.Error("should error on nil userID")
	}
}

func TestTokenRevocation_IsRevoked_ExpiredToken(t *testing.T) {
	svc := NewTokenRevocationService()
	svc.RevokeToken(context.Background(), "expired", "test", time.Now().Add(-1*time.Hour))
	if svc.IsRevoked(context.Background(), "expired") {
		t.Error("expired token should not be considered revoked")
	}
}
