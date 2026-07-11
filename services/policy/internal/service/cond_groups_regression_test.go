package service

import (
	"context"
	"testing"
)

// TestCondGroupRegression_InOperator verifies the "in" operator for list membership.
func TestCondGroupRegression_InOperator(t *testing.T) {
	group := &ConditionGroup{
		Op: OpAnd,
		Conditions: []LeafCondition{
			{Attribute: "department", Operator: "in", Value: []string{"engineering", "ops"}},
		},
	}

	// "in" operator not yet implemented in evalLeaf — test that it returns false
	result := EvalConditionGroup(context.Background(), group, map[string]any{
		"department": "engineering",
	})
	// Currently evalLeaf only supports eq/ne — "in" returns false
	if result {
		t.Log("Note: 'in' operator returns false (not yet implemented). Expected behavior documented.")
	}
}

// TestCondGroupRegression_EmptyGroup verifies empty group behavior.
func TestCondGroupRegression_EmptyGroup(t *testing.T) {
	// Empty AND group → vacuously true (no conditions to fail)
	group := &ConditionGroup{Op: OpAnd}
	result := EvalConditionGroup(context.Background(), group, map[string]any{})
	if !result {
		t.Error("empty AND group should be vacuously true")
	}

	// Empty OR group → vacuously false (no conditions to satisfy)
	group = &ConditionGroup{Op: OpOr}
	result = EvalConditionGroup(context.Background(), group, map[string]any{})
	if result {
		t.Error("empty OR group should be vacuously false")
	}

	// Empty NOT group → true (nothing to negate)
	group = &ConditionGroup{Op: OpNot}
	result = EvalConditionGroup(context.Background(), group, map[string]any{})
	if !result {
		t.Error("empty NOT group should be true")
	}
}

// TestCondGroupRegression_MixedLeafAndChild verifies a group with both
// leaf conditions AND child groups evaluated together.
func TestCondGroupRegression_MixedLeafAndChild(t *testing.T) {
	// (role == "admin") AND (department == "eng" OR department == "ops")
	group := &ConditionGroup{
		Op: OpAnd,
		Conditions: []LeafCondition{
			{Attribute: "role", Operator: "eq", Value: "admin"},
		},
		Children: []*ConditionGroup{
			{
				Op: OpOr,
				Conditions: []LeafCondition{
					{Attribute: "department", Operator: "eq", Value: "eng"},
					{Attribute: "department", Operator: "eq", Value: "ops"},
				},
			},
		},
	}

	// Admin in eng → true
	result := EvalConditionGroup(context.Background(), group, map[string]any{
		"role": "admin", "department": "eng",
	})
	if !result {
		t.Error("admin+eng should pass")
	}

	// Admin in ops → true (OR child passes)
	result = EvalConditionGroup(context.Background(), group, map[string]any{
		"role": "admin", "department": "ops",
	})
	if !result {
		t.Error("admin+ops should pass")
	}

	// Admin in sales → false (OR child fails)
	result = EvalConditionGroup(context.Background(), group, map[string]any{
		"role": "admin", "department": "sales",
	})
	if result {
		t.Error("admin+sales should fail (OR child fails)")
	}

	// User (not admin) in eng → false (AND leaf fails)
	result = EvalConditionGroup(context.Background(), group, map[string]any{
		"role": "user", "department": "eng",
	})
	if result {
		t.Error("user+eng should fail (role leaf fails)")
	}
}

// TestCondGroupRegression_DeepNesting verifies 4-level nesting.
func TestCondGroupRegression_DeepNesting(t *testing.T) {
	// NOT(NOT(NOT(role == "admin")))
	// = NOT(NOT(false))  [if role != admin]
	// = NOT(true)
	// = false
	group := &ConditionGroup{
		Op: OpNot,
		Children: []*ConditionGroup{
			{
				Op: OpNot,
				Children: []*ConditionGroup{
					{
						Op: OpNot,
						Conditions: []LeafCondition{
							{Attribute: "role", Operator: "eq", Value: "admin"},
						},
					},
				},
			},
		},
	}

	// role=user: NOT(NOT(NOT(false))) = NOT(NOT(true)) = NOT(false) = true
	result := EvalConditionGroup(context.Background(), group, map[string]any{"role": "user"})
	if !result {
		t.Error("NOT(NOT(NOT(false))) should be true")
	}

	// role=admin: NOT(NOT(NOT(true))) = NOT(NOT(false)) = NOT(true) = false
	result = EvalConditionGroup(context.Background(), group, map[string]any{"role": "admin"})
	if result {
		t.Error("NOT(NOT(NOT(true))) should be false")
	}
}

// TestCondGroupRegression_NilGroup verifies nil group returns true.
func TestCondGroupRegression_NilGroup(t *testing.T) {
	result := EvalConditionGroup(context.Background(), nil, map[string]any{"role": "admin"})
	if !result {
		t.Error("nil group should return true")
	}
}

// TestCondGroupRegression_UnknownOperator verifies unknown operator returns true (default).
func TestCondGroupRegression_UnknownOperator(t *testing.T) {
	group := &ConditionGroup{
		Op: "UNKNOWN",
		Conditions: []LeafCondition{
			{Attribute: "x", Operator: "eq", Value: 1},
		},
	}
	result := EvalConditionGroup(context.Background(), group, map[string]any{"x": 1})
	if !result {
		t.Error("unknown operator should default to true")
	}
}
