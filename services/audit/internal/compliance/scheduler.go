// Package compliance provides cron-based scheduling for automatic compliance
// report generation and delivery.
package compliance

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// ScheduleConfig configures the compliance report scheduler.
type ScheduleConfig struct {
	Interval    time.Duration // how often to run (e.g. 7*24h for weekly)
	ReportTypes []ReportType  // which report types to generate
	Recipients  []string      // email addresses to deliver to
	TenantIDs   []string      // tenants to generate reports for
}

// ScheduledReport is a generated report stored by the scheduler.
type ScheduledReport struct {
	ID        string        `json:"id"`
	Type      ReportType    `json:"type"`
	TenantID  string        `json:"tenant_id"`
	Period    string        `json:"period"`
	GeneratedAt time.Time   `json:"generated_at"`
	Report    *ComplianceReport `json:"report"`
}

// EmailSender sends compliance reports via email.
type EmailSender interface {
	Send(to []string, subject string, body []byte) error
}

// Scheduler generates compliance reports on a schedule and stores/delivers them.
type Scheduler struct {
	cfg      ScheduleConfig
	query    EventQuery
	emailer  EmailSender
	mu       sync.RWMutex
	reports  []*ScheduledReport
	stopCh   chan struct{}
	ticker   *time.Ticker
}

// NewScheduler creates a new compliance report scheduler.
func NewScheduler(cfg ScheduleConfig, query EventQuery, emailer EmailSender) *Scheduler {
	return &Scheduler{
		cfg:     cfg,
		query:   query,
		emailer: emailer,
		stopCh:  make(chan struct{}),
	}
}

// Start begins the scheduling loop in a goroutine.
func (s *Scheduler) Start() {
	if s.cfg.Interval <= 0 {
		s.cfg.Interval = 7 * 24 * time.Hour // default weekly
	}
	if len(s.cfg.ReportTypes) == 0 {
		s.cfg.ReportTypes = []ReportType{ReportSOC2}
	}
	s.ticker = time.NewTicker(s.cfg.Interval)
	go func() {
		for {
			select {
			case <-s.ticker.C:
				s.GenerateAll(context.Background())
			case <-s.stopCh:
				return
			}
		}
	}()
	log.Printf("Compliance scheduler started: interval=%v types=%v", s.cfg.Interval, s.cfg.ReportTypes)
}

// Stop halts the scheduler.
func (s *Scheduler) Stop() {
	if s.ticker != nil {
		s.ticker.Stop()
	}
	close(s.stopCh)
}

// GenerateAll generates reports for all configured tenants and types.
func (s *Scheduler) GenerateAll(ctx context.Context) {
	now := time.Now().UTC()
	from := now.Add(-s.cfg.Interval)

	gen := NewGenerator(s.query)

	for _, tenantID := range s.cfg.TenantIDs {
		for _, rt := range s.cfg.ReportTypes {
			report, err := gen.Generate(ctx, rt, from, now)
			if err != nil {
				log.Printf("Compliance scheduler: failed to generate %s for tenant %s: %v", rt, tenantID, err)
				continue
			}

			sr := &ScheduledReport{
				ID:        fmt.Sprintf("%s-%s-%d", tenantID, rt, now.Unix()),
				Type:      rt,
				TenantID:  tenantID,
				Period:    fmt.Sprintf("%s to %s", from.Format("2006-01-02"), now.Format("2006-01-02")),
				GeneratedAt: now,
				Report:    report,
			}

			s.mu.Lock()
			s.reports = append(s.reports, sr)
			s.mu.Unlock()

			// Email delivery
			if s.emailer != nil && len(s.cfg.Recipients) > 0 {
				subject := fmt.Sprintf("Compliance Report: %s (%s)", rt, sr.Period)
				body := []byte(fmt.Sprintf("Report ID: %s\nTenant: %s\nPeriod: %s\nSummary: %d total events, %d failed logins\n",
				sr.ID, tenantID, sr.Period, report.Summary.TotalEvents, report.Summary.FailedLogins))
				if err := s.emailer.Send(s.cfg.Recipients, subject, body); err != nil {
					log.Printf("Compliance scheduler: email delivery failed: %v", err)
				}
			}

			log.Printf("Compliance scheduler: generated %s report for tenant %s", rt, tenantID)
		}
	}
}

// ListReports returns all generated scheduled reports.
func (s *Scheduler) ListReports() []*ScheduledReport {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*ScheduledReport, len(s.reports))
	copy(result, s.reports)
	return result
}

// GetReport returns a specific report by ID.
func (s *Scheduler) GetReport(id string) *ScheduledReport {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, r := range s.reports {
		if r.ID == id {
			return r
		}
	}
	return nil
}

// GenerateNow generates reports immediately without waiting for the ticker.
func (s *Scheduler) GenerateNow(ctx context.Context) {
	s.GenerateAll(ctx)
}
