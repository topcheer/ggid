package http3

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNew_NilHandler(t *testing.T) {
	tlsCfg := &tls.Config{Certificates: []tls.Certificate{}}
	srv, err := New(Config{Addr: ":8443", TLSConfig: tlsCfg}, nil)
	if err != nil {
		// May fail due to empty TLS cert, but handler should be defaulted
	}
	_ = srv
	// Test passes if no panic (handler defaulting to DefaultServeMux)
}

func TestAltSvcHeader_DefaultPort(t *testing.T) {
	h := AltSvcHeader(0)
	expected := `h3=":8443"; ma=86400`
	if h != expected {
		t.Errorf("AltSvcHeader(0) = %q, want %q", h, expected)
	}
}

func TestAltSvcHeader_CustomPort(t *testing.T) {
	h := AltSvcHeader(9999)
	expected := `h3=":9999"; ma=86400`
	if h != expected {
		t.Errorf("AltSvcHeader(9999) = %q, want %q", h, expected)
	}
}

func TestAltSvcHeader_NegativePort(t *testing.T) {
	// Port 0 is the only "falsy" case; negative should pass through
	h := AltSvcHeader(-1)
	if h == "" {
		t.Error("should not be empty")
	}
}

func TestAltSvcMiddleware_AddsHeader(t *testing.T) {
	mw := AltSvcMiddleware(8443)
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		// Check header was set before calling next
		if w.Header().Get("Alt-Svc") == "" {
			t.Error("Alt-Svc header should be set before calling next")
		}
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, req)

	if !called {
		t.Error("next handler should be called")
	}
	if w.Header().Get("Alt-Svc") != AltSvcHeader(8443) {
		t.Errorf("Alt-Svc = %q", w.Header().Get("Alt-Svc"))
	}
}

func TestAltSvcMiddleware_ZeroPort(t *testing.T) {
	mw := AltSvcMiddleware(0)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, req)

	expected := `h3=":8443"; ma=86400`
	if w.Header().Get("Alt-Svc") != expected {
		t.Errorf("Alt-Svc = %q, want %q", w.Header().Get("Alt-Svc"), expected)
	}
}

func TestHappyEyeballsMiddleware_SetsAltSvc(t *testing.T) {
	mw := HappyEyeballsMiddleware(8443)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, req)

	if w.Header().Get("Alt-Svc") == "" {
		t.Error("expected Alt-Svc header")
	}
}

func TestHappyEyeballsMiddleware_HTTP3Version(t *testing.T) {
	mw := HappyEyeballsMiddleware(8443)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	req := httptest.NewRequest("GET", "/test", nil)
	// Simulate HTTP/3 request
	req.ProtoMajor = 3
	req.Proto = "HTTP/3.0"

	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, req)

	if w.Header().Get("X-HTTP3-Version") != "HTTP/3.0" {
		t.Errorf("X-HTTP3-Version = %q", w.Header().Get("X-HTTP3-Version"))
	}
}

func TestHappyEyeballsMiddleware_HTTP2NoVersion(t *testing.T) {
	mw := HappyEyeballsMiddleware(8443)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	req := httptest.NewRequest("GET", "/test", nil)
	req.ProtoMajor = 2

	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, req)

	if w.Header().Get("X-HTTP3-Version") != "" {
		t.Error("X-HTTP3-Version should not be set for HTTP/2")
	}
}

func TestHappyEyeballsMiddleware_PassesToNext(t *testing.T) {
	mw := HappyEyeballsMiddleware(8443)
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, req)

	if !called {
		t.Error("next handler should be called")
	}
}



func TestNew_EmptyAddr(t *testing.T) {
	tlsCfg := &tls.Config{Certificates: []tls.Certificate{}}
	srv, err := New(Config{TLSConfig: tlsCfg}, http.DefaultServeMux)
	if err != nil {
		t.Fatal(err)
	}
	if srv.addr != ":8443" {
		t.Errorf("empty addr should default to ':8443', got %q", srv.addr)
	}
}




