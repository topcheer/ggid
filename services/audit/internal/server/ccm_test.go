package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Test 1: RunAll produces 15+ control results.
func TestCCM_RunAll(t *testing.T) {
	engine := NewCCMEngine()
	results := engine.RunAll()

	if len(results) < 15 {
		t.Errorf("expected 15+ controls, got %d", len(results))
	}

	for _, r := range results {
		if r.ControlID == "" {
			t.Error("control_id should not be empty")
		}
		if r.Status != StatusPass && r.Status != StatusWarn && r.Status != StatusFail {
			t.Errorf("invalid status %s for control %s", r.Status, r.ControlID)
		}
	}
}

// Test 2: GetResults returns latest result per control.
func TestCCM_GetResults(t *testing.T) {
	engine := NewCCMEngine()
	engine.RunAll()

	results := engine.GetResults()
	if len(results) < 15 {
		t.Errorf("expected 15+ results, got %d", len(results))
	}
}

// Test 3: GetSummary produces correct counts.
func TestCCM_Summary(t *testing.T) {
	engine := NewCCMEngine()
	engine.RunAll()

	summary := engine.GetSummary()

	total, ok := summary["total_controls"]
	if !ok {
		t.Fatal("missing total_controls in summary")
	}
	if total.(int) < 15 {
		t.Errorf("expected total >=15, got %d", total)
	}

	pass := summary["pass"].(int)
	warn := summary["warn"].(int)
	fail := summary["fail"].(int)
	if pass+warn+fail != total.(int) {
		t.Error("pass+warn+fail should equal total")
	}

	score := summary["compliance_score"].(float64)
	if score < 0 || score > 100 {
		t.Errorf("compliance_score out of range: %f", score)
	}
}

// Test 4: GetHistory returns results after RunAll.
func TestCCM_History(t *testing.T) {
	engine := NewCCMEngine()
	engine.RunAll()
	engine.RunAll() // run twice for history

	history := engine.GetHistory("", 500)
	if len(history) < 30 { // 15 controls × 2 runs
		t.Errorf("expected 30+ history entries, got %d", len(history))
	}
}

// Test 5: GetHistory filters by control_id.
func TestCCM_HistoryFiltered(t *testing.T) {
	engine := NewCCMEngine()
	engine.RunAll()

	history := engine.GetHistory("mfa_coverage", 10)
	for _, h := range history {
		if h.ControlID != "mfa_coverage" {
			t.Errorf("expected all results to have control_id=mfa_coverage, got %s", h.ControlID)
		}
	}
}

// Test 6: evalStatus logic — pass when metric meets threshold.
func TestCCM_EvalStatus(t *testing.T) {
	// "lt" direction: metric should be >= threshold.
	if evalStatus(95, 90, "lt") != StatusPass {
		t.Error("95 >= 90 threshold should pass")
	}
	if evalStatus(85, 90, "lt") != StatusWarn {
		t.Error("85 < 90 should warn")
	}
	if evalStatus(50, 90, "lt") != StatusFail {
		t.Error("50 << 90 should fail")
	}

	// "gt" direction: metric should be <= threshold.
	if evalStatus(1, 5, "gt") != StatusPass {
		t.Error("1 <= 5 should pass")
	}
	if evalStatus(7, 5, "gt") != StatusWarn {
		t.Error("7 > 5 should warn")
	}
	if evalStatus(15, 5, "gt") != StatusFail {
		t.Error("15 >> 5 should fail")
	}
}

// Test 7: POST /ccm/run returns 200 with results.
func TestCCM_RunEndpoint(t *testing.T) {
	s := &HTTPServer{}

	req := httptest.NewRequest("POST", "/api/v1/audit/ccm/run", strings.NewReader(`{}`))
	w := httptest.NewRecorder()

	s.handleCCM(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp["controls_run"] == nil {
		t.Error("expected controls_run in response")
	}
}

// Test 8: GET /ccm/results returns results after run.
func TestCCM_ResultsEndpoint(t *testing.T) {
	s := &HTTPServer{ccmEngine: NewCCMEngine()}
	s.ccmEngine.RunAll()

	req := httptest.NewRequest("GET", "/api/v1/audit/ccm/results", nil)
	w := httptest.NewRecorder()

	s.handleCCM(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

// Test 9: GET /ccm/summary returns dashboard data.
func TestCCM_SummaryEndpoint(t *testing.T) {
	s := &HTTPServer{ccmEngine: NewCCMEngine()}
	s.ccmEngine.RunAll()

	req := httptest.NewRequest("GET", "/api/v1/audit/ccm/summary", nil)
	w := httptest.NewRecorder()

	s.handleCCM(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp["compliance_score"] == nil {
		t.Error("expected compliance_score in summary")
	}
}

// Test 10: GET /ccm/results without engine returns empty array.
func TestCCM_NoEngine(t *testing.T) {
	s := &HTTPServer{}

	req := httptest.NewRequest("GET", "/api/v1/audit/ccm/results", nil)
	w := httptest.NewRecorder()

	s.handleCCM(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

// Test 11: Wrong method returns 405.
func TestCCM_WrongMethod(t *testing.T) {
	s := &HTTPServer{}

	req := httptest.NewRequest("DELETE", "/api/v1/audit/ccm/results", nil)
	w := httptest.NewRecorder()

	s.handleCCM(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

// Test 12: GET /ccm/history returns results.
func TestCCM_HistoryEndpoint(t *testing.T) {
	s := &HTTPServer{ccmEngine: NewCCMEngine()}
	s.ccmEngine.RunAll()

	req := httptest.NewRequest("GET", "/api/v1/audit/ccm/history?control_id=mfa_coverage", nil)
	w := httptest.NewRecorder()

	s.handleCCM(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
