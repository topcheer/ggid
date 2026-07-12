package service

import (
	"testing"
	"time"

	"github.com/ggid/ggid/services/audit/internal/domain"
	"github.com/google/uuid"
)

func makeCorrelationEvent(action, actor, ip string, ts time.Time) domain.AuditEvent {
	return domain.AuditEvent{
		ID:        uuid.New(),
		TenantID:  uuid.New(),
		ActorType: domain.ActorUser,
		ActorName: actor,
		Action:    action,
		IPAddress: ip,
		CreatedAt: ts,
	}
}

func TestEventCorrelator_BruteForceDetection(t *testing.T) {
	ec := NewEventCorrelator()
	ec.AddRule(CorrelationRule{
		Name:      "brute_force",
		Pattern:   "user.login.failed",
		Window:    5 * time.Minute,
		MinEvents: 3,
		Action:    "alert",
	})

	now := time.Now()
	events := []domain.AuditEvent{
		makeCorrelationEvent("user.login.failed", "alice", "1.2.3.4", now),
		makeCorrelationEvent("user.login.failed", "alice", "1.2.3.4", now.Add(1*time.Minute)),
		makeCorrelationEvent("user.login.failed", "alice", "1.2.3.4", now.Add(2*time.Minute)),
	}

	results := ec.CorrelateAuditEvents(events)
	if len(results) == 0 {
		t.Fatal("should detect brute force pattern")
	}
	if results[0].RuleName != "brute_force" {
		t.Errorf("expected rule 'brute_force', got '%s'", results[0].RuleName)
	}
}

func TestEventCorrelator_NoCorrelation(t *testing.T) {
	ec := NewEventCorrelator()
	ec.AddRule(CorrelationRule{
		Name:      "brute_force",
		Pattern:   "user.login.failed",
		Window:    5 * time.Minute,
		MinEvents: 5,
		Action:    "alert",
	})

	now := time.Now()
	events := []domain.AuditEvent{
		makeCorrelationEvent("user.login.failed", "alice", "1.2.3.4", now),
		makeCorrelationEvent("user.login.failed", "alice", "1.2.3.4", now.Add(1*time.Minute)),
	}

	results := ec.CorrelateAuditEvents(events)
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestEventCorrelator_PatternWildcard(t *testing.T) {
	ec := NewEventCorrelator()
	ec.AddRule(CorrelationRule{
		Name:      "all_events",
		Pattern:   "*",
		Window:    10 * time.Minute,
		MinEvents: 2,
		Action:    "log",
	})

	now := time.Now()
	events := []domain.AuditEvent{
		makeCorrelationEvent("user.login", "alice", "1.2.3.4", now),
		makeCorrelationEvent("user.logout", "alice", "1.2.3.4", now.Add(1*time.Minute)),
	}

	results := ec.CorrelateAuditEvents(events)
	if len(results) == 0 {
		t.Fatal("wildcard pattern should match all events")
	}
}

func TestEventCorrelator_PatternPrefix(t *testing.T) {
	ec := NewEventCorrelator()
	ec.AddRule(CorrelationRule{
		Name:      "user_events",
		Pattern:   "user.*",
		Window:    10 * time.Minute,
		MinEvents: 2,
		Action:    "log",
	})

	now := time.Now()
	events := []domain.AuditEvent{
		makeCorrelationEvent("user.login", "alice", "1.2.3.4", now),
		makeCorrelationEvent("user.logout", "bob", "5.6.7.8", now.Add(1*time.Minute)),
		makeCorrelationEvent("role.assign", "carol", "9.0.1.2", now.Add(2*time.Minute)),
	}

	results := ec.CorrelateAuditEvents(events)
	if len(results) == 0 {
		t.Fatal("prefix pattern should match user.* events")
	}
	if len(results[0].Events) < 2 {
		t.Errorf("expected at least 2 events, got %d", len(results[0].Events))
	}
}

func TestEventCorrelator_Dedup(t *testing.T) {
	ec := NewEventCorrelator()
	ec.AddRule(CorrelationRule{Name: "test", Pattern: "*", Window: 10 * time.Minute, MinEvents: 2, Action: "log"})

	now := time.Now()
	events := []domain.AuditEvent{
		makeCorrelationEvent("a", "alice", "1.2.3.4", now),
		makeCorrelationEvent("b", "alice", "1.2.3.4", now.Add(1*time.Minute)),
	}

	results := ec.CorrelateAuditEvents(events)
	if len(results) != 1 {
		t.Errorf("expected 1 deduplicated result, got %d", len(results))
	}
}

func TestEventCorrelator_FalsePositiveFilter(t *testing.T) {
	ec := NewEventCorrelator()
	ec.AddRule(CorrelationRule{Name: "multi_actor", Pattern: "*", Window: 10 * time.Minute, MinEvents: 5, Action: "alert"})

	now := time.Now()
	// Single actor with many events but min_events > 3 → filtered.
	events := []domain.AuditEvent{
		makeCorrelationEvent("a", "alice", "1.2.3.4", now),
		makeCorrelationEvent("b", "alice", "1.2.3.4", now.Add(1*time.Minute)),
		makeCorrelationEvent("c", "alice", "1.2.3.4", now.Add(2*time.Minute)),
		makeCorrelationEvent("d", "alice", "1.2.3.4", now.Add(3*time.Minute)),
		makeCorrelationEvent("e", "alice", "1.2.3.4", now.Add(4*time.Minute)),
	}

	results := ec.CorrelateAuditEvents(events)
	if len(results) != 0 {
		t.Errorf("single actor with min_events>3 should be filtered, got %d results", len(results))
	}
}

func TestEventCorrelator_GetResults(t *testing.T) {
	ec := NewEventCorrelator()
	ec.AddRule(CorrelationRule{Name: "test", Pattern: "*", Window: 10 * time.Minute, MinEvents: 2, Action: "log"})

	now := time.Now()
	ec.CorrelateAuditEvents([]domain.AuditEvent{
		makeCorrelationEvent("a", "alice", "1.2.3.4", now),
		makeCorrelationEvent("b", "alice", "1.2.3.4", now.Add(1*time.Minute)),
	})

	results := ec.GetResults()
	if len(results) == 0 {
		t.Error("should have stored results")
	}
}

func TestEventCorrelator_Reset(t *testing.T) {
	ec := NewEventCorrelator()
	ec.AddRule(CorrelationRule{Name: "test", Pattern: "*", Window: 10 * time.Minute, MinEvents: 2, Action: "log"})
	ec.CorrelateAuditEvents([]domain.AuditEvent{
		makeCorrelationEvent("a", "alice", "1.2.3.4", time.Now()),
		makeCorrelationEvent("b", "alice", "1.2.3.4", time.Now().Add(time.Minute)),
	})
	ec.Reset()
	if len(ec.GetResults()) != 0 {
		t.Error("results should be empty after reset")
	}
}
