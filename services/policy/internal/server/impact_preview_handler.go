package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// POST /api/v1/policies/impact-preview
// Body: {"policy_id": "...", "changes": {...}}
// Previews impact of policy changes without modifying actual policy.
func (s *HTTPServer) handleImpactPreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		PolicyID string         `json:"policy_id"`
		Changes  map[string]any `json:"changes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.PolicyID == "" {
		req.PolicyID = "pol-preview"
	}

	// Simulated impact analysis
	affectedUsers := []map[string]any{
		{"user_id": "user-001", "username": "alice.eng", "impact": "loses_access", "resources": []string{"users/*", "audit/*"}},
		{"user_id": "user-003", "username": "carol.sec", "impact": "gains_access", "resources": []string{"policies/*"}},
		{"user_id": "user-005", "username": "eve.dev", "impact": "permission_modified", "resources": []string{"users/*"}},
		{"user_id": "user-008", "username": "henry.ops", "impact": "loses_access", "resources": []string{"config/*"}},
	}

	affectedResources := []map[string]any{
		{"resource": "users/*", "current_accessors": 145, "after_accessors": 142, "delta": -3},
		{"resource": "audit/*", "current_accessors": 38, "after_accessors": 37, "delta": -1},
		{"resource": "policies/*", "current_accessors": 12, "after_accessors": 13, "delta": 1},
		{"resource": "config/*", "current_accessors": 8, "after_accessors": 7, "delta": -1},
	}

	riskLevel := "medium"
	riskScore := 35
	breakingCount := 0
	for _, u := range affectedUsers {
		if u["impact"] == "loses_access" {
			breakingCount++
		}
	}
	if breakingCount > 5 {
		riskLevel = "high"
		riskScore = 65
	} else if breakingCount > 10 {
		riskLevel = "critical"
		riskScore = 85
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"preview_id":           uuid.New().String(),
		"policy_id":            req.PolicyID,
		"affected_users":       affectedUsers,
		"affected_resources":   affectedResources,
		"total_affected_users": len(affectedUsers),
		"breaking_changes":     breakingCount,
		"estimated_risk_level": riskLevel,
		"risk_score":           riskScore,
		"summary": map[string]string{
			"loses_access":       fmt.Sprintf("%d users will lose access", countImpact(affectedUsers, "loses_access")),
			"gains_access":       fmt.Sprintf("%d users will gain access", countImpact(affectedUsers, "gains_access")),
			"permission_modified": fmt.Sprintf("%d users will have modified permissions", countImpact(affectedUsers, "permission_modified")),
		},
		"previewed_at": time.Now().UTC().Format(time.RFC3339),
		"is_preview":   true,
	})
}

func countImpact(users []map[string]any, impact string) int {
	count := 0
	for _, u := range users {
		if u["impact"] == impact {
			count++
		}
	}
	return count
}
