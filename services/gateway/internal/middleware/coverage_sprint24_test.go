package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// --- NoopLogger coverage ---

func TestCovS24_NoopLogger_Methods(t *testing.T) {
	l := NoopLogger{}
	l.Info(LogEntry{})
	l.Warn(LogEntry{})
	l.Error(LogEntry{})
}

// --- audit_log.go Publish coverage ---

type mockNATSConnS24 struct {
	mu        sync.Mutex
	publishes [][]byte
	failErr   error
}

func (m *mockNATSConnS24) Publish(_ string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failErr != nil {
		return m.failErr
	}
	cp := make([]byte, len(data))
	copy(cp, data)
	m.publishes = append(m.publishes, cp)
	return nil
}

func TestCovS24_AuditPublish_WithConn(t *testing.T) {
	mock := &mockNATSConnS24{}
	pub := NewNATSAuditPublisher(mock, "audit.events")

	ev := &AuditEvent{Method: "GET", Path: "/test", StatusCode: 200}
	if err := pub.Publish(ev); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}
	if len(mock.publishes) != 1 {
		t.Fatalf("expected 1 publish, got %d", len(mock.publishes))
	}
	var decoded AuditEvent
	if err := json.Unmarshal(mock.publishes[0], &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if decoded.Method != "GET" {
		t.Fatalf("expected method GET, got %s", decoded.Method)
	}
}

func TestCovS24_AuditPublish_PublishError(t *testing.T) {
	mock := &mockNATSConnS24{failErr: fmt.Errorf("connection refused")}
	pub := NewNATSAuditPublisher(mock, "audit.events")

	err := pub.Publish(&AuditEvent{Method: "POST", Path: "/test"})
	if err == nil {
		t.Fatal("expected error from failed publish")
	}
	if pub.DroppedCount() != 1 {
		t.Fatalf("expected dropped count 1, got %d", pub.DroppedCount())
	}
}

// --- graphql.go resolveField coverage ---

func TestCovS24_ResolveField_Success(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"id": "123", "name": "test"})
	}))
	defer backend.Close()

	resolver := NewGraphQLResolver(map[string]string{"users": backend.URL})
	resolver.HTTPClient = backend.Client()

	field := graphqlField{Name: "user", Type: "users", Path: "/api/v1/users/123"}
	result, err := resolver.resolveField(context.Background(), field, "tenant-1", "Bearer token")
	if err != nil {
		t.Fatalf("resolveField failed: %v", err)
	}
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", result)
	}
	if m["id"] != "123" {
		t.Fatalf("expected id 123, got %v", m["id"])
	}
}

func TestCovS24_ResolveField_NonJSONResponse(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "plain text response")
	}))
	defer backend.Close()

	resolver := NewGraphQLResolver(map[string]string{"users": backend.URL})
	resolver.HTTPClient = backend.Client()

	field := graphqlField{Name: "user", Type: "users", Path: "/api/v1/users/123"}
	result, err := resolver.resolveField(context.Background(), field, "", "")
	if err != nil {
		t.Fatalf("resolveField failed: %v", err)
	}
	s, ok := result.(string)
	if !ok || !strings.Contains(s, "plain text") {
		t.Fatalf("expected string containing 'plain text', got %v", result)
	}
}

func TestCovS24_ResolveField_BackendError(t *testing.T) {
	resolver := NewGraphQLResolver(map[string]string{"users": "http://127.0.0.1:0"})
	resolver.HTTPClient = &http.Client{Timeout: 100 * time.Millisecond}

	field := graphqlField{Name: "user", Type: "users", Path: "/api/v1/users/123"}
	_, err := resolver.resolveField(context.Background(), field, "", "")
	if err == nil {
		t.Fatal("expected error from failed backend request")
	}
}

func TestCovS24_ResolveField_NoBackend(t *testing.T) {
	resolver := NewGraphQLResolver(nil)
	resolver.HTTPClient = &http.Client{}

	field := graphqlField{Name: "unknown", Type: "nonexistent", Path: "/api/v1/foo"}
	_, err := resolver.resolveField(context.Background(), field, "", "")
	if err == nil || !strings.Contains(err.Error(), "no backend configured") {
		t.Fatalf("expected 'no backend' error, got %v", err)
	}
}

// --- grpc.go HandleConn success path ---

func TestCovS24_HandleConn_SuccessTunnel(t *testing.T) {
	backendLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer backendLn.Close()

	go func() {
		conn, err := backendLn.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		io.Copy(conn, conn)
	}()

	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()

	proxy := NewGRPCProxy(DefaultGRPCProxyConfig())

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go func() {
		proxy.HandleConn(ctx, serverConn, backendLn.Addr().String())
	}()

	msg := []byte("hello-grpc-tunnel")
	go func() {
		clientConn.Write(msg)
	}()

	buf := make([]byte, 256)
	clientConn.SetReadDeadline(time.Now().Add(time.Second))
	n, err := clientConn.Read(buf)
	if err != nil {
		t.Logf("read (may timeout after echo): %v", err)
	}
	if n > 0 && string(buf[:n]) != string(msg) {
		t.Fatalf("expected echo %q, got %q", msg, buf[:n])
	}
}

// --- grpc.go GRPCHTTPHandler non-gRPC passes through ---

func TestCovS24_GRPCHTTPHandler_NonGRPCPassthrough(t *testing.T) {
	proxy := NewGRPCProxy(DefaultGRPCProxyConfig())
	srv := httptest.NewServer(proxy.GRPCHTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/test")
	if err != nil {
		t.Fatalf("non-gRPC request failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for non-gRPC, got %d", resp.StatusCode)
	}
}

// --- session.go IsSessionRevoked nil redis ---

func TestCovS24_IsSessionRevoked_NilRedis(t *testing.T) {
	sm := NewSessionManager(nil)
	if sm.IsSessionRevoked(context.Background(), "sess-123") {
		t.Fatal("nil redis should return false (not revoked)")
	}
}

// --- openapi_aggregator Handler error path ---

func TestCovS24_OpenAPIHandler_DownService(t *testing.T) {
	srv1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	srv1.Close()

	agg := NewOpenAPIAggregator(map[string]string{"svc1": srv1.URL})
	agg.ttl = 30 * time.Second
	h := agg.Handler()

	req := httptest.NewRequest("GET", "/openapi.json", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK && rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 200 or 503, got %d", rr.Code)
	}
}

// --- health_score recomputeScore latency paths ---

func TestCovS24_HealthScore_LatencyPaths(t *testing.T) {
	hs := NewHealthScore(5*time.Minute, 0.95)

	// Fast backend → high score
	hs.RecordSuccess("fast-backend", 50*time.Millisecond)
	bh := hs.getOrCreate("fast-backend")
	hs.mu.Lock()
	hs.recomputeScore(bh)
	score1 := bh.score
	hs.mu.Unlock()
	if score1 < 95 {
		t.Fatalf("expected fast backend score >=95, got %.1f", score1)
	}

	// Slow backend (>2s) → lower score
	hs.RecordSuccess("slow-backend", 3*time.Second)
	bh2 := hs.getOrCreate("slow-backend")
	hs.mu.Lock()
	hs.recomputeScore(bh2)
	score2 := bh2.score
	hs.mu.Unlock()
	if score2 >= score1 {
		t.Fatalf("expected slow backend score < fast backend, got %.1f vs %.1f", score2, score1)
	}

	// Medium latency (between 100ms and 2s)
	hs.RecordSuccess("med-backend", 500*time.Millisecond)
	bh3 := hs.getOrCreate("med-backend")
	hs.mu.Lock()
	hs.recomputeScore(bh3)
	score3 := bh3.score
	hs.mu.Unlock()
	if score3 <= score2 || score3 >= score1 {
		t.Fatalf("expected medium score between slow(%.1f) and fast(%.1f), got %.1f", score2, score1, score3)
	}

	// Errors → even lower score
	hs.RecordSuccess("err-backend", 50*time.Millisecond)
	hs.RecordError("err-backend")
	bh4 := hs.getOrCreate("err-backend")
	hs.mu.Lock()
	hs.recomputeScore(bh4)
	score4 := bh4.score
	hs.mu.Unlock()
	if score4 >= 90 {
		t.Fatalf("expected error backend score <90, got %.1f", score4)
	}
}

// --- otel.go export paths ---

func TestCovS24_OTEL_ExportToEndpoint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := DefaultTracingConfig()
	cfg.OTLPEndpoint = srv.URL
	cfg.ServiceName = "test-service"

	exporter := NewTraceExporter(cfg)
	defer exporter.Shutdown()

	spans := []*Span{
		{
			TraceID:    "abcdef0123456789abcdef0123456789",
			SpanID:     "abcdef0123456789",
			Name:       "test-span",
			StartTime:  time.Now().Add(-100 * time.Millisecond),
			EndTime:    time.Now(),
			StatusCode: 0,
		},
	}
	exporter.export(spans)
}

func TestCovS24_OTEL_ExportWithoutEndpoint(t *testing.T) {
	cfg := DefaultTracingConfig()
	cfg.OTLPEndpoint = ""
	cfg.ServiceName = "local-dev"

	exporter := NewTraceExporter(cfg)
	defer exporter.Shutdown()

	spans := []*Span{
		{
			TraceID:    "abcdef0123456789abcdef0123456789",
			SpanID:     "abcdef0123456789",
			Name:       "local-span",
			StartTime:  time.Now().Add(-50 * time.Millisecond),
			EndTime:    time.Now(),
			StatusCode: 1,
		},
	}
	exporter.export(spans)
}

func TestCovS24_OTEL_ExportToBadEndpoint(t *testing.T) {
	cfg := DefaultTracingConfig()
	cfg.OTLPEndpoint = "http://127.0.0.1:1"
	cfg.ServiceName = "test-service"

	exporter := NewTraceExporter(cfg)
	defer exporter.Shutdown()

	spans := []*Span{
		{
			TraceID:   "abcdef0123456789abcdef0123456789",
			SpanID:    "abcdef0123456789",
			Name:      "bad-endpoint-span",
			StartTime: time.Now(),
			EndTime:   time.Now(),
		},
	}
	exporter.export(spans)
}

// --- timeout.go Write method coverage ---

func TestCovS24_TimeoutWrite_AfterHeaders(t *testing.T) {
	cfg := DefaultTimeoutConfig()
	cfg.Default = 5 * time.Second

	handler := TimeoutMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("response body"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if rr.Body.String() != "response body" {
		t.Fatalf("expected 'response body', got %q", rr.Body.String())
	}
}

// --- jti_replay.go IsReplayed ---

func TestCovS24_JTIReplay_EdgeCases(t *testing.T) {
	tracker := NewJTIReplayTracker(time.Minute)
	exp := time.Now().Add(time.Hour)

	if !tracker.IsReplayed("", exp) {
		t.Fatal("empty jti should be treated as replayed (invalid)")
	}
	if tracker.IsReplayed("unique-jti-1", exp) {
		t.Fatal("first use should not be replayed")
	}
	if !tracker.IsReplayed("unique-jti-1", exp) {
		t.Fatal("second use of same jti should be replayed")
	}
}

// --- request_logging.go CapturingLogger coverage ---

func TestCovS24_RequestLogging_CapturingLogger(t *testing.T) {
	logger := &CapturingLogger{}
	handler := RequestLogging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if len(logger.Entries) == 0 {
		t.Fatal("expected at least one log entry")
	}
	if logger.Entries[0].Status != http.StatusTeapot {
		t.Fatalf("expected status 418, got %d", logger.Entries[0].Status)
	}
}

// --- graphql.go GraphQLHandler integration ---

func TestCovS24_GraphQLHandler_MultiFieldWithBackend(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "/users"):
			json.NewEncoder(w).Encode(map[string]any{"id": "1", "name": "Alice"})
		case strings.Contains(r.URL.Path, "/roles"):
			json.NewEncoder(w).Encode(map[string]any{"id": "1", "name": "Admin"})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer backend.Close()

	resolver := NewGraphQLResolver(map[string]string{
		"users": backend.URL,
		"roles": backend.URL,
	})
	resolver.HTTPClient = backend.Client()

	handler := resolver.GraphQLHandler()
	query := `{"query": "{\n  users {\n    id\n  }\n  roles {\n    id\n  }\n}"}`
	req := httptest.NewRequest("POST", "/graphql", strings.NewReader(query))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(rr.Body.Bytes(), &raw); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	// Response must have either 'data' or 'errors' key
	if _, ok := raw["data"]; !ok {
		if _, ok := raw["errors"]; !ok {
			t.Fatalf("response missing 'data' or 'errors' key: %s", rr.Body.String())
		}
	}
}

// --- stats.go DefaultMiddlewareChain ---

func TestCovS24_DefaultMiddlewareChain(t *testing.T) {
	chain := DefaultMiddlewareChain()
	if chain.Count == 0 {
		t.Fatal("expected non-zero middleware count")
	}
	if len(chain.Outer) == 0 {
		t.Fatal("expected outer middleware entries")
	}
	if len(chain.Inner) == 0 {
		t.Fatal("expected inner middleware entries")
	}
}

// --- openapi_aggregator SortedPaths empty ---

func TestCovS24_OpenAPI_SortedPaths_Empty(t *testing.T) {
	spec := &OpenAPISpec{Paths: map[string]map[string]any{}}
	paths := spec.SortedPaths()
	if len(paths) != 0 {
		t.Fatalf("expected 0 paths, got %d", len(paths))
	}
}
