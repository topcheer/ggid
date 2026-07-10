package http3

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAltSvcHeader(t *testing.T) {
	h := AltSvcHeader(8443)
	if h != `h3=":8443"; ma=86400` {
		t.Errorf("unexpected header: %q", h)
	}
	h2 := AltSvcHeader(0)
	if h2 != `h3=":8443"; ma=86400` {
		t.Errorf("unexpected default header: %q", h2)
	}
}

func TestAltSvcMiddleware(t *testing.T) {
	mw := AltSvcMiddleware(8443)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).ServeHTTP(w, req)
	if h := w.Header().Get("Alt-Svc"); h != `h3=":8443"; ma=86400` {
		t.Errorf("expected Alt-Svc header, got %q", h)
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
		t.Errorf("expected 100, got %d", cfg.MaxStreams)
	}
}

func TestNew_NoTLSConfig(t *testing.T) {
	_, err := New(Config{Addr: ":8443"}, nil)
	if err == nil {
		t.Error("expected error for missing TLS config")
	}
}

func TestNew_ValidTLSConfig(t *testing.T) {
	cfg := Config{
		Addr:      ":8443",
		TLSConfig: &tls.Config{Certificates: []tls.Certificate{{}}},
	}
	srv, err := New(cfg, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if srv == nil {
		t.Fatal("expected non-nil server")
	}
	if srv.addr != ":8443" {
		t.Errorf("expected :8443, got %s", srv.addr)
	}
}

func TestNew_EmptyAddrDefaults(t *testing.T) {
	cfg := Config{
		TLSConfig: &tls.Config{},
	}
	srv, err := New(cfg, http.NewServeMux())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if srv.addr != ":8443" {
		t.Errorf("expected default :8443, got %s", srv.addr)
	}
}

func TestNew_SetsNextProtos(t *testing.T) {
	tlsCfg := &tls.Config{}
	_, err := New(Config{Addr: ":8443", TLSConfig: tlsCfg}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tlsCfg.NextProtos) != 1 || tlsCfg.NextProtos[0] != "h3" {
		t.Errorf("expected NextProtos [h3], got %v", tlsCfg.NextProtos)
	}
}

func TestShutdown_NilServer(t *testing.T) {
	s := &Server{server: nil}
	if err := s.Shutdown(context.Background()); err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestHappyEyeballsMiddleware_HTTP3Proto(t *testing.T) {
	mw := HappyEyeballsMiddleware(8443)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.ProtoMajor = 3
	req.Proto = "HTTP/3"
	w := httptest.NewRecorder()
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).ServeHTTP(w, req)

	if v := w.Header().Get("X-HTTP3-Version"); v != "HTTP/3" {
		t.Errorf("expected X-HTTP3-Version HTTP/3, got %q", v)
	}
}

func TestHappyEyeballsMiddleware_HTTP1(t *testing.T) {
	mw := HappyEyeballsMiddleware(9000)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).ServeHTTP(w, req)

	if v := w.Header().Get("X-HTTP3-Version"); v != "" {
		t.Errorf("expected empty X-HTTP3-Version for HTTP/1, got %q", v)
	}
	if h := w.Header().Get("Alt-Svc"); h != `h3=":9000"; ma=86400` {
		t.Errorf("expected Alt-Svc for port 9000, got %q", h)
	}
}
