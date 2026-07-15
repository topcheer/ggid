package httpserver

import (
	"github.com/ggid/ggid/pkg/audit"
	"github.com/google/uuid"
)

// publishAuditEvent sends an audit event via NATS if publisher is configured.
// Fails silently — audit events are best-effort.
func (s *HTTPServer) publishAuditEvent(action, result, resourceType string, resourceID, tenantID uuid.UUID) {
	if s.auditPublisher == nil {
		return
	}
	event := audit.NewEvent(action, result, tenantID, uuid.Nil)
	event.ResourceType = resourceType
	event.ResourceID = resourceID
	s.auditPublisher.PublishAsync(event)
}
