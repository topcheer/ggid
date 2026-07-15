// Package domain defines the core domain models for the Audit Service.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// ActorType identifies the kind of actor in an audit event.
type ActorType string

const (
	ActorUser      ActorType = "user"
	ActorAPIKey    ActorType = "api_key"
	ActorSystem    ActorType = "system"
	ActorAnonymous ActorType = "anonymous"
)

// EventResult defines the outcome of an audited action.
type EventResult string

const (
	ResultSuccess EventResult = "success"
	ResultFailure EventResult = "failure"
	ResultDenied  EventResult = "denied"
)

// AuditEvent represents a single audit log entry.
type AuditEvent struct {
	ID           uuid.UUID      `json:"id"`
	TenantID     uuid.UUID      `json:"tenant_id"`
	ActorType    ActorType      `json:"actor_type"`
	ActorID      *uuid.UUID     `json:"actor_id,omitempty"`
	ActorName    string         `json:"actor_name,omitempty"`
	Action       string         `json:"action"`                 // e.g. "user.login", "role.assign"
	ResourceType string         `json:"resource_type,omitempty"`
	ResourceID   *uuid.UUID     `json:"resource_id,omitempty"`
	ResourceName string         `json:"resource_name,omitempty"`
	Result       EventResult    `json:"result"`
	IPAddress    string         `json:"ip_address,omitempty"`
	UserAgent    string         `json:"user_agent,omitempty"`
	RequestID    string         `json:"request_id,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	Hash         string         `json:"hash,omitempty"`         // HMAC-SHA256 chain hash for tamper detection
	PrevHash     string         `json:"prev_hash,omitempty"`    // Hash of the previous event in the chain
	CreatedAt    time.Time      `json:"created_at"`
}

// ListFilter holds parameters for querying audit events.
type ListFilter struct {
	TenantID     uuid.UUID
	ActorID      *uuid.UUID
	Action       string
	ResourceType string
	Result       EventResult
	StartTime    *time.Time
	EndTime      *time.Time
	OrderBy      string // created_at | action | actor_name
	Descending   bool
}
