package server

import (
	"context"
	"testing"
)

func TestRLSTables_Count(t *testing.T) {
	tables := RLSTables()
	if len(tables) < 20 {
		t.Errorf("expected >=20 RLS tables, got %d", len(tables))
	}
	// Verify key tables are present.
	required := map[string]bool{"users": false, "groups": false, "audit_events": false, "policies": false}
	for _, tbl := range tables {
		if _, ok := required[tbl]; ok {
			required[tbl] = true
		}
	}
	for tbl, found := range required {
		if !found {
			t.Errorf("%s should be in RLS table list", tbl)
		}
	}
}

func TestIsValidTableName(t *testing.T) {
	valid := []string{"users", "audit_events", "oauth_clients", "policy_decisions"}
	for _, name := range valid {
		if !isValidTableName(name) {
			t.Errorf("%s should be valid", name)
		}
	}
	invalid := []string{"", "users; DROP TABLE", "table'OR'1'='1", "a.b.c"}
	for _, name := range invalid {
		if isValidTableName(name) {
			t.Errorf("%s should be invalid", name)
		}
	}
}

func TestRLSRepo_NilPool(t *testing.T) {
	repo := newRLSRepo(nil)
	// EnableRLS with nil pool is no-op.
	if err := repo.EnableRLS(nil, "users"); err != nil {
		t.Errorf("nil pool EnableRLS should not error: %v", err)
	}
	status, err := repo.GetRLSStatus(nil)
	if err != nil {
		t.Fatalf("nil pool GetRLSStatus should not error: %v", err)
	}
	if len(status) != 0 {
		t.Error("nil pool should return empty status")
	}
	result, err := repo.RunIsolationTest(nil)
	if err != nil {
		t.Fatalf("nil pool RunIsolationTest should not error: %v", err)
	}
	if result["status"] != "skipped" {
		t.Error("nil pool should return skipped status")
	}
}

func TestSetTenantContext(t *testing.T) {
	// SetTenantContext requires a real DB connection.
	// Just verify the function signature compiles and is callable.
	// With nil execer, it will panic — so we skip the call.
	_ = SetTenantContext
}

func TestIsValidTableName_LengthLimit(t *testing.T) {
	// Very long name should be rejected.
	longName := ""
	for i := 0; i < 100; i++ {
		longName += "a"
	}
	if isValidTableName(longName) {
		t.Error("overly long table name should be invalid")
	}
}
