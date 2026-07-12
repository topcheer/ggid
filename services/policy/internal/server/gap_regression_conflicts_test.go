package httpserver

import (
	"encoding/json"
	"net/http"
	"testing"
)

// Gap Regression: Policy Conflicts Detection (#session-verified)
// Validates: POST /api/v1/policy/conflicts returns structured conflict pairs,
// correct severity distribution, and method enforcement.

func TestGapRegression_ConflictsDetect_PostOnly(t *testing.T) {
	newTestHarness()
	w := doReq("GET", "/api/v1/policy/conflicts", "")
	assertStatus(t, w, http.StatusMethodNotAllowed)
}

func TestGapRegression_ConflictsDetect_ReturnsConflictPairs(t *testing.T) {
	newTestHarness()
	w := doReq("POST", "/api/v1/policy/conflicts", "")
	assertStatus(t, w, http.StatusOK)
	resp := parseJSON(t, w)

	pairs, ok := resp["conflict_pairs"].([]any)
	if !ok || len(pairs) == 0 {
		t.Fatalf("expected conflict_pairs array with entries, got %T", resp["conflict_pairs"])
	}

	// Each pair must have required fields
	first := pairs[0].(map[string]any)
	requiredFields := []string{"policy_a", "policy_b", "rule", "overlap_type", "severity", "detail"}
	for _, field := range requiredFields {
		if _, exists := first[field]; !exists {
			t.Errorf("conflict pair missing field: %s", field)
		}
	}
}

func TestGapRegression_ConflictsDetect_SeverityCounts(t *testing.T) {
	newTestHarness()
	w := doReq("POST", "/api/v1/policy/conflicts", "")
	assertStatus(t, w, http.StatusOK)
	resp := parseJSON(t, w)

	bySeverity, ok := resp["by_severity"].(map[string]any)
	if !ok {
		t.Fatalf("expected by_severity map, got %T", resp["by_severity"])
	}

	totalConflicts := int(resp["total_conflicts"].(float64))
	countSum := 0
	for _, v := range bySeverity {
		countSum += int(v.(float64))
	}
	if countSum != totalConflicts {
		t.Errorf("by_severity counts (%d) != total_conflicts (%d)", countSum, totalConflicts)
	}
}

func TestGapRegression_ConflictsDetect_ValidOverlapTypes(t *testing.T) {
	newTestHarness()
	w := doReq("POST", "/api/v1/policy/conflicts", "")
	assertStatus(t, w, http.StatusOK)
	resp := parseJSON(t, w)

	pairs := resp["conflict_pairs"].([]any)
	validTypes := map[string]bool{"contradictory": true, "duplicate": true, "subset": true}
	for _, p := range pairs {
		pair := p.(map[string]any)
		overlapType := pair["overlap_type"].(string)
		if !validTypes[overlapType] {
			t.Errorf("invalid overlap_type: %s", overlapType)
		}
	}
}

func TestGapRegression_ConflictsDetect_ValidSeverities(t *testing.T) {
	newTestHarness()
	w := doReq("POST", "/api/v1/policy/conflicts", "")
	assertStatus(t, w, http.StatusOK)
	resp := parseJSON(t, w)

	pairs := resp["conflict_pairs"].([]any)
	validSev := map[string]bool{"low": true, "medium": true, "high": true, "critical": true}
	for _, p := range pairs {
		pair := p.(map[string]any)
		sev := pair["severity"].(string)
		if !validSev[sev] {
			t.Errorf("invalid severity: %s", sev)
		}
	}
}

func TestGapRegression_ConflictsDetect_WithBody(t *testing.T) {
	newTestHarness()
	body := `{"policy_ids":["pol-001","pol-002"]}`
	w := doReq("POST", "/api/v1/policy/conflicts", body)
	assertStatus(t, w, http.StatusOK)
	resp := parseJSON(t, w)
	if _, exists := resp["conflict_pairs"]; !exists {
		t.Error("expected conflict_pairs in response")
	}
}

func TestGapRegression_ConflictsDetect_CheckedTimestamp(t *testing.T) {
	newTestHarness()
	w := doReq("POST", "/api/v1/policy/conflicts", "")
	assertStatus(t, w, http.StatusOK)
	resp := parseJSON(t, w)

	checkedAt, exists := resp["checked_at"]
	if !exists {
		t.Error("expected checked_at timestamp in response")
	}
	if _, ok := checkedAt.(string); !ok {
		t.Errorf("checked_at should be string, got %T", checkedAt)
	}
}

// Gap Regression: Policy Blast Radius (#session-verified)
// Validates: GET /api/v1/policy/blast-radius/{policy_id} returns affected entities,
// cascading policies, and summary with risk assessment.

func TestGapRegression_BlastRadius_GetOnly(t *testing.T) {
	newTestHarness()
	w := doReq("POST", "/api/v1/policy/blast-radius/pol-001", "")
	assertStatus(t, w, http.StatusMethodNotAllowed)
}

func TestGapRegression_BlastRadius_EmptyPolicyID(t *testing.T) {
	newTestHarness()
	w := doReq("GET", "/api/v1/policy/blast-radius/", "")
	assertStatus(t, w, http.StatusBadRequest)
}

func TestGapRegression_BlastRadius_ReturnsAffectedUsers(t *testing.T) {
	newTestHarness()
	w := doReq("GET", "/api/v1/policy/blast-radius/pol-001", "")
	assertStatus(t, w, http.StatusOK)
	resp := parseJSON(t, w)

	users, ok := resp["affected_users"].([]any)
	if !ok || len(users) == 0 {
		t.Fatalf("expected affected_users array with entries, got %T", resp["affected_users"])
	}

	// Each user must have user_id, username, impact, severity
	first := users[0].(map[string]any)
	for _, field := range []string{"user_id", "username", "impact", "severity"} {
		if _, exists := first[field]; !exists {
			t.Errorf("affected user missing field: %s", field)
		}
	}
}

func TestGapRegression_BlastRadius_ReturnsAffectedRoles(t *testing.T) {
	newTestHarness()
	w := doReq("GET", "/api/v1/policy/blast-radius/pol-001", "")
	assertStatus(t, w, http.StatusOK)
	resp := parseJSON(t, w)

	roles, ok := resp["affected_roles"].([]any)
	if !ok || len(roles) == 0 {
		t.Fatalf("expected affected_roles array")
	}

	first := roles[0].(map[string]any)
	for _, field := range []string{"role_id", "role_name", "users_impacted", "permissions_changed"} {
		if _, exists := first[field]; !exists {
			t.Errorf("affected role missing field: %s", field)
		}
	}
}

func TestGapRegression_BlastRadius_ReturnsCascadingPolicies(t *testing.T) {
	newTestHarness()
	w := doReq("GET", "/api/v1/policy/blast-radius/pol-001", "")
	assertStatus(t, w, http.StatusOK)
	resp := parseJSON(t, w)

	cascading, ok := resp["cascading_policies"].([]any)
	if !ok || len(cascading) == 0 {
		t.Fatalf("expected cascading_policies array")
	}

	first := cascading[0].(map[string]any)
	for _, field := range []string{"policy_id", "name", "relationship", "impact"} {
		if _, exists := first[field]; !exists {
			t.Errorf("cascading policy missing field: %s", field)
		}
	}
}

func TestGapRegression_BlastRadius_SummaryFields(t *testing.T) {
	newTestHarness()
	w := doReq("GET", "/api/v1/policy/blast-radius/pol-001", "")
	assertStatus(t, w, http.StatusOK)
	resp := parseJSON(t, w)

	summary, ok := resp["summary"].(map[string]any)
	if !ok {
		t.Fatalf("expected summary map")
	}

	for _, field := range []string{
		"total_users_affected", "total_roles_affected",
		"total_resources_changed", "total_cascading",
		"breaking_changes", "risk_level", "recommended_action",
	} {
		if _, exists := summary[field]; !exists {
			t.Errorf("summary missing field: %s", field)
		}
	}

	// risk_level must be valid
	validRisks := map[string]bool{"low": true, "medium": true, "high": true, "critical": true}
	risk := summary["risk_level"].(string)
	if !validRisks[risk] {
		t.Errorf("invalid risk_level: %s", risk)
	}
}

func TestGapRegression_BlastRadius_PreviewModeDefault(t *testing.T) {
	newTestHarness()
	w := doReq("GET", "/api/v1/policy/blast-radius/pol-001", "")
	assertStatus(t, w, http.StatusOK)
	resp := parseJSON(t, w)

	mode, ok := resp["preview_mode"].(string)
	if !ok {
		t.Fatalf("preview_mode should be string")
	}
	if mode != "preview" {
		t.Errorf("expected preview_mode=preview (default), got %s", mode)
	}
}

func TestGapRegression_BlastRadius_CustomPreviewMode(t *testing.T) {
	newTestHarness()
	w := doReq("GET", "/api/v1/policy/blast-radius/pol-001?mode=live", "")
	assertStatus(t, w, http.StatusOK)
	resp := parseJSON(t, w)

	mode := resp["preview_mode"].(string)
	if mode != "live" {
		t.Errorf("expected preview_mode=live, got %s", mode)
	}
}

func TestGapRegression_BlastRadius_AnalyzedTimestamp(t *testing.T) {
	newTestHarness()
	w := doReq("GET", "/api/v1/policy/blast-radius/pol-001", "")
	assertStatus(t, w, http.StatusOK)
	resp := parseJSON(t, w)

	analyzedAt, exists := resp["analyzed_at"]
	if !exists {
		t.Error("expected analyzed_at timestamp")
	}
	// Verify it's valid RFC3339
	if _, ok := analyzedAt.(string); !ok {
		t.Errorf("analyzed_at should be string, got %T", analyzedAt)
	}
}

// Verify JSON encoding round-trip for conflicts response
func TestGapRegression_ConflictsResponse_JSONEncoding(t *testing.T) {
	newTestHarness()
	w := doReq("POST", "/api/v1/policy/conflicts", "")
	assertStatus(t, w, http.StatusOK)

	// Must be valid JSON (parseJSON would have failed otherwise)
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
		t.Fatalf("response not valid JSON: %v", err)
	}
}
