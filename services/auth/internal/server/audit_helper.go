package server

import (
	"net/http"

	"github.com/ggid/ggid/pkg/audit"
	"github.com/ggid/ggid/pkg/tenant"
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

// publishAuditEventWithMeta sends an audit event enriched with resource info
// and metadata. Extracts tenantID from request context, IP from RemoteAddr,
// and user agent from headers.
func (h *Handler) publishAuditEventWithMeta(
	r *http.Request,
	action, result, resourceType, resourceName string,
	resourceID uuid.UUID,
	metadata map[string]any,
) {
	if h.auditPublisher == nil {
		return
	}

	// Extract tenant from context.
	var tenantID uuid.UUID
	if tc, err := tenant.FromContext(r.Context()); err == nil {
		tenantID = tc.TenantID
	}

	event := audit.NewEvent(action, result, tenantID, uuid.Nil)
	event.ResourceType = resourceType
	event.ResourceID = resourceID
	event.ResourceName = resourceName
	event.IPAddress = r.RemoteAddr
	event.UserAgent = r.UserAgent()
	event.RequestID = r.Header.Get("X-Request-ID")
	event.Metadata = metadata

	h.auditPublisher.PublishAsync(event)
}
