package detection

import (
	"context"
	"fmt"
	"time"

	"github.com/ggid/ggid/services/audit/internal/domain"
	)

// BruteForceRule: same user, 5 failed logins in 5 minutes.
type BruteForceRule struct{}

func (r *BruteForceRule) ID() string         { return "brute_force" }
func (r *BruteForceRule) Name() string       { return "Brute Force Login" }
func (r *BruteForceRule) MITRE() string      { return "T1110" }
func (r *BruteForceRule) DefaultSeverity() domain.Severity { return domain.SeverityHigh }
func (r *BruteForceRule) Actions() []string  { return []string{"user.login"} }

func (r *BruteForceRule) Evaluate(ctx context.Context, evt *domain.AuditEvent, state StateStore, cfg domain.RuleConfig) (*domain.Detection, error) {
	// Only failed logins.
	if evt.Result != "failure" {
		return nil, nil
	}
	if evt.ActorID == nil {
		return nil, nil
	}

	// Threshold: default 5 failures in 5 minutes.
	threshold := int64(5)
	windowMin := 5
	if t, ok := cfg.Threshold["max_failures"].(float64); ok && t > 0 {
		threshold = int64(t)
	}

	key := fmt.Sprintf("bf:%s", evt.ActorID)
	count, err := state.Incr(ctx, key, time.Duration(windowMin)*time.Minute)
	if err != nil {
		return nil, err
	}
	if count < threshold {
		return nil, nil
	}

	actorID := *evt.ActorID
	det := domain.NewDetection(evt.TenantID, r.ID(), r.DefaultSeverity(), "Brute force login detected")
	det.ActorID = &actorID
	det.Detail = map[string]any{
		"failed_attempts": count,
		"window_minutes":  windowMin,
		"ip_address":      evt.IPAddress,
	}
	return det, nil
}
