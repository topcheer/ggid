package domain

import (
	"time"

	"github.com/google/uuid"
)

// Severity levels for ITDR detections.
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
)

// DetectionStatus represents the lifecycle state of a detection.
type DetectionStatus string

const (
	DetectionNew            DetectionStatus = "new"
	DetectionAcknowledged   DetectionStatus = "acknowledged"
	DetectionResolved       DetectionStatus = "resolved"
	DetectionFalsePositive  DetectionStatus = "false_positive"
)

// Detection represents a single threat detection result.
type Detection struct {
	ID         uuid.UUID      `json:"id"`
	TenantID   uuid.UUID      `json:"tenant_id"`
	RuleID     string         `json:"rule_id"`
	ActorID    *uuid.UUID     `json:"actor_id,omitempty"`
	Severity   Severity       `json:"severity"`
	Title      string         `json:"title"`
	Detail     map[string]any `json:"detail"`
	EventIDs   []uuid.UUID    `json:"event_ids"`
	Status     DetectionStatus `json:"status"`
	HitCount   int            `json:"hit_count"`
	DetectedAt time.Time      `json:"detected_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

// RuleConfig represents per-tenant configuration for a detection rule.
type RuleConfig struct {
	RuleID    string         `json:"rule_id"`
	Enabled   bool           `json:"enabled"`
	Severity  Severity       `json:"severity,omitempty"`
	Threshold map[string]any `json:"threshold,omitempty"`
}

// DetectionFilter holds query parameters for listing detections.
type DetectionFilter struct {
	TenantID  uuid.UUID
	Severity  *Severity
	Status    *DetectionStatus
	RuleID    *string
	ActorID   *uuid.UUID
	Since     *time.Time
	Page      int
	PageSize  int
}

// DetectionStats holds aggregated detection counts.
type DetectionStats struct {
	Total      int                       `json:"total"`
	BySeverity map[string]int            `json:"by_severity"`
	ByStatus   map[string]int            `json:"by_status"`
	ByRule     map[string]int            `json:"by_rule"`
}

// NewDetection creates a Detection with sensible defaults.
func NewDetection(tenantID uuid.UUID, ruleID string, severity Severity, title string) *Detection {
	now := time.Now().UTC()
	return &Detection{
		ID:         uuid.New(),
		TenantID:   tenantID,
		RuleID:     ruleID,
		Severity:   severity,
		Title:      title,
		Detail:     map[string]any{},
		EventIDs:   []uuid.UUID{},
		Status:     DetectionNew,
		DetectedAt: now,
		UpdatedAt:  now,
	}
}
