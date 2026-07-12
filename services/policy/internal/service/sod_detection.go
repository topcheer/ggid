package service

import (
	"fmt"
	"sync"
)

type ConflictLevel string

const (
	ConflictStrict   ConflictLevel = "strict"
	ConflictModerate ConflictLevel = "moderate"
	ConflictRelaxed  ConflictLevel = "relaxed"
)

type SODRule struct {
	RuleID        string        `json:"rule_id"`
	RoleA         string        `json:"role_a"`
	RoleB         string        `json:"role_b"`
	ConflictLevel ConflictLevel `json:"conflict_level"`
}

type SODConflict struct {
	RuleID    string        `json:"rule_id"`
	RoleA     string        `json:"role_a"`
	RoleB     string        `json:"role_b"`
	Level     ConflictLevel `json:"level"`
	Suggestion string       `json:"suggestion"`
}

type SODDetectionService struct {
	mu    sync.RWMutex
	rules map[string]*SODRule
	seq   int
}

func NewSODDetectionService() *SODDetectionService {
	return &SODDetectionService{rules: make(map[string]*SODRule)}
}

func (s *SODDetectionService) CreateSODRule(roleA, roleB string, level ConflictLevel) *SODRule {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	rule := &SODRule{
		RuleID:        fmt.Sprintf("sod_%d", s.seq),
		RoleA:         roleA,
		RoleB:         roleB,
		ConflictLevel: level,
	}
	s.rules[rule.RuleID] = rule
	return rule
}

func (s *SODDetectionService) CheckSODConflict(userID string, roles []string) []SODConflict {
	s.mu.RLock()
	defer s.mu.RUnlock()
	roleSet := make(map[string]bool)
	for _, r := range roles {
		roleSet[r] = true
	}
	var conflicts []SODConflict
	for _, rule := range s.rules {
		if roleSet[rule.RoleA] && roleSet[rule.RoleB] {
			conflicts = append(conflicts, SODConflict{
				RuleID:     rule.RuleID,
				RoleA:      rule.RoleA,
				RoleB:      rule.RoleB,
				Level:      rule.ConflictLevel,
				Suggestion: remediationFor(rule),
			})
		}
	}
	return conflicts
}

func (s *SODDetectionService) ListSODRules() []*SODRule {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var list []*SODRule
	for _, r := range s.rules {
		list = append(list, r)
	}
	return list
}

func (s *SODDetectionService) DeleteSODRule(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.rules[id]; !ok {
		return fmt.Errorf("rule not found")
	}
	delete(s.rules, id)
	return nil
}

func remediationFor(rule *SODRule) string {
	switch rule.ConflictLevel {
	case ConflictStrict:
		return "Remove one of the conflicting roles immediately"
	case ConflictModerate:
		return "Request manager approval for dual-role assignment"
	case ConflictRelaxed:
		return "Monitor access patterns for this role combination"
	}
	return "Review role assignment"
}