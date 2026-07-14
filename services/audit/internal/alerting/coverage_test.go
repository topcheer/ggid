package alerting

import (
	"context"
	"errors"
	"testing"
	"time"
)

type mockNotifier2 struct {
	called bool
	err    error
}

func (m *mockNotifier2) Notify(_ context.Context, _ *Alert, _ []AlertAction) error {
	m.called = true
	return m.err
}

func TestEvaluate2_NotifierError(t *testing.T) {
	n := &mockNotifier2{err: errors.New("send failed")}
	engine := NewAlertEngine(n)
	engine.AddRule(&AlertRule{
		ID:       "r-err",
		TenantID: "t1",
		Enabled:  true,
		Condition: AlertCondition{Field: "action", Operator: "eq", Value: "login"},
		Threshold: 1,
		Window:    time.Hour,
	})
	alerts := engine.Evaluate(context.Background(), &AlertEvent{TenantID: "t1", Action: "login"})
	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert even if notifier fails, got %d", len(alerts))
	}
}

func TestEvaluate2_NEQ(t *testing.T) {
	cond := AlertCondition{Field: "action", Operator: "neq", Value: "login"}
	ev := &AlertEvent{Action: "logout"}
	if !matchesCondition(cond, ev) {
		t.Error("neq should match different value")
	}
}

func TestEvaluate2_Contains(t *testing.T) {
	cond := AlertCondition{Field: "action", Operator: "contains", Value: "log"}
	ev := &AlertEvent{Action: "login"}
	if !matchesCondition(cond, ev) {
		t.Error("contains should match substring")
	}
}

func TestEvaluate2_UnknownOperator(t *testing.T) {
	cond := AlertCondition{Field: "action", Operator: "gt", Value: "x"}
	ev := &AlertEvent{Action: "login"}
	if matchesCondition(cond, ev) {
		t.Error("unknown operator should return false")
	}
}

func TestEvaluate2_CustomField(t *testing.T) {
	cond := AlertCondition{Field: "dept", Operator: "eq", Value: "eng"}
	ev := &AlertEvent{Fields: map[string]any{"dept": "eng"}}
	if !matchesCondition(cond, ev) {
		t.Error("custom field should match")
	}
}

func TestEvaluate2_ListRules(t *testing.T) {
	engine := NewAlertEngine(&mockNotifier2{})
	engine.AddRule(&AlertRule{ID: "r-l1", TenantID: "t1"})
	engine.AddRule(&AlertRule{ID: "r-l2", TenantID: "t2"})
	rules := engine.ListRules("t1")
	if len(rules) != 1 || rules[0].ID != "r-l1" {
		t.Errorf("expected 1 rule for t1, got %v", rules)
	}
}

func TestEvaluate2_RemoveRule(t *testing.T) {
	engine := NewAlertEngine(&mockNotifier2{})
	engine.AddRule(&AlertRule{ID: "r-rm", TenantID: "t1"})
	engine.RemoveRule("r-rm")
	rules := engine.ListRules("t1")
	if len(rules) != 0 {
		t.Error("rule should be removed")
	}
}

func TestContains2(t *testing.T) {
	if !contains("hello world", "world") {
		t.Error("should contain substring")
	}
	if contains("hello", "world") {
		t.Error("should not contain substring")
	}
	if !contains("hello", "") {
		t.Error("empty substring should match")
	}
}

func TestEvaluate2_WebhookNotifier(t *testing.T) {
	w := &WebhookNotifier{URL: "http://example.com/hook"}
	err := w.Notify(context.Background(), &Alert{RuleName: "test"}, nil)
	if err != nil {
		t.Errorf("webhook notify should not error: %v", err)
	}
}
