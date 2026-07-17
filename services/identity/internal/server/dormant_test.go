package server

import (
	"context"
	"testing"
	"time"
)

func TestEvaluateDormantState(t *testing.T) {
	tests := []struct {
		days       int
		wantState  string
		wantAction string
	}{
		{0, "active", "none"},
		{89, "active", "none"},
		{90, "dormant", "notify"},
		{100, "dormant", "notify"},
		{120, "suspended", "suspend"},
		{150, "archived", "archive"},
		{365, "archived", "archive"},
	}
	for _, tt := range tests {
		state, action := EvaluateDormantState(tt.days)
		if state != tt.wantState {
			t.Errorf("days=%d: state=%s want=%s", tt.days, state, tt.wantState)
		}
		if action != tt.wantAction {
			t.Errorf("days=%d: action=%s want=%s", tt.days, action, tt.wantAction)
		}
	}
}

func TestDormantRepo_NilPool(t *testing.T) {
	repo := newDormantRepo(nil)
	dormant, err := repo.ListDormant(nil)
	if err != nil { t.Fatalf("nil pool should not error: %v", err) }
	if len(dormant) != 0 { t.Error("nil pool should return empty") }
	ghosts, err := repo.ListGhosts(nil)
	if err != nil { t.Fatalf("nil pool should not error: %v", err) }
	if len(ghosts) != 0 { t.Error("nil pool should return empty") }
}

func TestProcessHREventJML(t *testing.T) {
	repo := newDormantRepo(nil)
	tests := []struct {
		eventType string
		want      string
	}{
		{"terminated", "jml:disable"},
		{"hired", "jml:create"},
		{"dept_change", "jml:access_review"},
		{"manager_change", "jml:update_manager"},
		{"unknown", "jml:none"},
	}
	for _, tt := range tests {
		got := repo.ProcessHREventJML(nil, &HREvent{EventType: tt.eventType})
		if got != tt.want {
			t.Errorf("event=%s: got=%s want=%s", tt.eventType, got, tt.want)
		}
	}
}

func TestDormantRepo_RunDormantScan_NilPool(t *testing.T) {
	repo := newDormantRepo(nil)
	now := time.Now().Add(-100 * 24 * time.Hour)
	count, err := repo.RunDormantScan(nil, func(_ context.Context) ([]UserLoginInfo, error) {
		return []UserLoginInfo{{UserID: "u1", LastLoginAt: &now}}, nil
	})
	// nil pool → 0 updates, no error
	if err != nil { t.Fatalf("nil pool should not error: %v", err) }
	if count != 0 { t.Error("nil pool should update 0") }
}

func TestGhostReconciliation_NilPool(t *testing.T) {
	repo := newDormantRepo(nil)
	users := []UserInfo{{UserID: "u1", Email: "a@b.com", EmployeeID: "emp-1", Status: "active"}}
	hrActive := map[string]bool{"emp-2": true} // emp-1 not in HR → ghost
	ghosts, err := repo.RunGhostReconciliation(nil, users, hrActive)
	if err != nil { t.Fatalf("nil pool should not error: %v", err) }
	if len(ghosts) != 0 { t.Error("nil pool should return empty ghosts") }
}
