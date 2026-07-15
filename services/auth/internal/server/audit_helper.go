package server

import (
	"github.com/ggid/ggid/pkg/audit"
	"github.com/google/uuid"
)

// publishAuditEvent sends an audit event via NATS if publisher is configured.
// Fails silently — audit events are best-effort.
func (h *Handler) publishAuditEvent(action, result string, tenantID, actorID uuid.UUID) {
	if h.auditPublisher == nil {
		return
	}
	event := audit.NewEvent(action, result, tenantID, actorID)
	h.auditPublisher.PublishAsync(event)
}
