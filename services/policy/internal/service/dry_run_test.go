package service

import (
	"context"
	"testing"

	"github.com/ggid/ggid/services/policy/internal/domain"
	"github.com/google/uuid"
)

func TestDryRun_AllowWithoutEnforce(t *testing.T) {
	ResetDryRunResults()
	e := NewEvaluator(nil, nil, nil)

	req := &domain.CheckRequest{
		TenantID:     uuid.New(),
		UserID:       uuid.New(),
		ResourceType: "document",
		Action:       "read",
		Resource:     "doc-1",
	}

	result, err := e.EvaluateDryRun(context.Background(), req)
	if err != nil {
		t.Fatalf("EvaluateDryRun: %v", err)
	}
	if result == nil {
		t.Fatal("result should not be nil")
	}
	if result.Reason == "" {
		t.Error("reason should explain what would happen")
	}
}

func TestDryRun_NilRequest(t *testing.T) {
	ResetDryRunResults()
	e := NewEvaluator(nil, nil, nil)

	result, _ := e.EvaluateDryRun(context.Background(), nil)
	if result.Allowed {
		t.Error("nil request should not be allowed")
	}
	if result.Reason != "nil request" {
		t.Errorf("expected 'nil request', got %s", result.Reason)
	}
}

func TestDryRun_LogsResults(t *testing.T) {
	ResetDryRunResults()
	e := NewEvaluator(nil, nil, nil)

	req := &domain.CheckRequest{
		TenantID: uuid.New(), UserID: uuid.New(),
		ResourceType: "file", Action: "write",
	}
	e.EvaluateDryRun(context.Background(), req)
	e.EvaluateDryRun(context.Background(), req)

	results := GetDryRunResults()
	if len(results) != 2 {
		t.Errorf("expected 2 logged results, got %d", len(results))
	}
}
