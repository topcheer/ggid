package httpserver

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/ggid/ggid/services/policy/internal/service"
	"github.com/google/uuid"
)

// TestHandleDecisionLog_Empty tests the endpoint when no decisions exist.
func TestHandleDecisionLog_Empty(t *testing.T) {
	newTestHarness()
	w := doReq("GET", "/api/v1/policies/decision-log", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["total"].(float64) != 0 {
		t.Fatalf("expected 0 total, got %v", resp["total"])
	}
}

func TestHandleDecisionLog_WithLimit(t *testing.T) {
	newTestHarness()
	w := doReq("GET", "/api/v1/policies/decision-log?limit=10", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["total"].(float64) != 0 {
		t.Fatalf("expected 0 total, got %v", resp["total"])
	}
}

func TestHandleDecisionLog_MethodNotAllowed(t *testing.T) {
	newTestHarness()
	w := doReq("POST", "/api/v1/policies/decision-log", "")
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

// TestHandleDecisionLog_WithDecisions verifies the endpoint correctly
// returns decisions that were logged by the evaluator.
func TestHandleDecisionLog_WithDecisions(t *testing.T) {
	// Clear any previous decisions
	clearTestDecisions()

	h := newTestHarness()

	// Record a few test decisions with the test tenant
	recordTestDecision(h.tenantID, true, "rbac", "user.read")
	recordTestDecision(h.tenantID, false, "deny policy:restrict", "")
	recordTestDecision(h.tenantID, true, "rbac", "user.write")

	w := doReq("GET", "/api/v1/policies/decision-log?limit=10", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	total := resp["total"].(float64)
	if total != 3 {
		t.Fatalf("expected 3 decisions, got %v", total)
	}

	allowCount := resp["allow_count"].(float64)
	if allowCount != 2 {
		t.Fatalf("expected 2 allow_count, got %v", allowCount)
	}

	denyCount := resp["deny_count"].(float64)
	if denyCount != 1 {
		t.Fatalf("expected 1 deny_count, got %v", denyCount)
	}
}

func TestHandleDecisionLog_InvalidLimit(t *testing.T) {
	newTestHarness()
	// Invalid limit should fall back to default (50), not error
	w := doReq("GET", "/api/v1/policies/decision-log?limit=abc", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for invalid limit, got %d", w.Code)
	}
}

// TestHandleDecisionLog_MissingTenant verifies fail-closed when no header.
func TestHandleDecisionLog_MissingTenant(t *testing.T) {
	newTestHarness()
	orig := testTenantHeader
	testTenantHeader = ""
	defer func() { testTenantHeader = orig }()
	w := doReq("GET", "/api/v1/policies/decision-log", "")
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

// --- Test helpers ---

var testMux *http.ServeMux

// clearTestDecisions resets the decision log between tests.
func clearTestDecisions() {
	// Reset the in-memory decision log.
	service.ClearDecisionLogForTest()
}

// recordTestDecision adds a synthetic decision entry for testing.
func recordTestDecision(tenantID uuid.UUID, allowed bool, matchedBy, action string) {
	service.AddTestDecisionForTest(tenantID, allowed, matchedBy, action)
}
