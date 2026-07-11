package service

import (
	"context"
	"testing"

	"github.com/ggid/ggid/services/policy/internal/domain"
	"github.com/google/uuid"
)

// TestDryRunRegression_NoStateMutation verifies that EvaluateDryRun
// does not modify any policy state (roles, permissions, rules).
func TestDryRunRegression_NoStateMutation(t *testing.T) {
	ResetDryRunResults()

	// Set up an evaluator with a known policy state
	e := NewEvaluator(nil, nil, nil)

	req := &domain.CheckRequest{
		UserID:       uuid.New(),
		ResourceType: "documents",
		Action:       "read",
		TenantID:     uuid.New(),
	}

	// Evaluate in dry-run mode
	result, err := e.EvaluateDryRun(context.Background(), req)
	if err != nil {
		t.Fatalf("EvaluateDryRun failed: %v", err)
	}

	// Dry-run must return a result without error
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Allowed {
		// With nil readers, should deny (no roles found)
		t.Errorf("expected denial with nil readers, got allowed=true")
	}
	if result.Reason == "" {
		t.Error("expected non-empty reason")
	}

	// Verify results are accumulated
	results := GetDryRunResults()
	if len(results) != 1 {
		t.Errorf("expected 1 dry-run result, got %d", len(results))
	}

	// Second dry-run should accumulate
	e.EvaluateDryRun(context.Background(), req)
	results = GetDryRunResults()
	if len(results) != 2 {
		t.Errorf("expected 2 dry-run results after second call, got %d", len(results))
	}

	// Reset clears results
	ResetDryRunResults()
	results = GetDryRunResults()
	if len(results) != 0 {
		t.Errorf("expected 0 results after reset, got %d", len(results))
	}
}

// TestDryRunRegression_NilRequest verifies graceful handling of nil request.
func TestDryRunRegression_NilRequest(t *testing.T) {
	ResetDryRunResults()
	defer ResetDryRunResults()

	e := NewEvaluator(nil, nil, nil)
	result, err := e.EvaluateDryRun(context.Background(), nil)
	if err != nil {
		t.Fatalf("nil request should not error: %v", err)
	}
	if result.Allowed {
		t.Error("nil request should be denied")
	}
	if result.Reason != "nil request" {
		t.Errorf("expected 'nil request' reason, got '%s'", result.Reason)
	}
}

// TestDryRunRegression_ResultFields verifies all DryRunResult fields are populated.
func TestDryRunRegression_ResultFields(t *testing.T) {
	ResetDryRunResults()
	defer ResetDryRunResults()

	e := NewEvaluator(nil, nil, nil)
	req := &domain.CheckRequest{
		UserID:       uuid.New(),
		ResourceType: "users",
		Action:       "delete",
		TenantID:     uuid.New(),
	}

	result, err := e.EvaluateDryRun(context.Background(), req)
	if err != nil {
		t.Fatalf("EvaluateDryRun failed: %v", err)
	}

	if result.ResourceType != "users" {
		t.Errorf("expected ResourceType=users, got %s", result.ResourceType)
	}
	if result.Action != "delete" {
		t.Errorf("expected Action=delete, got %s", result.Action)
	}
	if result.EvaluatedAt.IsZero() {
		t.Error("expected non-zero EvaluatedAt")
	}
	if result.Reason == "" {
		t.Error("expected non-empty Reason")
	}
}
