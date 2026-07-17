package server

import (
	"testing"
)

func TestHRConnectorRepo_NilPool(t *testing.T) {
	repo := newHRConnectorRepo(nil)
	connectors, err := repo.ListConnectors(nil)
	if err != nil {
		t.Fatalf("nil pool should not error: %v", err)
	}
	if len(connectors) != 0 {
		t.Error("nil pool should return empty")
	}
	log, err := repo.ListSyncLog(nil, 10)
	if err != nil {
		t.Fatalf("nil pool should not error: %v", err)
	}
	if len(log) != 0 {
		t.Error("nil pool should return empty")
	}
}

func TestHRConnectorRepo_CreateNilPool(t *testing.T) {
	repo := newHRConnectorRepo(nil)
	c := &HRConnectorConfig{Name: "workday-prod", Type: "workday", Enabled: true}
	if err := repo.CreateConnector(nil, c); err != nil {
		t.Errorf("nil pool CreateConnector should not error: %v", err)
	}
}

func TestSyncHREvents_Empty(t *testing.T) {
	events := syncHREvents(&HRConnectorConfig{Type: "workday"})
	if len(events) != 0 {
		t.Error("simulation mode should return no events")
	}
}

func TestHREvent_Struct(t *testing.T) {
	event := HREvent{
		EventType: "hired", EmployeeID: "emp-001",
		Email: "new@example.com", Department: "Engineering",
	}
	if event.EventType != "hired" { t.Error("type mismatch") }
	if event.Department != "Engineering" { t.Error("dept mismatch") }
}

func TestHRConnectorConfig_Defaults(t *testing.T) {
	c := HRConnectorConfig{Name: "bamboo", Type: "bamboohr"}
	if c.Type != "bamboohr" { t.Error("type mismatch") }
	if c.Enabled != false { t.Error("should default to false before creation") }
}
