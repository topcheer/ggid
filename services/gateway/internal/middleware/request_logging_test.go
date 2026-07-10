package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestLogging_2xx(t *testing.T) {
	cl := &CapturingLogger{}
	handler := RequestLogging(cl)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello"))
	}))

	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if len(cl.Entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(cl.Entries))
	}
	if cl.Entries[0].Status != 200 {
		t.Errorf("Status: want 200, got %d", cl.Entries[0].Status)
	}
	if cl.Levels[0] != "info" {
		t.Errorf("Level: want 'info', got '%s'", cl.Levels[0])
	}
	if cl.Entries[0].BytesOut != 5 {
		t.Errorf("Bytes: want 5, got %d", cl.Entries[0].BytesOut)
	}
	if cl.Entries[0].Path != "/api/v1/users" {
		t.Errorf("Path: got '%s'", cl.Entries[0].Path)
	}
}

func TestRequestLogging_4xx(t *testing.T) {
	cl := &CapturingLogger{}
	handler := RequestLogging(cl)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	req := httptest.NewRequest("GET", "/api/v1/missing", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if cl.Levels[0] != "warn" {
		t.Errorf("Level: want 'warn', got '%s'", cl.Levels[0])
	}
}

func TestRequestLogging_5xx(t *testing.T) {
	cl := &CapturingLogger{}
	handler := RequestLogging(cl)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	req := httptest.NewRequest("GET", "/api/v1/error", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if cl.Levels[0] != "error" {
		t.Errorf("Level: want 'error', got '%s'", cl.Levels[0])
	}
}

func TestRequestLogging_HealthCheckSkipped(t *testing.T) {
	cl := &CapturingLogger{}
	handler := RequestLogging(cl)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/healthz", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if len(cl.Entries) != 0 {
		t.Errorf("Health check should not be logged, got %d entries", len(cl.Entries))
	}
}

func TestRequestLogging_ReadyCheckSkipped(t *testing.T) {
	cl := &CapturingLogger{}
	handler := RequestLogging(cl)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/readyz", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if len(cl.Entries) != 0 {
		t.Error("Ready check should not be logged")
	}
}

func TestRequestLogging_WithTenantAndRequestID(t *testing.T) {
	cl := &CapturingLogger{}
	handler := RequestLogging(cl)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/api/v1/data", nil)
	ctx := context.WithValue(req.Context(), TenantIDKey, "tenant-42")
	ctx = context.WithValue(ctx, RequestIDKey, "req-99")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if cl.Entries[0].TenantID != "tenant-42" {
		t.Errorf("TenantID: got '%s'", cl.Entries[0].TenantID)
	}
	if cl.Entries[0].RequestID != "req-99" {
		t.Errorf("RequestID: got '%s'", cl.Entries[0].RequestID)
	}
}

func TestRequestLogging_NoExplicitWriteHeader(t *testing.T) {
	cl := &CapturingLogger{}
	handler := RequestLogging(cl)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("auto 200"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if cl.Entries[0].Status != 200 {
		t.Errorf("Status: want 200 (implicit), got %d", cl.Entries[0].Status)
	}
}

func TestStatusLogLevel(t *testing.T) {
	tests := []struct {
		status int
		want   LogLevel
	}{
		{200, LogLevelInfo},
		{201, LogLevelInfo},
		{301, LogLevelInfo},
		{400, LogLevelWarn},
		{404, LogLevelWarn},
		{429, LogLevelWarn},
		{500, LogLevelError},
		{502, LogLevelError},
		{503, LogLevelError},
	}
	for _, tt := range tests {
		if got := statusLogLevel(tt.status); got != tt.want {
			t.Errorf("statusLogLevel(%d) = %v, want %v", tt.status, got, tt.want)
		}
	}
}

func TestJSONLogger(t *testing.T) {
	var captured string
	l := NewJSONLogger(func(s string) { captured = s })
	l.Info(LogEntry{Method: "GET", Path: "/test", Status: 200})
	if captured == "" {
		t.Error("JSONLogger should have written output")
	}
}

func TestJSONLogger_NilWriter(t *testing.T) {
	l := NewJSONLogger(nil)
	// Should not panic
	l.Info(LogEntry{Method: "GET", Path: "/test", Status: 200})
}

func TestNoopLogger(t *testing.T) {
	l := NoopLogger{}
	// Should not panic
	l.Info(LogEntry{})
	l.Warn(LogEntry{})
	l.Error(LogEntry{})
}
