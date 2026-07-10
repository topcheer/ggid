package healthcheck

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestChecker_AllHealthy(t *testing.T) {
	srv1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv1.Close()
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv2.Close()

	checker := NewChecker(map[string]string{
		"auth":     srv1.URL + "/healthz",
		"identity": srv2.URL + "/healthz",
	})

	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	checker.Handler().ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var status AggregatedStatus
	json.NewDecoder(w.Body).Decode(&status)
	if status.Status != "healthy" {
		t.Errorf("expected healthy, got %s", status.Status)
	}
	if status.Total != 2 || status.Healthy != 2 {
		t.Errorf("expected 2/2 healthy, got %d/%d", status.Healthy, status.Total)
	}
}

func TestChecker_OneUnhealthy(t *testing.T) {
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer good.Close()
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer bad.Close()

	checker := NewChecker(map[string]string{
		"auth": good.URL + "/healthz",
		"oauth": bad.URL + "/healthz",
	})

	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	checker.Handler().ServeHTTP(w, req)

	if w.Code != 503 {
		t.Errorf("expected 503 when one unhealthy, got %d", w.Code)
	}

	var status AggregatedStatus
	json.NewDecoder(w.Body).Decode(&status)
	if status.Status != "degraded" {
		t.Errorf("expected degraded, got %s", status.Status)
	}
	if status.Healthy != 1 || status.Unhealthy != 1 {
		t.Errorf("expected 1 healthy + 1 unhealthy, got %d + %d", status.Healthy, status.Unhealthy)
	}
}

func TestChecker_Unreachable(t *testing.T) {
	checker := NewChecker(map[string]string{
		"dead": "http://127.0.0.1:1/healthz",
	})

	status := checker.CheckAll(context.Background())
	if status.Healthy != 0 || status.Unhealthy != 1 {
		t.Errorf("expected 0 healthy + 1 unhealthy, got %d + %d", status.Healthy, status.Unhealthy)
	}
	if status.Services["dead"].Status != "unhealthy" {
		t.Error("expected dead service to be unhealthy")
	}
}

func TestChecker_LatencyRecorded(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	checker := NewChecker(map[string]string{"auth": srv.URL})
	status := checker.CheckAll(context.Background())

	if status.Services["auth"].Latency < 0 {
		t.Error("latency should be non-negative")
	}
}

func TestChecker_Empty(t *testing.T) {
	checker := NewChecker(map[string]string{})
	status := checker.CheckAll(context.Background())
	if status.Total != 0 {
		t.Errorf("expected 0 services, got %d", status.Total)
	}
}
