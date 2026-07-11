package service

import (
	"sync"
	"time"
)

// SoDRuleInfo is a JSON-friendly representation of a SoD rule.
type SoDRuleInfo struct {
	ID        string    `json:"id"`
	Roles     []string  `json:"roles"`
	Reason    string    `json:"reason"`
	CreatedAt time.Time `json:"created_at"`
}

var (
	sodRulesInfoMu sync.RWMutex
	sodRulesInfo   = []SoDRuleInfo{
		{Roles: []string{"admin", "auditor"}, Reason: "admin + auditor mutually exclusive"},
		{Roles: []string{"admin", "compliance"}, Reason: "admin + compliance mutually exclusive"},
	}
)

// GetSoDRules returns all active SoD rules as JSON-friendly structs.
func GetSoDRules() []SoDRuleInfo {
	sodRulesInfoMu.RLock()
	defer sodRulesInfoMu.RUnlock()
	result := make([]SoDRuleInfo, len(sodRulesInfo))
	copy(result, sodRulesInfo)
	return result
}
