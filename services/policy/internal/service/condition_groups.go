package service

import (
	"context"

	"github.com/ggid/ggid/services/policy/internal/domain"
)

// ConditionOp is the logical operator for a condition group.
type ConditionOp string

const (
	OpAnd ConditionOp = "AND"
	OpOr  ConditionOp = "OR"
	OpNot ConditionOp = "NOT"
)

// ConditionGroup represents a nested AND/OR/NOT condition tree.
type ConditionGroup struct {
	Op         ConditionOp      // AND, OR, NOT
	Conditions []LeafCondition  // leaf conditions at this level
	Children   []*ConditionGroup // nested groups
}

// LeafCondition is a single attribute comparison.
type LeafCondition struct {
	Attribute string // e.g. "department", "role", "location"
	Operator  string // "eq", "ne", "in", "not_in"
	Value     any    // comparison value
}

// EvalConditionGroup recursively evaluates a condition group against the
// subject/resource attributes in a CheckRequest.
func EvalConditionGroup(ctx context.Context, group *ConditionGroup, attrs map[string]any) bool {
	if group == nil {
		return true
	}

	// Evaluate leaf conditions
	leafResults := make([]bool, 0, len(group.Conditions))
	for _, lc := range group.Conditions {
		leafResults = append(leafResults, evalLeaf(lc, attrs))
	}

	// Evaluate child groups
	childResults := make([]bool, 0, len(group.Children))
	for _, child := range group.Children {
		childResults = append(childResults, EvalConditionGroup(ctx, child, attrs))
	}

	all := append(leafResults, childResults...)

	switch group.Op {
	case OpAnd:
		for _, r := range all {
			if !r {
				return false
			}
		}
		return true
	case OpOr:
		for _, r := range all {
			if r {
				return true
			}
		}
		return false
	case OpNot:
		// NOT applies to the single child/leaf
		if len(all) == 0 {
			return true
		}
		return !all[0]
	default:
		return true
	}
}

func evalLeaf(lc LeafCondition, attrs map[string]any) bool {
	val, exists := attrs[lc.Attribute]
	if !exists {
		// Also check request conditions
		return false
	}

	switch lc.Operator {
	case "eq":
		return val == lc.Value
	case "ne":
		return val != lc.Value
	default:
		return false
	}
}

// CheckWithConditions performs a policy check but also evaluates
// ABAC condition groups against the request conditions.
func (e *Evaluator) CheckWithConditions(ctx context.Context, req *domain.CheckRequest, groups []*ConditionGroup) (*domain.CheckResult, error) {
	// First do the normal check
	result, err := e.Check(ctx, req)
	if err != nil {
		return nil, err
	}
	if !result.Allowed {
		return result, nil
	}

	// Then evaluate condition groups — all must pass (implicit AND at top level)
	attrs := req.Conditions
	if attrs == nil {
		attrs = map[string]any{}
	}
	attrs["resource_type"] = req.ResourceType
	attrs["action"] = req.Action

	for _, group := range groups {
		if !EvalConditionGroup(ctx, group, attrs) {
			return &domain.CheckResult{
				Allowed: false,
				Reason:  "ABAC condition group denied access",
			}, nil
		}
	}

	return result, nil
}
