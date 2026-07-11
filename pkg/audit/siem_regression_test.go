package audit

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestSIEMRegression_StopTwiceNoPanic verifies that calling Stop() multiple times
// does not panic due to double-close of the stopCh channel.
// This is a regression test for the sync.Once fix.
func TestSIEMRegression_StopTwiceNoPanic(t *testing.T) {
	cfg := DefaultSIEMConfig()
	cfg.Endpoint = "http://localhost:0" // won't actually connect
	cfg.FlushInterval = 50 * time.Millisecond

	f := NewSIEMForwarder(cfg)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	f.Start(ctx)

	// Forward some events so buffer is non-empty
	for i := 0; i < 5; i++ {
		f.Forward(Event{Action: "test", Result: "success"})
	}

	// Stop twice — must not panic
	f.Stop()
	f.Stop()
	f.Stop() // third time for good measure
}

// TestSIEMRegression_MaxRetriesZeroDefaults verifies that MaxRetries=0 in config
// gets defaulted to 3 inside NewSIEMForwarder.
// This is a regression test for the fix where MaxRetries=0 caused the retry loop
// to never execute (for attempt := 0; attempt < 0; never enters).
func TestSIEMRegression_MaxRetriesZeroDefaults(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusInternalServerError) // always fail
	}))
	defer srv.Close()

	cfg := SIEMConfig{
		Provider:   SIEMProviderGeneric,
		Endpoint:   srv.URL,
		APIKey:     "key",
		MaxRetries: 0, // intentionally zero — should default to 3
		Timeout:    2 * time.Second,
	}

	f := NewSIEMForwarder(cfg)

	// Verify the config was patched
	if f.config.MaxRetries != 3 {
		t.Fatalf("expected MaxRetries to default to 3, got %d", f.config.MaxRetries)
	}

	f.Forward(Event{Action: "test", Result: "fail"})
	f.flush()

	// With MaxRetries=3, the server should have been hit 3 times
	if got := atomic.LoadInt32(&attempts); got != 3 {
		t.Errorf("expected 3 attempts (MaxRetries default), got %d", got)
	}
}

// TestSIEMRegression_FullLifecycle verifies Start → Forward → periodic flush → Stop.
// This tests the actual production code path where events flow through the goroutine.
func TestSIEMRegression_FullLifecycle(t *testing.T) {
	var received atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.Add(1)
		body, _ := io.ReadAll(r.Body)
		if len(body) == 0 {
			t.Error("empty body received")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := DefaultSIEMConfig()
	cfg.Endpoint = srv.URL
	cfg.APIKey = "test-key"
	cfg.BatchSize = 100 // won't auto-flush, periodic flush will trigger
	cfg.FlushInterval = 100 * time.Millisecond
	cfg.MaxRetries = 2

	f := NewSIEMForwarder(cfg)
	ctx, cancel := context.WithCancel(context.Background())
	f.Start(ctx)

	// Forward 10 events
	for i := 0; i < 10; i++ {
		f.Forward(Event{
			Action:       "user.login",
			Result:       "success",
			TenantID:     uuid.New(),
			ActorType:    "user",
			ActorID:      uuid.New(),
			ResourceType: "session",
			IPAddress:    "10.0.0.1",
			CreatedAt:    time.Now(),
		})
	}

	// Wait for periodic flush
	time.Sleep(300 * time.Millisecond)

	// Stop should trigger final flush
	f.Stop()

	if got := received.Load(); got < 1 {
		t.Errorf("expected at least 1 batch received, got %d", got)
	}
	cancel()
}

// TestSIEMRegression_ElasticsearchAuth verifies Elasticsearch Bearer auth header.
func TestSIEMRegression_ElasticsearchAuth(t *testing.T) {
	var authHeader string
	var contentType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		contentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := DefaultSIEMConfig()
	cfg.Provider = SIEMProviderElasticsearch
	cfg.Endpoint = srv.URL
	cfg.APIKey = "es-token"
	cfg.IndexName = "audit-idx"
	cfg.MaxRetries = 1

	f := NewSIEMForwarder(cfg)
	f.Forward(Event{Action: "test", Result: "ok"})
	f.flush()

	if authHeader != "Bearer es-token" {
		t.Errorf("expected 'Bearer es-token', got '%s'", authHeader)
	}
	if contentType != "application/x-ndjson" {
		t.Errorf("expected 'application/x-ndjson', got '%s'", contentType)
	}
}

// TestSIEMRegression_BatchTriggerAutoFlush verifies that Forward auto-flushes
// when buffer reaches BatchSize.
func TestSIEMRegression_BatchTriggerAutoFlush(t *testing.T) {
	var received atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.Add(1)
		// Verify batch contains all events
		var events []Event
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &events); err != nil {
			t.Errorf("unmarshal failed: %v", err)
		}
		if len(events) != 3 {
			t.Errorf("expected 3 events in batch, got %d", len(events))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := DefaultSIEMConfig()
	cfg.Endpoint = srv.URL
	cfg.APIKey = "key"
	cfg.BatchSize = 3 // auto-flush at 3
	cfg.MaxRetries = 1
	cfg.Timeout = 2 * time.Second

	f := NewSIEMForwarder(cfg)
	// Forward exactly BatchSize events — should trigger async flush
	for i := 0; i < 3; i++ {
		f.Forward(Event{Action: "test", Result: "ok"})
	}

	// Wait for async flush
	time.Sleep(500 * time.Millisecond)

	if got := received.Load(); got != 1 {
		t.Errorf("expected 1 batch (auto-flush), got %d", got)
	}
}

// TestSIEMRegression_EmptyBufferNoFlush verifies that flush with empty buffer
// is a no-op (no HTTP call made).
func TestSIEMRegression_EmptyBufferNoFlush(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("unexpected HTTP call on empty buffer")
	}))
	defer srv.Close()

	cfg := DefaultSIEMConfig()
	cfg.Endpoint = srv.URL
	cfg.MaxRetries = 1

	f := NewSIEMForwarder(cfg)
	f.flush() // buffer is empty — should be no-op
	f.flush() // double-check still empty
}
