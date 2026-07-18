package shutdown

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestIsShuttingDown_Default(t *testing.T) {
	if IsShuttingDown() {
		t.Fatal("should be false by default")
	}
}

func TestHealthCheckMiddleware_Normal(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(200)
	})
	mw := HealthCheckMiddleware(next)

	req := httptest.NewRequest("GET", "/healthz", nil)
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	if !called {
		t.Fatal("handler should be called when not shutting down")
	}
}

func TestHealthCheckMiddleware_DuringShutdown(t *testing.T) {
	shuttingDown.Store(true)
	defer shuttingDown.Store(false)

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})
	mw := HealthCheckMiddleware(next)

	req := httptest.NewRequest("GET", "/healthz", nil)
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	if called {
		t.Fatal("handler should NOT be called during shutdown")
	}
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

func TestManager_Execute_OrderedShutdown(t *testing.T) {
	var order []int
	var mu sync.Mutex

	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
	}

	// Start a test server.
	ln := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	srv.Addr = ln.Listener.Addr().String()
	go srv.ListenAndServe()
	time.Sleep(50 * time.Millisecond)

	mgr := New(&Resources{
		HTTPServer: srv,
		OnShutdown: []func(ctx context.Context) error{
			func(ctx context.Context) error {
				mu.Lock()
				order = append(order, 2)
				mu.Unlock()
				return nil
			},
			func(ctx context.Context) error {
				mu.Lock()
				order = append(order, 3)
				mu.Unlock()
				return nil
			},
		},
	})

	mgr.execute()

	mu.Lock()
	defer mu.Unlock()

	// HTTP shutdown should be step 1, then custom handlers.
	if len(order) != 2 {
		t.Fatalf("expected 2 custom handlers, got %d", len(order))
	}
	if order[0] != 2 || order[1] != 3 {
		t.Fatalf("expected order [2,3], got %v", order)
	}

	// shuttingDown flag should be set.
	if !IsShuttingDown() {
		t.Fatal("should be shutting down after execute")
	}

	ln.Close()
}

func TestManager_NilResources(t *testing.T) {
	mgr := New(&Resources{})
	mgr.execute()
	// Should not panic with nil resources.
	if !IsShuttingDown() {
		t.Fatal("should set shuttingDown even with nil resources")
	}
}

func TestManager_WithTimeout(t *testing.T) {
	mgr := New(&Resources{}).WithTimeout(5 * time.Second)
	if mgr.timeout != 5*time.Second {
		t.Fatalf("expected 5s timeout, got %v", mgr.timeout)
	}
}

func TestRegister_ConvenienceFunc(t *testing.T) {
	// Just verify it doesn't panic with nil resources.
	// We can't test Wait() because it blocks for signal.
	r := &Resources{}
	m := New(r)
	if m == nil {
		t.Fatal("New should return non-nil manager")
	}
}
