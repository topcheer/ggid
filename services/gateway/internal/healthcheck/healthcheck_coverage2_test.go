package healthcheck

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCheckOne_HealthyWithVersion(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status":         "healthy",
			"version":        "1.2.3",
			"uptime_seconds": 3600,
		})
	}))
	defer ts.Close()

	c := NewChecker(nil)
	status := c.checkOne(context.Background(), "test-svc", ts.URL)
	if status.Status != "healthy" {
		t.Errorf("Status: want 'healthy', got '%s'", status.Status)
	}
	if status.Version != "1.2.3" {
		t.Errorf("Version: want '1.2.3', got '%s'", status.Version)
	}
	if status.UptimeSeconds != 3600 {
		t.Errorf("Uptime: want 3600, got %d", status.UptimeSeconds)
	}
}

func TestCheckOne_HealthyWithUptimeField(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"uptime": 7200,
		})
	}))
	defer ts.Close()

	c := NewChecker(nil)
	status := c.checkOne(context.Background(), "svc", ts.URL)
	if status.Status != "healthy" {
		t.Errorf("Status: want 'healthy', got '%s'", status.Status)
	}
	if status.UptimeSeconds != 7200 {
		t.Errorf("Uptime from 'uptime' field: want 7200, got %d", status.UptimeSeconds)
	}
}

func TestCheckOne_Non200Response(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer ts.Close()

	c := NewChecker(nil)
	status := c.checkOne(context.Background(), "svc", ts.URL)
	if status.Status != "unhealthy" {
		t.Errorf("Status: want 'unhealthy', got '%s'", status.Status)
	}
	if status.Error != "non-200 status" {
		t.Errorf("Error: got '%s'", status.Error)
	}
}

func TestCheckOne_ConnectionError(t *testing.T) {
	c := NewChecker(nil)
	status := c.checkOne(context.Background(), "down-svc", "http://127.0.0.1:1/health")
	if status.Status != "unhealthy" {
		t.Errorf("Status: want 'unhealthy', got '%s'", status.Status)
	}
	if status.Error == "" {
		t.Error("Should have error message")
	}
}

func TestCheckOne_InvalidURL(t *testing.T) {
	c := NewChecker(nil)
	status := c.checkOne(context.Background(), "bad-svc", "http://[::1]:namedport/health")
	if status.Status != "unhealthy" {
		t.Errorf("Status: want 'unhealthy', got '%s'", status.Status)
	}
}

func TestCheckOne_MalformedJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not json at all"))
	}))
	defer ts.Close()

	c := NewChecker(nil)
	status := c.checkOne(context.Background(), "svc", ts.URL)
	if status.Status != "healthy" {
		t.Errorf("Status: want 'healthy', got '%s'", status.Status)
	}
	if status.Version != "" {
		t.Errorf("Version should be empty: got '%s'", status.Version)
	}
}

func TestCheckAll_ConcurrentHealthChecks(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	c := NewChecker(map[string]string{
		"svc1": ts.URL,
		"svc2": ts.URL,
		"svc3": ts.URL,
	})

	result := c.CheckAll(context.Background())

	if result.Total != 3 {
		t.Errorf("Total: want 3, got %d", result.Total)
	}
	if result.Healthy != 3 {
		t.Errorf("Healthy: want 3, got %d", result.Healthy)
	}
	if len(result.Services) != 3 {
		t.Errorf("Services len: want 3, got %d", len(result.Services))
	}
}

func TestCheckAll_EmptyServices(t *testing.T) {
	c := NewChecker(nil)
	result := c.CheckAll(context.Background())
	if result.Total != 0 {
		t.Errorf("Total: want 0, got %d", result.Total)
	}
	if result.Healthy != 0 {
		t.Errorf("Healthy: want 0, got %d", result.Healthy)
	}
}

func TestCheckAll_MixedHealth(t *testing.T) {
	healthy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer healthy.Close()

	c := NewChecker(map[string]string{
		"good": healthy.URL,
		"bad":  "http://127.0.0.1:1/health",
	})

	result := c.CheckAll(context.Background())
	if result.Healthy != 1 {
		t.Errorf("Healthy: want 1, got %d", result.Healthy)
	}
	if result.Unhealthy != 1 {
		t.Errorf("Unhealthy: want 1, got %d", result.Unhealthy)
	}
}

func TestCheckAll_GatewayUptimePositive(t *testing.T) {
	c := NewChecker(nil)
	time.Sleep(10 * time.Millisecond)
	result := c.CheckAll(context.Background())
	if result.GatewayUptimeSeconds < 0 {
		t.Error("Uptime should not be negative")
	}
}

func TestCheckOne_LatencyRecorded(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	c := NewChecker(nil)
	status := c.checkOne(context.Background(), "svc", ts.URL)
	if status.Latency < 1 {
		t.Errorf("Latency should be >= 1ms, got %d", status.Latency)
	}
	_ = fmt.Sprintf("%v", status)
}
