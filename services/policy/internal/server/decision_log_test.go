package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ggid/ggid/services/policy/internal/service"
)

// TestHandleDecisionLog_Empty tests the endpoint when no decisions exist.
func TestHandleDecisionLog_Empty(t *testing.T) {
	srv := newTestServer()

	w := doRequest(srv, "GET", "/api/v1/policies/decision-log", "")
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
	srv := newTestServer()

	w := doRequest(srv, "GET", "/api/v1/policies/decision-log?limit=10", "")
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
	srv := newTestServer()
	w := doRequest(srv, "POST", "/api/v1/policies/decision-log", "")
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

// TestHandleDecisionLog_WithDecisions verifies the endpoint correctly
// returns decisions that were logged by the evaluator.
func TestHandleDecisionLog_WithDecisions(t *testing.T) {
	// Clear any previous decisions
	clearTestDecisions()

	// Record a few test decisions via the service layer
	recordTestDecision(true, "rbac", "user.read")
	recordTestDecision(false, "deny policy:restrict", "")
	recordTestDecision(true, "rbac", "user.write")

	srv := newTestServer()
	w := doRequest(srv, "GET", "/api/v1/policies/decision-log?limit=10", "")
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
	srv := newTestServer()
	// Invalid limit should fall back to default (50), not error
	w := doRequest(srv, "GET", "/api/v1/policies/decision-log?limit=abc", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for invalid limit, got %d", w.Code)
	}
}

// --- Test helpers ---

// newTestServer creates a minimal HTTPServer for decision-log tests.
func newTestServer() *HTTPServer {
	srv := &HTTPServer{}
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)
	// Override httptest handler
	testMux = mux
	return srv
}

var testMux *http.ServeMux

// doRequest creates and executes an HTTP request against the test mux.
func doRequest(srv *HTTPServer, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	if body != "" {
		req = httptest.NewRequest(method, path, stringReader(body))
	}
	w := httptest.NewRecorder()
	testMux.ServeHTTP(w, req)
	return w
}

func stringReader(s string) *stringReaderImpl {
	return &stringReaderImpl{data: []byte(s)}
}

type stringReaderImpl struct {
	data []byte
	pos  int
}

func (r *stringReaderImpl) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, nil
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

// clearTestDecisions resets the decision log between tests.
func clearTestDecisions() {
	service.ClearDecisionLogForTest()
}

// recordTestDecision adds a synthetic decision entry for testing.
func recordTestDecision(allowed bool, matchedBy, action string) {
	service.AddTestDecisionForTest(allowed, matchedBy, action)
}
