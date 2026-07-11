// Package retention provides structured audit data retention policies.
//
// A RetentionPolicy defines how long audit events are kept before deletion.
// It supports both time-based (MaxAge) and count-based (MaxEvents) limits.
package retention

import (
	"context"
	"log/slog"
	"time"
)

// EventDeleter is the interface for deleting audit events.
// Implementations typically wrap an audit repository.
type EventDeleter interface {
	DeleteOlderThan(ctx context.Context, before time.Time) (int64, error)
	Count(ctx context.Context) (int64, error)
	DeleteExcess(ctx context.Context, keep int64) (int64, error)
}

// RetentionPolicy defines audit event retention rules.
type RetentionPolicy struct {
	// MaxAge is the maximum age of events to keep. Events older than this are deleted.
	// Zero means no time-based deletion.
	MaxAge time.Duration

	// MaxEvents is the maximum number of events to keep. Excess events (oldest first)
	// are deleted. Zero means no count-based deletion.
	MaxEvents int64

	// Enabled controls whether the policy is active.
	Enabled bool
}

// Result describes the outcome of an Apply() run.
type Result struct {
	DeletedByAge   int64     `json:"deleted_by_age"`
	DeletedByCount int64     `json:"deleted_by_count"`
	TotalDeleted   int64     `json:"total_deleted"`
	Remaining      int64     `json:"remaining"`
	AppliedAt      time.Time `json:"applied_at"`
}

// Apply executes the retention policy against the given deleter.
// It first deletes events older than MaxAge, then trims to MaxEvents if set.
// Returns a Result with deletion counts, or an error.
func (p *RetentionPolicy) Apply(ctx context.Context, deleter EventDeleter) (*Result, error) {
	result := &Result{AppliedAt: time.Now()}

	if !p.Enabled {
		slog.Debug("retention policy disabled, skipping")
		return result, nil
	}

	// Phase 1: Delete by age
	if p.MaxAge > 0 {
		cutoff := time.Now().Add(-p.MaxAge)
		deleted, err := deleter.DeleteOlderThan(ctx, cutoff)
		if err != nil {
			return result, err
		}
		result.DeletedByAge = deleted
		slog.Info("retention: deleted by age", "count", deleted, "cutoff", cutoff)
	}

	// Phase 2: Delete excess by count
	if p.MaxEvents > 0 {
		count, err := deleter.Count(ctx)
		if err != nil {
			return result, err
		}
		if count > p.MaxEvents {
			deleted, err := deleter.DeleteExcess(ctx, p.MaxEvents)
			if err != nil {
				return result, err
			}
			result.DeletedByCount = deleted
			slog.Info("retention: deleted by count", "count", deleted, "kept", p.MaxEvents)
		}
	}

	result.TotalDeleted = result.DeletedByAge + result.DeletedByCount

	remaining, err := deleter.Count(ctx)
	if err != nil {
		return result, nil // non-fatal
	}
	result.Remaining = remaining

	return result, nil
}

// NewDefaultPolicy returns a policy with 90-day retention, no count limit.
func NewDefaultPolicy() *RetentionPolicy {
	return &RetentionPolicy{
		MaxAge:   90 * 24 * time.Hour,
		Enabled:  true,
	}
}

// NewDaysPolicy returns a policy with the given retention period in days.
func NewDaysPolicy(days int) *RetentionPolicy {
	return &RetentionPolicy{
		MaxAge:  time.Duration(days) * 24 * time.Hour,
		Enabled: true,
	}
}
