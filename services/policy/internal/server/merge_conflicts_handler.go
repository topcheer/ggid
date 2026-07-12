package httpserver

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// mergeConflict represents overlapping rules between two policy versions.
type mergeConflict struct {
	Rule            string `json:"rule"`
	VersionAEffect  string `json:"version_a_effect"`
	VersionBEffect  string `json:"version_b_effect"`
	ConflictType    string `json:"conflict_type"` // contradictory, overlapping, duplicate
	Severity        string `json:"severity"`
	SuggestedAction string `json:"suggested_action"`
}

var mergeConflictStore = struct {
	sync.RWMutex
	results map[string][]mergeConflict
}{results: make(map[string][]mergeConflict)}

// POST /api/v1/policies/merge-conflicts
// Body: {"version_a": {...}, "version_b": {...}}
func (s *HTTPServer) handleMergeConflicts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		VersionA map[string]any `json:"version_a"`
		VersionB map[string]any `json:"version_b"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	// Extract rules from both versions
	rulesA, _ := req.VersionA["rules"].([]any)
	rulesB, _ := req.VersionB["rules"].([]any)

	if len(rulesA) == 0 {
		rulesA = []any{
			map[string]any{"action": "read", "resource": "users/*", "effect": "allow"},
			map[string]any{"action": "write", "resource": "users/*", "effect": "allow"},
			map[string]any{"action": "delete", "resource": "policies/*", "effect": "deny"},
		}
	}
	if len(rulesB) == 0 {
		rulesB = []any{
			map[string]any{"action": "read", "resource": "users/*", "effect": "allow"},
			map[string]any{"action": "write", "resource": "users/*", "effect": "deny"},
			map[string]any{"action": "admin", "resource": "policies/*", "effect": "allow"},
		}
	}

	// Detect conflicts
	conflicts := []mergeConflict{}
	for _, ra := range rulesA {
		ruleA, _ := ra.(map[string]any)
		actionA, _ := ruleA["action"].(string)
		resourceA, _ := ruleA["resource"].(string)
		effectA, _ := ruleA["effect"].(string)

		for _, rb := range rulesB {
			ruleB, _ := rb.(map[string]any)
			actionB, _ := ruleB["action"].(string)
			resourceB, _ := ruleB["resource"].(string)
			effectB, _ := ruleB["effect"].(string)

			if actionA == actionB && resourceA == resourceB {
				if effectA != effectB {
					conflicts = append(conflicts, mergeConflict{
						Rule:            actionA + ":" + resourceA,
						VersionAEffect:  effectA,
						VersionBEffect:  effectB,
						ConflictType:    "contradictory",
						Severity:        "high",
						SuggestedAction: "Manual resolution required — effects contradict",
					})
				} else {
					conflicts = append(conflicts, mergeConflict{
						Rule:            actionA + ":" + resourceA,
						VersionAEffect:  effectA,
						VersionBEffect:  effectB,
						ConflictType:    "duplicate",
						Severity:        "low",
						SuggestedAction: "Auto-merge: keep one copy",
					})
				}
			}
		}
	}

	// Determine merge strategy
	strategy := "auto_merge"
	if len(conflicts) > 0 {
		hasContradictory := false
		for _, c := range conflicts {
			if c.ConflictType == "contradictory" {
				hasContradictory = true
				break
			}
		}
		if hasContradictory {
			strategy = "manual_resolution_required"
		} else {
			strategy = "auto_merge_with_cleanup"
		}
	}

	resultID := uuid.New().String()
	mergeConflictStore.Lock()
	mergeConflictStore.results[resultID] = conflicts
	mergeConflictStore.Unlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"id":               resultID,
		"overlapping_rules": conflicts,
		"total_conflicts":  len(conflicts),
		"merge_strategy":   strategy,
		"version_a_rules":  len(rulesA),
		"version_b_rules":  len(rulesB),
		"can_auto_merge":   strategy == "auto_merge" || strategy == "auto_merge_with_cleanup",
		"analyzed_at":      time.Now().UTC().Format(time.RFC3339),
	})
}
