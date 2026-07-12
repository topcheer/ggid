package service

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ggid/ggid/services/audit/internal/domain"
	"github.com/google/uuid"
)

// ExportFormat is the output format for audit event exports.
type ExportFormat string

const (
	ExportCSV     ExportFormat = "csv"
	ExportJSON    ExportFormat = "json"
	ExportParquet ExportFormat = "parquet"
)

// ExportFilter holds the filter criteria for selecting events to export.
type ExportFilter struct {
	TenantID     uuid.UUID
	ActorID      *uuid.UUID
	Action       string
	ResourceType string
	StartTime    *time.Time
	EndTime      *time.Time
	MaxRecords   int
	MaskPII      bool
}

// ExportResult is the result of an export operation.
type ExportResult struct {
	Format      ExportFormat `json:"format"`
	RecordCount int         `json:"record_count"`
	FilePath    string      `json:"file_path"`
	DownloadURL string      `json:"download_url"`
	ExportedAt  time.Time   `json:"exported_at"`
	FileSize    int64       `json:"file_size"`
	Truncated   bool        `json:"truncated"` // true if MaxRecords limit was hit
}

// ExportAuditEntry tracks the audit trail of export operations themselves.
type ExportAuditEntry struct {
	ExportID    uuid.UUID
	RequestedBy uuid.UUID
	Filter      ExportFilter
	Result      *ExportResult
	CreatedAt   time.Time
}

// ExportService handles exporting audit events to various formats.
type ExportService struct {
	mu           sync.RWMutex
	exportLog    []ExportAuditEntry
	maxRecords   int
	outputDir    string
}

// NewExportService creates a new ExportService.
func NewExportService(outputDir string, maxRecords int) *ExportService {
	if maxRecords <= 0 {
		maxRecords = 100000
	}
	if outputDir == "" {
		outputDir = os.TempDir()
	}
	return &ExportService{
		maxRecords: maxRecords,
		outputDir:  outputDir,
	}
}

// ExportEvents exports audit events matching the filter to the specified format.
// For large datasets, events are streamed to a file and a download URL is returned.
func (s *ExportService) ExportEvents(ctx context.Context, events []domain.AuditEvent, filter ExportFilter, format ExportFormat) (*ExportResult, error) {
	if filter.TenantID == uuid.Nil {
		return nil, fmt.Errorf("tenant_id is required")
	}
	if len(events) == 0 {
		return &ExportResult{
			Format:      format,
			RecordCount: 0,
			ExportedAt:  time.Now(),
		}, nil
	}

	// Apply filter and limit.
	filtered := s.filterEvents(events, filter)
	limit := s.maxRecords
	if filter.MaxRecords > 0 && filter.MaxRecords < limit {
		limit = filter.MaxRecords
	}
	truncated := false
	if len(filtered) > limit {
		filtered = filtered[:limit]
		truncated = true
	}

	// Generate output file.
	filename := fmt.Sprintf("audit_export_%s_%d.%s", filter.TenantID, time.Now().Unix(), format)
	filePath := filepath.Join(s.outputDir, filename)

	file, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("create export file: %w", err)
	}
	defer file.Close()

	var recordCount int
	switch format {
	case ExportCSV:
		recordCount, err = s.exportCSV(file, filtered, filter.MaskPII)
	case ExportJSON:
		recordCount, err = s.exportJSON(file, filtered, filter.MaskPII)
	case ExportParquet:
		// Parquet requires external libraries; fall back to JSON for now.
		recordCount, err = s.exportJSON(file, filtered, filter.MaskPII)
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
	if err != nil {
		return nil, fmt.Errorf("export %s: %w", format, err)
	}

	stat, _ := file.Stat()
	var fileSize int64
	if stat != nil {
		fileSize = stat.Size()
	}

	result := &ExportResult{
		Format:      format,
		RecordCount: recordCount,
		FilePath:    filePath,
		DownloadURL: fmt.Sprintf("/api/v1/audit/export/%s", filename),
		ExportedAt:  time.Now(),
		FileSize:    fileSize,
		Truncated:   truncated,
	}

	// Log the export.
	s.mu.Lock()
	s.exportLog = append(s.exportLog, ExportAuditEntry{
		ExportID:    uuid.New(),
		Filter:      filter,
		Result:      result,
		CreatedAt:   time.Now(),
	})
	s.mu.Unlock()

	return result, nil
}

// ExportStream streams filtered events to the provided writer in the specified format.
func (s *ExportService) ExportStream(ctx context.Context, events []domain.AuditEvent, filter ExportFilter, format ExportFormat, w io.Writer) (int, error) {
	filtered := s.filterEvents(events, filter)
	limit := s.maxRecords
	if filter.MaxRecords > 0 && filter.MaxRecords < limit {
		limit = filter.MaxRecords
	}
	if len(filtered) > limit {
		filtered = filtered[:limit]
	}

	switch format {
	case ExportCSV:
		return s.exportCSV(w, filtered, filter.MaskPII)
	case ExportJSON:
		return s.exportJSON(w, filtered, filter.MaskPII)
	default:
		return s.exportJSON(w, filtered, filter.MaskPII)
	}
}

// GetExportLog returns the audit trail of all export operations.
func (s *ExportService) GetExportLog() []ExportAuditEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]ExportAuditEntry, len(s.exportLog))
	copy(result, s.exportLog)
	return result
}

// filterEvents applies the export filter to the event list.
func (s *ExportService) filterEvents(events []domain.AuditEvent, filter ExportFilter) []domain.AuditEvent {
	var result []domain.AuditEvent
	for _, e := range events {
		if filter.TenantID != uuid.Nil && e.TenantID != filter.TenantID {
			continue
		}
		if filter.ActorID != nil && (e.ActorID == nil || *e.ActorID != *filter.ActorID) {
			continue
		}
		if filter.Action != "" && e.Action != filter.Action {
			continue
		}
		if filter.ResourceType != "" && e.ResourceType != filter.ResourceType {
			continue
		}
		if filter.StartTime != nil && e.CreatedAt.Before(*filter.StartTime) {
			continue
		}
		if filter.EndTime != nil && e.CreatedAt.After(*filter.EndTime) {
			continue
		}
		result = append(result, e)
	}
	return result
}

// exportCSV writes events as CSV to the writer. Returns the number of records written.
func (s *ExportService) exportCSV(w io.Writer, events []domain.AuditEvent, maskPII bool) (int, error) {
	cwt := csv.NewWriter(w)
	defer cwt.Flush()

	header := []string{"id", "tenant_id", "actor_type", "actor_id", "actor_name", "action", "resource_type", "resource_id", "resource_name", "result", "ip_address", "user_agent", "request_id", "created_at"}
	if err := cwt.Write(header); err != nil {
		return 0, err
	}
	count := 0
	for _, e := range events {
		actorID := ""
		if e.ActorID != nil {
			actorID = e.ActorID.String()
		}
		resourceID := ""
		if e.ResourceID != nil {
			resourceID = e.ResourceID.String()
		}
		actorName := e.ActorName
		ipAddress := e.IPAddress
		if maskPII {
			actorName = maskString(actorName)
			ipAddress = maskIP(ipAddress)
		}
		row := []string{
			e.ID.String(),
			e.TenantID.String(),
			string(e.ActorType),
			actorID,
			actorName,
			e.Action,
			e.ResourceType,
			resourceID,
			e.ResourceName,
			string(e.Result),
			ipAddress,
			e.UserAgent,
			e.RequestID,
			e.CreatedAt.Format(time.RFC3339),
		}
		if err := cwt.Write(row); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

// exportJSON writes events as JSON to the writer. Returns the number of records written.
func (s *ExportService) exportJSON(w io.Writer, events []domain.AuditEvent, maskPII bool) (int, error) {
	type exportEvent struct {
		ID           string         `json:"id"`
		TenantID     string         `json:"tenant_id"`
		ActorType    string         `json:"actor_type"`
		ActorID      string         `json:"actor_id,omitempty"`
		ActorName    string         `json:"actor_name,omitempty"`
		Action       string         `json:"action"`
		ResourceType string         `json:"resource_type"`
		ResourceID   string         `json:"resource_id,omitempty"`
		ResourceName string         `json:"resource_name,omitempty"`
		Result       string         `json:"result"`
		IPAddress    string         `json:"ip_address,omitempty"`
		UserAgent    string         `json:"user_agent,omitempty"`
		RequestID    string         `json:"request_id,omitempty"`
		CreatedAt    string         `json:"created_at"`
		Metadata     map[string]any `json:"metadata,omitempty"`
	}

	var records []exportEvent
	for _, e := range events {
		actorID := ""
		if e.ActorID != nil {
			actorID = e.ActorID.String()
		}
		resourceID := ""
		if e.ResourceID != nil {
			resourceID = e.ResourceID.String()
		}
		actorName := e.ActorName
		ipAddress := e.IPAddress
		if maskPII {
			actorName = maskString(actorName)
			ipAddress = maskIP(ipAddress)
		}
		records = append(records, exportEvent{
			ID:           e.ID.String(),
			TenantID:     e.TenantID.String(),
			ActorType:    string(e.ActorType),
			ActorID:      actorID,
			ActorName:    actorName,
			Action:       e.Action,
			ResourceType: e.ResourceType,
			ResourceID:   resourceID,
			ResourceName: e.ResourceName,
			Result:       string(e.Result),
			IPAddress:    ipAddress,
			UserAgent:    e.UserAgent,
			RequestID:    e.RequestID,
			CreatedAt:    e.CreatedAt.Format(time.RFC3339),
			Metadata:     e.Metadata,
		})
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(records); err != nil {
		return 0, err
	}
	return len(records), nil
}

// maskString masks a string for PII protection (keeps first 2 chars, masks rest).
func maskString(s string) string {
	if len(s) <= 2 {
		return strings.Repeat("*", len(s))
	}
	return s[:2] + strings.Repeat("*", len(s)-2)
}

// maskIP masks the last octets of an IP address.
func maskIP(ip string) string {
	parts := strings.Split(ip, ".")
	if len(parts) == 4 {
		return parts[0] + "." + parts[1] + ".*.*"
	}
	if strings.Contains(ip, ":") {
		idx := strings.LastIndex(ip, ":")
		if idx > 0 {
			return ip[:idx] + ":*:*"
		}
	}
	return "***"
}

// Reset clears the export log (for testing).
func (s *ExportService) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.exportLog = nil
}
