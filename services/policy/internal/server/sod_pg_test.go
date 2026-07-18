package httpserver

import "testing"

func TestSodPGRepo_NilPool(t *testing.T) {
	repo := NewSodPGRepo(nil)
	if repo == nil {
		t.Fatal("NewSodPGRepo returned nil")
	}
}

func TestSoDRulePG_JSON(t *testing.T) {
	rule := SoDRulePG{
		ID: "sod-test", RoleA: "admin", RoleB: "auditor",
		Description: "mutually exclusive", Enabled: true,
	}
	if rule.RoleA != "admin" || rule.RoleB != "auditor" {
		t.Error("unexpected role values")
	}
	if !rule.Enabled {
		t.Error("should be enabled")
	}
}

func TestSoDViolationPG_JSON(t *testing.T) {
	v := SoDViolationPG{
		ID: "sdv-test", UserID: "u1", RoleA: "admin", RoleB: "compliance",
		Reason: "conflict", Status: "open",
	}
	if v.Status != "open" {
		t.Error("expected open status")
	}
}
