package server

import (
	"github.com/ggid/ggid/pkg/audit"
	"github.com/google/uuid"
)

// publishAuditEvent sends an audit event via NATS if publisher is configured.
// Fails silently — audit events are best-effort.
func (h *HTTPHandler) publishAuditEvent(action, result, resourceType string, resourceID uuid.UUID, tenantID uuid.UUID, actorID uuid.UUID) {
	if h.auditPublisher == nil {
		return
	}
	event := audit.NewEvent(action, result, tenantID, actorID)
	event.ResourceType = resourceType
	event.ResourceID = resourceID
	h.auditPublisher.PublishAsync(event)
}
