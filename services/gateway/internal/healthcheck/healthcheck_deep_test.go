package healthcheck

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDeepHandler_AllHealthy_C21(t *testing.T) {
	// Create mock backends
	srv1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer srv1.Close()

	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer srv2.Close()

	checker := NewChecker(map[string]string{
		"auth":     srv1.URL + "/healthz",
		"identity": srv2.URL + "/healthz",
	})

	h := checker.DeepHandler()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/healthz/deep", nil)
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	var status AggregatedStatus
	if err := json.NewDecoder(rr.Body).Decode(&status); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if status.Healthy != 2 {
		t.Errorf("healthy = %d, want 2", status.Healthy)
	}
}

func TestDeepHandler_OneDown_C21(t *testing.T) {
	srvUp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srvUp.Close()

	srvDown := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srvDown.Close()

	checker := NewChecker(map[string]string{
		"auth": srvUp.URL + "/healthz",
		"org":  srvDown.URL + "/healthz",
	})

	h := checker.DeepHandler()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/healthz/deep", nil)
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusServiceUnavailable)
	}
	var status AggregatedStatus
	json.NewDecoder(rr.Body).Decode(&status)
	if status.Unhealthy != 1 {
		t.Errorf("unhealthy = %d, want 1", status.Unhealthy)
	}
}

func TestDeepHandler_Unreachable_C21(t *testing.T) {
	checker := NewChecker(map[string]string{
		"auth": "http://127.0.0.1:1/healthz", // unreachable
	})

	h := checker.DeepHandler()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/healthz/deep", nil)
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusServiceUnavailable)
	}
}

func TestDeepHandler_EmptyServices_C21(t *testing.T) {
	checker := NewChecker(map[string]string{})
	h := checker.DeepHandler()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/healthz/deep", nil)
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d (no services = all healthy)", rr.Code, http.StatusOK)
	}
}
