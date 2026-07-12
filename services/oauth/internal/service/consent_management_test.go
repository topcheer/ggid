package service

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestConsentManagement_GrantAndGet(t *testing.T) {
	cm := NewConsentManager()
	userID := uuid.New()
	clientID := "client-1"

	record, err := cm.GrantConsent(userID, clientID, []string{"read", "write"}, nil)
	if err != nil {
		t.Fatalf("GrantConsent: %v", err)
	}
	if record.UserID != userID {
		t.Error("user ID mismatch")
	}
	if len(record.Scopes) != 2 {
		t.Errorf("expected 2 scopes, got %d", len(record.Scopes))
	}

	got, err := cm.GetConsent(userID, clientID)
	if err != nil {
		t.Fatalf("GetConsent: %v", err)
	}
	if got == nil {
		t.Fatal("consent should exist")
	}
	if got.UserID != userID {
		t.Error("user ID mismatch")
	}
}

func TestConsentManagement_GrantConsent_NilUser(t *testing.T) {
	cm := NewConsentManager()
	_, err := cm.GrantConsent(uuid.Nil, "client-1", []string{"read"}, nil)
	if err == nil {
		t.Error("should error on nil user")
	}
}

func TestConsentManagement_GrantConsent_EmptyClient(t *testing.T) {
	cm := NewConsentManager()
	_, err := cm.GrantConsent(uuid.New(), "", []string{"read"}, nil)
	if err == nil {
		t.Error("should error on empty client")
	}
}

func TestConsentManagement_WithdrawConsent(t *testing.T) {
	cm := NewConsentManager()
	userID := uuid.New()
	cm.GrantConsent(userID, "client-1", []string{"read"}, nil)

	err := cm.WithdrawConsent(userID, "client-1")
	if err != nil {
		t.Fatalf("WithdrawConsent: %v", err)
	}

	valid, reason := cm.IsConsentValid(userID, "client-1", []string{"read"})
	if valid {
		t.Error("consent should be invalid after withdrawal")
	}
	if reason != "consent withdrawn" {
		t.Errorf("expected 'consent withdrawn', got '%s'", reason)
	}
}

func TestConsentManagement_WithdrawConsent_NotFound(t *testing.T) {
	cm := NewConsentManager()
	err := cm.WithdrawConsent(uuid.New(), "client-1")
	if err == nil {
		t.Error("should error when consent not found")
	}
}

func TestConsentManagement_IsConsentValid_Valid(t *testing.T) {
	cm := NewConsentManager()
	userID := uuid.New()
	cm.GrantConsent(userID, "client-1", []string{"read", "write"}, nil)

	valid, _ := cm.IsConsentValid(userID, "client-1", []string{"read"})
	if !valid {
		t.Error("consent should be valid")
	}
}

func TestConsentManagement_IsConsentValid_Expired(t *testing.T) {
	cm := NewConsentManager()
	userID := uuid.New()
	expired := time.Now().Add(-1 * time.Hour)
	cm.GrantConsent(userID, "client-1", []string{"read"}, &expired)

	valid, reason := cm.IsConsentValid(userID, "client-1", []string{"read"})
	if valid {
		t.Error("expired consent should be invalid")
	}
	if reason != "consent expired" {
		t.Errorf("expected 'consent expired', got '%s'", reason)
	}
}

func TestConsentManagement_IsConsentValid_ScopeNotGranted(t *testing.T) {
	cm := NewConsentManager()
	userID := uuid.New()
	cm.GrantConsent(userID, "client-1", []string{"read"}, nil)

	valid, reason := cm.IsConsentValid(userID, "client-1", []string{"delete"})
	if valid {
		t.Error("consent should be invalid for ungranted scope")
	}
	if reason == "" {
		t.Error("should have denial reason")
	}
}

func TestConsentManagement_IsConsentValid_NotFound(t *testing.T) {
	cm := NewConsentManager()
	valid, reason := cm.IsConsentValid(uuid.New(), "client-1", []string{"read"})
	if valid {
		t.Error("should be invalid when consent not found")
	}
	if reason != "consent not found" {
		t.Errorf("expected 'consent not found', got '%s'", reason)
	}
}

func TestConsentManagement_ListConsents(t *testing.T) {
	cm := NewConsentManager()
	userID := uuid.New()
	cm.GrantConsent(userID, "client-1", []string{"read"}, nil)
	cm.GrantConsent(userID, "client-2", []string{"write"}, nil)

	list, err := cm.ListConsents(userID)
	if err != nil {
		t.Fatalf("ListConsents: %v", err)
	}
	if len(list) != 2 { //nolint:staticcheck
		t.Errorf("expected 2 consents, got %d", len(list))
	}
}

func TestConsentManagement_ReGrantUpdatesScopes(t *testing.T) {
	cm := NewConsentManager()
	userID := uuid.New()
	cm.GrantConsent(userID, "client-1", []string{"read"}, nil)
	cm.GrantConsent(userID, "client-1", []string{"write"}, nil)

	record, _ := cm.GetConsent(userID, "client-1")
	if len(record.Scopes) != 2 {
		t.Errorf("expected 2 scopes after re-grant, got %d", len(record.Scopes))
	}
}
