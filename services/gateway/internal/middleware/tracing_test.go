package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTraceGenID(t *testing.T) {
	id1 := GenerateTraceID()
	id2 := GenerateTraceID()
	if len(id1) != 32 { t.Errorf("trace ID should be 32 chars, got %d", len(id1)) }
	if id1 == id2 { t.Error("trace IDs should be unique") }
}

func TestTraceGenSpan(t *testing.T) {
	id := GenerateSpanID()
	if len(id) != 16 { t.Errorf("span ID should be 16 chars, got %d", len(id)) }
}

func TestTraceMW_AddsHeaders(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	w := httptest.NewRecorder()

	called := false
	TraceMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		// Verify span is in context.
		span := SpanFromContext(r.Context())
		if span == nil { t.Error("span should be in context") }
		if span.TraceID == "" { t.Error("trace ID should be set") }
	})).ServeHTTP(w, req)

	if !called { t.Error("handler should be called") }
	if w.Header().Get("X-Trace-Id") == "" { t.Error("X-Trace-Id header should be set") }
	if w.Header().Get("X-Span-Id") == "" { t.Error("X-Span-Id header should be set") }
}

func TestTraceMW_SkipsHealth(t *testing.T) {
	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()

	TraceMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Span should NOT be in context for health checks.
		span := SpanFromContext(r.Context())
		if span != nil { t.Error("health check should not have span") }
	})).ServeHTTP(w, req)

	if w.Header().Get("X-Trace-Id") != "" { t.Error("health check should not have trace header") }
}

func TestSetSpanAttribute(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	w := httptest.NewRecorder()

	TraceMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		SetSpanAttribute(r.Context(), "user_id", "user-123")
		SetSpanAttribute(r.Context(), "tenant_id", "tenant-456")
		span := SpanFromContext(r.Context())
		if span.Attributes["user_id"] != "user-123" { t.Error("user_id attr should be set") }
		if span.Attributes["tenant_id"] != "tenant-456" { t.Error("tenant_id attr should be set") }
	})).ServeHTTP(w, req)

	_ = w
}

func TestSetSampleRate(t *testing.T) {
	SetSampleRate(1.0) // 100% sampling
	if getSampleRate() != 1.0 { t.Error("sample rate should be 1.0") }
	SetSampleRate(0.1) // reset to 10%
	if getSampleRate() != 0.1 { t.Error("sample rate should be 0.1") }
}

func TestGetRecentTraces_Empty(t *testing.T) {
	traces := GetRecentTraces(10)
	if traces == nil { t.Error("should return non-nil slice") }
}

func TestTraceMW_Propagates(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	req.Header.Set("X-Trace-Id", "existing-trace-id-1234567890123456")
	w := httptest.NewRecorder()

	TraceMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span := SpanFromContext(r.Context())
		if span.TraceID != "existing-trace-id-1234567890123456" {
			t.Errorf("should propagate existing trace ID, got %s", span.TraceID)
		}
	})).ServeHTTP(w, req)
}
