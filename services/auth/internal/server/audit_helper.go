package server

import (
	"net/http"
	"strings"

	"github.com/ggid/ggid/pkg/audit"
	"github.com/ggid/ggid/pkg/tenant"
	"github.com/google/uuid"
)

// publishAuditEvent sends an audit event via NATS if publisher is configured.
// Fails silently — audit events are best-effort.
func (h *Handler) publishAuditEvent(action, result string, tenantID, actorID uuid.UUID) {
	h.publishAuditEventFull(nil, action, result, tenantID, actorID, "", "", "")
}

// publishAuditEventWithRequest sends an audit event enriched with request context
// (IP, User-Agent, Request-ID).
func (h *Handler) publishAuditEventWithRequest(
	r *http.Request,
	action, result string,
	tenantID, actorID uuid.UUID,
) {
	ip := clientIPFromRequest(r)
	ua := r.UserAgent()
	reqID := r.Header.Get("X-Request-ID")
	h.publishAuditEventFull(r, action, result, tenantID, actorID, ip, ua, reqID)
}

func (h *Handler) publishAuditEventFull(
	r *http.Request,
	action, result string,
	tenantID, actorID uuid.UUID,
	ip, ua, reqID string,
) {
	if h.auditPublisher == nil {
		return
	}

	// Resolve username from identity service for actor_name
	actorName := ""
	if actorID != uuid.Nil && r != nil {
		if ic := h.authSvc.IdentityClient(); ic != nil {
			if u, err := ic.GetUser(r.Context(), tenantID, actorID); err == nil && u != nil {
				actorName = u.Username
			}
		}
	}

	event := audit.NewEvent(action, result, tenantID, actorID)
	if actorName != "" {
		event.ActorName = actorName
	}
	if ip != "" {
		event.IPAddress = ip
	}
	if ua != "" {
		event.UserAgent = ua
	}
	if reqID != "" {
		event.RequestID = reqID
	}
	h.auditPublisher.PublishAsync(event)
}

// clientIPFromRequest extracts the real client IP from X-Forwarded-For or falls back to RemoteAddr.
func clientIPFromRequest(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the chain
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return strings.TrimSpace(xff[:i])
			}
		}
		return strings.TrimSpace(xff)
	}
	if r.RemoteAddr != "" {
		// Strip port
		if idx := strings.LastIndex(r.RemoteAddr, ":"); idx > 0 {
			return r.RemoteAddr[:idx]
		}
		return r.RemoteAddr
	}
	return ""
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
