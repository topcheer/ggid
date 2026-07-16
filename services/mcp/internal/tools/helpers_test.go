package tools

import "testing"

func TestArgStr_Present(t *testing.T) {
	args := map[string]any{"name": "test-user", "id": float64(123)}
	if got := argStr(args, "name"); got != "test-user" {
		t.Errorf("expected test-user, got %s", got)
	}
	if got := argStr(args, "id"); got != "123" {
		t.Errorf("expected 123, got %s", got)
	}
}

func TestArgStr_Missing(t *testing.T) {
	args := map[string]any{}
	if got := argStr(args, "missing"); got != "" {
		t.Errorf("expected empty string, got %s", got)
	}
}

func TestArgStr_Nil(t *testing.T) {
	if got := argStr(nil, "key"); got != "" {
		t.Errorf("expected empty string for nil args, got %s", got)
	}
}

func TestArgInt_Float64(t *testing.T) {
	args := map[string]any{"page": float64(42)}
	if got := argInt(args, "page", 1); got != 42 {
		t.Errorf("expected 42, got %d", got)
	}
}

func TestArgInt_Int(t *testing.T) {
	args := map[string]any{"count": 10}
	if got := argInt(args, "count", 1); got != 10 {
		t.Errorf("expected 10, got %d", got)
	}
}

func TestArgInt_MissingDefault(t *testing.T) {
	args := map[string]any{}
	if got := argInt(args, "missing", 99); got != 99 {
		t.Errorf("expected default 99, got %d", got)
	}
}

func TestArgInt_InvalidType(t *testing.T) {
	args := map[string]any{"page": "not-a-number"}
	if got := argInt(args, "page", 5); got != 5 {
		t.Errorf("expected default 5 for string type, got %d", got)
	}
}

func TestArgInt_Nil(t *testing.T) {
	if got := argInt(nil, "key", 7); got != 7 {
		t.Errorf("expected default 7 for nil args, got %d", got)
	}
}

func TestRegistryAll(t *testing.T) {
	r := NewRegistry()
	all := r.All()
	if len(all) == 0 {
		t.Error("expected non-empty tool list")
	}
}

func TestRegistryFilterByScopes_MultiScope(t *testing.T) {
	r := NewRegistry()
	total := len(r.All())

	// Admin sees all
	admin := r.FilterByScopes([]string{"admin"})
	if len(admin) != total {
		t.Errorf("admin should see all %d, got %d", total, len(admin))
	}

	// Specific scopes only see matching tools
	readOnly := r.FilterByScopes([]string{"users:read"})
	if len(readOnly) >= total {
		t.Errorf("users:read should see fewer tools than admin (%d vs %d)", len(readOnly), total)
	}
}

func TestRegistryFind_Existing(t *testing.T) {
	r := NewRegistry()
	tool, ok := r.Find("list_users")
	if !ok {
		t.Fatal("expected to find list_users")
	}
	if tool.Name != "list_users" {
		t.Errorf("expected list_users, got %s", tool.Name)
	}
}

func TestRegistryFind_Missing(t *testing.T) {
	r := NewRegistry()
	_, ok := r.Find("nonexistent_tool")
	if ok {
		t.Error("should not find nonexistent tool")
	}
}
