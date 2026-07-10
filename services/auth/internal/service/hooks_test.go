package service

import (
	"context"
	"testing"
)

func TestHookManager_RegisterAndRemove(t *testing.T) {
	mgr := NewHookManager()

	hook := &AuthHook{
		ID:    "hook-1",
		Event: HookPostLogin,
		URL:   "http://localhost:9999/hook",
		Enabled: true,
	}
	mgr.RegisterHook(hook)

	// Execute post-hooks (fire-and-forget, no error even if server is down).
	err := mgr.ExecuteHooks(context.Background(), HookPostLogin, &HookPayload{
		Event:    HookPostLogin,
		TenantID: "test-tenant",
	})
	// Post-hooks should not return errors.
	if err != nil {
		t.Errorf("post-hook should not error: %v", err)
	}

	mgr.RemoveHook("hook-1")
	// Should execute 0 hooks now.
	err = mgr.ExecuteHooks(context.Background(), HookPostLogin, &HookPayload{})
	if err != nil {
		t.Errorf("no hooks should not error: %v", err)
	}
}

func TestHookManager_PreLoginDeny(t *testing.T) {
	mgr := NewHookManager()
	// Pre-login hook pointing to non-existent server.
	mgr.RegisterHook(&AuthHook{
		ID:    "hook-denied",
		Event: HookPreLogin,
		URL:   "http://localhost:1/deny",
		Enabled: true,
	})

	// Pre-hooks with connection errors should return error.
	err := mgr.ExecuteHooks(context.Background(), HookPreLogin, &HookPayload{
		Event: HookPreLogin,
	})
	if err == nil {
		t.Error("pre-login hook with unreachable URL should return error")
	}
}

func TestHookManager_PostHookErrorIgnored(t *testing.T) {
	mgr := NewHookManager()
	mgr.RegisterHook(&AuthHook{
		ID:    "hook-post",
		Event: HookPostRegister,
		URL:   "http://localhost:1/fail",
		Enabled: true,
	})

	err := mgr.ExecuteHooks(context.Background(), HookPostRegister, &HookPayload{})
	if err != nil {
		t.Errorf("post hooks should not error: %v", err)
	}
}

func TestHookManager_DisabledHookSkipped(t *testing.T) {
	mgr := NewHookManager()
	mgr.RegisterHook(&AuthHook{
		ID:    "hook-disabled",
		Event: HookPreLogin,
		URL:   "http://localhost:1/fail",
		Enabled: false,
	})

	err := mgr.ExecuteHooks(context.Background(), HookPreLogin, &HookPayload{})
	if err != nil {
		t.Errorf("disabled hooks should be skipped: %v", err)
	}
}
