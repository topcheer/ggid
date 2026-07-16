package detection

import (
	"github.com/ggid/ggid/services/audit/internal/domain"
)

// RuleRegistry holds built-in rules and per-tenant overrides.
type RuleRegistry struct {
	rules     []Rule
	overrides map[string]domain.RuleConfig // key: "tenantID:ruleID"
}

// NewRuleRegistry creates a registry with all built-in rules registered.
func NewRuleRegistry() *RuleRegistry {
	r := &RuleRegistry{
		overrides: make(map[string]domain.RuleConfig),
	}
	r.registerBuiltins()
	return r
}

func (r *RuleRegistry) Register(rule Rule) {
	r.rules = append(r.rules, rule)
}

func (r *RuleRegistry) SetOverride(tenantID string, ruleID string, cfg domain.RuleConfig) {
	r.overrides[tenantID+":"+ruleID] = cfg
}

// RulesFor returns rules that care about the given audit action.
func (r *RuleRegistry) RulesFor(action string) []Rule {
	var matching []Rule
	for _, rule := range r.rules {
		for _, a := range rule.Actions() {
			if a == "*" || a == action {
				matching = append(matching, rule)
				break
			}
		}
	}
	return matching
}

// ConfigFor returns the effective config for a rule (override or default).
func (r *RuleRegistry) ConfigFor(tenantID interface{ String() string }, ruleID string) domain.RuleConfig {
	key := tenantID.String() + ":" + ruleID
	if cfg, ok := r.overrides[key]; ok {
		return cfg
	}
	// Default: enabled with built-in severity.
	for _, rule := range r.rules {
		if rule.ID() == ruleID {
			return domain.RuleConfig{
				RuleID:  ruleID,
				Enabled: true,
			}
		}
	}
	return domain.RuleConfig{RuleID: ruleID, Enabled: false}
}

func (r *RuleRegistry) registerBuiltins() {
	r.Register(&BruteForceRule{})
	r.Register(&CredentialStuffingRule{})
	r.Register(&ImpossibleTravelRule{})
}

// All returns all registered rules.
func (r *RuleRegistry) All() []Rule {
	return r.rules
}
