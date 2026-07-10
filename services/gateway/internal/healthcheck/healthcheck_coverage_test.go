package healthcheck

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestChecker_CheckOne_Non2xxStatus(t *testing.T) {
	// Server returning 503
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	c := NewChecker(map[string]string{
		"svc1": srv.URL,
	})
	result := c.CheckAll(context.Background())
	if result.Status != "degraded" {
		t.Errorf("expected degraded, got %s", result.Status)
	}
	if len(result.Services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(result.Services))
	}
	if result.Services["svc1"].Status != "unhealthy" {
		t.Errorf("expected unhealthy, got %s", result.Services["svc1"].Status)
	}
	if result.Services["svc1"].Error == "" {
		t.Error("expected non-empty error message for unhealthy service")
	}
}

func TestChecker_CheckOne_InvalidURL(t *testing.T) {
	c := NewChecker(map[string]string{
		"bad": "http://localhost:1/health",
	})
	result := c.CheckAll(context.Background())
	if result.Services["bad"].Status != "unhealthy" {
		t.Errorf("expected unhealthy for invalid URL, got %s", result.Services["bad"].Status)
	}
}

func TestChecker_CheckOne_BadURLFormat(t *testing.T) {
	c := NewChecker(map[string]string{
		"badfmt": "://not-a-url",
	})
	result := c.CheckAll(context.Background())
	// Should not panic, should report unhealthy
	if result.Services["badfmt"].Status != "unhealthy" {
		t.Errorf("expected unhealthy for bad URL, got %s", result.Services["badfmt"].Status)
	}
}
