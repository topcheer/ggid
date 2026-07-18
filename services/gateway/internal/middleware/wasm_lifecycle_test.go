package middleware

import (
	"context"
	"testing"
)

func TestLifecycleManager_RegisterUnregister(t *testing.T) {
	mgr := NewLifecycleManager(nil, nil)
	mgr.RegisterPlugin("plugin-a", []LifecycleHook{HookPreAuth, HookPostAuth})
	mgr.RegisterPlugin("plugin-b", []LifecycleHook{HookPreAuth})

	hooks := mgr.GetRegisteredHooks("plugin-a")
	if len(hooks) != 2 {
		t.Fatalf("expected 2 hooks for plugin-a, got %d", len(hooks))
	}

	// PreAuth should have 2 plugins.
	mgr.mu.RLock()
	preAuth := mgr.hookPlugins[HookPreAuth]
	mgr.mu.RUnlock()
	if len(preAuth) != 2 {
		t.Fatalf("expected 2 plugins for pre_auth, got %d", len(preAuth))
	}

	// Unregister plugin-a.
	mgr.UnregisterPlugin("plugin-a")
	hooks = mgr.GetRegisteredHooks("plugin-a")
	if len(hooks) != 0 {
		t.Fatalf("expected 0 hooks after unregister, got %d", len(hooks))
	}

	// PreAuth should now have 1.
	mgr.mu.RLock()
	preAuth = mgr.hookPlugins[HookPreAuth]
	mgr.mu.RUnlock()
	if len(preAuth) != 1 {
		t.Fatalf("expected 1 plugin for pre_auth after unregister, got %d", len(preAuth))
	}
}

func TestLifecycleManager_NoPlugins(t *testing.T) {
	mgr := NewLifecycleManager(nil, nil)
	result := mgr.InvokeHook(context.Background(), HookPreAuth, PluginContext{})
	if result != nil {
		t.Fatal("expected nil result with no plugins")
	}
}

func TestToWasmPhase(t *testing.T) {
	tests := []struct {
		hook   LifecycleHook
		expect WasmPluginPhase
	}{
		{HookPreAuth, PhaseRequest},
		{HookPostAuth, PhaseRequest},
		{HookPrePolicy, PhaseRequest},
		{HookPostPolicy, PhaseResponse},
		{HookPreResponse, PhaseResponse},
		{HookOnAudit, PhaseRequest},
	}
	for _, tt := range tests {
		got := toWasmPhase(tt.hook)
		if got != tt.expect {
			t.Errorf("toWasmPhase(%s) = %s, want %s", tt.hook, got, tt.expect)
		}
	}
}

func TestPluginStore_EnsureSchema_NilPool(t *testing.T) {
	store := NewPluginStore(nil)
	if err := store.EnsureSchema(context.Background()); err != nil {
		t.Fatalf("nil pool should not error: %v", err)
	}
}

func TestPluginStore_ListPlugins_NilPool(t *testing.T) {
	store := NewPluginStore(nil)
	plugins, err := store.ListPlugins(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plugins != nil {
		t.Fatal("nil pool should return nil")
	}
}

func TestLifecycleManager_ReloadVersion(t *testing.T) {
	mgr := NewLifecycleManager(nil, NewPluginStore(nil))
	if mgr.ReloadVersion() != 0 {
		t.Fatal("initial version should be 0")
	}
	// ReloadFromStore with nil pool should succeed and increment version.
	if err := mgr.ReloadFromStore(context.Background()); err != nil {
		t.Fatalf("reload should not error: %v", err)
	}
	if mgr.ReloadVersion() != 1 {
		t.Fatalf("expected version 1 after reload, got %d", mgr.ReloadVersion())
	}
}

func TestAllHooks(t *testing.T) {
	if len(AllHooks) != 6 {
		t.Fatalf("expected 6 hooks, got %d", len(AllHooks))
	}
	// Verify all hooks are distinct.
	seen := map[LifecycleHook]bool{}
	for _, h := range AllHooks {
		if seen[h] {
			t.Fatalf("duplicate hook: %s", h)
		}
		seen[h] = true
	}
}

func TestLifecycleManager_OnAuditEvent(t *testing.T) {
	mgr := NewLifecycleManager(nil, nil)
	// Should not panic with no plugins.
	mgr.OnAuditEvent(context.Background(), "user.login", "user-1", "user.login")
}
