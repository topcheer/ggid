package middleware

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// --- apikey.go: String() 0% coverage ---

func TestAPIKeyScopesKey_String(t *testing.T) {
	k := APIKeyScopesKey
	s := k.String()
	if s != string(k) {
		t.Errorf("expected %q, got %q", string(k), s)
	}
}

// --- audit_log.go: DroppedCount() 0% coverage ---

func TestNATSAuditPublisher_DroppedCount(t *testing.T) {
	// nil conn → Publish is no-op, dropped stays 0
	p := NewNATSAuditPublisher(nil, "audit.events")
	_ = p.Publish(&AuditEvent{Method: "GET", Path: "/t", Timestamp: time.Now()})
	if p.DroppedCount() != 1 {
		t.Errorf("expected 1 dropped for nil conn, got %d", p.DroppedCount())
	}

	// failing conn → Publish increments dropped counter
	failNC := &failingNATSConn{}
	p2 := NewNATSAuditPublisher(failNC, "audit.events")
	_ = p2.Publish(&AuditEvent{Method: "GET", Path: "/t", Timestamp: time.Now()})
	if p2.DroppedCount() != 1 {
		t.Errorf("expected 1 dropped for failing conn, got %d", p2.DroppedCount())
	}
}

type failingNATSConn struct{}

func (failingNATSConn) Publish(string, []byte) error { return fmt.Errorf("connection refused") }

// --- compress.go: WriteHeader, level pools ---

func TestCompressWriter_WriteHeader_BothPaths(t *testing.T) {
	// Test WriteHeader with skip=true (binary content type)
	w1 := httptest.NewRecorder()
	cw1 := &compressWriter{
		ResponseWriter: w1,
		wroteHeader:     false,
		skip:            true,
		supportsBrotli:  false,
	}
	cw1.WriteHeader(http.StatusTeapot)
	if w1.Code != http.StatusTeapot {
		t.Errorf("expected 418, got %d", w1.Code)
	}

	// Test WriteHeader with skip=false
	w2 := httptest.NewRecorder()
	cw2 := &compressWriter{
		ResponseWriter: w2,
		wroteHeader:     false,
		skip:            false,
		supportsBrotli:  false,
	}
	cw2.WriteHeader(http.StatusOK)
	if w2.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w2.Code)
	}
}

func TestGzipSyncPool_GetPut(t *testing.T) {
	pool := newGzipPool()
	w1 := pool.Get(6)
	if w1 == nil {
		t.Fatal("expected non-nil writer")
	}
	pool.Put(6, w1)
	w2 := pool.Get(6)
	if w2 == nil {
		t.Fatal("expected non-nil writer after Put")
	}
}

func TestBrotliSyncPool_GetPut(t *testing.T) {
	pool := newBrotliPool()
	w1 := pool.Get(4)
	if w1 == nil {
		t.Fatal("expected non-nil writer")
	}
	pool.Put(4, w1)
	w2 := pool.Get(4)
	if w2 == nil {
		t.Fatal("expected non-nil writer after Put")
	}
}

// --- ipallowlist.go: extractClientIP with comma in X-Forwarded-For ---

func TestExtractClientIP_MultipleForwarded(t *testing.T) {
	// Multiple IPs in X-Forwarded-For
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.5, 198.51.100.1, 10.0.0.1")
	ip := extractClientIP(req)
	if ip != "203.0.113.5" {
		t.Errorf("expected 203.0.113.5, got %s", ip)
	}

	// X-Real-IP only
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set("X-Real-IP", "192.0.2.10")
	ip2 := extractClientIP(req2)
	if ip2 != "192.0.2.10" {
		t.Errorf("expected 192.0.2.10, got %s", ip2)
	}

	// RemoteAddr with port
	req3 := httptest.NewRequest(http.MethodGet, "/", nil)
	req3.RemoteAddr = "198.51.100.42:54321"
	ip3 := extractClientIP(req3)
	if ip3 != "198.51.100.42" {
		t.Errorf("expected 198.51.100.42, got %s", ip3)
	}
}

// --- coalesce.go: Header() 66.7% ---

func TestCoalesceRecorder_Header_ExplicitSet(t *testing.T) {
	hdr := http.Header{}
	hdr.Set("X-Custom", "value")
	r := &coalesceRecorder{status: 200, header: hdr}
	h := r.Header()
	if h.Get("X-Custom") != "value" {
		t.Error("expected custom header")
	}
}

func TestCoalesceRecorder_Header_NilBoth(t *testing.T) {
	// Both header and ResponseWriter nil → should init a new map
	r := &coalesceRecorder{status: 200}
	h := r.Header()
	if h == nil {
		t.Error("expected non-nil headers")
	}
}

// --- graphql.go: inlineFragments 30.4% ---

func TestInlineFragments_WithFragment(t *testing.T) {
	query := `query GetUser {
  user(id: 1) {
    ...UserFields
  }
}
fragment UserFields on User {
  id
  name
  email
}`
	result := inlineFragments(query)
	// Should expand the fragment inline
	if !strings.Contains(result, "id") || !strings.Contains(result, "name") {
		t.Error("expected fragment fields to be inlined")
	}
}

func TestInlineFragments_NoFragment(t *testing.T) {
	query := `{ user { id name } }`
	result := inlineFragments(query)
	if result == "" {
		t.Error("expected non-empty result for query without fragments")
	}
}

func TestInlineFragments_Empty(t *testing.T) {
	result := inlineFragments("")
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

// --- session.go: touchSessionTTL 0% ---

func TestSessionManager_TouchSessionTTL_NilRedis(t *testing.T) {
	sm := NewSessionManager(nil)
	// Should not panic with nil redis
	sm.touchSessionTTL(context.Background(), "test-session", 30*time.Minute)
}

// --- circuitbreaker String() unknown state ---

func TestCircuitState_String_Unknown(t *testing.T) {
	s := CircuitState(99)
	if s.String() != "unknown" {
		t.Errorf("expected 'unknown', got %q", s.String())
	}
}

// --- compress.go setupWriter already done case ---

func TestCompressWriter_SetupWriter_AlreadySetup(t *testing.T) {
	w := httptest.NewRecorder()
	cw := &compressWriter{
		ResponseWriter: w,
		wroteHeader:     true,
	}
	// Should be a no-op since already set up
	cw.setupWriter()
	// No Content-Encoding should be set
	if ce := w.Header().Get("Content-Encoding"); ce != "" {
		t.Errorf("expected no Content-Encoding, got %q", ce)
	}
}
