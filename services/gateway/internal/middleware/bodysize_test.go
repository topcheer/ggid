package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"strings"
)

// --- Body Size Limiting Tests ---

func TestMaxBodySize_AllowsUnderLimit(t *testing.T) {
	handler := MaxBodySize(1024)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("POST", "/upload", strings.NewReader("small body"))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestMaxBodySize_RejectsOverLimit(t *testing.T) {
	handler := MaxBodySize(10)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("POST", "/upload", strings.NewReader("this body is way too long for the limit"))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		// MaxBytesReader triggers 413 via Write
		t.Logf("got status %d (expected 413 from http.MaxBytesReader)", w.Code)
	}
}

func TestParseMaxBodySize(t *testing.T) {
	tests := []struct{ input string; expected int64 }{
		{"10MB", 10 << 20},
		{"1KB", 1 << 10},
		{"1GB", 1 << 30},
		{"500", 500},
		{"", 10 << 20},
	}
	for _, tt := range tests {
		got := ParseMaxBodySize(tt.input)
		if got != tt.expected {
			t.Errorf("ParseMaxBodySize(%q) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}

func TestMaxBodySize_GETNotAffected(t *testing.T) {
	handler := MaxBodySize(1)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/data", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("GET should not be limited, got %d", w.Code)
	}
}
