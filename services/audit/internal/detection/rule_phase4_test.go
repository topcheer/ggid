package detection

import (
	"context"
	"testing"
	"time"

	"github.com/ggid/ggid/services/audit/internal/domain"
	"github.com/google/uuid"
)

// --- OffHoursAdminRule tests ---

func TestOffHoursAdmin_Detects(t *testing.T) {
	rule := &OffHoursAdminRule{}
	tenantID := uuid.New()
	actorID := uuid.New()

	evt := &domain.AuditEvent{
		Action:    "role.assign",
		Result:    "success",
		TenantID:  tenantID,
		ActorID:   &actorID,
		CreatedAt: time.Date(2025, 1, 1, 23, 0, 0, 0, time.UTC), // 23:00 = off-hours
		IPAddress: "10.0.0.1",
	}
	det, err := rule.Evaluate(context.Background(), evt, &MemStateStore{}, domain.RuleConfig{Enabled: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if det == nil {
		t.Fatal("expected detection for off-hours admin action")
	}
	if det.Severity != domain.SeverityMedium {
		t.Errorf("expected medium severity, got %s", det.Severity)
	}
}

func TestOffHoursAdmin_NoDetectDuringBusinessHours(t *testing.T) {
	rule := &OffHoursAdminRule{}
	actorID := uuid.New()

	evt := &domain.AuditEvent{
		Action:    "role.assign",
		Result:    "success",
		TenantID:  uuid.New(),
		ActorID:   &actorID,
		CreatedAt: time.Date(2025, 1, 1, 14, 0, 0, 0, time.UTC), // 14:00 = business hours
	}
	det, _ := rule.Evaluate(context.Background(), evt, &MemStateStore{}, domain.RuleConfig{Enabled: true})
	if det != nil {
		t.Fatal("should not detect during business hours")
	}
}

func TestOffHoursAdmin_IgnoresFailedEvents(t *testing.T) {
	rule := &OffHoursAdminRule{}
	actorID := uuid.New()

	evt := &domain.AuditEvent{
		Action:    "role.assign",
		Result:    "failure",
		ActorID:   &actorID,
		CreatedAt: time.Date(2025, 1, 1, 23, 0, 0, 0, time.UTC),
	}
	det, _ := rule.Evaluate(context.Background(), evt, &MemStateStore{}, domain.RuleConfig{Enabled: true})
	if det != nil {
		t.Fatal("should not detect failed events")
	}
}

// --- NewDevicePrivilegedRule tests ---

func TestNewDevicePrivileged_DetectsFirstUse(t *testing.T) {
	rule := &NewDevicePrivilegedRule{}
	actorID := uuid.New()

	evt := &domain.AuditEvent{
		Action:    "policy.update",
		Result:    "success",
		TenantID:  uuid.New(),
		ActorID:   &actorID,
		IPAddress: "192.168.1.100",
		CreatedAt: time.Now(),
		ID:        uuid.New(),
	}
	state := NewMemStateStore()
	det, err := rule.Evaluate(context.Background(), evt, state, domain.RuleConfig{Enabled: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if det == nil {
		t.Fatal("expected detection for new device privileged action")
	}
	if det.Severity != domain.SeverityHigh {
		t.Errorf("expected high severity, got %s", det.Severity)
	}
}

func TestNewDevicePrivileged_NoDetectOnKnownDevice(t *testing.T) {
	rule := &NewDevicePrivilegedRule{}
	actorID := uuid.New()
	state := NewMemStateStore()

	// Pre-populate the device as seen.
	key := "dev:" + actorID.String() + ":192.168.1.100"
	_ = state.AddEvent(context.Background(), key, time.Now().Unix()-60, "prev", 24*time.Hour)

	evt := &domain.AuditEvent{
		Action:    "policy.update",
		Result:    "success",
		ActorID:   &actorID,
		IPAddress: "192.168.1.100",
		CreatedAt: time.Now(),
		ID:        uuid.New(),
	}
	det, _ := rule.Evaluate(context.Background(), evt, state, domain.RuleConfig{Enabled: true})
	if det != nil {
		t.Fatal("should not detect for known device")
	}
}

func TestNewDevicePrivileged_IgnoresEmptyIP(t *testing.T) {
	rule := &NewDevicePrivilegedRule{}
	actorID := uuid.New()

	evt := &domain.AuditEvent{
		Action:    "policy.update",
		Result:    "success",
		ActorID:   &actorID,
		IPAddress: "", // no IP
		CreatedAt: time.Now(),
	}
	det, _ := rule.Evaluate(context.Background(), evt, &MemStateStore{}, domain.RuleConfig{Enabled: true})
	if det != nil {
		t.Fatal("should not detect with empty IP")
	}
}

// --- TokenReplayRule tests ---

func TestTokenReplay_DetectsAfterThreshold(t *testing.T) {
	rule := &TokenReplayRule{}
	actorID := uuid.New()
	state := NewMemStateStore()

	// Simulate 3 denied requests with revocation reason (threshold=3).
	for i := 0; i < 3; i++ {
		evt := &domain.AuditEvent{
			Action:    "api.request",
			Result:    "denied",
			TenantID:  uuid.New(),
			ActorID:   &actorID,
			IPAddress: "10.0.0.5",
			CreatedAt: time.Now(),
			Metadata:  map[string]any{"cae_reason": "session revoked (CAE)"},
		}
		det, err := rule.Evaluate(context.Background(), evt, state, domain.RuleConfig{Enabled: true})
		if err != nil {
			t.Fatalf("unexpected error on iteration %d: %v", i, err)
		}
		if i < 2 && det != nil {
			t.Fatalf("should not detect before threshold (iteration %d)", i)
		}
		if i == 2 && det == nil {
			t.Fatal("should detect at threshold (iteration 2)")
		}
	}
}

func TestTokenReplay_NoDetectWithoutCAEReason(t *testing.T) {
	rule := &TokenReplayRule{}
	actorID := uuid.New()

	evt := &domain.AuditEvent{
		Action:    "api.request",
		Result:    "denied",
		ActorID:   &actorID,
		CreatedAt: time.Now(),
		Metadata:  map[string]any{"other": "something"}, // no cae_reason
	}
	det, _ := rule.Evaluate(context.Background(), evt, &MemStateStore{}, domain.RuleConfig{Enabled: true})
	if det != nil {
		t.Fatal("should not detect without CAE reason in metadata")
	}
}

func TestTokenReplay_IgnoresNonDeniedResults(t *testing.T) {
	rule := &TokenReplayRule{}
	actorID := uuid.New()

	evt := &domain.AuditEvent{
		Action:    "api.request",
		Result:    "success", // not denied
		ActorID:   &actorID,
		CreatedAt: time.Now(),
		Metadata:  map[string]any{"cae_reason": "session revoked"},
	}
	det, _ := rule.Evaluate(context.Background(), evt, &MemStateStore{}, domain.RuleConfig{Enabled: true})
	if det != nil {
		t.Fatal("should not detect non-denied results")
	}
}
