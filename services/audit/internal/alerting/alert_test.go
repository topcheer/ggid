package alerting

import (
	"context"
	"testing"
	"time"
)

type mockNotifier struct {
	alerts []*Alert
}

func (m *mockNotifier) Notify(_ context.Context, alert *Alert, _ []AlertAction) error {
	m.alerts = append(m.alerts, alert)
	return nil
}

func newTestEngine() (*AlertEngine, *mockNotifier) {
	n := &mockNotifier{}
	return NewAlertEngine(n), n
}

// 1. TestEvaluate_SingleMatch
func TestEvaluate_SingleMatch(t *testing.T) {
	engine, notifier := newTestEngine()
	engine.AddRule(&AlertRule{
		ID: "r1", TenantID: "t1", Enabled: true,
		Condition: AlertCondition{Field: "action", Operator: "eq", Value: "login.failed"},
		Threshold: 1, Window: time.Minute,
		Actions: []AlertAction{{Type: "webhook", Target: "http://example.com/hook"}},
	})

	alerts := engine.Evaluate(context.Background(), &AlertEvent{
		TenantID: "t1", Action: "login.failed",
	})
	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(alerts))
	}
	if len(notifier.alerts) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(notifier.alerts))
	}
}

// 2. TestEvaluate_NoMatch
func TestEvaluate_NoMatch(t *testing.T) {
	engine, _ := newTestEngine()
	engine.AddRule(&AlertRule{
		ID: "r1", TenantID: "t1", Enabled: true,
		Condition: AlertCondition{Field: "action", Operator: "eq", Value: "login.failed"},
		Threshold: 1, Window: time.Minute,
	})

	alerts := engine.Evaluate(context.Background(), &AlertEvent{
		TenantID: "t1", Action: "login.success",
	})
	if len(alerts) != 0 {
		t.Fatalf("expected 0 alerts, got %d", len(alerts))
	}
}

// 3. TestEvaluate_ThresholdNotMet
func TestEvaluate_ThresholdNotMet(t *testing.T) {
	engine, notifier := newTestEngine()
	engine.AddRule(&AlertRule{
		ID: "r1", TenantID: "t1", Enabled: true,
		Condition: AlertCondition{Field: "action", Operator: "eq", Value: "login.failed"},
		Threshold: 3, Window: time.Minute,
	})

	engine.Evaluate(context.Background(), &AlertEvent{TenantID: "t1", Action: "login.failed"})
	engine.Evaluate(context.Background(), &AlertEvent{TenantID: "t1", Action: "login.failed"})

	if len(notifier.alerts) != 0 {
		t.Fatalf("expected 0 alerts before threshold, got %d", len(notifier.alerts))
	}
}

// 4. TestEvaluate_ThresholdMet
func TestEvaluate_ThresholdMet(t *testing.T) {
	engine, notifier := newTestEngine()
	engine.AddRule(&AlertRule{
		ID: "r1", TenantID: "t1", Enabled: true,
		Condition: AlertCondition{Field: "action", Operator: "eq", Value: "login.failed"},
		Threshold: 3, Window: time.Minute,
	})

	for i := 0; i < 3; i++ {
		engine.Evaluate(context.Background(), &AlertEvent{TenantID: "t1", Action: "login.failed"})
	}
	if len(notifier.alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(notifier.alerts))
	}
	if notifier.alerts[0].Count != 3 {
		t.Fatalf("expected count 3, got %d", notifier.alerts[0].Count)
	}
}

// 5. TestEvaluate_TenantIsolation
func TestEvaluate_TenantIsolation(t *testing.T) {
	engine, _ := newTestEngine()
	engine.AddRule(&AlertRule{
		ID: "r1", TenantID: "t1", Enabled: true,
		Condition: AlertCondition{Field: "action", Operator: "eq", Value: "login.failed"},
		Threshold: 1, Window: time.Minute,
	})

	alerts := engine.Evaluate(context.Background(), &AlertEvent{
		TenantID: "t2", Action: "login.failed",
	})
	if len(alerts) != 0 {
		t.Fatal("expected 0 alerts for different tenant")
	}
}

// 6. TestEvaluate_DisabledRule
func TestEvaluate_DisabledRule(t *testing.T) {
	engine, notifier := newTestEngine()
	engine.AddRule(&AlertRule{
		ID: "r1", TenantID: "t1", Enabled: false,
		Condition: AlertCondition{Field: "action", Operator: "eq", Value: "login.failed"},
		Threshold: 1, Window: time.Minute,
	})

	alerts := engine.Evaluate(context.Background(), &AlertEvent{
		TenantID: "t1", Action: "login.failed",
	})
	if len(alerts) != 0 || len(notifier.alerts) != 0 {
		t.Fatal("disabled rule should not fire")
	}
}

// 7. TestAddRemoveRule
func TestAddRemoveRule(t *testing.T) {
	engine, _ := newTestEngine()
	rule := &AlertRule{ID: "r1", TenantID: "t1", Enabled: true, Threshold: 1, Window: time.Minute}
	engine.AddRule(rule)

	if len(engine.ListRules("t1")) != 1 {
		t.Fatal("expected 1 rule")
	}

	engine.RemoveRule("r1")
	if len(engine.ListRules("t1")) != 0 {
		t.Fatal("expected 0 rules after remove")
	}
}

// 8. TestResetAfterFire
func TestResetAfterFire(t *testing.T) {
	engine, notifier := newTestEngine()
	engine.AddRule(&AlertRule{
		ID: "r1", TenantID: "t1", Enabled: true,
		Condition: AlertCondition{Field: "action", Operator: "eq", Value: "x"},
		Threshold: 2, Window: time.Minute,
	})

	engine.Evaluate(context.Background(), &AlertEvent{TenantID: "t1", Action: "x"})
	engine.Evaluate(context.Background(), &AlertEvent{TenantID: "t1", Action: "x"})
	// Should fire once
	if len(notifier.alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(notifier.alerts))
	}

	// Next single event should not fire (counter reset)
	engine.Evaluate(context.Background(), &AlertEvent{TenantID: "t1", Action: "x"})
	if len(notifier.alerts) != 1 {
		t.Fatalf("expected still 1 alert after reset, got %d", len(notifier.alerts))
	}
}
