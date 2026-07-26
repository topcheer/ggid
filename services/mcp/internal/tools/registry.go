// Package tools defines MCP tools exposed to LLM agents.
package tools

import (
	"context"
	"fmt"

	"github.com/ggid/ggid/services/mcp/internal/client"
)

// Tool represents a single MCP tool callable by LLM agents.
type Tool struct {
	Name           string         `json:"name"`
	Description    string         `json:"description"`
	InputSchema    map[string]any `json:"inputSchema"`
	RequiredScopes []string       `json:"-"`
	Handler        ToolHandler     `json:"-"`
}

// ToolHandler executes a tool and returns its result.
type ToolHandler func(ctx context.Context, c *client.Client, args map[string]any) (any, error)

// Registry holds all registered tools.
type Registry struct {
	tools []Tool
}

// NewRegistry creates and populates the tool registry with all tools.
func NewRegistry() *Registry {
	r := &Registry{}
	r.register(userTools...)
	r.register(roleTools...)
	r.register(policyTools...)
	r.register(auditTools...)
	r.register(adminTools...)
	return r
}

func (r *Registry) register(tools ...Tool) {
	r.tools = append(r.tools, tools...)
}

// All returns every registered tool.
func (r *Registry) All() []Tool {
	return r.tools
}

// FilterByScopes returns only tools whose RequiredScopes are satisfied.
func (r *Registry) FilterByScopes(scopes []string) []Tool {
	scopeSet := make(map[string]bool, len(scopes))
	for _, s := range scopes {
		scopeSet[s] = true
	}
	var available []Tool
	for _, t := range r.tools {
		// SECURITY: Tools with no RequiredScopes should NOT be accessible
		// to anyone - they must explicitly declare required scopes.
		if len(t.RequiredScopes) == 0 {
			continue // Skip tools with no scope requirements
		}
		// Even admin scopes must satisfy RequiredScopes - this prevents
		// tenant:admin from tenant A from accessing tools intended for tenant B.
		if hasAllScopes(scopeSet, t.RequiredScopes) {
			available = append(available, t)
		}
	}
	return available
}

// Find returns a tool by name.
func (r *Registry) Find(name string) (*Tool, bool) {
	for i := range r.tools {
		if r.tools[i].Name == name {
			return &r.tools[i], true
		}
	}
	return nil, false
}

func hasAllScopes(have map[string]bool, need []string) bool {
	// SECURITY: If RequiredScopes is nil, treat it as empty (no access allowed)
	if need == nil {
		return false
	}
	for _, s := range need {
		if !have[s] {
			return false
		}
	}
	return true
}

// argStr extracts a string arg, returning "" if missing.
func argStr(args map[string]any, key string) string {
	if v, ok := args[key]; ok {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

// argAny extracts an arg with a default value.
func argAny(args map[string]any, key string, def any) any {
	if v, ok := args[key]; ok {
		return v
	}
	return def
}

// argStrSlice extracts a []string arg, accepting []any or []string.
func argStrSlice(args map[string]any, key string) []string {
	if v, ok := args[key]; ok {
		switch s := v.(type) {
		case []string:
			return s
		case []any:
			result := make([]string, 0, len(s))
			for _, item := range s {
				result = append(result, fmt.Sprintf("%v", item))
			}
			return result
		}
	}
	return nil
}

// argInt extracts an int arg.
func argInt(args map[string]any, key string, def int) int {
	if v, ok := args[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		}
	}
	return def
}

// RegisterTool is the exported wrapper for register, allowing external
// packages (e.g., tests in server package) to register test tools.
func (r *Registry) RegisterTool(tools ...Tool) {
	r.register(tools...)
}
