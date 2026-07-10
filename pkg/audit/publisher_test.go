package audit

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewEvent(t *testing.T) {
	tid := uuid.New()
	uid := uuid.New()
	e := NewEvent("user.login", "success", tid, uid)

	if e.Action != "user.login" {
		t.Errorf("expected action 'user.login', got %s", e.Action)
	}
	if e.Result != "success" {
		t.Errorf("expected result 'success', got %s", e.Result)
	}
	if e.TenantID != tid {
		t.Errorf("expected tenant ID %s, got %s", tid, e.TenantID)
	}
	if e.ActorID != uid {
		t.Errorf("expected actor ID %s, got %s", uid, e.ActorID)
	}
	if e.ActorType != "user" {
		t.Errorf("expected actor type 'user', got %s", e.ActorType)
	}
	if e.CreatedAt.IsZero() {
		t.Error("expected non-zero created_at")
	}
}

func TestEvent_JSONRoundtrip(t *testing.T) {
	e := Event{
		ID:           uuid.New(),
		TenantID:     uuid.New(),
		ActorType:    "user",
		ActorID:      uuid.New(),
		ActorName:    "alice",
		Action:       "role.assign",
		ResourceType: "role",
		ResourceID:   uuid.New(),
		ResourceName: "admin",
		Result:       "success",
		IPAddress:    "192.168.1.100",
		UserAgent:    "Mozilla/5.0",
		RequestID:    "req-123",
		Metadata:     map[string]any{"key": "value"},
		CreatedAt:    time.Now().UTC(),
	}

	data, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var e2 Event
	if err := json.Unmarshal(data, &e2); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if e2.Action != e.Action {
		t.Errorf("action mismatch: %s != %s", e2.Action, e.Action)
	}
	if e2.ActorName != e.ActorName {
		t.Errorf("actor name mismatch: %s != %s", e2.ActorName, e.ActorName)
	}
	if e2.IPAddress != e.IPAddress {
		t.Errorf("IP mismatch: %s != %s", e2.IPAddress, e.IPAddress)
	}
	if e2.Metadata["key"] != "value" {
		t.Errorf("metadata mismatch: %v", e2.Metadata)
	}
}

func TestEvent_MetadataOmitEmpty(t *testing.T) {
	e := Event{
		ID:       uuid.New(),
		TenantID: uuid.New(),
		Action:   "test.action",
	}

	data, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	str := string(data)
	if strings.Contains(str, `"metadata":`) {
		t.Errorf("expected metadata to be omitted with omitempty, got: %s", str)
	}
}

func TestEvent_NilIDs(t *testing.T) {
	e := Event{
		Action: "test",
	}
	data, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var e2 Event
	if err := json.Unmarshal(data, &e2); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if e2.Action != "test" {
		t.Errorf("action mismatch: %s", e2.Action)
	}
}

func TestDefaultNames(t *testing.T) {
	if DefaultStreamName != "AUDIT" {
		t.Errorf("expected AUDIT, got %s", DefaultStreamName)
	}
	if DefaultSubjectName != "audit.events" {
		t.Errorf("expected audit.events, got %s", DefaultSubjectName)
	}
}
