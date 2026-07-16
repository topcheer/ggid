package detection

import (
	"context"
	"testing"
	"time"

	"github.com/ggid/ggid/services/audit/internal/domain"
	"github.com/google/uuid"
)

func makeLoginEvent(result domain.EventResult, actorID *uuid.UUID, ip string) *domain.AuditEvent {
	return &domain.AuditEvent{
		ID:        uuid.New(),
		TenantID:  uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		ActorID:   actorID,
		ActorName: "testuser",
		Action:    "user.login",
		Result:    result,
		IPAddress: ip,
		CreatedAt: time.Now().UTC(),
	}
}

func actorPtr() *uuid.UUID {
	id := uuid.New()
	return &id
}

// --- BruteForce tests ---

func TestBruteForce_BelowThreshold(t *testing.T) {
	rule := &BruteForceRule{}
	state := NewMemStateStore()
	actor := actorPtr()

	for i := 0; i < 4; i++ { // 4 failures, threshold is 5
		evt := makeLoginEvent("failure", actor, "1.2.3.4")
		det, err := rule.Evaluate(context.Background(), evt, state, domain.RuleConfig{Enabled: true})
		if err != nil {
			t.Fatalf("eval error: %v", err)
		}
		if det != nil {
			t.Fatalf("should not trigger on %d failures", i+1)
		}
	}
}

func TestBruteForce_AtThreshold(t *testing.T) {
	rule := &BruteForceRule{}
	state := NewMemStateStore()
	actor := actorPtr()
	var det *domain.Detection

	for i := 0; i < 5; i++ {
		evt := makeLoginEvent("failure", actor, "1.2.3.4")
		det, _ = rule.Evaluate(context.Background(), evt, state, domain.RuleConfig{Enabled: true})
	}
	if det == nil {
		t.Fatal("should trigger at 5 failures")
	}
	if det.Severity != domain.SeverityHigh {
		t.Errorf("expected high severity, got %s", det.Severity)
	}
}

func TestBruteForce_SuccessLoginIgnored(t *testing.T) {
	rule := &BruteForceRule{}
	state := NewMemStateStore()
	actor := actorPtr()

	evt := makeLoginEvent("success", actor, "1.2.3.4")
	det, _ := rule.Evaluate(context.Background(), evt, state, domain.RuleConfig{Enabled: true})
	if det != nil {
		t.Fatal("should not trigger on successful login")
	}
}

// --- CredentialStuffing tests ---

func TestCredentialStuffing_BelowThreshold(t *testing.T) {
	rule := &CredentialStuffingRule{}
	state := NewMemStateStore()

	for i := 0; i < 9; i++ { // 9 unique accounts, threshold is 10
		id := uuid.New()
		evt := makeLoginEvent("failure", &id, "1.2.3.4")
		evt.ActorName = "user" + string(rune('a'+i))
		det, _ := rule.Evaluate(context.Background(), evt, state, domain.RuleConfig{Enabled: true})
		if det != nil {
			t.Fatalf("should not trigger on %d accounts", i+1)
		}
	}
}

func TestCredentialStuffing_AtThreshold(t *testing.T) {
	rule := &CredentialStuffingRule{}
	state := NewMemStateStore()
	var det *domain.Detection

	for i := 0; i < 10; i++ {
		id := uuid.New()
		evt := makeLoginEvent("failure", &id, "1.2.3.4")
		evt.ActorName = "user" + string(rune('a'+i))
		det, _ = rule.Evaluate(context.Background(), evt, state, domain.RuleConfig{Enabled: true})
	}
	if det == nil {
		t.Fatal("should trigger at 10 unique accounts")
	}
	if det.Severity != domain.SeverityCritical {
		t.Errorf("expected critical, got %s", det.Severity)
	}
}

func TestCredentialStuffing_NoIP(t *testing.T) {
	rule := &CredentialStuffingRule{}
	state := NewMemStateStore()
	actor := actorPtr()

	evt := makeLoginEvent("failure", actor, "")
	det, _ := rule.Evaluate(context.Background(), evt, state, domain.RuleConfig{Enabled: true})
	if det != nil {
		t.Fatal("should not trigger without IP")
	}
}

// --- ImpossibleTravel tests ---

func TestImpossibleTravel_NormalSpeed(t *testing.T) {
	rule := &ImpossibleTravelRule{}
	state := NewMemStateStore()
	actor := actorPtr()

	// First login from New York.
	evt1 := makeLoginEvent("success", actor, "1.1.1.1")
	evt1.Metadata = map[string]any{"latitude": 40.7128, "longitude": -74.0060}
	evt1.CreatedAt = time.Now().UTC().Add(-2 * time.Hour)
	rule.Evaluate(context.Background(), evt1, state, domain.RuleConfig{Enabled: true})

	// Second login from Boston 2 hours later (~350km, ~175 km/h — normal).
	evt2 := makeLoginEvent("success", actor, "2.2.2.2")
	evt2.Metadata = map[string]any{"latitude": 42.3601, "longitude": -71.0589}
	evt2.CreatedAt = time.Now().UTC()
	det, _ := rule.Evaluate(context.Background(), evt2, state, domain.RuleConfig{Enabled: true})
	if det != nil {
		t.Fatal("should not trigger for normal travel speed")
	}
}

func TestImpossibleTravel_ImpossiblyFast(t *testing.T) {
	rule := &ImpossibleTravelRule{}
	state := NewMemStateStore()
	actor := actorPtr()

	// Login from New York 5 minutes ago.
	evt1 := makeLoginEvent("success", actor, "1.1.1.1")
	evt1.Metadata = map[string]any{"latitude": 40.7128, "longitude": -74.0060}
	evt1.CreatedAt = time.Now().UTC().Add(-5 * time.Minute)
	rule.Evaluate(context.Background(), evt1, state, domain.RuleConfig{Enabled: true})

	// Login from London now (~5570km in 5 min — impossible).
	evt2 := makeLoginEvent("success", actor, "2.2.2.2")
	evt2.Metadata = map[string]any{"latitude": 51.5074, "longitude": -0.1278}
	evt2.CreatedAt = time.Now().UTC()
	det, _ := rule.Evaluate(context.Background(), evt2, state, domain.RuleConfig{Enabled: true})
	if det == nil {
		t.Fatal("should trigger for impossible travel")
	}
	speed := det.Detail["speed_kmh"].(float64)
	if speed < 900 {
		t.Errorf("expected speed >900, got %.0f", speed)
	}
}

func TestImpossibleTravel_NoGeoData(t *testing.T) {
	rule := &ImpossibleTravelRule{}
	state := NewMemStateStore()
	actor := actorPtr()

	evt := makeLoginEvent("success", actor, "1.1.1.1")
	// No geo metadata
	det, _ := rule.Evaluate(context.Background(), evt, state, domain.RuleConfig{Enabled: true})
	if det != nil {
		t.Fatal("should not trigger without geo data")
	}
}

// --- Engine tests ---

type mockRepo struct {
	detections []*domain.Detection
}

func (m *mockRepo) InsertDetection(_ context.Context, d *domain.Detection) error {
	m.detections = append(m.detections, d)
	return nil
}

func TestEngine_EvaluatePanicRecovery(t *testing.T) {
	engine := NewEngine(&mockRepo{}, NewMemStateStore())
	// Should not panic with nil event.
	engine.Evaluate(context.Background(), nil)
}

func TestEngine_RulesRegistered(t *testing.T) {
	engine := NewEngine(&mockRepo{}, NewMemStateStore())
	rules := engine.Registry().All()
	if len(rules) < 3 {
		t.Errorf("expected ≥3 rules, got %d", len(rules))
	}
}
