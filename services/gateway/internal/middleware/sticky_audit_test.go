package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// --- Sticky Session Tests ---

func TestStickyRouter_SingleBackend(t *testing.T) {
	sr := NewStickyRouter(&StickySessionConfig{
		CookieName: "sticky",
		Backends:   []string{"http://b1:8080"},
	})
	r := httptest.NewRequest("GET", "/", nil)
	r.AddCookie(&http.Cookie{Name: "sticky", Value: "user1"})
	if got := sr.ResolveBackend(r); got != "http://b1:8080" {
		t.Errorf("expected single backend, got %s", got)
	}
}

func TestStickyRouter_NoBackends(t *testing.T) {
	sr := NewStickyRouter(&StickySessionConfig{Backends: []string{}})
	r := httptest.NewRequest("GET", "/", nil)
	if got := sr.ResolveBackend(r); got != "" {
		t.Errorf("expected empty, got %s", got)
	}
}

func TestStickyRouter_ConsistentRouting(t *testing.T) {
	sr := NewStickyRouter(&StickySessionConfig{
		CookieName: "sticky",
		Backends:   []string{"http://b1:8080", "http://b2:8080", "http://b3:8080"},
		TTL:        5 * time.Minute,
	})
	r := httptest.NewRequest("GET", "/", nil)
	r.AddCookie(&http.Cookie{Name: "sticky", Value: "user-abc"})

	first := sr.ResolveBackend(r)
	for i := 0; i < 10; i++ {
		got := sr.ResolveBackend(r)
		if got != first {
			t.Errorf("expected consistent %s, got %s on iter %d", first, got, i)
		}
	}
	if sr.BindingCount() != 1 {
		t.Errorf("expected 1 binding, got %d", sr.BindingCount())
	}
}

func TestStickyRouter_HeaderKey(t *testing.T) {
	sr := NewStickyRouter(&StickySessionConfig{
		HeaderName: "X-Sticky-Key",
		Backends:   []string{"http://b1:8080", "http://b2:8080"},
	})
	r1 := httptest.NewRequest("GET", "/", nil)
	r1.Header.Set("X-Sticky-Key", "user-x")
	b1 := sr.ResolveBackend(r1)

	r2 := httptest.NewRequest("GET", "/", nil)
	r2.Header.Set("X-Sticky-Key", "user-x")
	b2 := sr.ResolveBackend(r2)

	if b1 != b2 {
		t.Errorf("expected same backend for same key, got %s and %s", b1, b2)
	}
}

func TestStickyRouter_NoKey(t *testing.T) {
	sr := NewStickyRouter(&StickySessionConfig{
		Backends: []string{"http://b1:8080", "http://b2:8080"},
	})
	r := httptest.NewRequest("GET", "/", nil)
	got := sr.ResolveBackend(r)
	if got != "http://b1:8080" {
		t.Errorf("expected first backend, got %s", got)
	}
}

func TestStickyRouter_TTLExpiry(t *testing.T) {
	sr := NewStickyRouter(&StickySessionConfig{
		CookieName: "sticky",
		Backends:   []string{"http://b1:8080", "http://b2:8080"},
		TTL:        1 * time.Millisecond,
	})
	r := httptest.NewRequest("GET", "/", nil)
	r.AddCookie(&http.Cookie{Name: "sticky", Value: "user1"})
	sr.ResolveBackend(r)
	if sr.BindingCount() != 1 {
		t.Fatal("expected 1 binding")
	}
	time.Sleep(5 * time.Millisecond)
	sr.CleanupExpired()
	if sr.BindingCount() != 0 {
		t.Errorf("expected 0 after cleanup, got %d", sr.BindingCount())
	}
}

func TestStickyRouter_SetCookie(t *testing.T) {
	sr := NewStickyRouter(&StickySessionConfig{
		CookieName: "sticky",
		Backends:   []string{"http://b1:8080"},
	})
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	sr.SetStickyCookie(w, r)
	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected cookie to be set")
	}
	if cookies[0].Name != "sticky" {
		t.Errorf("expected sticky, got %s", cookies[0].Name)
	}
}

func TestStickyRouter_SetCookie_AlreadyHasKey(t *testing.T) {
	sr := NewStickyRouter(&StickySessionConfig{
		CookieName: "sticky",
		Backends:   []string{"http://b1:8080"},
	})
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r.AddCookie(&http.Cookie{Name: "sticky", Value: "existing"})
	sr.SetStickyCookie(w, r)
	if len(w.Result().Cookies()) != 0 {
		t.Error("should not set cookie when one already exists")
	}
}

func TestStickyMiddleware(t *testing.T) {
	sr := NewStickyRouter(&StickySessionConfig{
		CookieName: "sticky",
		Backends:   []string{"http://b1:8080", "http://b2:8080"},
	})
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Header.Get("X-Sticky-Backend") == "" {
			t.Error("expected X-Sticky-Backend header")
		}
	})
	h := StickyMiddleware(sr, next)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r.AddCookie(&http.Cookie{Name: "sticky", Value: "user1"})
	h.ServeHTTP(w, r)
	if !called {
		t.Error("next handler not called")
	}
}

func TestDefaultStickyConfig(t *testing.T) {
	cfg := DefaultStickyConfig()
	if cfg.CookieName != "ggid_sticky" {
		t.Errorf("expected ggid_sticky, got %s", cfg.CookieName)
	}
	if cfg.TTL != 30*time.Minute {
		t.Errorf("expected 30m, got %v", cfg.TTL)
	}
}

func TestNewStickyRouter_Defaults(t *testing.T) {
	sr := NewStickyRouter(nil)
	if sr.config.CookieName != "ggid_sticky" {
		t.Errorf("expected default cookie name, got %s", sr.config.CookieName)
	}
}

// --- Audit Logging Tests ---

// mockNATSConn captures published messages for testing.
type mockNATSConn struct {
	messages [][]byte
}

func (m *mockNATSConn) Publish(_ string, data []byte) error {
	m.messages = append(m.messages, data)
	return nil
}

func TestNATSAuditPublisher_Publish(t *testing.T) {
	mock := &mockNATSConn{}
	pub := NewNATSAuditPublisher(mock, "audit.events")
	err := pub.Publish(&AuditEvent{Method: "GET", Path: "/test"})
	if err != nil {
		t.Fatal(err)
	}
	if len(mock.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(mock.messages))
	}
}

func TestNATSAuditPublisher_NilConn(t *testing.T) {
	pub := NewNATSAuditPublisher(nil, "audit.events")
	err := pub.Publish(&AuditEvent{Method: "GET"})
	if err != nil {
		t.Errorf("expected nil error for nil conn, got %v", err)
	}
}

func TestAuditLogger_AsyncPublish(t *testing.T) {
	mock := &mockNATSConn{}
	pub := NewNATSAuditPublisher(mock, "audit")
	logger := NewAuditLogger(pub, 100)

	logger.Enqueue(&AuditEvent{Method: "GET", Path: "/a"})
	logger.Enqueue(&AuditEvent{Method: "POST", Path: "/b"})

	// Wait for async processing
	time.Sleep(100 * time.Millisecond)
	logger.Stop()

	if len(mock.messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(mock.messages))
	}
}

func TestAuditLogger_QueueFullDrops(t *testing.T) {
	pub := noopAuditPublisher{}
	logger := NewAuditLogger(pub, 1) // tiny queue
	// Fill queue + overflow
	for i := 0; i < 100; i++ {
		logger.Enqueue(&AuditEvent{Method: "GET"})
	}
	// Should not block or panic
	logger.Stop()
}

func TestAuditMiddleware_CapturesRequest(t *testing.T) {
	mock := &mockNATSConn{}
	pub := NewNATSAuditPublisher(mock, "audit")
	logger := NewAuditLogger(pub, 100)
	defer logger.Stop()

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("hello"))
	})
	h := AuditMiddleware(logger)(next)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/v1/users", nil)
	r.Header.Set("X-Tenant-ID", "t1")
	r.Header.Set("X-Request-ID", "req-1")
	h.ServeHTTP(w, r)

	// Wait for async
	time.Sleep(100 * time.Millisecond)

	if len(mock.messages) != 1 {
		t.Fatalf("expected 1 audit message, got %d", len(mock.messages))
	}
	// Verify the event content
	var event AuditEvent
	jsonUnmarshal(t, mock.messages[0], &event)
	if event.Method != "POST" {
		t.Errorf("expected POST, got %s", event.Method)
	}
	if event.StatusCode != http.StatusCreated {
		t.Errorf("expected 201, got %d", event.StatusCode)
	}
	if event.TenantID != "t1" {
		t.Errorf("expected t1, got %s", event.TenantID)
	}
	if event.BytesSent != 5 {
		t.Errorf("expected 5 bytes, got %d", event.BytesSent)
	}
}

// helper
func jsonUnmarshal(t *testing.T, data []byte, v any) {
	t.Helper()
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatal(err)
	}
}
