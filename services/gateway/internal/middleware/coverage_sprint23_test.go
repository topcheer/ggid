package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Task 1: JTI replay tracker (was 0%)

func TestJTIReplay_New_C23(t *testing.T) {
	tracker := NewJTIReplayTracker(5 * time.Minute)
	if tracker == nil {
		t.Fatal("nil tracker")
	}
}

func TestJTIReplay_FirstUse_C23(t *testing.T) {
	tracker := NewJTIReplayTracker(5 * time.Minute)
	// First use should return false (not replayed)
	if tracker.IsReplayed("jti-001", time.Now().Add(5*time.Minute)) {
		t.Error("first use should not be replayed")
	}
}

func TestJTIReplay_ReplayDetected_C23(t *testing.T) {
	tracker := NewJTIReplayTracker(5 * time.Minute)
	// First use
	tracker.IsReplayed("jti-002", time.Now().Add(5*time.Minute))
	// Second use with same jti should be replayed
	if !tracker.IsReplayed("jti-002", time.Now().Add(5*time.Minute)) {
		t.Error("second use should be replayed")
	}
}

func TestJTIReplay_DifferentJTI_C23(t *testing.T) {
	tracker := NewJTIReplayTracker(5 * time.Minute)
	tracker.IsReplayed("jti-a", time.Now().Add(5*time.Minute))
	if tracker.IsReplayed("jti-b", time.Now().Add(5*time.Minute)) {
		t.Error("different jti should not be replayed")
	}
}

func TestJTIReplay_Expired_C23(t *testing.T) {
	tracker := NewJTIReplayTracker(1 * time.Millisecond)
	tracker.IsReplayed("jti-old", time.Now().Add(1*time.Millisecond))
	time.Sleep(10 * time.Millisecond)
	// After expiry, same jti should be accepted again (cleanup removes it)
	if tracker.IsReplayed("jti-old", time.Now().Add(5*time.Minute)) {
		t.Error("expired jti should be accepted again")
	}
}

// Task 1: JSONLogger Info/Warn/Error (was 0%)

func TestJSONLogger_Info_C23(t *testing.T) {
	var output string
	logger := NewJSONLogger(func(s string) { output = s })
	logger.Info(LogEntry{Method: "GET", Path: "/ok", Status: 200})
	if output == "" {
		t.Error("Info should produce output")
	}
	if !strings.Contains(output, "info") {
		t.Errorf("output should contain 'info': %s", output)
	}
}

func TestJSONLogger_Warn_C23(t *testing.T) {
	var output string
	logger := NewJSONLogger(func(s string) { output = s })
	logger.Warn(LogEntry{Method: "POST", Path: "/warn", Status: 404})
	if !strings.Contains(output, "warn") {
		t.Errorf("output should contain 'warn': %s", output)
	}
}

func TestJSONLogger_Error_C23(t *testing.T) {
	var output string
	logger := NewJSONLogger(func(s string) { output = s })
	logger.Error(LogEntry{Method: "PUT", Path: "/err", Status: 500})
	if !strings.Contains(output, "error") {
		t.Errorf("output should contain 'error': %s", output)
	}
}

func TestJSONLogger_NilWriter_C23(t *testing.T) {
	logger := &JSONLogger{writer: nil}
	// Should not panic with nil writer
	logger.Info(LogEntry{Method: "GET", Path: "/", Status: 200})
}

// Task 1: HostValidation integration

func TestHostValidation_LambdaStyle_C23(t *testing.T) {
	called := false
	h := HostValidation(HostValidationConfig{
		AllowedHosts: []string{"api.ggid.dev"},
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(200)
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Host = "api.ggid.dev"
	h.ServeHTTP(rr, req)
	if !called {
		t.Error("allowed host should reach handler")
	}
}

// Task 1: SSRF-related middleware coverage

func TestIPAllowlist_AllowAll_C23(t *testing.T) {
	al := NewIPAllowlist(nil)
	called := false
	h := al.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(200)
	}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	h.ServeHTTP(rr, req)
	if !called {
		t.Error("nil CIDRs should allow all")
	}
}
