package server

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// AgentAuditEntry captures a single MCP tool invocation by an agent or user.
type AgentAuditEntry struct {
	Timestamp time.Time      `json:"timestamp"`
	Tool      string         `json:"tool"`
	Status    string         `json:"status"`
	UserID    string         `json:"user_id,omitempty"`
	TenantID  string         `json:"tenant_id,omitempty"`
	AgentID   string         `json:"agent_id,omitempty"`
	AgentType string         `json:"agent_type,omitempty"`
	ActorSub  string         `json:"actor_sub,omitempty"` // who delegated to this agent
	Args      map[string]any `json:"args,omitempty"`
}

// AgentAuditLog is a thread-safe ring buffer for recent MCP tool invocations.
// In production, entries should also be published to the audit service via NATS.
type AgentAuditLog struct {
	mu      sync.RWMutex
	entries []AgentAuditEntry
	maxLen  int
}

// NewAgentAuditLog creates an audit log with a 1000-entry ring buffer.
func NewAgentAuditLog() *AgentAuditLog {
	return &AgentAuditLog{
		entries: make([]AgentAuditEntry, 0, 1000),
		maxLen:  1000,
	}
}

// Append adds an entry, trimming oldest when at capacity.
func (a *AgentAuditLog) Append(e *AgentAuditEntry) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.entries = append(a.entries, *e)
	if len(a.entries) > a.maxLen {
		a.entries = a.entries[len(a.entries)-a.maxLen:]
	}
}

// Recent returns the last N audit entries.
func (a *AgentAuditLog) Recent(n int) []AgentAuditEntry {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if n > len(a.entries) {
		n = len(a.entries)
	}
	return a.entries[len(a.entries)-n:]
}

// ByAgent returns recent entries for a specific agent ID.
func (a *AgentAuditLog) ByAgent(agentID string, n int) []AgentAuditEntry {
	a.mu.RLock()
	defer a.mu.RUnlock()
	var result []AgentAuditEntry
	for i := len(a.entries) - 1; i >= 0 && len(result) < n; i-- {
		if a.entries[i].AgentID == agentID {
			result = append(result, a.entries[i])
		}
	}
	return result
}

// HandleAuditQuery handles GET /mcp/audit — returns recent agent tool invocations.
func (s *Server) HandleAuditQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	limit := 50
	recent := s.auditLog.Recent(limit)
	writeJSON(w, http.StatusOK, map[string]any{
		"entries": recent,
		"count":   len(recent),
	})
}

// MarshalJSON for AgentAuditEntry ensures consistent output.
func (e AgentAuditEntry) MarshalJSON() ([]byte, error) {
	type alias AgentAuditEntry
	return json.Marshal(alias(e))
}
