package service

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ggid/ggid/services/audit/internal/domain"
	"github.com/google/uuid"
)

func makeTestEvent(tenantID uuid.UUID, action string) domain.AuditEvent {
	return domain.AuditEvent{
		ID:           uuid.New(),
		TenantID:     tenantID,
		ActorType:    domain.ActorUser,
		ActorName:    "john.doe@example.com",
		Action:       action,
		ResourceType: "document",
		ResourceName: "secret-doc",
		Result:       domain.ResultSuccess,
		IPAddress:    "192.168.1.100",
		UserAgent:    "Mozilla/5.0",
		RequestID:    "req-123",
		CreatedAt:    time.Now(),
	}
}

func TestExportService_ExportCSV(t *testing.T) {
	tenantID := uuid.New()
	events := []domain.AuditEvent{
		makeTestEvent(tenantID, "user.login"),
		makeTestEvent(tenantID, "user.logout"),
	}
	svc := NewExportService("", 1000)

	result, err := svc.ExportEvents(context.Background(), events, ExportFilter{TenantID: tenantID}, ExportCSV)
	if err != nil {
		t.Fatalf("ExportEvents CSV: %v", err)
	}
	if result.RecordCount != 2 {
		t.Errorf("expected 2 records, got %d", result.RecordCount)
	}
	if result.FilePath == "" {
		t.Error("file path should not be empty")
	}
	if result.DownloadURL == "" {
		t.Error("download URL should not be empty")
	}
	defer os.Remove(result.FilePath)
}

func TestExportService_ExportJSON(t *testing.T) {
	tenantID := uuid.New()
	events := []domain.AuditEvent{
		makeTestEvent(tenantID, "user.login"),
		makeTestEvent(tenantID, "user.logout"),
		makeTestEvent(tenantID, "role.assign"),
	}
	svc := NewExportService("", 1000)

	result, err := svc.ExportEvents(context.Background(), events, ExportFilter{TenantID: tenantID}, ExportJSON)
	if err != nil {
		t.Fatalf("ExportEvents JSON: %v", err)
	}
	if result.RecordCount != 3 {
		t.Errorf("expected 3 records, got %d", result.RecordCount)
	}
	defer os.Remove(result.FilePath)
}

func TestExportService_ExportWithPIIMasking(t *testing.T) {
	tenantID := uuid.New()
	events := []domain.AuditEvent{
		makeTestEvent(tenantID, "user.login"),
	}
	svc := NewExportService("", 1000)

	result, err := svc.ExportEvents(context.Background(), events, ExportFilter{TenantID: tenantID, MaskPII: true}, ExportCSV)
	if err != nil {
		t.Fatalf("ExportEvents: %v", err)
	}
	defer os.Remove(result.FilePath)

	// Read the file and verify PII is masked.
	content, err := os.ReadFile(result.FilePath)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if strings.Contains(string(content), "john.doe@example.com") {
		t.Error("PII should be masked in export")
	}
	if !strings.Contains(string(content), "jo*") {
		t.Error("masked name should contain 'jo*' prefix")
	}
	if strings.Contains(string(content), "192.168.1.100") {
		t.Error("IP should be masked")
	}
	if !strings.Contains(string(content), "192.168.*.*") {
		t.Error("masked IP should be 192.168.*.*")
	}
}

func TestExportService_ExportWithMaxRecords(t *testing.T) {
	tenantID := uuid.New()
	events := make([]domain.AuditEvent, 10)
	for i := range events {
		events[i] = makeTestEvent(tenantID, "user.login")
	}
	svc := NewExportService("", 10000)

	result, err := svc.ExportEvents(context.Background(), events, ExportFilter{TenantID: tenantID, MaxRecords: 3}, ExportCSV)
	if err != nil {
		t.Fatalf("ExportEvents: %v", err)
	}
	if result.RecordCount != 3 {
		t.Errorf("expected 3 records, got %d", result.RecordCount)
	}
	if !result.Truncated {
		t.Error("result should be truncated")
	}
	defer os.Remove(result.FilePath)
}

func TestExportService_ExportFilterByAction(t *testing.T) {
	tenantID := uuid.New()
	events := []domain.AuditEvent{
		makeTestEvent(tenantID, "user.login"),
		makeTestEvent(tenantID, "user.logout"),
		makeTestEvent(tenantID, "user.login"),
	}
	svc := NewExportService("", 1000)

	result, err := svc.ExportEvents(context.Background(), events, ExportFilter{TenantID: tenantID, Action: "user.login"}, ExportJSON)
	if err != nil {
		t.Fatalf("ExportEvents: %v", err)
	}
	if result.RecordCount != 2 {
		t.Errorf("expected 2 filtered records, got %d", result.RecordCount)
	}
	defer os.Remove(result.FilePath)
}

func TestExportService_ExportEmptyEvents(t *testing.T) {
	svc := NewExportService("", 1000)
	result, err := svc.ExportEvents(context.Background(), nil, ExportFilter{TenantID: uuid.New()}, ExportCSV)
	if err != nil {
		t.Fatalf("ExportEvents: %v", err)
	}
	if result.RecordCount != 0 {
		t.Errorf("expected 0 records, got %d", result.RecordCount)
	}
}

func TestExportService_ExportNilTenant(t *testing.T) {
	svc := NewExportService("", 1000)
	_, err := svc.ExportEvents(context.Background(), nil, ExportFilter{}, ExportCSV)
	if err == nil {
		t.Error("should error on nil tenant")
	}
}

func TestExportService_ExportLog(t *testing.T) {
	tenantID := uuid.New()
	events := []domain.AuditEvent{makeTestEvent(tenantID, "user.login")}
	svc := NewExportService("", 1000)

	svc.ExportEvents(context.Background(), events, ExportFilter{TenantID: tenantID}, ExportCSV)
	svc.ExportEvents(context.Background(), events, ExportFilter{TenantID: tenantID}, ExportJSON)

	log := svc.GetExportLog()
	if len(log) != 2 {
		t.Errorf("expected 2 log entries, got %d", len(log))
	}
}

func TestExportService_ExportStream(t *testing.T) {
	tenantID := uuid.New()
	events := []domain.AuditEvent{
		makeTestEvent(tenantID, "user.login"),
		makeTestEvent(tenantID, "user.logout"),
	}
	svc := NewExportService("", 1000)

	var buf strings.Builder
	count, err := svc.ExportStream(context.Background(), events, ExportFilter{TenantID: tenantID}, ExportCSV, &buf)
	if err != nil {
		t.Fatalf("ExportStream: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 records, got %d", count)
	}
}

func TestExportService_maskString(t *testing.T) {
	tests := []struct {
		input, expected string
	}{
		{"john", "jo**"},
		{"ab", "**"},
		{"", ""},
		{"x", "*"},
	}
	for _, tt := range tests {
		got := maskString(tt.input)
		if got != tt.expected {
			t.Errorf("maskString(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestExportService_maskIP(t *testing.T) {
	tests := []struct {
		input, expected string
	}{
		{"192.168.1.100", "192.168.*.*"},
		{"10.0.0.1", "10.0.*.*"},
	}
	for _, tt := range tests {
		got := maskIP(tt.input)
		if got != tt.expected {
			t.Errorf("maskIP(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
