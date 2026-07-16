package tools

import (
	"testing"
)

func TestRegistryFilterByScopes(t *testing.T) {
	r := NewRegistry()
	total := len(r.All())
	if total < 10 {
		t.Fatalf("expected at least 10 tools, got %d", total)
	}

	// Admin sees all
	all := r.FilterByScopes([]string{"admin"})
	if len(all) != total {
		t.Errorf("admin should see all %d tools, got %d", total, len(all))
	}

	// users:read only sees read tools
	readOnly := r.FilterByScopes([]string{"users:read"})
	for _, tool := range readOnly {
		if tool.Name == "create_user" || tool.Name == "lock_user" {
			t.Errorf("users:read should not see %s", tool.Name)
		}
	}

	// No scopes = no tools
	none := r.FilterByScopes([]string{})
	if len(none) != 0 {
		t.Errorf("no scopes should return 0 tools, got %d", len(none))
	}
}

func TestRegistryFind(t *testing.T) {
	r := NewRegistry()
	tool, ok := r.Find("list_users")
	if !ok {
		t.Fatal("expected to find list_users tool")
	}
	if tool.Name != "list_users" {
		t.Errorf("got %s", tool.Name)
	}

	_, ok = r.Find("nonexistent")
	if ok {
		t.Error("should not find nonexistent tool")
	}
}

func TestHasAllScopes(t *testing.T) {
	have := map[string]bool{"a": true, "b": true}
	if !hasAllScopes(have, []string{"a"}) {
		t.Error("should have a")
	}
	if !hasAllScopes(have, []string{"a", "b"}) {
		t.Error("should have a and b")
	}
	if hasAllScopes(have, []string{"c"}) {
		t.Error("should not have c")
	}
}
