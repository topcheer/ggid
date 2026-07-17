package detection

import (
	"context"
	"testing"

	"github.com/ggid/ggid/services/audit/internal/domain"
	"github.com/google/uuid"
)

// fakeThreatChecker implements ThreatIntelChecker for testing.
type fakeThreatChecker struct {
	indicators map[string]*ThreatIntelHit // key: "type:value"
}

func (f *fakeThreatChecker) CheckIndicator(ctx context.Context, tenantID uuid.UUID, indType, value string) (*ThreatIntelHit, error) {
	key := indType + ":" + value
	if hit, ok := f.indicators[key]; ok {
		return hit, nil
	}
	return nil, nil
}

func TestThreatIntelRule_IPMatch(t *testing.T) {
	checker := &fakeThreatChecker{
		indicators: map[string]*ThreatIntelHit{
			"ip:203.0.113.50": {
				IndicatorType:  "ip",
				IndicatorValue: "203.0.113.50",
				Severity:       "high",
				Confidence:     90,
				SourceID:       uuid.New(),
			},
		},
	}
	rule := NewThreatIntelRule(checker)

	evt := &domain.AuditEvent{
		ID:        uuid.New(),
		TenantID:  uuid.New(),
		Action:    "user.login",
		IPAddress: "203.0.113.50",
	}

	det, err := rule.Evaluate(context.Background(), evt, nil, domain.RuleConfig{Enabled: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if det == nil {
		t.Fatal("expected detection for known-bad IP, got nil")
	}
	if det.RuleID != "threat_intel_hit" {
		t.Fatalf("expected rule_id threat_intel_hit, got %s", det.RuleID)
	}
	if det.Severity != domain.Severity("high") {
		t.Fatalf("expected severity high, got %s", det.Severity)
	}
	if det.Detail["matched_on"] != "ip" {
		t.Fatalf("expected matched_on=ip, got %v", det.Detail["matched_on"])
	}
	if det.Detail["confidence"].(int) != 90 {
		t.Fatalf("expected confidence 90, got %v", det.Detail["confidence"])
	}
}

func TestThreatIntelRule_NoMatch(t *testing.T) {
	checker := &fakeThreatChecker{
		indicators: map[string]*ThreatIntelHit{},
	}
	rule := NewThreatIntelRule(checker)

	evt := &domain.AuditEvent{
		ID:        uuid.New(),
		TenantID:  uuid.New(),
		Action:    "user.login",
		IPAddress: "192.168.1.1",
	}

	det, err := rule.Evaluate(context.Background(), evt, nil, domain.RuleConfig{Enabled: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if det != nil {
		t.Fatalf("expected nil detection for clean IP, got %+v", det)
	}
}

func TestThreatIntelRule_EmailMatch(t *testing.T) {
	checker := &fakeThreatChecker{
		indicators: map[string]*ThreatIntelHit{
			"email:bad@evil.com": {
				IndicatorType:  "email",
				IndicatorValue: "bad@evil.com",
				Severity:       "critical",
				Confidence:     95,
				SourceID:       uuid.New(),
			},
		},
	}
	rule := NewThreatIntelRule(checker)

	actorID := uuid.New()
	evt := &domain.AuditEvent{
		ID:        uuid.New(),
		TenantID:  uuid.New(),
		Action:    "user.login",
		ActorName: "bad@evil.com",
		ActorID:   &actorID,
	}

	det, err := rule.Evaluate(context.Background(), evt, nil, domain.RuleConfig{Enabled: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if det == nil {
		t.Fatal("expected detection for known-bad email, got nil")
	}
	if det.Severity != domain.SeverityCritical {
		t.Fatalf("expected severity critical, got %s", det.Severity)
	}
	if det.Detail["recommendation"] != "block_session" {
		t.Fatalf("expected recommendation block_session for critical, got %v", det.Detail["recommendation"])
	}
}
