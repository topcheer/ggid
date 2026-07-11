package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// --- error_writer.go coverage ---

func TestCovS25_WriteError_WithRequestID(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	ctx := context.WithValue(req.Context(), RequestIDContextKey{}, "req-12345")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	WriteError(rr, req, http.StatusUnauthorized, "unauthorized", "missing token")

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
	if rr.Header().Get("X-Request-ID") != "req-12345" {
		t.Fatalf("expected X-Request-ID 'req-12345', got %q", rr.Header().Get("X-Request-ID"))
	}
	var resp ErrorResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Error.Code != "unauthorized" || resp.Error.RequestID != "req-12345" {
		t.Fatalf("bad error body: %+v", resp)
	}
}

func TestCovS25_WriteError_NoRequestID(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	WriteError(rr, req, http.StatusForbidden, "forbidden", "no access")

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
	var resp ErrorResponse
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp.Error.RequestID == "" {
		t.Fatal("expected non-empty auto-generated request_id")
	}
}

func TestCovS25_WriteErrorNoRequest(t *testing.T) {
	rr := httptest.NewRecorder()
	WriteErrorNoRequest(rr, http.StatusInternalServerError, "internal", "oops")

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
	var resp ErrorResponse
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp.Error.Code != "internal" || resp.Error.RequestID == "" {
		t.Fatalf("bad error body: %+v", resp)
	}
}

func TestCovS25_WriteError_NilRequest(t *testing.T) {
	rr := httptest.NewRecorder()
	WriteError(rr, nil, http.StatusBadRequest, "bad_request", "nil req")

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

// --- circuitbreaker.go String() ---

func TestCovS25_CircuitBreaker_String(t *testing.T) {
	states := []CircuitState{CircuitClosed, CircuitOpen, CircuitHalfOpen}
	for _, s := range states {
		str := s.String()
		if str == "" {
			t.Fatalf("String() empty for state %d", s)
		}
	}
	invalid := CircuitState(99)
	if invalid.String() == "" {
		t.Fatal("expected non-empty string for invalid state")
	}
}

// --- circuitbreaker RecordFailure → open ---

func TestCovS25_CircuitBreaker_FailureOpens(t *testing.T) {
	registry := NewCircuitRegistry(CircuitConfig{MaxFailures: 3, Timeout: 30 * time.Second})
	cb := registry.Get("test-cb-fail")

	for i := 0; i < 3; i++ {
		cb.RecordFailure()
	}
	if cb.State() != CircuitOpen {
		t.Fatalf("expected open after 3 failures, got %s", cb.State())
	}

	// Allow should fail fast when open
	if cb.Allow() {
		t.Fatal("expected Allow() to return false when circuit is open")
	}
}

// --- circuitbreaker RecordSuccess transitions half-open → closed ---

func TestCovS25_CircuitBreaker_Recovery(t *testing.T) {
	cb := NewCircuitBreaker(CircuitConfig{
		MaxFailures:     2,
		Timeout:         50 * time.Millisecond,
		HalfOpenSuccess: 2,
	})

	// Trip the circuit
	cb.RecordFailure()
	cb.RecordFailure()
	if cb.State() != CircuitOpen {
		t.Fatalf("expected open, got %s", cb.State())
	}

	// Wait for timeout → half-open
	time.Sleep(60 * time.Millisecond)
	if !cb.Allow() {
		t.Fatal("expected Allow() to return true in half-open")
	}

	// Record successes to close circuit (HalfOpenSuccess=2)
	cb.RecordSuccess()
	if cb.State() == CircuitClosed {
		t.Fatal("should still be half-open after 1 success")
	}
	cb.RecordSuccess()
	if cb.State() != CircuitClosed {
		t.Fatalf("expected closed after 2 successes, got %s", cb.State())
	}
}

// --- circuitbreaker Stats ---

func TestCovS25_CircuitBreaker_Stats(t *testing.T) {
	cb := NewCircuitBreaker(CircuitConfig{MaxFailures: 5, Timeout: 30 * time.Second})
	cb.RecordSuccess()
	cb.RecordFailure()

	stats := cb.Stats()
	if stats.State != CircuitClosed {
		t.Fatalf("expected closed state, got %s", stats.State)
	}
}

// --- audit_log Publish with error path ---
