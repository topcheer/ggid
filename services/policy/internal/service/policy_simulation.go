package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ggid/ggid/services/policy/internal/domain"
	"github.com/google/uuid"
)

// SimulationRequest is a sample request for policy simulation.
type SimulationRequest struct {
	UserID       uuid.UUID        `json:"user_id"`
	TenantID     uuid.UUID        `json:"tenant_id"`
	ResourceType string           `json:"resource_type"`
	Action       string           `json:"action"`
	Resource     string           `json:"resource"`
	Conditions   map[string]any   `json:"conditions,omitempty"`
}

// SimulationDecision is the result of a single simulation.
type SimulationDecision struct {
	Request       SimulationRequest `json:"request"`
	Allowed       bool              `json:"allowed"`
	Reason        string            `json:"reason"`
	MatchedBy     string            `json:"matched_by"`
	MatchedRules  []string          `json:"matched_rules"`
	MissedRules   []string          `json:"missed_rules"`
	EvaluatedAt   time.Time         `json:"evaluated_at"`
	Duration      time.Duration     `json:"duration"`
}

// SimulationTrace shows the step-by-step evaluation of a request.
type SimulationTrace struct {
	Steps []TraceStep `json:"steps"`
}

// TraceStep is a single step in the evaluation trace.
type TraceStep struct {
	RuleID    string `json:"rule_id"`
	RuleName  string `json:"rule_name"`
	Effect    string `json:"effect"` // allow | deny
	Matched   bool   `json:"matched"`
	Reason    string `json:"reason"`
}

// ImpactAnalysis summarizes the potential impact of a policy change.
type ImpactAnalysis struct {
	AffectedUsersCount int                `json:"affected_users_count"`
	TotalRequests      int                `json:"total_requests"`
	AllowedCount       int                `json:"allowed_count"`
	DeniedCount        int                `json:"denied_count"`
	ChangeCount        int                `json:"change_count"` // requests that flipped from allow->deny or deny->allow
}

// PolicySimulator simulates policy evaluation against sample requests.
type PolicySimulator struct {
	mu       sync.RWMutex
	evaluator *Evaluator
	results   map[string]*SimulationDecision // keyed by request hash
}

// NewPolicySimulator creates a new PolicySimulator.
func NewPolicySimulator(evaluator *Evaluator) *PolicySimulator {
	return &PolicySimulator{
		evaluator: evaluator,
		results:   make(map[string]*SimulationDecision),
	}
}

// SimulatePolicy evaluates a single sample request and returns the decision with a trace.
func (ps *PolicySimulator) SimulatePolicy(ctx context.Context, policyID uuid.UUID, req SimulationRequest) (*SimulationDecision, *SimulationTrace, error) {
	if req.TenantID == uuid.Nil {
		return nil, nil, fmt.Errorf("tenant_id is required")
	}
	start := time.Now()

	checkReq := &domain.CheckRequest{
		UserID:       req.UserID,
		TenantID:     req.TenantID,
		ResourceType: req.ResourceType,
		Action:       req.Action,
		Resource:     req.Resource,
		Conditions:   req.Conditions,
	}

	// Use dry-run evaluation if available, otherwise evaluate normally.
	var allowed bool
	var reason string
	var matchedBy string
	if ps.evaluator != nil {
		dr, err := ps.evaluator.EvaluateDryRun(ctx, checkReq)
		if err != nil {
			allowed = false
			reason = fmt.Sprintf("evaluation error: %v", err)
			matchedBy = "error"
		} else {
			allowed = dr.Allowed
			reason = dr.Reason
			matchedBy = "dry-run"
		}
	} else {
		allowed = true
		reason = "no evaluator configured"
		matchedBy = "default"
	}

	decision := &SimulationDecision{
		Request:      req,
		Allowed:      allowed,
		Reason:       reason,
		MatchedBy:    matchedBy,
		MatchedRules: []string{matchedBy},
		MissedRules:  []string{},
		EvaluatedAt:  time.Now(),
		Duration:     time.Since(start),
	}

	trace := &SimulationTrace{
		Steps: []TraceStep{
			{
				RuleID:   matchedBy,
				RuleName: matchedBy,
				Effect:   boolToEffect(allowed),
				Matched:  true,
				Reason:   reason,
			},
		},
	}

	ps.mu.Lock()
	ps.results[reqHash(req)] = decision
	ps.mu.Unlock()

	return decision, trace, nil
}

// SimulateBatch evaluates multiple sample requests and returns all decisions.
func (ps *PolicySimulator) SimulateBatch(ctx context.Context, policyID uuid.UUID, requests []SimulationRequest) ([]SimulationDecision, error) {
	if len(requests) == 0 {
		return nil, fmt.Errorf("no requests to simulate")
	}

	results := make([]SimulationDecision, 0, len(requests))
	for _, req := range requests {
		decision, _, err := ps.SimulatePolicy(ctx, policyID, req)
		if err != nil {
			results = append(results, SimulationDecision{
				Request: req,
				Allowed: false,
				Reason:  fmt.Sprintf("simulation error: %v", err),
				EvaluatedAt: time.Now(),
			})
			continue
		}
		results = append(results, *decision)
	}
	return results, nil
}

// AnalyzeImpact compares simulation results against baseline decisions to determine impact.
func (ps *PolicySimulator) AnalyzeImpact(decisions []SimulationDecision, baseline map[string]bool) *ImpactAnalysis {
	analysis := &ImpactAnalysis{
		TotalRequests: len(decisions),
	}
	userSet := make(map[uuid.UUID]bool)
	for _, d := range decisions {
		userSet[d.Request.UserID] = true
		if d.Allowed {
			analysis.AllowedCount++
		} else {
			analysis.DeniedCount++
		}
		hash := reqHash(d.Request)
		if baselineVal, ok := baseline[hash]; ok && baselineVal != d.Allowed {
			analysis.ChangeCount++
		}
	}
	analysis.AffectedUsersCount = len(userSet)
	return analysis
}

// GetResults returns all stored simulation results.
func (ps *PolicySimulator) GetResults() []SimulationDecision {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	results := make([]SimulationDecision, 0, len(ps.results))
	for _, d := range ps.results {
		results = append(results, *d)
	}
	return results
}

// Reset clears all simulation results.
func (ps *PolicySimulator) Reset() {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.results = make(map[string]*SimulationDecision)
}

func boolToEffect(allowed bool) string {
	if allowed {
		return "allow"
	}
	return "deny"
}

func reqHash(req SimulationRequest) string {
	return fmt.Sprintf("%s:%s:%s:%s", req.TenantID, req.UserID, req.ResourceType, req.Action)
}
