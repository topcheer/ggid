package service

import (
	"fmt"
	"sync"
	"time"

	"github.com/ggid/ggid/services/audit/internal/domain"
)

// CorrelationRule defines a pattern for correlating audit events.
type CorrelationRule struct {
	Name      string        `json:"name"`
	Pattern   string        `json:"pattern"`
	Window    time.Duration `json:"window"`
	MinEvents int           `json:"min_events"`
	Action    string        `json:"action"`
}

// CorrelationResult holds the result of correlating events.
type CorrelationResult struct {
	RuleName     string              `json:"rule_name"`
	Events       []domain.AuditEvent `json:"events"`
	Score        float64             `json:"score"`
	Action       string              `json:"action"`
	CorrelatedAt time.Time           `json:"correlated_at"`
}

// EventCorrelator correlates audit events based on rules.
type EventCorrelator struct {
	mu      sync.RWMutex
	rules   []CorrelationRule
	results []CorrelationResult
}

// NewEventCorrelator creates a new EventCorrelator.
func NewEventCorrelator() *EventCorrelator {
	return &EventCorrelator{}
}

// AddRule adds a correlation rule.
func (ec *EventCorrelator) AddRule(rule CorrelationRule) {
	if rule.Window <= 0 {
		rule.Window = 5 * time.Minute
	}
	if rule.MinEvents <= 0 {
		rule.MinEvents = 2
	}
	ec.mu.Lock()
	defer ec.mu.Unlock()
	ec.rules = append(ec.rules, rule)
}

// CorrelateAuditEvents correlates events against all rules using sliding windows.
func (ec *EventCorrelator) CorrelateAuditEvents(events []domain.AuditEvent) []CorrelationResult {
	ec.mu.RLock()
	rules := make([]CorrelationRule, len(ec.rules))
	copy(rules, ec.rules)
	ec.mu.RUnlock()

	var results []CorrelationResult
	seen := make(map[string]bool)

	for _, rule := range rules {
		ruleResults := ec.applyRule(events, rule)
		for _, r := range ruleResults {
			dedupKey := fmt.Sprintf("%s:%d", r.RuleName, len(r.Events))
			if !seen[dedupKey] {
				seen[dedupKey] = true
				results = append(results, r)
			}
		}
	}

	ec.mu.Lock()
	ec.results = append(ec.results, results...)
	ec.mu.Unlock()
	return results
}

func (ec *EventCorrelator) applyRule(events []domain.AuditEvent, rule CorrelationRule) []CorrelationResult {
	var matching []domain.AuditEvent
	for _, e := range events {
		if ec.matchPattern(e.Action, rule.Pattern) {
			matching = append(matching, e)
		}
	}
	if len(matching) < rule.MinEvents {
		return nil
	}

	var results []CorrelationResult
	for i := 0; i < len(matching); i++ {
		var windowEvents []domain.AuditEvent
		for j := i; j < len(matching); j++ {
			if matching[j].CreatedAt.Sub(matching[i].CreatedAt) <= rule.Window {
				windowEvents = append(windowEvents, matching[j])
			} else {
				break
			}
		}
		if len(windowEvents) >= rule.MinEvents {
			// False positive filter: single actor with high min events.
			actorSet := make(map[string]bool)
			for _, e := range windowEvents {
				actorSet[string(e.ActorType)+":"+e.ActorName] = true
			}
			if len(actorSet) == 1 && rule.MinEvents > 3 {
				continue
			}
			results = append(results, CorrelationResult{
				RuleName:     rule.Name,
				Events:       windowEvents,
				Score:        ec.calculateScore(windowEvents, rule),
				Action:       rule.Action,
				CorrelatedAt: time.Now(),
			})
			i += len(windowEvents) - 1
		}
	}
	return results
}

func (ec *EventCorrelator) matchPattern(action, pattern string) bool {
	if pattern == "*" {
		return true
	}
	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(action) >= len(prefix) && action[:len(prefix)] == prefix
	}
	return action == pattern
}

func (ec *EventCorrelator) calculateScore(events []domain.AuditEvent, rule CorrelationRule) float64 {
	if len(events) == 0 || rule.MinEvents == 0 {
		return 0
	}
	score := float64(len(events)) / float64(rule.MinEvents)
	if score > 1.0 {
		score = 1.0
	}
	actorSet := make(map[string]bool)
	ipSet := make(map[string]bool)
	for _, e := range events {
		actorSet[string(e.ActorType)+":"+e.ActorName] = true
		if e.IPAddress != "" {
			ipSet[e.IPAddress] = true
		}
	}
	if len(actorSet) > 1 {
		score = score + 0.2
		if score > 1.0 {
			score = 1.0
		}
	}
	if len(ipSet) == 1 {
		score = score + 0.1
		if score > 1.0 {
			score = 1.0
		}
	}
	return score
}

// GetResults returns all stored correlation results.
func (ec *EventCorrelator) GetResults() []CorrelationResult {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	result := make([]CorrelationResult, len(ec.results))
	copy(result, ec.results)
	return result
}

// Reset clears all rules and results.
func (ec *EventCorrelator) Reset() {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	ec.rules = nil
	ec.results = nil
}
