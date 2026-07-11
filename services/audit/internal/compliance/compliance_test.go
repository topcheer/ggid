package compliance

import (
	"context"
	"testing"
	"time"
)

type mockQuery struct {
	events []AuditEvent
}

func (m *mockQuery) QueryEvents(_ context.Context, _, _ time.Time, _ []string) ([]AuditEvent, error) {
	return m.events, nil
}

// 1. TestGenerate_SOC2
func TestGenerate_SOC2(t *testing.T) {
	now := time.Now()
	q := &mockQuery{events: []AuditEvent{
		{ID: "1", Action: "login.success", Success: true, Timestamp: now},
		{ID: "2", Action: "login.failed", Success: false, Timestamp: now},
		{ID: "3", Action: "policy.change", Success: true, Timestamp: now},
	}}
	gen := NewGenerator(q)
	report, err := gen.Generate(context.Background(), ReportSOC2, now.Add(-24*time.Hour), now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Type != ReportSOC2 {
		t.Fatal("expected soc2 type")
	}
	if report.Summary.TotalEvents != 3 {
		t.Fatalf("expected 3 events, got %d", report.Summary.TotalEvents)
	}
	if report.Summary.FailedLogins != 1 {
		t.Fatalf("expected 1 failed login, got %d", report.Summary.FailedLogins)
	}
	if len(report.Sections) < 2 {
		t.Fatalf("expected >=2 sections, got %d", len(report.Sections))
	}
}

// 2. TestGenerate_HIPAA
func TestGenerate_HIPAA(t *testing.T) {
	now := time.Now()
	q := &mockQuery{events: []AuditEvent{
		{ID: "1", Action: "phi.access", Success: true, Timestamp: now},
		{ID: "2", Action: "phi.access", Success: false, Timestamp: now},
	}}
	gen := NewGenerator(q)
	report, err := gen.Generate(context.Background(), ReportHIPAA, now.Add(-24*time.Hour), now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Type != ReportHIPAA {
		t.Fatal("expected hipaa type")
	}
	if len(report.Sections) < 2 {
		t.Fatalf("expected >=2 sections, got %d", len(report.Sections))
	}
}

// 3. TestGenerate_GDPR
func TestGenerate_GDPR(t *testing.T) {
	now := time.Now()
	q := &mockQuery{events: []AuditEvent{
		{ID: "1", Action: "data.access", Success: true, Timestamp: now},
		{ID: "2", Action: "data.delete", Success: true, Timestamp: now},
	}}
	gen := NewGenerator(q)
	report, err := gen.Generate(context.Background(), ReportGDPR, now.Add(-24*time.Hour), now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Type != ReportGDPR {
		t.Fatal("expected gdpr type")
	}
	if report.Summary.DataAccessed != 1 {
		t.Fatalf("expected 1 data access, got %d", report.Summary.DataAccessed)
	}
}

// 4. TestGenerate_InvalidType
func TestGenerate_InvalidType(t *testing.T) {
	q := &mockQuery{}
	gen := NewGenerator(q)
	_, err := gen.Generate(context.Background(), "invalid", time.Now().Add(-time.Hour), time.Now())
	if err == nil {
		t.Fatal("expected error for invalid type")
	}
}

// 5. TestGenerate_EmptyEvents
func TestGenerate_EmptyEvents(t *testing.T) {
	q := &mockQuery{events: nil}
	gen := NewGenerator(q)
	report, err := gen.Generate(context.Background(), ReportSOC2, time.Now().Add(-time.Hour), time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Summary.TotalEvents != 0 {
		t.Fatalf("expected 0 events, got %d", report.Summary.TotalEvents)
	}
}

// 6. TestStatusFromEvents
func TestStatusFromEvents(t *testing.T) {
	events := []AuditEvent{
		{Action: "login.failed", Success: true},
	}
	if statusFromEvents(events, "login.failed") != "pass" {
		t.Fatal("expected pass for successful event")
	}

	events = append(events, AuditEvent{Action: "login.failed", Success: false})
	if statusFromEvents(events, "login.failed") != "warning" {
		t.Fatal("expected warning for failed event")
	}

	if statusFromEvents(events, "nonexistent") != "pass" {
		t.Fatal("expected pass for nonexistent action")
	}
}
