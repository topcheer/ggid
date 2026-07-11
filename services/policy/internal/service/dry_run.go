package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ggid/ggid/services/policy/internal/domain"
)

// DryRunResult captures what would happen if a policy were enforced.
type DryRunResult struct {
	ResourceType string
	Action       string
	Allowed      bool
	Reason       string
	EvaluatedAt  time.Time
}

var (
	dryRunMu      sync.Mutex
	dryRunResults = []*DryRunResult{}
)

// EvaluateDryRun evaluates a permission check WITHOUT enforcing the decision.
// Returns what the policy engine WOULD decide, with detailed reasoning.
func (e *Evaluator) EvaluateDryRun(ctx context.Context, req *domain.CheckRequest) (*DryRunResult, error) {
	result := &DryRunResult{
		EvaluatedAt: time.Now().UTC(),
	}

	if req == nil {
		result.Allowed = false
		result.Reason = "nil request"
		return result, nil
	}

	result.ResourceType = req.ResourceType
	result.Action = req.Action

	decision, err := e.Check(ctx, req)
	if err != nil {
		result.Allowed = false
		result.Reason = fmt.Sprintf("evaluation error: %v", err)
	} else {
		result.Allowed = decision.Allowed
		if decision.Allowed {
			result.Reason = fmt.Sprintf("policy would ALLOW: %s", decision.Reason)
		} else {
			result.Reason = fmt.Sprintf("policy would DENY: %s", decision.Reason)
		}
	}

	dryRunMu.Lock()
	dryRunResults = append(dryRunResults, result)
	dryRunMu.Unlock()

	return result, nil
}

// GetDryRunResults returns accumulated dry-run results.
func GetDryRunResults() []*DryRunResult {
	dryRunMu.Lock()
	defer dryRunMu.Unlock()
	out := make([]*DryRunResult, len(dryRunResults))
	copy(out, dryRunResults)
	return out
}

// ResetDryRunResults clears results (for testing).
func ResetDryRunResults() {
	dryRunMu.Lock()
	defer dryRunMu.Unlock()
	dryRunResults = dryRunResults[:0]
}
