package middleware

import (
	"testing"

	"github.com/google/uuid"
)

func TestPluginRepo_NilPool(t *testing.T) {
	repo := NewPluginRepo(nil)

	// List should return empty, not panic.
	plugins, err := repo.List(nil, uuid.New())
	if err != nil {
		t.Fatalf("nil pool should not error: %v", err)
	}
	if len(plugins) != 0 {
		t.Error("nil pool should return empty")
	}

	// Create should be no-op.
	p := &PluginRecord{Name: "test", TenantID: uuid.New()}
	if err := repo.Create(nil, p); err != nil {
		t.Errorf("nil pool Create should not error: %v", err)
	}

	// Hook bindings should return empty.
	bindings, err := repo.ListHookBindings(nil, uuid.New(), "on_request")
	if err != nil {
		t.Fatalf("nil pool should not error: %v", err)
	}
	if len(bindings) != 0 {
		t.Error("nil pool should return empty")
	}
}

func TestPluginRecord_Defaults(t *testing.T) {
	p := &PluginRecord{
		Name: "rate-limiter",
		Hooks: []string{"on_request"},
		MaxMemoryMB: 8,
		TimeoutMs: 50,
	}
	if p.Name != "rate-limiter" {
		t.Error("name mismatch")
	}
	if len(p.Hooks) != 1 {
		t.Error("hooks mismatch")
	}
	if p.MaxMemoryMB != 8 {
		t.Error("memory mismatch")
	}
}

func TestPluginRepo_SetEnabled_NilPool(t *testing.T) {
	repo := NewPluginRepo(nil)
	if err := repo.SetEnabled(nil, uuid.New(), true); err != nil {
		t.Errorf("nil pool should not error: %v", err)
	}
}

func TestPluginRepo_Delete_NilPool(t *testing.T) {
	repo := NewPluginRepo(nil)
	if err := repo.Delete(nil, uuid.New()); err != nil {
		t.Errorf("nil pool should not error: %v", err)
	}
}

func TestPluginRepo_GetByName_NilPool(t *testing.T) {
	repo := NewPluginRepo(nil)
	_, err := repo.GetByName(nil, "test")
	if err == nil {
		t.Error("nil pool GetByName should return error")
	}
}

func TestHookBinding_Defaults(t *testing.T) {
	hb := &HookBinding{
		HookName: "on_response",
		Priority: 50,
		Enabled:  true,
	}
	if hb.HookName != "on_response" {
		t.Error("hook name mismatch")
	}
	if hb.Priority != 50 {
		t.Error("priority mismatch")
	}
	if !hb.Enabled {
		t.Error("should be enabled")
	}
}
