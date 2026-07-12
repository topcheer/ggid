package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// POST /api/v1/policy/versions/{id}/diff
// Body: {"version_a": 1, "version_b": 2}
func (s *HTTPServer) handlePolicyVersionDiff(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract policy ID from path
	policyID := strings.TrimPrefix(r.URL.Path, "/api/v1/policy/versions/")
	policyID = strings.TrimSuffix(policyID, "/diff")
	policyID = strings.TrimSuffix(policyID, "/")
	if policyID == "" {
		writeJSONError(w, http.StatusBadRequest, "policy_id is required")
		return
	}

	var req struct {
		VersionA int `json:"version_a"`
		VersionB int `json:"version_b"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.VersionA == 0 {
		req.VersionA = 1
	}
	if req.VersionB == 0 {
		req.VersionB = 2
	}

	// Simulated version data
	versionA := map[string]any{
		"version": req.VersionA, "name": "Allow Read Access",
		"effect": "allow", "actions": []string{"read"}, "resources": []string{"users/*"},
		"priority": 10,
	}
	versionB := map[string]any{
		"version": req.VersionB, "name": "Allow Read-Write Access",
		"effect": "allow", "actions": []string{"read", "write"}, "resources": []string{"users/*", "audit/*"},
		"priority": 15,
	}

	// Compute diff
	added := []map[string]any{}
	removed := []map[string]any{}
	modified := []map[string]any{}

	// Check actions
	aActions := toSlice(versionA["actions"])
	bActions := toSlice(versionB["actions"])
	for _, a := range bActions {
		if !sliceContains(aActions, a) {
			added = append(added, map[string]any{"field": "actions", "value": a})
		}
	}
	for _, a := range aActions {
		if !sliceContains(bActions, a) {
			removed = append(removed, map[string]any{"field": "actions", "value": a})
		}
	}

	// Check resources
	aRes := toSlice(versionA["resources"])
	bRes := toSlice(versionB["resources"])
	for _, res := range bRes {
		if !sliceContains(aRes, res) {
			added = append(added, map[string]any{"field": "resources", "value": res})
		}
	}

	// Check scalar fields
	if versionA["name"] != versionB["name"] {
		modified = append(modified, map[string]any{"field": "name", "old": versionA["name"], "new": versionB["name"]})
	}
	if versionA["priority"] != versionB["priority"] {
		modified = append(modified, map[string]any{"field": "priority", "old": versionA["priority"], "new": versionB["priority"]})
	}

	breakingChanges := len(removed) > 0

	writeJSON(w, http.StatusOK, map[string]any{
		"policy_id":     policyID,
		"version_a":     req.VersionA,
		"version_b":     req.VersionB,
		"field_changes": map[string]any{
			"added":    added,
			"removed":  removed,
			"modified": modified,
		},
		"total_added":     len(added),
		"total_removed":   len(removed),
		"total_modified":  len(modified),
		"breaking_changes": breakingChanges,
		"impact_summary": map[string]string{
			"scope":       "expanded_access",
			"description": "Version B grants additional read+write on audit/* and write on users/*",
			"risk":        func() string {
				if breakingChanges {
					return "high — breaking changes detected"
				}
				return "low — additive changes only"
			}(),
		},
		"version_a_snapshot": versionA,
		"version_b_snapshot": versionB,
		"diffed_at":          time.Now().UTC().Format(time.RFC3339),
	})
}

func toSlice(v any) []string {
	if s, ok := v.([]string); ok {
		return s
	}
	if arr, ok := v.([]any); ok {
		result := []string{}
		for _, item := range arr {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
		return result
	}
	return []string{}
}

func sliceContains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
