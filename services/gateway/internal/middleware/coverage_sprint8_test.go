package middleware

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// --- circuitbreaker: state transitions ---

func TestCircuitBreaker_HalfOpenToOpenOnFailure(t *testing.T) {
	cb := NewCircuitBreaker(CircuitConfig{MaxFailures: 1, Timeout: 1 * time.Second})
	// Force to half-open
	cb.state = CircuitHalfOpen
	cb.RecordFailure()
	if cb.State() != CircuitOpen {
		t.Errorf("expected CircuitOpen after half-open failure, got %d", cb.State())
	}
}

func TestCircuitBreaker_HalfOpenToClosedOnSuccess(t *testing.T) {
	cb := NewCircuitBreaker(CircuitConfig{MaxFailures: 1, Timeout: 1 * time.Second, HalfOpenSuccess: 1})
	cb.state = CircuitHalfOpen
	cb.RecordSuccess()
	if cb.State() != CircuitClosed {
		t.Errorf("expected CircuitClosed after half-open success, got %d", cb.State())
	}
}

func TestCircuitBreaker_ClosedResetOnSuccess(t *testing.T) {
	cb := NewCircuitBreaker(CircuitConfig{MaxFailures: 5, Timeout: 30 * time.Second})
	cb.failures = 3
	cb.RecordSuccess()
	if cb.failures != 0 {
		t.Errorf("expected failures reset to 0, got %d", cb.failures)
	}
}

func TestCircuitBreaker_OpenTimeoutToHalfOpen(t *testing.T) {
	cb := NewCircuitBreaker(CircuitConfig{MaxFailures: 1, Timeout: 50 * time.Millisecond})
	cb.state = CircuitOpen
	cb.lastFailure = time.Now().Add(-1 * time.Second)
	allowed := cb.Allow()
	if !allowed {
		t.Error("expected Allow=true after timeout (half-open transition)")
	}
	if cb.State() != CircuitHalfOpen {
		t.Error("expected state=HalfOpen after timeout")
	}
}

func TestCircuitBreaker_HalfOpenMaxRequests(t *testing.T) {
	cb := NewCircuitBreaker(CircuitConfig{MaxFailures: 1, Timeout: 50 * time.Millisecond, HalfOpenMax: 1})
	cb.state = CircuitHalfOpen
	cb.halfOpenReq = 1
	allowed := cb.Allow()
	if allowed {
		t.Error("expected Allow=false when halfOpenReq >= HalfOpenMax")
	}
}

func TestCircuitBreaker_StatsSnapshot(t *testing.T) {
	cb := NewCircuitBreaker(CircuitConfig{MaxFailures: 3, Timeout: 30 * time.Second})
	cb.RecordFailure()
	stats := cb.Stats()
	if stats.State != CircuitClosed {
		t.Error("expected Closed state")
	}
	if stats.Failures != 1 {
		t.Errorf("expected 1 failure, got %d", stats.Failures)
	}
}

// --- canary routing ---

func TestCanaryRouter_HeaderTrue(t *testing.T) {
	cr := NewCanaryRouter(nil)
	cfg := &CanaryConfig{Percentage: 0, Header: "X-Canary"}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Canary", "true")
	if !cr.ShouldRouteCanary(cfg, req) {
		t.Error("expected canary=true when header=true")
	}
}

func TestCanaryRouter_HeaderFalse(t *testing.T) {
	cr := NewCanaryRouter(nil)
	cfg := &CanaryConfig{Percentage: 100, Header: "X-Canary"}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Canary", "false")
	if cr.ShouldRouteCanary(cfg, req) {
		t.Error("expected canary=false when header=false")
	}
}

func TestCanaryRouter_CookieCanary(t *testing.T) {
	cr := NewCanaryRouter(nil)
	cfg := &CanaryConfig{Percentage: 0, CookieName: "canary"}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "canary", Value: "canary"})
	if !cr.ShouldRouteCanary(cfg, req) {
		t.Error("expected canary=true when cookie=canary")
	}
}

func TestCanaryRouter_CookieStable(t *testing.T) {
	cr := NewCanaryRouter(nil)
	cfg := &CanaryConfig{Percentage: 100, CookieName: "canary"}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "canary", Value: "stable"})
	if cr.ShouldRouteCanary(cfg, req) {
		t.Error("expected canary=false when cookie=stable")
	}
}

func TestCanaryRouter_Percentage100(t *testing.T) {
	cr := NewCanaryRouter(nil)
	cfg := &CanaryConfig{Percentage: 100}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if !cr.ShouldRouteCanary(cfg, req) {
		t.Error("expected canary=true at 100%")
	}
}

func TestCanaryRouter_Percentage0(t *testing.T) {
	cr := NewCanaryRouter(nil)
	cfg := &CanaryConfig{Percentage: 0}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if cr.ShouldRouteCanary(cfg, req) {
		t.Error("expected canary=false at 0%")
	}
}

// --- ipallowlist matching ---

func mustCIDR(s string) *net.IPNet {
	_, n, _ := net.ParseCIDR(s)
	return n
}

func TestIPAllowlist_IPDenied(t *testing.T) {
	al := NewIPAllowlist(map[string][]*net.IPNet{
		"tenant1": {mustCIDR("10.0.0.0/8")},
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	ctx := context.WithValue(req.Context(), TenantIDKey, "tenant1")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	called := false
	al.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})).ServeHTTP(w, req)

	if called {
		t.Error("expected handler NOT called for denied IP")
	}
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestIPAllowlist_IPAllowed(t *testing.T) {
	al := NewIPAllowlist(map[string][]*net.IPNet{
		"tenant1": {mustCIDR("10.0.0.0/8")},
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.1.2.3:12345"
	ctx := context.WithValue(req.Context(), TenantIDKey, "tenant1")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	called := false
	al.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(w, req)

	if !called {
		t.Error("expected handler called for allowed IP")
	}
}

func TestIPAllowlist_NoTenantPasses(t *testing.T) {
	al := NewIPAllowlist(map[string][]*net.IPNet{
		"tenant1": {mustCIDR("10.0.0.0/8")},
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	called := false
	al.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})).ServeHTTP(w, req)
	if !called {
		t.Error("expected handler called when no tenant")
	}
}

func TestIPAllowlist_InvalidIP(t *testing.T) {
	al := NewIPAllowlist(map[string][]*net.IPNet{
		"tenant1": {mustCIDR("10.0.0.0/8")},
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "not-an-ip"
	ctx := context.WithValue(req.Context(), TenantIDKey, "tenant1")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	al.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for invalid IP, got %d", w.Code)
	}
}

// --- audit_log: NewAuditLogger ---

func TestNewAuditLogger_NilPublisher(t *testing.T) {
	al := NewAuditLogger(nil, 0)
	if al == nil {
		t.Fatal("expected non-nil AuditLogger")
	}
	al.Enqueue(&AuditEvent{Method: "GET", Path: "/test", Timestamp: time.Now()})
	al.Stop()
}

func TestNewAuditLogger_WithPublisher(t *testing.T) {
	pub := NewNATSAuditPublisher(nil, "test")
	al := NewAuditLogger(pub, 10)
	if al == nil {
		t.Fatal("expected non-nil AuditLogger")
	}
	al.Enqueue(&AuditEvent{Method: "POST", Path: "/data", StatusCode: 201, Timestamp: time.Now()})
	al.Stop()
}

// --- OTel sampling ---

func TestShouldSample_Boundaries(t *testing.T) {
	if !shouldSample(1.0) {
		t.Error("expected sampling at rate 1.0")
	}
	if shouldSample(0) {
		t.Error("expected no sampling at rate 0")
	}
}
