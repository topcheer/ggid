package service

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestImpersonation_Issue(t *testing.T) {
	ResetImpersonationStore()
	admin := uuid.New()
	target := uuid.New()
	tenant := uuid.New()

	tok, err := IssueImpersonationToken(admin, target, tenant, "support escalation")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if tok.ImpersonatorID != admin || tok.TargetUserID != target {
		t.Error("IDs mismatch")
	}
	if tok.Reason != "support escalation" {
		t.Error("reason mismatch")
	}
}

func TestImpersonation_SelfImpersonation(t *testing.T) {
	ResetImpersonationStore()
	id := uuid.New()
	_, err := IssueImpersonationToken(id, id, uuid.New(), "test")
	if err == nil {
		t.Error("should reject self-impersonation")
	}
}

func TestImpersonation_MissingReason(t *testing.T) {
	ResetImpersonationStore()
	_, err := IssueImpersonationToken(uuid.New(), uuid.New(), uuid.New(), "")
	if err == nil {
		t.Error("should require reason")
	}
}

func TestImpersonation_Validate(t *testing.T) {
	ResetImpersonationStore()
	admin, target := uuid.New(), uuid.New()
	tok, _ := IssueImpersonationToken(admin, target, uuid.New(), "audit")

	valid, err := ValidateImpersonationToken(tok.TokenID)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if valid.TargetUserID != target {
		t.Error("target mismatch")
	}
}

func TestImpersonation_Revoke(t *testing.T) {
	ResetImpersonationStore()
	tok, _ := IssueImpersonationToken(uuid.New(), uuid.New(), uuid.New(), "test")

	RevokeImpersonationToken(tok.TokenID)

	_, err := ValidateImpersonationToken(tok.TokenID)
	if err == nil {
		t.Error("revoked token should fail validation")
	}
}

func TestImpersonation_Expired(t *testing.T) {
	ResetImpersonationStore()
	tok, _ := IssueImpersonationToken(uuid.New(), uuid.New(), uuid.New(), "test")
	tok.ExpiresAt = time.Now().UTC().Add(-1 * time.Second)

	_, err := ValidateImpersonationToken(tok.TokenID)
	if err == nil {
		t.Error("expired token should fail validation")
	}
}

func TestImpersonation_ListActive(t *testing.T) {
	ResetImpersonationStore()
	IssueImpersonationToken(uuid.New(), uuid.New(), uuid.New(), "a")
	IssueImpersonationToken(uuid.New(), uuid.New(), uuid.New(), "b")

	active := ListActiveImpersonations()
	if len(active) != 2 {
		t.Errorf("expected 2 active, got %d", len(active))
	}
}

// --- Session Revocation Tests ---

func TestSessionRevocation_BlockJTI(t *testing.T) {
	ResetJTIBlocklist()
	RevokeAllUserSessions([]string{"jti-1", "jti-2"})

	if !IsJTIRevoked("jti-1") {
		t.Error("jti-1 should be revoked")
	}
	if !IsJTIRevoked("jti-2") {
		t.Error("jti-2 should be revoked")
	}
	if IsJTIRevoked("jti-3") {
		t.Error("jti-3 should NOT be revoked")
	}
}

func TestSessionRevocation_Empty(t *testing.T) {
	ResetJTIBlocklist()
	RevokeAllUserSessions(nil)
	// should not panic
}

// --- JWT Expiry Notification Tests ---

func TestExpiryNotification_Schedule(t *testing.T) {
	ResetExpiryNotifs()
	userID := uuid.New()
	expiry := time.Now().UTC().Add(5 * time.Minute)

	ScheduleExpiryNotification(userID, "token-123", expiry)

	notif := GetExpiryNotification(userID)
	if notif == nil {
		t.Fatal("notification should exist")
	}
	if notif.Message == "" {
		t.Error("message should not be empty")
	}
}

func TestExpiryNotification_Channel(t *testing.T) {
	ResetExpiryNotifs()
	userID := uuid.New()
	ch := RegisterExpiryChannel(userID)

	ScheduleExpiryNotification(userID, "tok", time.Now().UTC().Add(5*time.Minute))

	select {
	case notif := <-ch:
		if notif.UserID != userID {
			t.Error("userID mismatch")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("should receive notification via channel")
	}
}
