// Package compliance generates compliance reports (SOC2, HIPAA, GDPR)
// from audit event data.
package compliance

import (
	"context"
	"fmt"
	"time"
)

// ReportType identifies the compliance framework.
type ReportType string

const (
	ReportSOC2  ReportType = "soc2"
	ReportHIPAA ReportType = "hipaa"
	ReportGDPR  ReportType = "gdpr"
)

// EventQuery can query audit events for report generation.
type EventQuery interface {
	QueryEvents(ctx context.Context, from, to time.Time, actionTypes []string) ([]AuditEvent, error)
}

// AuditEvent represents a single audit event for compliance reporting.
type AuditEvent struct {
	ID         string    `json:"id"`
	TenantID   string    `json:"tenant_id"`
	UserID     string    `json:"user_id"`
	Action     string    `json:"action"`
	Resource   string    `json:"resource"`
	IPAddress  string    `json:"ip_address"`
	Timestamp  time.Time `json:"timestamp"`
	Success    bool      `json:"success"`
}

// ComplianceReport is the generated report.
type ComplianceReport struct {
	Type        ReportType     `json:"type"`
	Period      string         `json:"period"`
	GeneratedAt time.Time      `json:"generated_at"`
	Summary     ReportSummary  `json:"summary"`
	Sections    []ReportSection `json:"sections"`
}

// ReportSummary contains high-level metrics.
type ReportSummary struct {
	TotalEvents     int `json:"total_events"`
	FailedLogins    int `json:"failed_logins"`
	PrivilegedAccess int `json:"privileged_access"`
	DataAccessed    int `json:"data_accessed"`
	PolicyChanges   int `json:"policy_changes"`
}

// ReportSection groups related findings.
type ReportSection struct {
	Title    string       `json:"title"`
	Criteria string       `json:"criteria"`
	Status   string       `json:"status"` // "pass", "warning", "fail"
	Details  []AuditEvent `json:"details,omitempty"`
}

// Generator creates compliance reports from audit events.
type Generator struct {
	query EventQuery
}

// NewGenerator creates a compliance report generator.
func NewGenerator(query EventQuery) *Generator {
	return &Generator{query: query}
}

// Generate creates a compliance report for the given period.
func (g *Generator) Generate(ctx context.Context, reportType ReportType, from, to time.Time) (*ComplianceReport, error) {
	var actionTypes []string

	switch reportType {
	case ReportSOC2:
		actionTypes = []string{"login.success", "login.failed", "user.create", "user.delete", "role.assign", "policy.change"}
	case ReportHIPAA:
		actionTypes = []string{"login.success", "login.failed", "phi.access", "phi.export", "user.create", "audit.export"}
	case ReportGDPR:
		actionTypes = []string{"login.success", "data.access", "data.export", "data.delete", "consent.change", "user.delete"}
	default:
		return nil, fmt.Errorf("unsupported report type: %s", reportType)
	}

	events, err := g.query.QueryEvents(ctx, from, to, actionTypes)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}

	report := &ComplianceReport{
		Type:        reportType,
		Period:      fmt.Sprintf("%s to %s", from.Format(time.RFC3339), to.Format(time.RFC3339)),
		GeneratedAt: time.Now(),
	}

	// Build summary
	for _, e := range events {
		report.Summary.TotalEvents++
		if !e.Success {
			report.Summary.FailedLogins++
		}
		switch e.Action {
		case "role.assign":
			report.Summary.PrivilegedAccess++
		case "policy.change":
			report.Summary.PrivilegedAccess++
			report.Summary.PolicyChanges++
		case "phi.access", "data.access", "data.export":
			report.Summary.DataAccessed++
		}
	}

	// Build sections based on report type
	switch reportType {
	case ReportSOC2:
		report.Sections = g.buildSOC2Sections(events)
	case ReportHIPAA:
		report.Sections = g.buildHIPAASections(events)
	case ReportGDPR:
		report.Sections = g.buildGDPRSections(events)
	}

	return report, nil
}

func (g *Generator) buildSOC2Sections(events []AuditEvent) []ReportSection {
	return []ReportSection{
		{
			Title:    "Access Control (CC6.1)",
			Criteria: "Logical and physical access controls are implemented",
			Status:   statusFromEvents(events, "login.failed"),
			Details:  filterEvents(events, "login.failed"),
		},
		{
			Title:    "Change Management (CC8.1)",
			Criteria: "Changes are authorized, documented, and tested",
			Status:   statusFromEvents(events, "policy.change"),
			Details:  filterEvents(events, "policy.change"),
		},
	}
}

func (g *Generator) buildHIPAASections(events []AuditEvent) []ReportSection {
	return []ReportSection{
		{
			Title:    "Access Audit (§164.312(b))",
			Criteria: "Audit controls record and examine activity",
			Status:   statusFromEvents(events, "phi.access"),
			Details:  filterEvents(events, "phi.access"),
		},
		{
			Title:    "Access Control (§164.312(a))",
			Criteria: "Unique user identification and emergency access",
			Status:   statusFromEvents(events, "login.success"),
		},
	}
}

func (g *Generator) buildGDPRSections(events []AuditEvent) []ReportSection {
	return []ReportSection{
		{
			Title:    "Data Access Log (Art. 30)",
			Criteria: "Records of processing activities maintained",
			Status:   statusFromEvents(events, "data.access"),
			Details:  filterEvents(events, "data.access"),
		},
		{
			Title:    "Right to Erasure (Art. 17)",
			Criteria: "Data deletion requests tracked",
			Status:   statusFromEvents(events, "data.delete"),
			Details:  filterEvents(events, "data.delete"),
		},
	}
}

func filterEvents(events []AuditEvent, action string) []AuditEvent {
	var result []AuditEvent
	for _, e := range events {
		if e.Action == action {
			result = append(result, e)
		}
	}
	return result
}

func statusFromEvents(events []AuditEvent, action string) string {
	for _, e := range events {
		if e.Action == action && !e.Success {
			return "warning"
		}
	}
	return "pass"
}
