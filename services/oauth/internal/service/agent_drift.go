package service

import (
	"sync"
	"time"
)

type DriftType string

const (
	DriftScopeExpansion  DriftType = "scope_expansion"
	DriftNewMCPTool       DriftType = "new_mcp_tool_access"
	DriftUnauthorizedOp  DriftType = "unauthorized_operation"
)

type DriftReport struct {
	AgentID         string     `json:"agent_id"`
	DetectedScopes  []string   `json:"detected_scopes"`
	DeclaredScopes  []string   `json:"declared_scopes"`
	DriftType       DriftType  `json:"drift_type"`
	Severity        string     `json:"severity"`
	DetectedAt      time.Time  `json:"detected_at"`
	Description     string     `json:"description"`
}

type DriftDetector struct {
	mu     sync.RWMutex
	reports map[string][]DriftReport
}

func NewDriftDetector() *DriftDetector {
	return &DriftDetector{reports: make(map[string][]DriftReport)}
}

func (d *DriftDetector) DetectDrift(agentID string, detectedScopes, declaredScopes []string, mcpToolsAccessed []string, declaredMCPTools []string) *DriftReport {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check for scope expansion
	declaredSet := make(map[string]bool)
	for _, s := range declaredScopes {
		declaredSet[s] = true
	}
	var extraScopes []string
	for _, s := range detectedScopes {
		if !declaredSet[s] {
			extraScopes = append(extraScopes, s)
		}
	}

	if len(extraScopes) > 0 {
		report := &DriftReport{
			AgentID:        agentID,
			DetectedScopes: detectedScopes,
			DeclaredScopes: declaredScopes,
			DriftType:      DriftScopeExpansion,
			Severity:       "high",
			DetectedAt:     time.Now(),
			Description:    "agent has scopes not in declared set: " + driftJoinScopes(extraScopes),
		}
		d.reports[agentID] = append(d.reports[agentID], *report)
		return report
	}

	// Check for unauthorized MCP tool access
	declaredToolSet := make(map[string]bool)
	for _, t := range declaredMCPTools {
		declaredToolSet[t] = true
	}
	var unknownTools []string
	for _, t := range mcpToolsAccessed {
		if !declaredToolSet[t] {
			unknownTools = append(unknownTools, t)
		}
	}

	if len(unknownTools) > 0 {
		report := &DriftReport{
			AgentID:        agentID,
			DetectedScopes: detectedScopes,
			DeclaredScopes: declaredScopes,
			DriftType:      DriftNewMCPTool,
			Severity:       "medium",
			DetectedAt:     time.Now(),
			Description:    "agent accessed undeclared MCP tools: " + driftJoinScopes(unknownTools),
		}
		d.reports[agentID] = append(d.reports[agentID], *report)
		return report
	}

	return nil
}

func (d *DriftDetector) GetReports(agentID string) []DriftReport {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.reports[agentID]
}

func driftJoinScopes(s []string) string {
	result := ""
	for i, v := range s {
		if i > 0 {
			result += ", "
		}
		result += v
	}
	return result
}