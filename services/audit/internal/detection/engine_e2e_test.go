package detection

import (
	"context"
	"testing"
	"time"

	"github.com/ggid/ggid/services/audit/internal/domain"
	"github.com/google/uuid"
)

// TestEngine_BruteForceEndToEnd proves the full pipeline:
// 10 failed login events → engine.Evaluate → detection persisted.
func TestEngine_BruteForceEndToEnd(t *testing.T) {
	repo := &mockRepo{}
	engine := NewEngine(repo, NewMemStateStore())

	actor := uuid.New()
	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	for i := 0; i < 10; i++ {
		evt := &domain.AuditEvent{
			ID:        uuid.New(),
			TenantID:  tenantID,
			ActorID:   &actor,
			ActorName: "victim",
			Action:    "user.login",
			Result:    "failure",
			IPAddress: "192.168.1.100",
			CreatedAt: time.Now().Add(-time.Duration(10-i) * time.Minute / 2),
		}
		engine.Evaluate(context.Background(), evt)
	}

	if len(repo.detections) == 0 {
		t.Fatal("expected at least 1 detection after 10 failed logins")
	}

	det := repo.detections[0]
	if det.RuleID != "brute_force" {
		t.Errorf("expected brute_force, got %s", det.RuleID)
	}
	if det.Severity != domain.SeverityHigh {
		t.Errorf("expected high, got %s", det.Severity)
	}
	if det.ActorID == nil || *det.ActorID != actor {
		t.Error("actor ID mismatch")
	}

	t.Logf("brute force detection created: rule=%s severity=%s hits=%d",
		det.RuleID, det.Severity, len(repo.detections))
}

// TestEngine_MultipleRulesNoInterference proves rules don't interfere.
func TestEngine_MultipleRulesNoInterference(t *testing.T) {
	repo := &mockRepo{}
	engine := NewEngine(repo, NewMemStateStore())

	// A successful login should not trigger brute_force.
	actor := uuid.New()
	evt := &domain.AuditEvent{
		ID:        uuid.New(),
		TenantID:  uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		ActorID:   &actor,
		Action:    "user.login",
		Result:    "success",
		IPAddress: "1.2.3.4",
		CreatedAt: time.Now(),
	}
	engine.Evaluate(context.Background(), evt)

	if len(repo.detections) != 0 {
		t.Errorf("expected 0 detections for successful login, got %d", len(repo.detections))
	}
}

// TestEngine_DisabledRuleSkipped proves disabled rules are skipped.
func TestEngine_DisabledRuleSkipped(t *testing.T) {
	repo := &mockRepo{}
	engine := NewEngine(repo, NewMemStateStore())

	// Disable brute_force for this tenant.
	engine.Registry().SetOverride(
		"00000000-0000-0000-0000-000000000001",
		"brute_force",
		domain.RuleConfig{Enabled: false},
	)

	actor := uuid.New()
	for i := 0; i < 10; i++ {
		evt := &domain.AuditEvent{
			ID:        uuid.New(),
			TenantID:  uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			ActorID:   &actor,
			Action:    "user.login",
			Result:    "failure",
			IPAddress: "1.2.3.4",
			CreatedAt: time.Now(),
		}
		engine.Evaluate(context.Background(), evt)
	}

	if len(repo.detections) != 0 {
		t.Errorf("disabled rule should not produce detections, got %d", len(repo.detections))
	}
}
