package healthcheck

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLiveHandler(t *testing.T) {
	checker := NewChecker(nil)
	req := httptest.NewRequest("GET", "/healthz/live", nil)
	w := httptest.NewRecorder()
	checker.LiveHandler().ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "alive" {
		t.Errorf("expected 'alive', got '%s'", resp["status"])
	}
}

func TestReadyHandler_AllHealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	checker := NewChecker(map[string]string{"auth": srv.URL + "/healthz"})
	req := httptest.NewRequest("GET", "/healthz/ready", nil)
	w := httptest.NewRecorder()
	checker.ReadyHandler().ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200 when all healthy, got %d", w.Code)
	}
	var status AggregatedStatus
	json.NewDecoder(w.Body).Decode(&status)
	if status.Status != "healthy" {
		t.Errorf("expected healthy, got %s", status.Status)
	}
}

func TestReadyHandler_Unhealthy(t *testing.T) {
	checker := NewChecker(map[string]string{"dead": "http://127.0.0.1:1/healthz"})
	req := httptest.NewRequest("GET", "/healthz/ready", nil)
	w := httptest.NewRecorder()
	checker.ReadyHandler().ServeHTTP(w, req)

	if w.Code != 503 {
		t.Fatalf("expected 503 when unhealthy, got %d", w.Code)
	}
}

func TestHandlerWithMode_Live(t *testing.T) {
	checker := NewChecker(map[string]string{"dead": "http://127.0.0.1:1/healthz"})
	req := httptest.NewRequest("GET", "/healthz?mode=live", nil)
	w := httptest.NewRecorder()
	checker.HandlerWithMode().ServeHTTP(w, req)

	// Live should return 200 even with dead backend
	if w.Code != 200 {
		t.Errorf("expected 200 for live mode, got %d", w.Code)
	}
}

func TestHandlerWithMode_Ready(t *testing.T) {
	checker := NewChecker(map[string]string{"dead": "http://127.0.0.1:1/healthz"})
	req := httptest.NewRequest("GET", "/healthz?mode=ready", nil)
	w := httptest.NewRecorder()
	checker.HandlerWithMode().ServeHTTP(w, req)

	if w.Code != 503 {
		t.Errorf("expected 503 for ready mode with dead backend, got %d", w.Code)
	}
}

func TestHandlerWithMode_Default(t *testing.T) {
	checker := NewChecker(map[string]string{"dead": "http://127.0.0.1:1/healthz"})
	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	checker.HandlerWithMode().ServeHTTP(w, req)

	// Default mode = full check, should be 503 with dead backend
	if w.Code != 503 {
		t.Errorf("expected 503 for default mode, got %d", w.Code)
	}
}

func TestReadyHandler_Empty(t *testing.T) {
	checker := NewChecker(map[string]string{})
	req := httptest.NewRequest("GET", "/healthz/ready", nil)
	w := httptest.NewRecorder()
	checker.ReadyHandler().ServeHTTP(w, req)

	// No backends = healthy (0 unhealthy)
	if w.Code != 200 {
		t.Errorf("expected 200 for empty checker, got %d", w.Code)
	}
}

func TestLiveHandler_ContextNotBlocked(t *testing.T) {
	// Even with a cancelled context, live should return 200 immediately
	checker := NewChecker(nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := httptest.NewRequestWithContext(ctx, "GET", "/healthz/live", nil)
	w := httptest.NewRecorder()
	checker.LiveHandler().ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("expected 200 even with cancelled context, got %d", w.Code)
	}
}
