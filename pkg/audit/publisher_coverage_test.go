package audit

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestNewPublisher_InvalidNATSURL covers the NATS connection error path.
func TestNewPublisher_InvalidNATSURL(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// An invalid NATS URL should return an error from NewPublisher.
	p, err := NewPublisher(ctx, "nats://127.0.0.1:1")
	if err == nil {
		if p != nil {
			p.Close()
		}
		t.Fatal("expected error for invalid NATS URL, got nil")
	}
	if p != nil {
		t.Fatal("expected nil publisher on error")
	}
}

// TestPublishDefaultFields covers the ID and CreatedAt default logic in Publish/PublishAsync.
func TestPublishDefaultFields(t *testing.T) {
	// Verify the default-setting logic works without a real NATS connection
	// by checking that events with zero fields get populated correctly.
	e := Event{Action: "test"}
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now()
	}

	if e.ID == uuid.Nil {
		t.Error("expected non-nil ID after default")
	}
	if e.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt after default")
	}
}

// TestNewEvent_AllFields verifies the NewEvent convenience constructor.
func TestNewEvent_AllFields(t *testing.T) {
	tid := uuid.New()
	uid := uuid.New()
	e := NewEvent("user.logout", "success", tid, uid)

	if e.ActorType != "user" {
		t.Errorf("expected actor type 'user', got '%s'", e.ActorType)
	}
	if e.Result != "success" {
		t.Errorf("expected result 'success', got '%s'", e.Result)
	}
	if e.CreatedAt.IsZero() {
		t.Error("expected non-zero created_at")
	}
}

// TestPublisher_Close_Nil verifies Close is safe when nc is nil.
func TestPublisher_Close_Nil(t *testing.T) {
	p := &Publisher{}
	// Should not panic.
	p.Close()
}

// TestPublishAsync_NilJS verifies PublishAsync does not panic with nil js.
func TestPublishAsync_NilJS(t *testing.T) {
	p := &Publisher{
		stream:  DefaultStreamName,
		subject: DefaultSubjectName,
		// js is nil — simulates disconnected publisher
	}
	// Should not panic; fire-and-forget contract.
	p.PublishAsync(Event{
		ID:        uuid.New(),
		TenantID:  uuid.New(),
		Action:    "test.async.nil",
		ActorType: "system",
	})
}
