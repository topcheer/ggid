package compliance

import (
	"context"
	"sync"
	"testing"
	"time"
)

// schedulerMockQuery returns deterministic events for scheduler tests.
type schedulerMockQuery struct{}

func (m *schedulerMockQuery) QueryEvents(ctx context.Context, from, to time.Time, actionTypes []string) ([]AuditEvent, error) {
	return []AuditEvent{
		{ID: "1", TenantID: "tenant-1", Action: "user.login", Success: true, Timestamp: time.Now()},
		{ID: "2", TenantID: "tenant-1", Action: "user.login", Success: false, Timestamp: time.Now()},
		{ID: "3", TenantID: "tenant-1", Action: "role.assign", Success: true, Timestamp: time.Now()},
	}, nil
}

type mockEmailer struct {
	mu       sync.Mutex
	sent     int
	lastTo   []string
	lastSubj string
}

func (m *mockEmailer) Send(to []string, subject string, body []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent++
	m.lastTo = to
	m.lastSubj = subject
	return nil
}

func TestScheduler_GenerateNow(t *testing.T) {
	cfg := ScheduleConfig{
		Interval:    24 * time.Hour,
		ReportTypes: []ReportType{ReportSOC2, ReportHIPAA},
		TenantIDs:   []string{"tenant-1"},
		Recipients:  []string{"compliance@example.com"},
	}
	s := NewScheduler(cfg, &schedulerMockQuery{}, &mockEmailer{})
	s.GenerateNow(context.Background())

	reports := s.ListReports()
	if len(reports) != 2 {
		t.Fatalf("expected 2 reports, got %d", len(reports))
	}
	for _, r := range reports {
		if r.TenantID != "tenant-1" {
			t.Errorf("expected tenant-1, got %s", r.TenantID)
		}
		if r.Report == nil {
			t.Error("report should not be nil")
		}
	}
}

func TestScheduler_EmailDelivery(t *testing.T) {
	emailer := &mockEmailer{}
	cfg := ScheduleConfig{
		Interval:    7 * 24 * time.Hour,
		ReportTypes: []ReportType{ReportSOC2},
		TenantIDs:   []string{"t1"},
		Recipients:  []string{"audit@example.com", "sec@example.com"},
	}
	s := NewScheduler(cfg, &schedulerMockQuery{}, emailer)
	s.GenerateNow(context.Background())

	emailer.mu.Lock()
	defer emailer.mu.Unlock()
	if emailer.sent != 1 {
		t.Errorf("expected 1 email sent, got %d", emailer.sent)
	}
	if len(emailer.lastTo) != 2 {
		t.Errorf("expected 2 recipients, got %d", len(emailer.lastTo))
	}
}

func TestScheduler_GetReport(t *testing.T) {
	cfg := ScheduleConfig{
		ReportTypes: []ReportType{ReportGDPR},
		TenantIDs:   []string{"t1"},
	}
	s := NewScheduler(cfg, &schedulerMockQuery{}, nil)
	s.GenerateNow(context.Background())

	reports := s.ListReports()
	if len(reports) == 0 {
		t.Fatal("expected at least 1 report")
	}

	found := s.GetReport(reports[0].ID)
	if found == nil {
		t.Error("GetReport should find the report by ID")
	}

	missing := s.GetReport("nonexistent")
	if missing != nil {
		t.Error("GetReport should return nil for nonexistent ID")
	}
}

func TestScheduler_MultipleTenants(t *testing.T) {
	cfg := ScheduleConfig{
		ReportTypes: []ReportType{ReportSOC2},
		TenantIDs:   []string{"t1", "t2", "t3"},
	}
	s := NewScheduler(cfg, &schedulerMockQuery{}, nil)
	s.GenerateNow(context.Background())

	reports := s.ListReports()
	if len(reports) != 3 {
		t.Errorf("expected 3 reports (one per tenant), got %d", len(reports))
	}
}

func TestScheduler_StartStop(t *testing.T) {
	cfg := ScheduleConfig{
		Interval:    50 * time.Millisecond,
		ReportTypes: []ReportType{ReportSOC2},
		TenantIDs:   []string{"t1"},
	}
	s := NewScheduler(cfg, &schedulerMockQuery{}, nil)
	s.Start()

	// Wait for at least one tick
	time.Sleep(200 * time.Millisecond)
	s.Stop()

	reports := s.ListReports()
	if len(reports) == 0 {
		t.Error("scheduler should have generated reports after Start+tick")
	}
}

func TestScheduler_DefaultInterval(t *testing.T) {
	cfg := ScheduleConfig{
		ReportTypes: nil,
		TenantIDs:   []string{"t1"},
	}
	s := NewScheduler(cfg, &schedulerMockQuery{}, nil)
	s.Start()
	s.Stop()

	// Should default to weekly without panicking
	if s.cfg.Interval != 7*24*time.Hour {
		t.Errorf("expected default weekly interval, got %v", s.cfg.Interval)
	}
	if len(s.cfg.ReportTypes) != 1 || s.cfg.ReportTypes[0] != ReportSOC2 {
		t.Errorf("expected default ReportSOC2, got %v", s.cfg.ReportTypes)
	}
}

func TestScheduler_NoRecipientsNoEmail(t *testing.T) {
	emailer := &mockEmailer{}
	cfg := ScheduleConfig{
		ReportTypes: []ReportType{ReportSOC2},
		TenantIDs:   []string{"t1"},
		Recipients:  nil, // no recipients
	}
	s := NewScheduler(cfg, &schedulerMockQuery{}, emailer)
	s.GenerateNow(context.Background())

	emailer.mu.Lock()
	defer emailer.mu.Unlock()
	if emailer.sent != 0 {
		t.Error("should not send email when no recipients configured")
	}
}
