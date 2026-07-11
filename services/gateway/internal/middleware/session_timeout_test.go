package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestSessionTimeoutConfig_Defaults(t *testing.T) {
	cfg := DefaultSessionTimeoutConfig()
	if cfg.AbsoluteTimeout != 8*time.Hour {
		t.Errorf("expected 8h absolute, got %v", cfg.AbsoluteTimeout)
	}
	if cfg.IdleTimeout != 30*time.Minute {
		t.Errorf("expected 30m idle, got %v", cfg.IdleTimeout)
	}
}

func TestCheckSessionTimeoutRedis_NilRedis(t *testing.T) {
	// Nil Redis should fail open (return nil).
	err := CheckSessionTimeoutRedis(context.Background(), nil, "sess-1", DefaultSessionTimeoutConfig())
	if err != nil {
		t.Errorf("expected nil error with nil Redis, got %v", err)
	}
}

func TestCheckSessionTimeoutRedis_Errors(t *testing.T) {
	if ErrSessionTimeoutAbsolute == nil {
		t.Error("ErrSessionTimeoutAbsolute should not be nil")
	}
	if ErrSessionTimeoutIdle == nil {
		t.Error("ErrSessionTimeoutIdle should not be nil")
	}
}

func TestSessionTimeoutMiddleware_PublicPath(t *testing.T) {
	sm := &SessionManager{rdb: nil}
	mw := sm.SessionTimeoutMiddleware(DefaultSessionTimeoutConfig())

	called := false
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest("GET", "/healthz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("expected handler to be called for public path")
	}
}

func TestSessionTimeoutMiddleware_NoSession(t *testing.T) {
	sm := &SessionManager{rdb: nil} // nil Redis — fail open
	mw := sm.SessionTimeoutMiddleware(DefaultSessionTimeoutConfig())

	called := false
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("expected handler called (no session, fail open)")
	}
}

// Suppress unused import guard.
var _ = redis.NewClient
