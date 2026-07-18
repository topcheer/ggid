package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ggid/ggid/services/auth/internal/repository"
	"github.com/google/uuid"
)

func newTestHandlerWithCAP() *Handler {
	return &Handler{
		capRepo: repository.NewConditionalAccessRepository(nil),
	}
}

// Test 1: POST create policy returns 201.
func TestConditionalAccess_Create(t *testing.T) {
	h := newTestHandlerWithCAP()
	tid := uuid.New()

	body := `{"name":"Block high risk","conditions":{"risk_score_greater_than":60,"geo_countries":["CN","RU"]},"action":"block","priority":10,"enabled":true}`
	req := httptest.NewRequest("POST", "/api/v1/auth/conditional-access/policies", bytes.NewBufferString(body))
	req = reqWithTenantContext(req, tid)
	w := httptest.NewRecorder()

	h.handleConditionalAccess(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var policy repository.ConditionalAccessPolicy
	if err := json.Unmarshal(w.Body.Bytes(), &policy); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if policy.Name != "Block high risk" {
		t.Errorf("expected name=Block high risk, got %s", policy.Name)
	}
	if policy.Action != repository.ActionBlock {
		t.Errorf("expected action=block, got %s", policy.Action)
	}
}

// Test 2: GET list returns 200 with array.
func TestConditionalAccess_List(t *testing.T) {
	h := newTestHandlerWithCAP()
	tid := uuid.New()

	req := httptest.NewRequest("GET", "/api/v1/auth/conditional-access/policies", nil)
	req = reqWithTenantContext(req, tid)
	w := httptest.NewRecorder()

	h.handleConditionalAccess(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// Test 3: POST evaluate with no policies returns allow.
func TestConditionalAccess_EvaluateNoPolicies(t *testing.T) {
	h := newTestHandlerWithCAP()
	tid := uuid.New()

	body := `{"device_posture":50,"risk_score":30,"geo_country":"US","auth_method":"password","ip_address":"1.2.3.4"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/conditional-access/evaluate", bytes.NewBufferString(body))
	req = reqWithTenantContext(req, tid)
	w := httptest.NewRecorder()

	h.handleConditionalAccess(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp["action"] != repository.ActionAllow {
		t.Errorf("expected action=allow with no policies, got %v", resp["action"])
	}
	if resp["matched"] != false {
		t.Error("expected matched=false with no policies")
	}
}

// Test 4: POST create with invalid action returns 400.
func TestConditionalAccess_CreateInvalidAction(t *testing.T) {
	h := newTestHandlerWithCAP()
	tid := uuid.New()

	body := `{"name":"test","action":"bogus"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/conditional-access/policies", bytes.NewBufferString(body))
	req = reqWithTenantContext(req, tid)
	w := httptest.NewRecorder()

	h.handleConditionalAccess(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// Test 5: POST create without name returns 400.
func TestConditionalAccess_CreateNoName(t *testing.T) {
	h := newTestHandlerWithCAP()
	tid := uuid.New()

	body := `{"action":"block"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/conditional-access/policies", bytes.NewBufferString(body))
	req = reqWithTenantContext(req, tid)
	w := httptest.NewRecorder()

	h.handleConditionalAccess(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// Test 6: DELETE with invalid UUID returns 400.
func TestConditionalAccess_DeleteInvalidID(t *testing.T) {
	h := newTestHandlerWithCAP()
	tid := uuid.New()

	req := httptest.NewRequest("DELETE", "/api/v1/auth/conditional-access/policies/not-a-uuid", nil)
	req = reqWithTenantContext(req, tid)
	w := httptest.NewRecorder()

	h.handleConditionalAccess(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// Test 7: Missing tenant returns 401.
func TestConditionalAccess_NoTenant(t *testing.T) {
	h := newTestHandlerWithCAP()

	req := httptest.NewRequest("GET", "/api/v1/auth/conditional-access/policies", nil)
	w := httptest.NewRecorder()

	h.handleConditionalAccess(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

// Test 8: Evaluate engine — condition matching with nil pool returns allow.
func TestConditionalAccess_EvaluateNilPool(t *testing.T) {
	repo := repository.NewConditionalAccessRepository(nil)
	evalCtx := repository.EvalContext{
		DevicePosture: 30,
		RiskScore:     80,
		GeoCountry:    "CN",
	}
	action, policy := repo.Evaluate(reqWithTenantContext(
		httptest.NewRequest("GET", "/", nil), uuid.New()).Context(), uuid.New(), evalCtx)

	if action != repository.ActionAllow {
		t.Errorf("with nil pool, expected allow, got %s", action)
	}
	if policy != nil {
		t.Error("with nil pool, policy should be nil")
	}
}

// Test 9: matchesConditions — risk score condition.
func TestConditionalAccess_MatchRiskScore(t *testing.T) {
	threshold := 60
	conds := repository.Conditions{RiskScoreGreaterThan: &threshold}

	// Risk 80 > 60 → matches.
	if !repository.MatchesConditionsPublic(conds, repository.EvalContext{RiskScore: 80}) {
		t.Error("risk_score=80 should match risk_score_greater_than=60")
	}
	// Risk 50 < 60 → no match.
	if repository.MatchesConditionsPublic(conds, repository.EvalContext{RiskScore: 50}) {
		t.Error("risk_score=50 should NOT match risk_score_greater_than=60")
	}
}

// Test 10: matchesConditions — geo country condition.
func TestConditionalAccess_MatchGeo(t *testing.T) {
	conds := repository.Conditions{GeoCountries: []string{"CN", "RU"}}

	if !repository.MatchesConditionsPublic(conds, repository.EvalContext{GeoCountry: "CN"}) {
		t.Error("geo=CN should match GeoCountries=[CN,RU]")
	}
	if repository.MatchesConditionsPublic(conds, repository.EvalContext{GeoCountry: "US"}) {
		t.Error("geo=US should NOT match GeoCountries=[CN,RU]")
	}
}

// Test 11: matchesConditions — empty conditions = no match.
func TestConditionalAccess_MatchEmpty(t *testing.T) {
	conds := repository.Conditions{}
	if repository.MatchesConditionsPublic(conds, repository.EvalContext{RiskScore: 99}) {
		t.Error("empty conditions should not match anything")
	}
}

// Test 12: matchesConditions — device posture condition.
func TestConditionalAccess_MatchDevicePosture(t *testing.T) {
	threshold := 70
	conds := repository.Conditions{DevicePostureLessThan: &threshold}

	if !repository.MatchesConditionsPublic(conds, repository.EvalContext{DevicePosture: 50}) {
		t.Error("posture=50 should match device_posture_less_than=70")
	}
	if repository.MatchesConditionsPublic(conds, repository.EvalContext{DevicePosture: 80}) {
		t.Error("posture=80 should NOT match device_posture_less_than=70")
	}
}

// Test 13: matchesConditions — auth method condition.
func TestConditionalAccess_MatchAuthMethod(t *testing.T) {
	conds := repository.Conditions{AuthMethodNotIn: []string{"password"}}

	// Using password → condition met (password is NOT allowed).
	if !repository.MatchesConditionsPublic(conds, repository.EvalContext{AuthMethod: "password"}) {
		t.Error("auth_method=password should match AuthMethodNotIn=[password]")
	}
	// Using webauthn → no match.
	if repository.MatchesConditionsPublic(conds, repository.EvalContext{AuthMethod: "webauthn"}) {
		t.Error("auth_method=webauthn should NOT match AuthMethodNotIn=[password]")
	}
}
