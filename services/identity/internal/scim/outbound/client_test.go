package outbound

import (
	"context"
	"testing"
	"time"
)

func TestMapGGIDToSCIM(t *testing.T) {
	user := GGIDUser{
		ID:          "u1",
		UserName:    "john.doe",
		DisplayName: "John Doe",
		Email:       "john@example.com",
		Active:      true,
		Groups:      []string{"engineers", "admins"},
	}

	scim := mapGGIDToSCIM(user)
	if scim.UserName != "john.doe" {
		t.Fatalf("expected userName john.doe, got %s", scim.UserName)
	}
	if !scim.Active {
		t.Fatal("expected active=true")
	}
	if len(scim.Emails) != 1 || scim.Emails[0].Value != "john@example.com" {
		t.Fatalf("expected 1 email, got %+v", scim.Emails)
	}
	if len(scim.Groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(scim.Groups))
	}
	if scim.Schemas[0] != "urn:ietf:params:scim:schemas:core:2.0:User" {
		t.Fatal("missing SCIM schema")
	}
}

func TestClient_Execute_TargetNotFound(t *testing.T) {
	client := NewClient(nil)
	log, err := client.Execute(context.Background(), "nonexistent", OpCreateUser, GGIDUser{ID: "u1"})
	if err == nil {
		t.Fatal("expected error for unknown target")
	}
	if log.Status != "failed" {
		t.Fatalf("expected status failed, got %s", log.Status)
	}
}

func TestClient_AddAndListTargets(t *testing.T) {
	client := NewClient(nil)
	client.AddTarget(&Target{
		Name: "aws-ic", Endpoint: "https://scim.aws.amazon.com/sso",
		Enabled: true,
	})
	targets := client.ListTargets()
	if len(targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(targets))
	}
	if targets[0].Name != "aws-ic" {
		t.Fatalf("expected aws-ic, got %s", targets[0].Name)
	}
	if targets[0].ID == "" {
		t.Fatal("expected non-empty ID")
	}
}

func TestCircuitBreaker(t *testing.T) {
	cb := newCircuitBreaker(3, 100*time.Millisecond)

	// Should allow when closed.
	if !cb.allow() {
		t.Fatal("should allow when closed")
	}

	// Record failures to trip breaker.
	cb.recordFailure()
	cb.recordFailure()
	cb.recordFailure()

	// Should block when open.
	if cb.allow() {
		t.Fatal("should block when open")
	}

	// After reset timeout, should allow (half-open).
	time.Sleep(110 * time.Millisecond)
	if !cb.allow() {
		t.Fatal("should allow after reset timeout (half-open)")
	}

	// Success should close breaker.
	cb.recordSuccess()
	if cb.state != "closed" {
		t.Fatal("should be closed after success")
	}
}

func TestEnsureSchema_NilPool(t *testing.T) {
	client := NewClient(nil)
	err := client.EnsureSchema(context.Background())
	if err != nil {
		t.Fatalf("nil pool should not error: %v", err)
	}
}

func TestGetSyncLog_NilPool(t *testing.T) {
	client := NewClient(nil)
	logs, err := client.GetSyncLog(context.Background(), "aws-ic", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if logs != nil {
		t.Fatal("nil pool should return nil")
	}
}

func TestClient_DisabledTarget(t *testing.T) {
	client := NewClient(nil)
	client.AddTarget(&Target{
		Name: "disabled-target", Endpoint: "https://example.com",
		Enabled: false,
	})

	log, err := client.Execute(context.Background(), "disabled-target", OpCreateUser, GGIDUser{ID: "u1"})
	if err == nil {
		t.Fatal("expected error for disabled target")
	}
	if log.Status != "failed" {
		t.Fatalf("expected failed, got %s", log.Status)
	}
}
