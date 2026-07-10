package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/andybalholm/brotli"
)

func TestGzipBrotli_NoAcceptEncoding(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world"))
	})
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	GzipBrotli(next).ServeHTTP(w, req)
	if enc := w.Header().Get("Content-Encoding"); enc != "" {
		t.Errorf("expected no encoding, got %s", enc)
	}
}

func TestGzipBrotli_GzipEncoding(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(strings.Repeat("hello world ", 100)))
	})
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	GzipBrotli(next).ServeHTTP(w, req)
	if w.Header().Get("Content-Encoding") != "gzip" {
		t.Errorf("expected gzip encoding, got %s", w.Header().Get("Content-Encoding"))
	}
	gr, err := gzip.NewReader(w.Body)
	if err != nil {
		t.Fatalf("gzip reader error: %v", err)
	}
	decompressed, _ := io.ReadAll(gr)
	if !strings.Contains(string(decompressed), "hello") {
		t.Error("decompressed body should contain 'hello'")
	}
}

func TestGzipBrotli_BrotliPreferred(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message":"` + strings.Repeat("x", 1000) + `"}`))
	})
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip, br")
	w := httptest.NewRecorder()
	GzipBrotli(next).ServeHTTP(w, req)
	if w.Header().Get("Content-Encoding") != "br" {
		t.Errorf("expected br (brotli preferred), got %s", w.Header().Get("Content-Encoding"))
	}
	br := brotli.NewReader(w.Body)
	decompressed, _ := io.ReadAll(br)
	if !strings.Contains(string(decompressed), "message") {
		t.Error("decompressed body should contain 'message'")
	}
}

func TestGzipBrotli_SkipBinary(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write([]byte("binary data"))
	})
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip, br")
	w := httptest.NewRecorder()
	GzipBrotli(next).ServeHTTP(w, req)
	if enc := w.Header().Get("Content-Encoding"); enc != "" {
		t.Errorf("should not compress image/png, got encoding %s", enc)
	}
}

func TestCompressionLevelForType(t *testing.T) {
	tests := []struct {
		ct   string
		want int
	}{
		{"text/html", 6},
		{"text/html; charset=utf-8", 6},
		{"text/css", 6},
		{"application/javascript", 6},
		{"application/json", 4},
		{"application/xml", 4},
		{"text/plain", 4},
		{"application/octet-stream", 1},
	}
	for _, tt := range tests {
		got := compressionLevelForType(tt.ct)
		if got != tt.want {
			t.Errorf("compressionLevelForType(%q) = %d, want %d", tt.ct, got, tt.want)
		}
	}
}

func TestAcceptEncoding(t *testing.T) {
	tests := []struct {
		header string
		enc    string
		want   bool
	}{
		{"gzip, br", "br", true},
		{"gzip, br", "gzip", true},
		{"gzip", "br", false},
		{"br", "gzip", false},
		{"", "gzip", false},
		{"gzip;q=0", "gzip", true}, // prefix match still works
	}
	for _, tt := range tests {
		got := acceptEncoding(tt.header, tt.enc)
		if got != tt.want {
			t.Errorf("acceptEncoding(%q, %q) = %v, want %v", tt.header, tt.enc, got, tt.want)
		}
	}
}

func TestParseContentEncoding(t *testing.T) {
	result := parseContentEncoding("gzip;q=0.8, br;q=1.0")
	if result["gzip"] != 0.8 {
		t.Errorf("expected gzip q=0.8, got %v", result["gzip"])
	}
	if result["br"] != 1.0 {
		t.Errorf("expected br q=1.0, got %v", result["br"])
	}
}

func TestGzipBrotli_PassesThrough(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(204)
	})
	req := httptest.NewRequest("GET", "/", nil)
	// No Accept-Encoding header
	w := httptest.NewRecorder()
	GzipBrotli(next).ServeHTTP(w, req)
	if w.Code != 204 {
		t.Errorf("expected 204, got %d", w.Code)
	}
}
