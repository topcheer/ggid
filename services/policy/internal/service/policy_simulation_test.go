package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestPolicySimulation_SimulatePolicy(t *testing.T) {
	evaluator := NewEvaluator(nil, nil, nil)
	sim := NewPolicySimulator(evaluator)
	policyID := uuid.New()

	req := SimulationRequest{
		UserID:       uuid.New(),
		TenantID:     uuid.New(),
		ResourceType: "document",
		Action:       "read",
		Resource:     "doc-1",
	}

	decision, trace, err := sim.SimulatePolicy(context.Background(), policyID, req)
	if err != nil {
		t.Fatalf("SimulatePolicy: %v", err)
	}
	if decision == nil {
		t.Fatal("decision should not be nil")
	}
	if trace == nil {
		t.Fatal("trace should not be nil")
	}
	if len(trace.Steps) == 0 {
		t.Error("trace should have at least one step")
	}
	if decision.Request.Action != "read" {
		t.Errorf("expected action 'read', got '%s'", decision.Request.Action)
	}
}

func TestPolicySimulation_SimulatePolicy_NilTenant(t *testing.T) {
	sim := NewPolicySimulator(nil)
	req := SimulationRequest{
		UserID:   uuid.New(),
		Action:   "read",
		Resource: "doc-1",
	}
	_, _, err := sim.SimulatePolicy(context.Background(), uuid.New(), req)
	if err == nil {
		t.Error("should error on nil tenant")
	}
}

func TestPolicySimulation_SimulateBatch(t *testing.T) {
	evaluator := NewEvaluator(nil, nil, nil)
	sim := NewPolicySimulator(evaluator)
	policyID := uuid.New()
	tenantID := uuid.New()

	requests := []SimulationRequest{
		{UserID: uuid.New(), TenantID: tenantID, ResourceType: "doc", Action: "read", Resource: "a"},
		{UserID: uuid.New(), TenantID: tenantID, ResourceType: "doc", Action: "write", Resource: "b"},
		{UserID: uuid.New(), TenantID: tenantID, ResourceType: "doc", Action: "delete", Resource: "c"},
	}

	decisions, err := sim.SimulateBatch(context.Background(), policyID, requests)
	if err != nil {
		t.Fatalf("SimulateBatch: %v", err)
	}
	if len(decisions) != 3 {
		t.Errorf("expected 3 decisions, got %d", len(decisions))
	}
}

func TestPolicySimulation_SimulateBatch_Empty(t *testing.T) {
	sim := NewPolicySimulator(nil)
	_, err := sim.SimulateBatch(context.Background(), uuid.New(), nil)
	if err == nil {
		t.Error("should error on empty requests")
	}
}

func TestPolicySimulation_AnalyzeImpact(t *testing.T) {
	sim := NewPolicySimulator(nil)
	tenantID := uuid.New()
	user1 := uuid.New()
	user2 := uuid.New()

	decisions := []SimulationDecision{
		{Request: SimulationRequest{UserID: user1, TenantID: tenantID, Action: "read"}, Allowed: true},
		{Request: SimulationRequest{UserID: user2, TenantID: tenantID, Action: "write"}, Allowed: false},
		{Request: SimulationRequest{UserID: user1, TenantID: tenantID, Action: "delete"}, Allowed: true},
	}

	baseline := map[string]bool{
		reqHash(decisions[0].Request): false, // was denied, now allowed → change
		reqHash(decisions[1].Request): true, // was allowed, now denied → change
		reqHash(decisions[2].Request): true, // was allowed, still allowed → no change
	}

	analysis := sim.AnalyzeImpact(decisions, baseline)
	if analysis.TotalRequests != 3 {
		t.Errorf("expected 3 total requests, got %d", analysis.TotalRequests)
	}
	if analysis.AllowedCount != 2 {
		t.Errorf("expected 2 allowed, got %d", analysis.AllowedCount)
	}
	if analysis.DeniedCount != 1 {
		t.Errorf("expected 1 denied, got %d", analysis.DeniedCount)
	}
	if analysis.ChangeCount != 2 {
		t.Errorf("expected 2 changes, got %d", analysis.ChangeCount)
	}
	if analysis.AffectedUsersCount != 2 {
		t.Errorf("expected 2 affected users, got %d", analysis.AffectedUsersCount)
	}
}

func TestPolicySimulation_GetResults(t *testing.T) {
	sim := NewPolicySimulator(NewEvaluator(nil, nil, nil))
	sim.SimulatePolicy(context.Background(), uuid.New(), SimulationRequest{
		UserID:   uuid.New(),
		TenantID: uuid.New(),
		Action:   "read",
	})
	results := sim.GetResults()
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

func TestPolicySimulation_Reset(t *testing.T) {
	sim := NewPolicySimulator(NewEvaluator(nil, nil, nil))
	sim.SimulatePolicy(context.Background(), uuid.New(), SimulationRequest{
		UserID:   uuid.New(),
		TenantID: uuid.New(),
		Action:   "read",
	})
	sim.Reset()
	if len(sim.GetResults()) != 0 {
		t.Error("results should be empty after reset")
	}
}
