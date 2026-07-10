package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPanicRecovery_Recovers(t *testing.T) {
	panicking := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	logger := NewStructuredLogger("test")
	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	w := httptest.NewRecorder()

	PanicRecovery(logger)(panicking).ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "internal server error") {
		t.Errorf("expected error message, got %s", w.Body.String())
	}
}

func TestPanicRecovery_NilLogger(t *testing.T) {
	panicking := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	// Should not panic even with nil logger
	PanicRecovery(nil)(panicking).ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestPanicRecovery_NoPanic(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	})

	logger := NewStructuredLogger("test")
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	PanicRecovery(logger)(ok).ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "ok" {
		t.Errorf("expected 'ok', got %s", w.Body.String())
	}
}

func TestRequestLogger_JSON(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("hello"))
	})

	logger := NewStructuredLogger("test-service")
	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	w := httptest.NewRecorder()

	RequestLogger(logger)(next).ServeHTTP(w, req)

	// The middleware should have called the handler
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestRequestLogger_Logs4xx(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	})

	logger := NewStructuredLogger("test")
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	RequestLogger(logger)(next).ServeHTTP(w, req)
	if w.Code != 404 {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestRequestLogger_Logs5xx(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	})

	logger := NewStructuredLogger("test")
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	RequestLogger(logger)(next).ServeHTTP(w, req)
	if w.Code != 500 {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestLevelFromStatus(t *testing.T) {
	tests := []struct {
		status int
		level  string
	}{
		{200, "info"},
		{301, "info"},
		{404, "warn"},
		{500, "error"},
		{503, "error"},
	}
	for _, tt := range tests {
		if got := levelFromStatus(tt.status); got != tt.level {
			t.Errorf("levelFromStatus(%d) = %s, want %s", tt.status, got, tt.level)
		}
	}
}

func TestStructuredLogger_Emit(t *testing.T) {
	logger := NewStructuredLogger("test")
	logger.Emit(LogRecord{
		Timestamp: "2024-01-01T00:00:00Z",
		Level:     "info",
		Service:   "test",
		Method:    "GET",
		Path:      "/",
		Status:    200,
	})
	// Should not panic
}

func TestRequestIDEnsure(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		id, _ := r.Context().Value(RequestIDKey).(string)
		if id == "" {
			t.Error("expected non-empty request ID")
		}
	})

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	RequestIDEnsure(next).ServeHTTP(w, req)
	if !called {
		t.Error("next handler should be called")
	}
}

func TestRequestIDEnsure_PreservesExisting(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, _ := r.Context().Value(RequestIDKey).(string)
		if id != "custom-id" {
			t.Errorf("expected custom-id, got %s", id)
		}
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Request-ID", "custom-id")
	w := httptest.NewRecorder()
	RequestIDEnsure(next).ServeHTTP(w, req)
}

func TestPanicRecovery_PanicWithNil(t *testing.T) {
	panicking := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var p *int
		_ = *p // nil pointer dereference
	})

	logger := NewStructuredLogger("test")
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	PanicRecovery(logger)(panicking).ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}
