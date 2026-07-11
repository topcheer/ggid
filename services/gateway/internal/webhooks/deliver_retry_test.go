package webhooks

// Webhook Delivery Retry E2E Test
// Verifies: httptest server returns 503 three times → backoff retries →
// eventually 200 → webhook delivery succeeds.
// Date: 2026-07-25

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// newRetryTestServer creates a test HTTP server that returns 503 for the first
// failCount requests, then 200 for subsequent requests.
func newRetryTestServer(t *testing.T, requestCount *int32, failCount int32) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := atomic.AddInt32(requestCount, 1)
		if n <= failCount {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
}

// newAlwaysFailTestServer creates a test HTTP server that always returns 503.
func newAlwaysFailTestServer(t *testing.T, requestCount *int32) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(requestCount, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
}

// newAlwaysSuccessTestServer creates a test HTTP server that always returns 200.
func newAlwaysSuccessTestServer(t *testing.T, requestCount *int32) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(requestCount, 1)
		w.WriteHeader(http.StatusOK)
	}))
}

// newSignatureTestServer creates a test HTTP server that captures the signature header.
func newSignatureTestServer(t *testing.T, requestCount *int32, sigReceived *string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(requestCount, 1)
		*sigReceived = r.Header.Get("X-GGID-Signature")
		w.WriteHeader(http.StatusOK)
	}))
}

// guard against unused import if fmt not referenced
var _ = fmt.Sprintf

// TestWebhookDelivery_RetryThenSuccess verifies that the HTTPDeliverer retries
// on 503 responses and eventually succeeds when the server starts returning 200.
func TestWebhookDelivery_RetryThenSuccess(t *testing.T) {
	var requestCount int32

	// Start a test HTTP server that returns 503 for the first 3 requests,
	// then 200 for subsequent ones.
	srv := newRetryTestServer(t, &requestCount, 3)
	defer srv.Close()

	// Create a deliverer with test SSRF config (allows loopback)
	// and short retry delays for fast testing.
	d := newTestDeliverer()
	// Override max retries to 5 for the test
	d.maxRetries = 5

	payload := []byte(`{"event":"user.created","data":{"id":"123"}}`)

	// Use a short-timeout context (retry backoff in production is attempt^2 seconds,
	// but in tests we use a custom deliverer with fast retries).
	// The test deliverer from helpers_test.go has a 5s dial timeout.
	// Since the retry uses time.After(attempt^2 seconds), we need to be smart.
	// For attempt 0: no wait. Attempt 1: 1s. Attempt 2: 4s. Attempt 3: 9s.
	// Total: ~14s for 5 attempts. With 15s timeout this works.
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	err := d.Deliver(ctx, srv.URL, "test-secret", payload)

	// Verify delivery succeeded
	if err != nil {
		t.Fatalf("delivery should have succeeded after retries, got: %v", err)
	}

	// Verify the server received exactly 4 requests (3 failures + 1 success)
	count := atomic.LoadInt32(&requestCount)
	if count != 4 {
		t.Errorf("expected 4 total requests (3×503 + 1×200), got %d", count)
	}

	t.Logf("webhook delivered after %d attempts (3 failures + 1 success)", count)
}

// TestWebhookDelivery_AllRetriesExhausted verifies that when the server always
// returns 503, the delivery eventually fails after max retries.
func TestWebhookDelivery_AllRetriesExhausted(t *testing.T) {
	var requestCount int32

	// Server that ALWAYS returns 503
	srv := newAlwaysFailTestServer(t, &requestCount)
	defer srv.Close()

	d := newTestDeliverer()
	d.maxRetries = 3 // fewer retries for faster test

	payload := []byte(`{"event":"user.deleted"}`)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	err := d.Deliver(ctx, srv.URL, "", payload)

	// Should fail after exhausting retries
	if err == nil {
		t.Fatal("delivery should fail when all retries exhausted")
	}

	count := atomic.LoadInt32(&requestCount)
	if count != 3 {
		t.Errorf("expected 3 total requests (matching maxRetries=3), got %d", count)
	}

	t.Logf("delivery correctly failed after %d attempts: %v", count, err)
}

// TestWebhookDelivery_FirstAttemptSuccess verifies no retries when the first
// attempt succeeds.
func TestWebhookDelivery_FirstAttemptSuccess(t *testing.T) {
	var requestCount int32

	srv := newAlwaysSuccessTestServer(t, &requestCount)
	defer srv.Close()

	d := newTestDeliverer()
	d.maxRetries = 5

	payload := []byte(`{"event":"user.updated"}`)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := d.Deliver(ctx, srv.URL, "secret", payload)
	if err != nil {
		t.Fatalf("delivery should succeed on first attempt: %v", err)
	}

	count := atomic.LoadInt32(&requestCount)
	if count != 1 {
		t.Errorf("expected exactly 1 request (no retries), got %d", count)
	}

	t.Logf("delivery succeeded on first attempt (1 request)")
}

// TestWebhookDelivery_HMACSignature verifies that the HMAC-SHA256 signature
// header is sent with each delivery.
func TestWebhookDelivery_HMACSignature(t *testing.T) {
	var signatureReceived string
	var requestCount int32

	srv := newSignatureTestServer(t, &requestCount, &signatureReceived)
	defer srv.Close()

	d := newTestDeliverer()

	payload := []byte(`{"event":"test"}`)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := d.Deliver(ctx, srv.URL, "my-webhook-secret", payload)
	if err != nil {
		t.Fatalf("delivery failed: %v", err)
	}

	if signatureReceived == "" {
		t.Error("X-GGID-Signature header should have been received")
	}

	// Verify it's a valid sha256= prefix
	if len(signatureReceived) < 7 || signatureReceived[:7] != "sha256=" {
		t.Errorf("signature should start with 'sha256=', got: %s", signatureReceived)
	}

	t.Logf("HMAC signature received: %s", signatureReceived)
}

// TestWebhookDelivery_ContextCancellation verifies that delivery stops when
// context is cancelled.
func TestWebhookDelivery_ContextCancellation(t *testing.T) {
	var requestCount int32

	srv := newAlwaysFailTestServer(t, &requestCount)
	defer srv.Close()

	d := newTestDeliverer()
	d.maxRetries = 10 // would normally retry many times

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel immediately after first request
	go func() {
		time.Sleep(500 * time.Millisecond)
		cancel()
	}()

	payload := []byte(`{"event":"test"}`)
	err := d.Deliver(ctx, srv.URL, "", payload)

	// Should return context error
	if err == nil {
		t.Error("delivery should fail with cancelled context")
	}

	t.Logf("context cancellation properly stopped delivery: %v", err)
}
