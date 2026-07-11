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
	ID           uuid.UUID
	TenantID     uuid.UUID
	ActorType    ActorType
	ActorID      *uuid.UUID
	ActorName    string
	Action       string // e.g. "user.login", "role.assign"
	ResourceType string
	ResourceID   *uuid.UUID
	ResourceName string
	Result       EventResult
	IPAddress    string
	UserAgent    string
	RequestID    string
	Metadata     map[string]any
	Hash         string // HMAC-SHA256 chain hash for tamper detection
	PrevHash     string // Hash of the previous event in the chain
	CreatedAt    time.Time
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
