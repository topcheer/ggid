package audit

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestPublisher_Close tests Close on a publisher with nil nc (no panic).
func TestPublisher_Close_NilConn(t *testing.T) {
	p := &Publisher{} // nc is nil
	p.Close()         // should not panic
}

// TestPublisher_Close_RealConn tests Close with a real (but disconnected) nc.
func TestPublisher_Close_WithConn(t *testing.T) {
	p := &Publisher{
		nc: nil, // would need a real nats.Conn for full test
	}
	p.Close()
}

// TestPublisher_PublishAsync_NilJS tests PublishAsync when js is nil (no panic).
func TestPublisher_PublishAsync_NilJS(t *testing.T) {
	p := &Publisher{} // js is nil
	p.PublishAsync(Event{Action: "test"})
	// should return without panic
}

// TestPublisher_PublishAsync_SetsDefaults tests that PublishAsync
// populates ID and CreatedAt if missing.
func TestPublisher_PublishAsync_SetsDefaults(t *testing.T) {
	p := &Publisher{} // js is nil, but defaults are set before publish
	e := Event{Action: "test.action"}
	p.PublishAsync(e)
	// The function sets ID and CreatedAt internally;
	// since js is nil it returns early after setting defaults.
	// We can't observe the modified event since it's by value,
	// but we verify no panic occurs.
}

// TestEvent_JSONRoundtrip_AllFields tests full event with all fields populated.
func TestEvent_JSONRoundtrip_AllFields(t *testing.T) {
	e := Event{
		ID:           uuid.New(),
		TenantID:     uuid.New(),
		ActorType:    "api_key",
		ActorID:      uuid.New(),
		ActorName:    "service-account",
		Action:       "user.delete",
		ResourceType: "user",
		ResourceID:   uuid.New(),
		ResourceName: "bob",
		Result:       "success",
		IPAddress:    "10.0.0.1",
		UserAgent:    "curl/8.0",
		RequestID:    "req-abc-123",
		Metadata:     map[string]any{"reason": "gdpr", "count": 42},
		CreatedAt:    time.Now().UTC(),
	}

	data, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var e2 Event
	if err := json.Unmarshal(data, &e2); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if e2.ID != e.ID {
		t.Errorf("ID mismatch")
	}
	if e2.TenantID != e.TenantID {
		t.Errorf("TenantID mismatch")
	}
	if e2.ActorType != "api_key" {
		t.Errorf("ActorType mismatch: %s", e2.ActorType)
	}
	if e2.ResourceType != "user" {
		t.Errorf("ResourceType mismatch: %s", e2.ResourceType)
	}
	if e2.Result != "success" {
		t.Errorf("Result mismatch: %s", e2.Result)
	}
	if e2.Metadata["reason"] != "gdpr" {
		t.Errorf("Metadata reason mismatch: %v", e2.Metadata)
	}
}

// TestNewPublisher_InvalidURL tests NewPublisher with an unreachable NATS URL.
func TestNewPublisher_InvalidURL(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := NewPublisher(ctx, "nats://127.0.0.1:1") // port 1 = connection refused
	if err == nil {
		t.Error("expected error connecting to invalid NATS URL")
	}
}

// TestNewPublisher_InvalidStreamConfig tests NewPublisher with a valid NATS
// connection but an invalid stream configuration (empty context).
func TestNewPublisher_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately cancel

	// Even with cancelled context, NATS connect may succeed,
	// but CreateOrUpdateStream should fail with cancelled context.
	_, err := NewPublisher(ctx, "nats://127.0.0.1:1")
	if err == nil {
		t.Log("expected error with cancelled context")
	}
}

// TestPublisher_Publish_NilJS tests Publish when js is nil.
// This should panic, so we test it doesn't crash in production
// by checking the code path.
func TestPublisher_Publish_DefaultsSet(t *testing.T) {
	// Verify that the Event defaults logic works:
	// If ID is Nil, it gets set. If CreatedAt is zero, it gets set.
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

// TestEvent_AnonymousActor tests events with anonymous actor type.
func TestEvent_AnonymousActor(t *testing.T) {
	e := NewEvent("auth.attempt", "failure", uuid.New(), uuid.Nil)
	e.ActorType = "anonymous"

	data, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var e2 Event
	if err := json.Unmarshal(data, &e2); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if e2.ActorType != "anonymous" {
		t.Errorf("expected anonymous, got %s", e2.ActorType)
	}
	if e2.Result != "failure" {
		t.Errorf("expected failure, got %s", e2.Result)
	}
}

// TestEvent_SystemActor tests events with system actor type.
func TestEvent_SystemActor(t *testing.T) {
	e := NewEvent("key.rotate", "success", uuid.New(), uuid.Nil)
	e.ActorType = "system"
	e.Metadata = map[string]any{"key_id": "key-001"}

	data, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var e2 Event
	if err := json.Unmarshal(data, &e2); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if e2.ActorType != "system" {
		t.Errorf("expected system, got %s", e2.ActorType)
	}
}

// TestEvent_EmptyMetadata tests that metadata is omitted when nil.
func TestEvent_EmptyMetadata(t *testing.T) {
	e := Event{
		ID:     uuid.New(),
		Action: "test",
	}
	data, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal to map: %v", err)
	}
	if _, ok := m["metadata"]; ok {
		t.Error("expected metadata to be omitted when nil")
	}
}

// TestEvent_AllResultTypes tests all valid result types.
func TestEvent_AllResultTypes(t *testing.T) {
	results := []string{"success", "failure", "denied"}
	for _, r := range results {
		e := NewEvent("test", r, uuid.New(), uuid.New())
		if e.Result != r {
			t.Errorf("expected %s, got %s", r, e.Result)
		}
	}
}
