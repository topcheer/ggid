// Package detection implements the ITDR detection engine.
package detection

import (
	"context"
	"log/slog"
	"time"

	"github.com/ggid/ggid/services/audit/internal/domain"
	"github.com/google/uuid"
)

// Rule evaluates a single audit event and returns a Detection if matched.
type Rule interface {
	ID() string
	Name() string
	MITRE() string
	DefaultSeverity() domain.Severity
	Actions() []string // which audit actions this rule cares about
	Evaluate(ctx context.Context, evt *domain.AuditEvent, state StateStore, cfg domain.RuleConfig) (*domain.Detection, error)
}

// StateStore provides sliding-window state for stateful rules.
type StateStore interface {
	AddEvent(ctx context.Context, key string, ts int64, member string, windowTTL time.Duration) error
	EventsSince(ctx context.Context, key string, since int64) ([]string, error)
	Incr(ctx context.Context, key string, windowTTL time.Duration) (int64, error)
}

// DetectionRepo is the minimum interface the engine needs for persisting detections.
type DetectionRepo interface {
	InsertDetection(ctx context.Context, d *domain.Detection) error
}

// DetectionCallback is invoked after a detection is persisted.
// Used for ITDR → CAE linkage: critical detections can trigger session revocation.
type DetectionCallback func(ctx context.Context, det *domain.Detection)

// Engine evaluates audit events against registered rules.
type Engine struct {
	registry     *RuleRegistry
	state        StateStore
	repo         DetectionRepo
	callback     DetectionCallback // optional: invoked after each detection is persisted
	threatIntel  ThreatIntelChecker // optional: checks external threat indicators
}

// NewEngine creates a detection engine.
func NewEngine(repo DetectionRepo, state StateStore) *Engine {
	return &Engine{
		registry: NewRuleRegistry(),
		state:    state,
		repo:     repo,
	}
}

// SetThreatIntelChecker injects the threat intel checker for ITDR enrichment.
// When set, the engine will query threat indicators during Evaluate and
// auto-registers the threat_intel_hit rule.
func (e *Engine) SetThreatIntelChecker(checker ThreatIntelChecker) {
	e.threatIntel = checker
	if checker != nil {
		e.registry.Register(NewThreatIntelRule(checker))
	}
}

// SetCallback injects a post-detection callback (e.g. for CAE session revocation).
func (e *Engine) SetCallback(cb DetectionCallback) {
	e.callback = cb
}

// Registry returns the rule registry so callers can register custom rules.
func (e *Engine) Registry() *RuleRegistry {
	return e.registry
}

// Evaluate evaluates an audit event against all matching rules.
// A single rule failure is logged and does not block other rules.
func (e *Engine) Evaluate(ctx context.Context, evt *domain.AuditEvent) {
	if evt == nil || e.registry == nil {
		return
	}

	// Recover from panics — never block the audit pipeline.
	defer func() {
		if r := recover(); r != nil {
			slog.Error("ITDR engine panic", "error", r, "action", evt.Action)
		}
	}()

	rules := e.registry.RulesFor(evt.Action)
	for _, rule := range rules {
		cfg := e.registry.ConfigFor(evt.TenantID, rule.ID())
		if !cfg.Enabled {
			continue
		}

		det, err := rule.Evaluate(ctx, evt, e.state, cfg)
		if err != nil {
			slog.Warn("ITDR rule evaluate error", "rule", rule.ID(), "error", err)
			continue
		}
		if det != nil {
			// Set event ID for evidence trail.
			det.EventIDs = appendUniqueUUID(det.EventIDs, evt.ID)
			if e.repo != nil {
				if err := e.repo.InsertDetection(ctx, det); err != nil {
					slog.Warn("ITDR insert detection error", "rule", rule.ID(), "error", err)
				}
			}
			// ITDR → CAE linkage: invoke callback (e.g. publish session.revoke for critical).
			if e.callback != nil {
				e.callback(ctx, det)
			}
		}
	}
}

func appendUniqueUUID(ids []uuid.UUID, id uuid.UUID) []uuid.UUID {
	for _, existing := range ids {
		if existing == id {
			return ids
		}
	}
	return append(ids, id)
}
