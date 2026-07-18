package server

import (
	"testing"
)

func TestDefaultSeverityRouting(t *testing.T) {
	tests := []struct {
		severity string
		minChans int
	}{
		{"critical", 5},
		{"high", 2},
		{"medium", 1},
		{"low", 1},
		{"unknown", 1},
	}
	for _, tt := range tests {
		channels := DefaultSeverityRouting(tt.severity)
		if len(channels) < tt.minChans {
			t.Errorf("%s: expected >= %d channels, got %d", tt.severity, tt.minChans, len(channels))
		}
	}
}

func TestNotificationRepo_NilPool(t *testing.T) {
	repo := newNotificationRepo(nil)
	rules, err := repo.ListRules(nil)
	if err != nil { t.Fatalf("nil pool should not error: %v", err) }
	if len(rules) != 0 { t.Error("nil pool should return empty") }
	log, err := repo.ListLog(nil, 10)
	if err != nil { t.Fatalf("nil pool should not error: %v", err) }
	if len(log) != 0 { t.Error("nil pool should return empty") }
}

func TestNotificationRepo_CreateRuleNilPool(t *testing.T) {
	repo := newNotificationRepo(nil)
	rule := &NotificationRule{Severity: "high", Channels: []NotificationChannel{ChSlack}}
	if err := repo.CreateRule(nil, rule); err != nil {
		t.Errorf("nil pool should not error: %v", err)
	}
}

func TestNotificationChannel_Constants(t *testing.T) {
	if ChEmail != "email" { t.Error("email mismatch") }
	if ChSlack != "slack" { t.Error("slack mismatch") }
	if ChPagerDuty != "pagerduty" { t.Error("pagerduty mismatch") }
}
