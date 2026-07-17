package detection

import (
	"context"
	"testing"
	"time"

	"github.com/ggid/ggid/services/audit/internal/domain"
	"github.com/google/uuid"
)

func TestProfileBuilder_LearnAndRetrieve(t *testing.T) {
	b := NewProfileBuilder()
	tenantID := uuid.New()
	userID := uuid.New()

	for i := 0; i < 10; i++ {
		b.IngestEvent(context.Background(), tenantID, userID, 10, "192.168.1.1", "Chrome/macOS", "user.login")
	}

	p := b.GetProfile(tenantID, userID)
	if p == nil {
		t.Fatal("profile should exist")
	}
	if p.EventCount != 10 {
		t.Errorf("expected 10 events, got %d", p.EventCount)
	}
	if p.LoginHours[10] != 10 {
		t.Errorf("expected 10 events at hour 10, got %f", p.LoginHours[10])
	}
}

func TestProfileBuilder_Normalize(t *testing.T) {
	b := NewProfileBuilder()
	tenantID := uuid.New()
	userID := uuid.New()

	b.IngestEvent(context.Background(), tenantID, userID, 9, "10.0.0.1", "Chrome", "login")
	b.IngestEvent(context.Background(), tenantID, userID, 9, "10.0.0.1", "Chrome", "login")
	b.IngestEvent(context.Background(), tenantID, userID, 14, "10.0.0.2", "Safari", "login")

	p := b.GetProfile(tenantID, userID)
	p.Normalize()

	total := 0.0
	for _, v := range p.LoginHours {
		total += v
	}
	if total < 0.99 || total > 1.01 {
		t.Errorf("normalized login hours should sum to 1, got %f", total)
	}
}

func TestBaselineDeviation_ColdStart(t *testing.T) {
	rule := &BaselineDeviationRule{}
	state := NewMemStateStore()
	actorID := uuid.New()
	evt := &domain.AuditEvent{
		TenantID:  uuid.New(),
		ActorID:   &actorID,
		CreatedAt: time.Now(),
	}
	det, err := rule.Evaluate(context.Background(), evt, state, domain.RuleConfig{Enabled: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if det != nil {
		t.Fatal("should not detect during cold start (no profile)")
	}
}

func TestBaselineDeviation_OffHoursDetected(t *testing.T) {
	rule := &BaselineDeviationRule{}
	state := NewMemStateStore()
	actorID := uuid.New()
	tenantID := uuid.New()

	for i := 0; i < 55; i++ {
		state.AddEvent(context.Background(), "ueba:"+tenantID.String()+":"+actorID.String(), time.Now().Unix(), "evt", 365*24*time.Hour)
	}

	evt := &domain.AuditEvent{
		TenantID:  tenantID,
		ActorID:   &actorID,
		CreatedAt: time.Date(2026, 1, 1, 3, 0, 0, 0, time.UTC),
		IPAddress: "203.0.113.99",
	}
	det, err := rule.Evaluate(context.Background(), evt, state, domain.RuleConfig{Enabled: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if det == nil {
		t.Fatal("should detect off-hours + new IP deviation")
	}
}

func TestBaselineDeviation_BusinessHoursNoDetect(t *testing.T) {
	rule := &BaselineDeviationRule{}
	state := NewMemStateStore()
	actorID := uuid.New()
	tenantID := uuid.New()

	for i := 0; i < 55; i++ {
		state.AddEvent(context.Background(), "ueba:"+tenantID.String()+":"+actorID.String(), time.Now().Unix(), "evt", 365*24*time.Hour)
	}

	state.AddEvent(context.Background(), "ueba_ip:"+tenantID.String()+":"+actorID.String()+":192.168.1.1", time.Now().Add(-1*time.Hour).Unix(), "prev", 30*24*time.Hour)

	evt := &domain.AuditEvent{
		TenantID:  tenantID,
		ActorID:   &actorID,
		CreatedAt: time.Date(2026, 1, 1, 14, 0, 0, 0, time.UTC),
		IPAddress: "192.168.1.1",
	}
	det, _ := rule.Evaluate(context.Background(), evt, state, domain.RuleConfig{Enabled: true})
	if det != nil {
		t.Fatal("should not detect normal business hours + known IP")
	}
}
