package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// --- Threat Feed Stream SSE Tests ---

func TestThreatFeedStream_SSEHeaders(t *testing.T) {
	srv := newTestServer(nil, nil)
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req := httptest.NewRequest("GET", "/api/v1/audit/threat-feed/stream?tenant_id="+testTenantID.String(), nil)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		mux.ServeHTTP(w, req)
		close(done)
	}()
	time.Sleep(100 * time.Millisecond)
	cancel()
	<-done

	if ct := w.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("expected Content-Type text/event-stream, got %s", ct)
	}
	if cc := w.Header().Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("expected Cache-Control no-cache, got %s", cc)
	}
	if conn := w.Header().Get("Connection"); conn != "keep-alive" {
		t.Errorf("expected Connection keep-alive, got %s", conn)
	}

	body := w.Body.String()
	if !strings.Contains(body, "event: connected") {
		t.Error("expected initial 'connected' SSE event in body")
	}
	if !strings.Contains(body, `data: {"status":"ok"}`) {
		t.Error("expected connected data payload")
	}
}

func TestThreatFeedStream_MethodNotAllowed(t *testing.T) {
	srv := newTestServer(nil, nil)
	w := doRequest(srv, "POST", "/api/v1/audit/threat-feed/stream", "")
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestThreatFeedStream_InvalidTenantID_NoPanic(t *testing.T) {
	srv := newTestServer(nil, nil)
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req := httptest.NewRequest("GET", "/api/v1/audit/threat-feed/stream?tenant_id=invalid-uuid", nil)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		mux.ServeHTTP(w, req)
		close(done)
	}()
	time.Sleep(100 * time.Millisecond)
	cancel()
	<-done

	if ct := w.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("expected text/event-stream even with invalid tenant, got %s", ct)
	}
	if !strings.Contains(w.Body.String(), "event: connected") {
		t.Error("expected connected event even with invalid tenant")
	}
}

func TestThreatFeedStream_NoTenantID_DoesNotCrash(t *testing.T) {
	srv := newTestServer(nil, nil)
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req := httptest.NewRequest("GET", "/api/v1/audit/threat-feed/stream", nil)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		mux.ServeHTTP(w, req)
		close(done)
	}()
	time.Sleep(100 * time.Millisecond)
	cancel()
	<-done

	if w.Header().Get("Content-Type") != "text/event-stream" {
		t.Error("expected SSE Content-Type")
	}
}

func TestPollThreatEvents_NilService_ReturnsNil(t *testing.T) {
	srv := &HTTPServer{svc: nil}
	req := httptest.NewRequest("GET", "/", nil)
	events := srv.pollThreatEvents(req, testTenantID, time.Now())
	if events != nil {
		t.Errorf("expected nil for nil service, got %d events", len(events))
	}
}

func TestPollThreatEvents_NilTenantID_ReturnsNil(t *testing.T) {
	srv := newTestServer(nil, nil)
	req := httptest.NewRequest("GET", "/", nil)
	events := srv.pollThreatEvents(req, uuid.Nil, time.Now())
	if events != nil {
		t.Errorf("expected nil for nil tenant, got %d events", len(events))
	}
}

func TestSSEThreatEvent_JSONSerialization(t *testing.T) {
	evt := SSEThreatEvent{
		ID:          "test-1",
		Severity:    "high",
		Type:        "brute_force",
		Description: "Multiple failed logins",
		SourceIP:    "203.0.113.50",
		Indicators:  []string{"rapid_retries"},
		Target:      "login",
		Source:      "audit_engine",
		CreatedAt:   "2025-01-01T00:00:00Z",
	}

	data, err := json.Marshal(evt)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	jsonStr := string(data)

	expectedFields := []string{`"id":"test-1"`, `"severity":"high"`, `"type":"brute_force"`, `"source_ip":"203.0.113.50"`, `"source":"audit_engine"`}
	for _, field := range expectedFields {
		if !strings.Contains(jsonStr, field) {
			t.Errorf("expected %s in JSON output, got: %s", field, jsonStr)
		}
	}
}
