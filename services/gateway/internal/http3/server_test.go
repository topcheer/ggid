package http3

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAltSvcHeader(t *testing.T) {
	h := AltSvcHeader(8443)
	if h != `h3=":8443"; ma=86400` {
		t.Errorf("unexpected header: %q", h)
	}

	// Default port
	h2 := AltSvcHeader(0)
	if h2 != `h3=":8443"; ma=86400` {
		t.Errorf("unexpected default header: %q", h2)
	}
}

func TestAltSvcMiddleware(t *testing.T) {
	mw := AltSvcMiddleware(8443)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	called := false
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})).ServeHTTP(w, req)

	if !called {
		t.Error("expected handler to be called")
	}
	if h := w.Header().Get("Alt-Svc"); h != `h3=":8443"; ma=86400` {
		t.Errorf("expected Alt-Svc header, got %q", h)
	}
}

func TestHappyEyeballsMiddleware(t *testing.T) {
	mw := HappyEyeballsMiddleware(8443)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).ServeHTTP(w, req)

	if h := w.Header().Get("Alt-Svc"); h == "" {
		t.Error("expected Alt-Svc header")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Addr != ":8443" {
		t.Errorf("expected :8443, got %s", cfg.Addr)
	}
	if !cfg.EnableH2 {
		t.Error("expected EnableH2=true")
	}
	if cfg.MaxStreams != 100 {
		t.Errorf("expected 100 streams, got %d", cfg.MaxStreams)
	}
}

func TestNew_NoTLSConfig(t *testing.T) {
	_, err := New(Config{Addr: ":8443"}, nil)
	if err == nil {
		t.Error("expected error for missing TLS config")
	}
}
