package domain

import (
	"time"

	"github.com/google/uuid"
)

// AccessRequestStatus represents the lifecycle state of an access request.
type AccessRequestStatus string

const (
	AccessRequestPending  AccessRequestStatus = "pending"
	AccessRequestApproved AccessRequestStatus = "approved"
	AccessRequestDenied   AccessRequestStatus = "denied"
	AccessRequestExpired  AccessRequestStatus = "expired"
)

// ResourceType enumerates the kinds of resources that can be requested.
type ResourceType string

const (
	ResourceTypeRole       ResourceType = "role"
	ResourceTypeGroup      ResourceType = "group"
	ResourceTypePermission ResourceType = "permission"
)

// AccessRequest represents a governance workflow request for additional access.
// Users submit requests; approvers review and approve/deny them.
// Requests auto-expire after ExpiresAt.
type AccessRequest struct {
	ID           uuid.UUID          `json:"id"`
	TenantID     uuid.UUID          `json:"tenant_id"`
	RequesterID  uuid.UUID          `json:"requester_id"`
	ResourceType ResourceType       `json:"resource_type"`
	ResourceID   string             `json:"resource_id"`
	Reason       string             `json:"reason"`
	Status       AccessRequestStatus `json:"status"`
	ApproverID   *uuid.UUID         `json:"approver_id,omitempty"`
	DenialReason string             `json:"denial_reason,omitempty"`
	CreatedAt    time.Time          `json:"created_at"`
	ResolvedAt   *time.Time         `json:"resolved_at,omitempty"`
	ExpiresAt    time.Time          `json:"expires_at"`
}

// IsValid validates the AccessRequest fields.
func (ar *AccessRequest) IsValid() bool {
	if ar.TenantID == uuid.Nil {
		return false
	}
	if ar.RequesterID == uuid.Nil {
		return false
	}
	if ar.ResourceType != ResourceTypeRole &&
		ar.ResourceType != ResourceTypeGroup &&
		ar.ResourceType != ResourceTypePermission {
		return false
	}
	if ar.ResourceID == "" {
		return false
	}
	if ar.ExpiresAt.IsZero() {
		return false
	}
	return true
}

// IsExpired returns true if the request has passed its expiry time and is still pending.
func (ar *AccessRequest) IsExpired() bool {
	return ar.Status == AccessRequestPending && time.Now().After(ar.ExpiresAt)
}
