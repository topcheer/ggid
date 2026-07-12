package server

import (
	"net/http"
	"sync"
)

// dependencyNode represents a client in the dependency graph.
type dependencyNode struct {
	ClientID    string   `json:"client_id"`
	ClientName  string   `json:"client_name"`
	Type        string   `json:"type"` // first_party, third_party, service
	SharedScopes []string `json:"shared_scopes"`
	DependsOn   []string `json:"depends_on"` // client IDs this client delegates to
	DelegatedBy []string `json:"delegated_by"` // client IDs that delegate to this one
}

var dependencyGraphStore = struct {
	sync.RWMutex
	nodes map[string]*dependencyNode
}{nodes: map[string]*dependencyNode{
	"web-app": {
		ClientID: "web-app", ClientName: "Web Application", Type: "first_party",
		SharedScopes: []string{"openid", "profile", "email"},
		DependsOn:    []string{"service-backend"},
		DelegatedBy:  []string{},
	},
	"mobile-ios": {
		ClientID: "mobile-ios", ClientName: "iOS Mobile App", Type: "first_party",
		SharedScopes: []string{"openid", "profile", "email", "offline_access"},
		DependsOn:    []string{"service-backend"},
		DelegatedBy:  []string{},
	},
	"admin-cli": {
		ClientID: "admin-cli", ClientName: "Admin CLI", Type: "first_party",
		SharedScopes: []string{"openid", "admin", "read:users", "read:audit"},
		DependsOn:    []string{},
		DelegatedBy:  []string{"service-backend"},
	},
	"service-backend": {
		ClientID: "service-backend", ClientName: "Backend Service", Type: "service",
		SharedScopes: []string{"read:users", "write:users", "read:audit"},
		DependsOn:    []string{},
		DelegatedBy:  []string{"web-app", "mobile-ios"},
	},
	"analytics-3p": {
		ClientID: "analytics-3p", ClientName: "Analytics Platform", Type: "third_party",
		SharedScopes: []string{"read:audit"},
		DependsOn:    []string{},
		DelegatedBy:  []string{"admin-cli"},
	},
}}

// GET /api/v1/oauth/clients/dependency-graph
// Returns client inter-dependencies: shared scopes, delegation chains.
func handleDependencyGraph(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	dependencyGraphStore.RLock()
	defer dependencyGraphStore.RUnlock()

	nodes := []*dependencyNode{}
	for _, n := range dependencyGraphStore.nodes {
		nodes = append(nodes, n)
	}

	// Build edges from depends_on
	edges := []map[string]any{}
	for _, n := range nodes {
		for _, dep := range n.DependsOn {
			edgeType := "delegation"
			// Check shared scopes
			if target, ok := dependencyGraphStore.nodes[dep]; ok {
				shared := 0
				for _, s := range n.SharedScopes {
					for _, ts := range target.SharedScopes {
						if s == ts {
							shared++
						}
					}
				}
				edges = append(edges, map[string]any{
					"from":         n.ClientID,
					"to":           dep,
					"type":         edgeType,
					"shared_scopes": shared,
				})
			}
		}
	}

	// Find delegation chains (depth > 1)
	delegationChains := [][]string{}
	for _, n := range nodes {
		if len(n.DependsOn) > 0 {
			for _, dep := range n.DependsOn {
				chain := []string{n.ClientID, dep}
				// Follow chain
				if target, ok := dependencyGraphStore.nodes[dep]; ok {
					for _, dep2 := range target.DependsOn {
						chain = append(chain, dep2)
						delegationChains = append(delegationChains, append([]string{}, chain...))
						chain = chain[:len(chain)-1] // backtrack
					}
				}
				if len(chain) == 2 {
					delegationChains = append(delegationChains, chain)
				}
			}
		}
	}

	// Compute impact analysis: if client X is revoked, which others are affected?
	impactAnalysis := map[string]any{}
	for _, n := range nodes {
		affected := []string{}
		for _, other := range nodes {
			if other.ClientID == n.ClientID {
				continue
			}
			for _, dep := range other.DependsOn {
				if dep == n.ClientID {
					affected = append(affected, other.ClientID)
				}
			}
		}
		if len(affected) > 0 {
			impactAnalysis[n.ClientID] = affected
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"nodes":              nodes,
		"edges":              edges,
		"delegation_chains":  delegationChains,
		"impact_analysis":    impactAnalysis,
		"total_clients":      len(nodes),
		"total_edges":        len(edges),
		"total_shared_scope_pairs": func() int {
			count := 0
			for _, e := range edges {
				if s, ok := e["shared_scopes"].(int); ok && s > 0 {
					count++
				}
			}
			return count
		}(),
	})
}
