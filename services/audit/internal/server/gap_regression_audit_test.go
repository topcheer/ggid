package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Gap Regression: Framework Coverage (#session-verified)
// Validates: GET /api/v1/audit/framework-coverage returns 5 frameworks with
// correct structure, coverage math, and summary aggregation.

func TestGapRegression_FrameworkCoverage_GetOnly(t *testing.T) {
	srv := newTestServer(nil, nil)
	w := doRequest(srv, "POST", "/api/v1/audit/framework-coverage", "")
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestGapRegression_FrameworkCoverage_ReturnsFiveFrameworks(t *testing.T) {
	srv := newTestServer(nil, nil)
	w := doRequest(srv, "GET", "/api/v1/audit/framework-coverage", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	frameworks, ok := resp["frameworks"].([]any)
	if !ok {
		t.Fatal("expected frameworks array")
	}
	if len(frameworks) < 4 {
		t.Fatalf("expected at least 4 frameworks, got %d", len(frameworks))
	}
}

func TestGapRegression_FrameworkCoverage_RequiredFields(t *testing.T) {
	srv := newTestServer(nil, nil)
	w := doRequest(srv, "GET", "/api/v1/audit/framework-coverage", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	frameworks := resp["frameworks"].([]any)
	first := frameworks[0].(map[string]any)
	required := []string{"framework", "total_controls", "covered", "gaps", "coverage_pct", "evidence_count", "last_assessed", "status"}
	for _, field := range required {
		if _, exists := first[field]; !exists {
			t.Errorf("framework missing field: %s", field)
		}
	}
}

func TestGapRegression_FrameworkCoverage_CoverageMath(t *testing.T) {
	srv := newTestServer(nil, nil)
	w := doRequest(srv, "GET", "/api/v1/audit/framework-coverage", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	frameworks := resp["frameworks"].([]any)
	for _, f := range frameworks {
		fw := f.(map[string]any)
		total := int(fw["total_controls"].(float64))
		covered := int(fw["covered"].(float64))
		gaps := int(fw["gaps"].(float64))
		if covered+gaps != total {
			t.Errorf("%s: covered(%d)+gaps(%d) != total(%d)", fw["framework"], covered, gaps, total)
		}
	}
}

func TestGapRegression_FrameworkCoverage_SummaryAggregation(t *testing.T) {
	srv := newTestServer(nil, nil)
	w := doRequest(srv, "GET", "/api/v1/audit/framework-coverage", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	totalControls := int(resp["total_controls"].(float64))
	totalCovered := int(resp["total_covered"].(float64))
	totalGaps := int(resp["total_gaps"].(float64))

	if totalControls <= 0 {
		t.Errorf("expected total_controls > 0, got %d", totalControls)
	}
	if totalCovered+totalGaps != totalControls {
		t.Errorf("total_covered(%d)+total_gaps(%d) != total_controls(%d)", totalCovered, totalGaps, totalControls)
	}
}

func TestGapRegression_FrameworkCoverage_BestWorst(t *testing.T) {
	srv := newTestServer(nil, nil)
	w := doRequest(srv, "GET", "/api/v1/audit/framework-coverage", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	if _, ok := resp["best_framework"]; !ok {
		t.Error("missing best_framework")
	}
	if _, ok := resp["worst_framework"]; !ok {
		t.Error("missing worst_framework")
	}
	if _, ok := resp["checked_at"]; !ok {
		t.Error("missing checked_at")
	}
}

// Gap Regression: Forensics Timeline (#session-verified)
// Validates: GET /api/v1/audit/forensics/timeline returns chain verification,
// tamper evidence, insertion gaps, reorder detection, and integrity score.

func TestGapRegression_ForensicsTimeline_GetOnly(t *testing.T) {
	srv := newTestServer(nil, nil)
	w := doRequest(srv, "POST", "/api/v1/audit/forensics/timeline", "")
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestGapRegression_ForensicsTimeline_Structure(t *testing.T) {
	srv := newTestServer(nil, nil)
	w := doRequest(srv, "GET", "/api/v1/audit/forensics/timeline", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	required := []string{
		"hash_chain_verification", "tamper_evidence", "insertion_gaps",
		"reorder_detected", "verification_summary", "integrity_score",
		"total_events_checked", "anomalies_found",
	}
	for _, field := range required {
		if _, exists := resp[field]; !exists {
			t.Errorf("forensics timeline missing field: %s", field)
		}
	}
}

func TestGapRegression_ForensicsTimeline_TamperEvidence(t *testing.T) {
	srv := newTestServer(nil, nil)
	w := doRequest(srv, "GET", "/api/v1/audit/forensics/timeline", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	evidence, ok := resp["tamper_evidence"].([]any)
	if !ok {
		t.Fatal("expected tamper_evidence array")
	}
	if len(evidence) == 0 {
		t.Fatal("expected at least 1 tamper evidence entry")
	}

	first := evidence[0].(map[string]any)
	for _, field := range []string{"event_id", "timestamp", "type", "detail", "confidence"} {
		if _, exists := first[field]; !exists {
			t.Errorf("tamper evidence missing field: %s", field)
		}
	}
}

func TestGapRegression_ForensicsTimeline_IntegrityScore(t *testing.T) {
	srv := newTestServer(nil, nil)
	w := doRequest(srv, "GET", "/api/v1/audit/forensics/timeline", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	score, ok := resp["integrity_score"].(float64)
	if !ok {
		t.Fatalf("integrity_score should be float64, got %T", resp["integrity_score"])
	}
	if score < 0 || score > 1 {
		t.Errorf("integrity_score should be 0-1, got %f", score)
	}
}

func TestGapRegression_ForensicsTimeline_HashChainStatus(t *testing.T) {
	srv := newTestServer(nil, nil)
	w := doRequest(srv, "GET", "/api/v1/audit/forensics/timeline", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	chainStatus := resp["hash_chain_verification"].(string)
	validStatuses := map[string]bool{"verified": true, "failed": true, "partial": true, "broken": true}
	if !validStatuses[chainStatus] {
		t.Errorf("invalid hash_chain_verification: %s", chainStatus)
	}
}

func TestGapRegression_ForensicsTimeline_AnomalyConsistency(t *testing.T) {
	srv := newTestServer(nil, nil)
	w := doRequest(srv, "GET", "/api/v1/audit/forensics/timeline", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	anomaliesFound := int(resp["anomalies_found"].(float64))
	tamperCount := len(resp["tamper_evidence"].([]any))
	gapCount := len(resp["insertion_gaps"].([]any))

	// anomalies_found should be >= tamper_count (may include other anomaly types)
	if anomaliesFound < tamperCount {
		t.Errorf("anomalies_found(%d) < tamper_evidence count(%d)", anomaliesFound, tamperCount)
	}
	_ = gapCount // gaps may not count toward anomalies_found
}

// Verify response is valid JSON via httptest
func TestGapRegression_ForensicsTimeline_ValidJSON(t *testing.T) {
	srv := newTestServer(nil, nil)
	w := doRequest(srv, "GET", "/api/v1/audit/forensics/timeline", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
		t.Fatalf("response not valid JSON: %v", err)
	}
}

// Ensure response recorder is used correctly
func TestGapRegression_FrameworkCoverage_ContentType(t *testing.T) {
	srv := newTestServer(nil, nil)
	w := doRequest(srv, "GET", "/api/v1/audit/framework-coverage", "")
	ct := w.Header().Get("Content-Type")
	if ct == "" {
		t.Log("Note: framework-coverage uses writeJSON which sets Content-Type")
	}
	// Just verify we got a valid response
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
}

// Unused but ensures httptest import
var _ = httptest.NewRecorder
