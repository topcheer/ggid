package server

import (
	"fmt"
	"testing"
)

func TestEvaluateAccessPolicy_EmptyPolicy(t *testing.T) {
	result := evaluateAccessPolicy(map[string]any{}, map[string]any{}, map[string]any{})
	if result.Decision != "allow" {
		t.Errorf("empty policy should allow, got %s", result.Decision)
	}
}

func TestEvaluateAccessPolicy_DenyOnConditionFail(t *testing.T) {
	policy := map[string]any{
		"conditions": map[string]any{
			"and": []any{
				map[string]any{"$user.role": "sre"},
			},
		},
	}
	user := map[string]any{"role": "viewer"}
	result := evaluateAccessPolicy(policy, user, map[string]any{})
	if result.Decision != "deny" {
		t.Errorf("should deny when role mismatches, got %s", result.Decision)
	}
}

func TestEvaluateAccessPolicy_AllowOnAllConditionsMet(t *testing.T) {
	policy := map[string]any{
		"conditions": map[string]any{
			"and": []any{
				map[string]any{"$user.role": "sre"},
				map[string]any{"$security.device_trusted": true},
			},
		},
	}
	user := map[string]any{"role": "sre"}
	security := map[string]any{"device_trusted": true}
	result := evaluateAccessPolicy(policy, user, security)
	if result.Decision != "allow" {
		t.Errorf("should allow when all conditions met, got %s", result.Decision)
	}
}

func TestEvaluateAccessPolicy_InOperator(t *testing.T) {
	policy := map[string]any{
		"conditions": map[string]any{
			"and": []any{
				map[string]any{"$user.role": map[string]any{"$in": []any{"sre", "platform-eng"}}},
			},
		},
	}
	user := map[string]any{"role": "platform-eng"}
	result := evaluateAccessPolicy(policy, user, map[string]any{})
	if result.Decision != "allow" {
		t.Errorf("$in should match, got %s", result.Decision)
	}
}

func TestResolveAttribute(t *testing.T) {
	user := map[string]any{"email": "alice@example.com"}
	sec := map[string]any{"device_trusted": true}
	if resolveAttribute("$user.email", user, sec) != "alice@example.com" {
		t.Error("should resolve $user.email")
	}
	if fmt.Sprintf("%v", resolveAttribute("$security.device_trusted", user, sec)) != "true" {
		t.Error("should resolve $security.device_trusted")
	}
}
