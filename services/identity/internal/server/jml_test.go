package server

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestEventToTrigger_Joiner(t *testing.T) {
	if eventToTrigger("user.created") != "joiner" {
		t.Error("user.created should map to joiner")
	}
}

func TestEventToTrigger_Leaver(t *testing.T) {
	if eventToTrigger("user.deleted") != "leaver" {
		t.Error("user.deleted should map to leaver")
	}
}

func TestEventToTrigger_Unknown(t *testing.T) {
	if eventToTrigger("user.random") != "" {
		t.Error("unknown event should map to empty string")
	}
}

func TestMatchConditions_ExactMatch(t *testing.T) {
	conditions := map[string]any{"department": "engineering"}
	attrs := map[string]any{"department": "engineering"}
	if !matchConditions(conditions, attrs) {
		t.Error("exact match should return true")
	}
}

func TestMatchConditions_NoMatch(t *testing.T) {
	conditions := map[string]any{"department": "engineering"}
	attrs := map[string]any{"department": "sales"}
	if matchConditions(conditions, attrs) {
		t.Error("mismatched values should return false")
	}
}

func TestMatchConditions_Wildcard(t *testing.T) {
	conditions := map[string]any{"department": "*"}
	attrs := map[string]any{"department": "anything"}
	if !matchConditions(conditions, attrs) {
		t.Error("wildcard should match any value")
	}
}

func TestMatchConditions_EmptyConditions(t *testing.T) {
	if !matchConditions(map[string]any{}, map[string]any{"x": "y"}) {
		t.Error("empty conditions should match all")
	}
}

func TestMatchConditions_MissingKey(t *testing.T) {
	conditions := map[string]any{"department": "engineering"}
	attrs := map[string]any{"title": "engineer"}
	if matchConditions(conditions, attrs) {
		t.Error("missing key in attrs should return false")
	}
}

func TestJMLEngine_ProcessEvent_NoRules(t *testing.T) {
	engine := &JMLEngine{repo: newLifecycleRepo(nil)}
	// With nil pool, FindMatchingRules returns empty → no panic.
	engine.ProcessEvent(context.Background(), LifecycleEvent{
		EventType: "user.created",
		UserID:    uuid.New(),
	})
	// No panic = pass
}
