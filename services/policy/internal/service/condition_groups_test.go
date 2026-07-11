package service

import (
	"context"
	"testing"

	"github.com/ggid/ggid/services/policy/internal/domain"
	"github.com/google/uuid"
)

func TestConditionGroup_SimpleAND(t *testing.T) {
	group := &ConditionGroup{
		Op: OpAnd,
		Conditions: []LeafCondition{
			{Attribute: "department", Operator: "eq", Value: "engineering"},
			{Attribute: "level", Operator: "eq", Value: 5},
		},
	}
	attrs := map[string]any{"department": "engineering", "level": 5}
	if !EvalConditionGroup(context.Background(), group, attrs) {
		t.Error("AND should pass when all conditions match")
	}

	attrs["level"] = 3
	if EvalConditionGroup(context.Background(), group, attrs) {
		t.Error("AND should fail when one condition doesn't match")
	}
}

func TestConditionGroup_SimpleOR(t *testing.T) {
	group := &ConditionGroup{
		Op: OpOr,
		Conditions: []LeafCondition{
			{Attribute: "role", Operator: "eq", Value: "admin"},
			{Attribute: "role", Operator: "eq", Value: "superadmin"},
		},
	}

	if !EvalConditionGroup(context.Background(), group, map[string]any{"role": "admin"}) {
		t.Error("OR should pass when one matches")
	}
	if !EvalConditionGroup(context.Background(), group, map[string]any{"role": "superadmin"}) {
		t.Error("OR should pass when other matches")
	}
	if EvalConditionGroup(context.Background(), group, map[string]any{"role": "user"}) {
		t.Error("OR should fail when none match")
	}
}

func TestConditionGroup_SimpleNOT(t *testing.T) {
	group := &ConditionGroup{
		Op: OpNot,
		Conditions: []LeafCondition{
			{Attribute: "blocked", Operator: "eq", Value: true},
		},
	}

	if !EvalConditionGroup(context.Background(), group, map[string]any{"blocked": false}) {
		t.Error("NOT should pass when condition is false")
	}
	if EvalConditionGroup(context.Background(), group, map[string]any{"blocked": true}) {
		t.Error("NOT should fail when condition is true")
	}
}

func TestConditionGroup_Nested(t *testing.T) {
	// (department=eng AND (level>=5 OR role=admin))
	group := &ConditionGroup{
		Op: OpAnd,
		Conditions: []LeafCondition{
			{Attribute: "department", Operator: "eq", Value: "eng"},
		},
		Children: []*ConditionGroup{
			{
				Op: OpOr,
				Conditions: []LeafCondition{
					{Attribute: "level", Operator: "eq", Value: 5},
					{Attribute: "role", Operator: "eq", Value: "admin"},
				},
			},
		},
	}

	// dept=eng, level=5 → pass
	ok := EvalConditionGroup(context.Background(), group, map[string]any{"department": "eng", "level": 5, "role": "user"})
	if !ok {
		t.Error("should pass: dept=eng AND (level=5)")
	}

	// dept=eng, role=admin → pass
	ok = EvalConditionGroup(context.Background(), group, map[string]any{"department": "eng", "level": 1, "role": "admin"})
	if !ok {
		t.Error("should pass: dept=eng AND (role=admin)")
	}

	// dept=eng, neither → fail
	ok = EvalConditionGroup(context.Background(), group, map[string]any{"department": "eng", "level": 1, "role": "user"})
	if ok {
		t.Error("should fail: dept=eng but neither child condition met")
	}

	// dept=sales → fail (top-level AND)
	ok = EvalConditionGroup(context.Background(), group, map[string]any{"department": "sales", "level": 5, "role": "admin"})
	if ok {
		t.Error("should fail: dept=sales fails top-level AND")
	}
}

func TestConditionGroup_DeepNested(t *testing.T) {
	// NOT(AND(role=user, NOT(dept=eng)))
	// = NOT(role=user AND dept!=eng)
	// = role!=user OR dept==eng
	group := &ConditionGroup{
		Op: OpNot,
		Children: []*ConditionGroup{
			{
				Op: OpAnd,
				Conditions: []LeafCondition{
					{Attribute: "role", Operator: "eq", Value: "user"},
				},
				Children: []*ConditionGroup{
					{
						Op: OpNot,
						Conditions: []LeafCondition{
							{Attribute: "dept", Operator: "eq", Value: "eng"},
						},
					},
				},
			},
		},
	}

	// role=admin → NOT(false) → true
	ok := EvalConditionGroup(context.Background(), group, map[string]any{"role": "admin", "dept": "sales"})
	if !ok {
		t.Error("role=admin should pass deep nested NOT(AND)")
	}

	// role=user, dept=eng → AND(role=user, NOT(dept=eng)) = AND(true, false) = false → NOT → true
	ok = EvalConditionGroup(context.Background(), group, map[string]any{"role": "user", "dept": "eng"})
	if !ok {
		t.Error("role=user+dept=eng should pass")
	}

	// role=user, dept=sales → AND(true, NOT(true)) = AND(true, false) → wait
	// dept=eng is false for "sales", so NOT(dept=eng=eng) = NOT(false) = true
	// AND(role=user=true, NOT(dept=eng=false→true)) = AND(true,true) = true
	// NOT(true) = false
	ok = EvalConditionGroup(context.Background(), group, map[string]any{"role": "user", "dept": "sales"})
	if ok {
		t.Error("role=user+dept=sales should fail")
	}
}

func TestConditionGroup_NilGroup(t *testing.T) {
	if !EvalConditionGroup(context.Background(), nil, nil) {
		t.Error("nil group should return true")
	}
}

func TestConditionGroup_NotEqual(t *testing.T) {
	group := &ConditionGroup{
		Op: OpAnd,
		Conditions: []LeafCondition{
			{Attribute: "status", Operator: "ne", Value: "suspended"},
		},
	}

	if !EvalConditionGroup(context.Background(), group, map[string]any{"status": "active"}) {
		t.Error("ne should pass when value differs")
	}
	if EvalConditionGroup(context.Background(), group, map[string]any{"status": "suspended"}) {
		t.Error("ne should fail when value matches")
	}
}

func TestConditionGroup_MissingAttribute(t *testing.T) {
	group := &ConditionGroup{
		Op: OpAnd,
		Conditions: []LeafCondition{
			{Attribute: "nonexistent", Operator: "eq", Value: "x"},
		},
	}

	if EvalConditionGroup(context.Background(), group, map[string]any{}) {
		t.Error("missing attribute should fail condition")
	}
}

func TestCheckWithConditions(t *testing.T) {
	e := NewEvaluator(nil, nil, nil)
	req := &domain.CheckRequest{
		UserID:       uuid.New(),
		ResourceType: "doc",
		Action:       "read",
		Conditions:   map[string]any{"department": "eng"},
	}

	// No condition groups → normal check result
	result, err := e.CheckWithConditions(context.Background(), req, nil)
	if err != nil {
		t.Fatalf("CheckWithConditions: %v", err)
	}
	_ = result // evaluator with nil readers returns allowed=false, that's fine
}
