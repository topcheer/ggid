package middleware

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// --- gRPC HandleConn ---

func TestGRPCProxy_HandleConn_UnknownTarget(t *testing.T) {
	p := NewGRPCProxy(GRPCProxyConfig{})
	conn := &mockConnS9{}
	p.HandleConn(nil, conn, "unknown:9090")
}

type mockConnS9 struct{}

func (m *mockConnS9) Read(b []byte) (int, error)         { return 0, nil }
func (m *mockConnS9) Write(b []byte) (int, error)        { return len(b), nil }
func (m *mockConnS9) Close() error                       { return nil }
func (m *mockConnS9) LocalAddr() net.Addr                { return &net.IPAddr{} }
func (m *mockConnS9) RemoteAddr() net.Addr               { return &net.IPAddr{} }
func (m *mockConnS9) SetDeadline(t time.Time) error      { return nil }
func (m *mockConnS9) SetReadDeadline(t time.Time) error  { return nil }
func (m *mockConnS9) SetWriteDeadline(t time.Time) error { return nil }

// --- gRPC GRPCHTTPHandler ---

func TestGRPCProxy_HTTPHandler_UnknownPath(t *testing.T) {
	p := NewGRPCProxy(GRPCProxyConfig{})
	p.AddBackend("auth", "localhost:9090")
	mw := p.GRPCHTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodPost, "/api/grpc/unknown/method", nil)
	req.Header.Set("Content-Type", "application/grpc+proto")
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)
}

func TestGRPCProxy_HTTPHandler_NonGRPCContentType(t *testing.T) {
	p := NewGRPCProxy(GRPCProxyConfig{})
	p.AddBackend("auth", "localhost:9090")
	mw := p.GRPCHTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodPost, "/api/grpc/auth.Login", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mw.ServeHTTP(w, req)
}

// --- health_score ---

func TestHealthScore_RecordSuccessAndScore_S9(t *testing.T) {
	hs := NewHealthScore(5*time.Minute, 0.8)
	hs.RecordSuccess("svc1", 50*time.Millisecond)
	hs.RecordSuccess("svc1", 30*time.Millisecond)
	score := hs.Score("svc1")
	if score <= 0 {
		t.Errorf("expected positive score, got %f", score)
	}
}

func TestHealthScore_AllFailures_S9(t *testing.T) {
	hs := NewHealthScore(5*time.Minute, 0.5)
	hs.RecordError("svc2")
	hs.RecordError("svc2")
	hs.RecordError("svc2")
	if hs.IsHealthy("svc2", 0.5) {
		t.Error("expected unhealthy after 3 failures")
	}
}

func TestHealthScore_AllScores_S9(t *testing.T) {
	hs := NewHealthScore(5*time.Minute, 0.8)
	hs.RecordSuccess("svc1", 10*time.Millisecond)
	hs.RecordError("svc2")
	scores := hs.AllScores()
	if len(scores) != 2 {
		t.Errorf("expected 2 backends, got %d", len(scores))
	}
}

func TestHealthScore_Reset_S9(t *testing.T) {
	hs := NewHealthScore(5*time.Minute, 0.8)
	hs.RecordError("svc1")
	hs.Reset("svc1")
	score := hs.Score("svc1")
	if score != 100.0 {
		t.Errorf("expected 100 after reset, got %f", score)
	}
}

// --- graphql proxy ---

func TestGraphQLHandler_MalformedQuery_S9(t *testing.T) {
	resolver := NewGraphQLResolver(map[string]string{"user": "http://localhost:18080"})
	handler := resolver.GraphQLHandler()
	req := httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader(`{ malformed !!! }`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
}

func TestGraphQLHandler_EmptyBody_S9(t *testing.T) {
	resolver := NewGraphQLResolver(nil)
	handler := resolver.GraphQLHandler()
	req := httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader(""))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
}

func TestGraphQLHandler_GetMethod_S9(t *testing.T) {
	resolver := NewGraphQLResolver(nil)
	handler := resolver.GraphQLHandler()
	req := httptest.NewRequest(http.MethodGet, "/graphql?query={users{id}}", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
}

// --- shadow traffic ---

func TestShadowTrafficMirror_Percentage0_S9(t *testing.T) {
	stm := NewShadowTrafficMirror(ShadowTrafficConfig{
		ShadowBackend: "http://shadow:8080",
		Percentage:    0,
	})
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	if stm.shouldMirror(req) {
		t.Error("expected no mirror at 0%")
	}
}
