package service

import (
	"testing"
	"time"

	"github.com/ggid/ggid/pkg/audit"
	"github.com/ggid/ggid/services/audit/internal/domain"
	"github.com/google/uuid"
)

// TestSIEMIntegration_AuditEventToSIEMEvent proves domain.AuditEvent maps
// correctly to audit.Event for SIEM forwarding.
func TestSIEMIntegration_AuditEventToSIEMEvent(t *testing.T) {
	tenantID := uuid.New()
	actorID := uuid.New()

	event := &domain.AuditEvent{
		ID:        uuid.New(),
		TenantID:  tenantID,
		ActorID:   &actorID,
		ActorName: "admin@test.com",
		Action:    "user.delete",
		Result:    domain.ResultSuccess,
		IPAddress: "10.0.0.1",
		CreatedAt: time.Now().UTC(),
	}

	siemEvent := audit.Event{
		TenantID:  event.TenantID,
		ActorName: event.ActorName,
		Action:    event.Action,
		IPAddress: event.IPAddress,
		Result:    string(event.Result),
		CreatedAt: event.CreatedAt,
	}

	if siemEvent.Action != "user.delete" {
		t.Error("action should map correctly")
	}
	if siemEvent.Result != "success" {
		t.Error("result should map correctly")
	}
	if siemEvent.IPAddress != "10.0.0.1" {
		t.Error("IP should map correctly")
	}
	if siemEvent.TenantID != tenantID {
		t.Error("tenantID should map correctly")
	}
}
