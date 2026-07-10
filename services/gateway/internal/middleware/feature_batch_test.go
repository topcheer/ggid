package middleware

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// --- WebSocket Subprotocol Tests ---

func TestParseSubprotocols(t *testing.T) {
	cases := []struct {
		input string
		want  int
	}{
		{"chat, notification", 2},
		{"chat", 1},
		{"", 0},
		{"a, b, c", 3},
		{"  spaced  ,  trimmed  ", 2},
	}
	for _, c := range cases {
		got := ParseSubprotocols(c.input)
		if len(got) != c.want {
			t.Errorf("ParseSubprotocols(%q) = %v, want %d items", c.input, got, c.want)
		}
	}
}

func TestNegotiateSubprotocol(t *testing.T) {
	server := []string{"chat", "notification", "graphql-ws"}

	cases := []struct {
		client []string
		want   string
	}{
		{[]string{"chat"}, "chat"},
		{[]string{"chat", "notification"}, "chat"}, // first match wins
		{[]string{"notification"}, "notification"},
		{[]string{"unknown"}, ""},
		{[]string{}, ""},
		{[]string{"GRAPHQL-WS"}, "graphql-ws"}, // case insensitive
	}
	for _, c := range cases {
		got := NegotiateSubprotocol(c.client, server)
		if got != c.want {
			t.Errorf("NegotiateSubprotocol(%v, %v) = %q, want %q", c.client, server, got, c.want)
		}
	}
}

func TestDefaultWebSocketConfig(t *testing.T) {
	cfg := DefaultWebSocketConfig()
	if cfg.PingInterval != 30*time.Second {
		t.Errorf("expected 30s ping interval, got %v", cfg.PingInterval)
	}
	if cfg.PongTimeout != 10*time.Second {
		t.Errorf("expected 10s pong timeout, got %v", cfg.PongTimeout)
	}
	if len(cfg.SupportedSubprotocols) != 3 {
		t.Errorf("expected 3 subprotocols, got %d", len(cfg.SupportedSubprotocols))
	}
}

// --- WSKeepalive Tests ---

func TestWSKeepalive_RecordPong(t *testing.T) {
	k := NewWSKeepalive(1*time.Second, 500*time.Millisecond)
	before := k.LastPong()
	time.Sleep(10 * time.Millisecond)
	k.RecordPong()
	after := k.LastPong()
	if !after.After(before) {
		t.Error("expected LastPong to advance after RecordPong")
	}
}

func TestWSKeepalive_Stop(t *testing.T) {
	k := NewWSKeepalive(100*time.Millisecond, 50*time.Millisecond)
	k.Start(nil)
	time.Sleep(50 * time.Millisecond)
	k.Stop()
	// Should not panic on double stop
	k.Stop()
}

func TestWSKeepalive_TimeoutCallback(t *testing.T) {
	k := NewWSKeepalive(50*time.Millisecond, 10*time.Millisecond)
	// Set lastPong far in the past so first tick triggers timeout
	k.mu.Lock()
	k.lastPong = time.Now().Add(-1 * time.Hour)
	k.mu.Unlock()

	var mu sync.Mutex
	called := false
	k.Start(func() {
		mu.Lock()
		called = true
		mu.Unlock()
	})

	time.Sleep(200 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()
	if !called {
		t.Error("expected timeout callback to fire")
	}
}

func TestWSKeepalive_Defaults(t *testing.T) {
	k := NewWSKeepalive(0, 0)
	if k.interval != 30*time.Second {
		t.Errorf("expected default 30s interval, got %v", k.interval)
	}
	if k.timeout != 10*time.Second {
		t.Errorf("expected default 10s timeout, got %v", k.timeout)
	}
}

// --- API Key Rotation Tests ---

func TestRotatableAPIKeyValidator_ActiveKey(t *testing.T) {
	v := NewRotatableAPIKeyValidator(7 * 24 * time.Hour)
	v.AddKey("ggid_active", "tenant1", "user1", []string{"read"},
		time.Now().Add(24*time.Hour))

	tid, uid, scopes, err := v.Validate(nil, "ggid_active")
	if err != nil {
		t.Fatalf("expected valid: %v", err)
	}
	if tid != "tenant1" || uid != "user1" {
		t.Errorf("unexpected identity: %s/%s", tid, uid)
	}
	if len(scopes) != 1 || scopes[0] != "read" {
		t.Errorf("unexpected scopes: %v", scopes)
	}
}

func TestRotatableAPIKeyValidator_ExpiredActive(t *testing.T) {
	v := NewRotatableAPIKeyValidator(7 * 24 * time.Hour)
	v.AddKey("ggid_expired", "tenant1", "user1", []string{"read"},
		time.Now().Add(-1*time.Hour)) // expired 1h ago

	_, _, _, err := v.Validate(nil, "ggid_expired")
	if err == nil {
		t.Error("expected error for expired active key")
	}
}

func TestRotatableAPIKeyValidator_RotatedWithinGrace(t *testing.T) {
	v := NewRotatableAPIKeyValidator(7 * 24 * time.Hour)
	v.AddKey("ggid_old", "tenant1", "user1", []string{"read"},
		time.Now().Add(-1*time.Hour)) // expired 1h ago

	v.RotateKey("ggid_old", "ggid_new", "tenant1", "user1", []string{"read", "write"},
		time.Now().Add(24*time.Hour))

	// Old key should still work during grace period
	tid, _, _, err := v.Validate(nil, "ggid_old")
	if err != nil {
		t.Fatalf("expected old key valid during grace: %v", err)
	}
	if tid != "tenant1" {
		t.Errorf("expected tenant1, got %s", tid)
	}

	// New key should work normally
	_, _, scopes, err := v.Validate(nil, "ggid_new")
	if err != nil {
		t.Fatalf("expected new key valid: %v", err)
	}
	if len(scopes) != 2 {
		t.Errorf("expected 2 scopes, got %d", len(scopes))
	}
}

func TestRotatableAPIKeyValidator_RotatedPastGrace(t *testing.T) {
	v := NewRotatableAPIKeyValidator(1 * time.Hour) // 1h grace
	v.AddKey("ggid_old", "tenant1", "user1", []string{"read"},
		time.Now().Add(-2*time.Hour)) // expired 2h ago, grace is 1h

	v.RotateKey("ggid_old", "ggid_new", "tenant1", "user1", []string{"read"},
		time.Now().Add(24*time.Hour))

	// Old key should be rejected (past grace)
	_, _, _, err := v.Validate(nil, "ggid_old")
	if err == nil {
		t.Error("expected error for key past grace period")
	}
}

func TestRotatableAPIKeyValidator_UnknownKey(t *testing.T) {
	v := NewRotatableAPIKeyValidator(24 * time.Hour)
	_, _, _, err := v.Validate(nil, "nonexistent")
	if err == nil {
		t.Error("expected error for unknown key")
	}
}

func TestRotatableAPIKeyValidator_IsRotated(t *testing.T) {
	v := NewRotatableAPIKeyValidator(24 * time.Hour)
	v.AddKey("ggid_old", "t1", "u1", []string{"read"}, time.Now().Add(24*time.Hour))
	v.RotateKey("ggid_old", "ggid_new", "t1", "u1", []string{"read"}, time.Now().Add(24*time.Hour))

	if !v.IsRotated("ggid_old") {
		t.Error("expected ggid_old to be rotated")
	}
	if v.IsRotated("ggid_new") {
		t.Error("expected ggid_new to NOT be rotated")
	}
}

func TestRotatableAPIKeyValidator_ReplacementKey(t *testing.T) {
	v := NewRotatableAPIKeyValidator(24 * time.Hour)
	v.AddKey("ggid_old", "t1", "u1", []string{"read"}, time.Now().Add(24*time.Hour))
	v.RotateKey("ggid_old", "ggid_new", "t1", "u1", []string{"read"}, time.Now().Add(24*time.Hour))

	if v.ReplacementKey("ggid_old") != "ggid_new" {
		t.Errorf("expected ggid_new, got %s", v.ReplacementKey("ggid_old"))
	}
	if v.ReplacementKey("nonexistent") != "" {
		t.Error("expected empty for unknown key")
	}
}

func TestDefaultRotationConfig(t *testing.T) {
	cfg := DefaultRotationConfig()
	if cfg.GracePeriod != 7*24*time.Hour {
		t.Errorf("expected 7d grace, got %v", cfg.GracePeriod)
	}
}

func TestNewRotatableAPIKeyValidator_DefaultGrace(t *testing.T) {
	v := NewRotatableAPIKeyValidator(0)
	if v.gracePd != 7*24*time.Hour {
		t.Errorf("expected default 7d, got %v", v.gracePd)
	}
}

// --- GeoRouter Tests ---

func TestGeoRouter_ResolveBackend(t *testing.T) {
	router := NewGeoRouter("us-east")
	router.AddRegion(&GeoRegion{
		Name:       "us-east",
		BackendURL: "http://us-east-svc:8080",
		Countries:  []string{"US", "CA"},
	})
	router.AddRegion(&GeoRegion{
		Name:       "eu-west",
		BackendURL: "http://eu-west-svc:8080",
		Countries:  []string{"GB", "DE", "FR"},
	})
	router.AddRegion(&GeoRegion{
		Name:       "ap-southeast",
		BackendURL: "http://ap-svc:8080",
		Countries:  []string{"JP", "CN", "SG"},
	})

	cases := []struct {
		header string
		want   string
	}{
		{"US", "http://us-east-svc:8080"},
		{"CA", "http://us-east-svc:8080"},
		{"GB", "http://eu-west-svc:8080"},
		{"DE", "http://eu-west-svc:8080"},
		{"JP", "http://ap-svc:8080"},
		{"CN", "http://ap-svc:8080"},
		{"BR", "http://us-east-svc:8080"}, // fallback
		{"", "http://us-east-svc:8080"},  // fallback
	}

	for _, c := range cases {
		r := httptest.NewRequest("GET", "/", nil)
		if c.header != "" {
			r.Header.Set("CF-IPCountry", c.header)
		}
		got := router.ResolveBackend(r)
		if got != c.want {
			t.Errorf("ResolveBackend(country=%q) = %q, want %q", c.header, got, c.want)
		}
	}
}

func TestGeoRouter_ResolveRegion(t *testing.T) {
	router := NewGeoRouter("us-east")
	router.AddRegion(&GeoRegion{
		Name:       "us-east",
		BackendURL: "http://us-east-svc:8080",
		Countries:  []string{"US"},
	})
	router.AddRegion(&GeoRegion{
		Name:       "eu-west",
		BackendURL: "http://eu-west-svc:8080",
		Countries:  []string{"DE"},
	})

	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("CF-IPCountry", "DE")
	if region := router.ResolveRegion(r); region != "eu-west" {
		t.Errorf("expected eu-west, got %s", region)
	}

	// No header → fallback
	r2 := httptest.NewRequest("GET", "/", nil)
	if region := router.ResolveRegion(r2); region != "us-east" {
		t.Errorf("expected us-east fallback, got %s", region)
	}
}

func TestGeoRouter_AlternateHeaders(t *testing.T) {
	router := NewGeoRouter("us-east")
	router.AddRegion(&GeoRegion{
		Name:       "eu-west",
		BackendURL: "http://eu-svc:8080",
		Countries:  []string{"GB"},
	})

	// X-GeoIP-Country header
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-GeoIP-Country", "GB")
	if got := router.ResolveBackend(r); got != "http://eu-svc:8080" {
		t.Errorf("expected eu backend via X-GeoIP-Country, got %s", got)
	}
}

func TestGeoRouter_SetHeaderName(t *testing.T) {
	router := NewGeoRouter("us-east")
	router.AddRegion(&GeoRegion{
		Name:       "eu-west",
		BackendURL: "http://eu-svc:8080",
		Countries:  []string{"DE"},
	})
	router.SetHeaderName("X-Custom-Country")

	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-Custom-Country", "DE")
	if got := router.ResolveBackend(r); got != "http://eu-svc:8080" {
		t.Errorf("expected eu backend via custom header, got %s", got)
	}
}

func TestGeoRoutingMiddleware(t *testing.T) {
	router := NewGeoRouter("us-east")
	router.AddRegion(&GeoRegion{
		Name:       "eu-west",
		BackendURL: "http://eu-svc:8080",
		Countries:  []string{"DE"},
	})

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})
	h := GeoRoutingMiddleware(router)(next)

	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("CF-IPCountry", "DE")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if !called {
		t.Error("next handler not called")
	}
	if w.Header().Get("X-Served-By") != "eu-west" {
		t.Errorf("expected X-Served-By=eu-west, got %s", w.Header().Get("X-Served-By"))
	}
}

func TestGeoRouter_Regions(t *testing.T) {
	router := NewGeoRouter("us-east")
	router.AddRegion(&GeoRegion{Name: "us-east", BackendURL: "a", Countries: []string{"US"}})
	router.AddRegion(&GeoRegion{Name: "eu-west", BackendURL: "b", Countries: []string{"DE"}})

	regions := router.Regions()
	if len(regions) != 2 {
		t.Errorf("expected 2 regions, got %d", len(regions))
	}
}

func TestGeoRouter_FallbackRegion(t *testing.T) {
	router := NewGeoRouter("ap-southeast")
	if router.FallbackRegion() != "ap-southeast" {
		t.Errorf("expected ap-southeast, got %s", router.FallbackRegion())
	}
}

func TestGeoRouter_NoFallbackMatch(t *testing.T) {
	router := NewGeoRouter("nonexistent")
	router.AddRegion(&GeoRegion{
		Name:       "us-east",
		BackendURL: "http://us-svc:8080",
		Countries:  []string{"US"},
	})

	// Request with no matching country and no valid fallback
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("CF-IPCountry", "DE")
	if got := router.ResolveBackend(r); got != "" {
		t.Errorf("expected empty for no match, got %s", got)
	}
}

// --- Request Coalescing Concurrency Test ---

func TestCoalesceMiddleware_ConcurrentSameURL(t *testing.T) {
	rc := NewRequestCoalescer(0) // no caching, just in-flight dedup

	var callCount int
	var mu sync.Mutex

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond) // simulate slow backend
		mu.Lock()
		callCount++
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("result"))
	})

	h := CoalesceMiddleware(rc)(next)

	// Launch 10 concurrent requests for the same URL
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/api/v1/users?page=1", nil)
			h.ServeHTTP(w, r)
		}()
	}
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	if callCount != 1 {
		t.Errorf("expected 1 backend call (coalesced), got %d", callCount)
	}
}

func TestCoalesceMiddleware_DifferentURLsNotCoalesced(t *testing.T) {
	rc := NewRequestCoalescer(0)

	var callCount int
	var mu sync.Mutex

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		callCount++
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	})

	h := CoalesceMiddleware(rc)(next)

	// Different URLs should NOT be coalesced
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/api/v1/users?page=1", nil))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/api/v1/users?page=2", nil))

	mu.Lock()
	defer mu.Unlock()
	if callCount != 2 {
		t.Errorf("expected 2 backend calls, got %d", callCount)
	}
}
