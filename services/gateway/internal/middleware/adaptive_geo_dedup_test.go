package middleware

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// --- Adaptive Rate Limiter Tests ---

func TestAdaptiveRateLimiter_Allow(t *testing.T) {
	al := NewAdaptiveRateLimiter(100, 10, 1000)
	for i := 0; i < 100; i++ {
		if !al.Allow("t1") {
			t.Fatalf("expected allow on iteration %d", i)
		}
	}
	// 101st should be rate-limited (bucket exhausted)
	if al.Allow("t1") {
		t.Error("expected rate limit on 101st request")
	}
}

func TestAdaptiveRateLimiter_DifferentKeys(t *testing.T) {
	al := NewAdaptiveRateLimiter(5, 1, 100)
	for i := 0; i < 5; i++ {
		if !al.Allow("t1") {
			t.Fatal("expected allow for t1")
		}
	}
	// t2 should be independent
	if !al.Allow("t2") {
		t.Error("expected allow for t2")
	}
}

func TestAdaptiveRateLimiter_RecordLatencyDecrease(t *testing.T) {
	al := NewAdaptiveRateLimiter(100, 5, 1000)
	al.adjustInterval = 1 * time.Millisecond

	// Record slow requests, waiting past adjustInterval each time
	for i := 0; i < 20; i++ {
		al.Allow("t1")
		al.RecordLatency("t1", 2*time.Second)
		time.Sleep(2 * time.Millisecond)
	}

	limit := al.Limit("t1")
	if limit >= 100 {
		t.Errorf("expected limit < 100 after slow responses, got %f", limit)
	}
}

func TestAdaptiveRateLimiter_RecordLatencyIncrease(t *testing.T) {
	al := NewAdaptiveRateLimiter(50, 5, 1000)
	al.adjustInterval = 1 * time.Millisecond

	// Record fast responses, waiting past adjustInterval each time
	for i := 0; i < 20; i++ {
		al.Allow("t1")
		al.RecordLatency("t1", 10*time.Millisecond)
		time.Sleep(2 * time.Millisecond)
	}

	limit := al.Limit("t1")
	if limit <= 50 {
		t.Errorf("expected limit > 50 after fast responses, got %f", limit)
	}
}

func TestAdaptiveRateLimiter_SetLimit(t *testing.T) {
	al := NewAdaptiveRateLimiter(100, 10, 1000)
	al.SetLimit("t1", 200)
	if al.Limit("t1") != 200 {
		t.Errorf("expected 200, got %f", al.Limit("t1"))
	}
}

func TestAdaptiveRateLimiter_AllLimits(t *testing.T) {
	al := NewAdaptiveRateLimiter(100, 10, 1000)
	al.Allow("t1")
	al.Allow("t2")
	limits := al.AllLimits()
	if len(limits) != 2 {
		t.Errorf("expected 2 limits, got %d", len(limits))
	}
}

func TestAdaptiveRateLimiter_Defaults(t *testing.T) {
	al := NewAdaptiveRateLimiter(0, 0, 0)
	if al.baseLimit != 100 {
		t.Errorf("expected default 100, got %f", al.baseLimit)
	}
}

// --- Request Deduplication Tests ---

func TestRequestDeduplicator_GetSet(t *testing.T) {
	d := NewRequestDeduplicator(5 * time.Minute)
	d.Set("key1", http.StatusOK, []byte("hello"), http.Header{"X-Custom": []string{"val"}})

	entry, ok := d.Get("key1")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if entry.statusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", entry.statusCode)
	}
	if string(entry.body) != "hello" {
		t.Errorf("expected hello, got %s", entry.body)
	}
}

func TestRequestDeduplicator_Expired(t *testing.T) {
	d := NewRequestDeduplicator(1 * time.Millisecond)
	d.Set("key1", http.StatusOK, []byte("data"), http.Header{})
	time.Sleep(5 * time.Millisecond)

	_, ok := d.Get("key1")
	if ok {
		t.Error("expected cache miss for expired entry")
	}
}

func TestRequestDeduplicator_Count(t *testing.T) {
	d := NewRequestDeduplicator(5 * time.Minute)
	d.Set("k1", 200, []byte("a"), http.Header{})
	d.Set("k2", 200, []byte("b"), http.Header{})
	if d.Count() != 2 {
		t.Errorf("expected 2, got %d", d.Count())
	}
}

func TestRequestDeduplicator_Delete(t *testing.T) {
	d := NewRequestDeduplicator(5 * time.Minute)
	d.Set("k1", 200, []byte("a"), http.Header{})
	d.Delete("k1")
	if d.Count() != 0 {
		t.Error("expected 0 after delete")
	}
}

func TestDedupMiddleware_CacheHit(t *testing.T) {
	d := NewRequestDeduplicator(5 * time.Minute)
	d.Set("idem-1", http.StatusOK, []byte(`{"cached":true}`),
		http.Header{"Content-Type": []string{"application/json"}})

	callCount := 0
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
	})
	h := DedupMiddleware(d)(next)

	// First request: cache hit
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/data", nil)
	r.Header.Set("Idempotency-Key", "idem-1")
	h.ServeHTTP(w, r)

	if callCount != 0 {
		t.Error("backend should NOT be called on cache hit")
	}
	if w.Body.String() != `{"cached":true}` {
		t.Errorf("expected cached body, got %s", w.Body.String())
	}
	if w.Header().Get("X-Deduplicated") != "true" {
		t.Error("expected X-Deduplicated header")
	}
}

func TestDedupMiddleware_CacheMiss_StoresResponse(t *testing.T) {
	d := NewRequestDeduplicator(5 * time.Minute)
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"fresh":true}`))
	})
	h := DedupMiddleware(d)(next)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/data", nil)
	r.Header.Set("Idempotency-Key", "new-key")
	h.ServeHTTP(w, r)

	// Response should be cached now
	if d.Count() != 1 {
		t.Errorf("expected 1 cached entry, got %d", d.Count())
	}
}

func TestDedupMiddleware_PostNotCached(t *testing.T) {
	d := NewRequestDeduplicator(5 * time.Minute)
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
	h := DedupMiddleware(d)(next)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/v1/data", nil)
	r.Header.Set("Idempotency-Key", "post-key")
	h.ServeHTTP(w, r)

	if d.Count() != 0 {
		t.Error("POST should not be cached")
	}
}

func TestDedupMiddleware_NoKey(t *testing.T) {
	d := NewRequestDeduplicator(5 * time.Minute)
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	h := DedupMiddleware(d)(next)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api", nil) // no Idempotency-Key
	h.ServeHTTP(w, r)

	if !called {
		t.Error("backend should be called when no key")
	}
}

// --- Geo Enrichment Tests ---

func TestGeoEnricher_DefaultLookup(t *testing.T) {
	g := NewGeoEnricher()
	country, city := g.Lookup("127.0.0.1")
	if country != "local" {
		t.Errorf("expected local, got %s", country)
	}
	if city != "loopback" {
		t.Errorf("expected loopback, got %s", city)
	}
}

func TestGeoEnricher_InternalIP(t *testing.T) {
	g := NewGeoEnricher()
	country, _ := g.Lookup("10.0.0.1")
	if country != "internal" {
		t.Errorf("expected internal, got %s", country)
	}
}

func TestGeoEnricher_UnknownIP(t *testing.T) {
	g := NewGeoEnricher()
	country, _ := g.Lookup("8.8.8.8")
	if country != "unknown" {
		t.Errorf("expected unknown, got %s", country)
	}
}

func TestGeoEnricher_AddRange(t *testing.T) {
	g := NewGeoEnricher()
	g.AddRange("203.0.113.", "US", "New York")
	country, city := g.Lookup("203.0.113.50")
	if country != "US" {
		t.Errorf("expected US, got %s", country)
	}
	if city != "New York" {
		t.Errorf("expected New York, got %s", city)
	}
}

func TestGeoEnrichMiddleware(t *testing.T) {
	g := NewGeoEnricher()
	g.AddRange("198.51.100.", "DE", "Berlin")

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Geo-Country") != "DE" {
			t.Errorf("expected DE, got %s", r.Header.Get("X-Geo-Country"))
		}
		if r.Header.Get("X-Geo-City") != "Berlin" {
			t.Errorf("expected Berlin, got %s", r.Header.Get("X-Geo-City"))
		}
		w.WriteHeader(http.StatusOK)
	})
	h := GeoEnrichMiddleware(g)(next)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "198.51.100.10:12345"
	h.ServeHTTP(w, r)
}

func TestGeoEnrichMiddleware_PreservesExisting(t *testing.T) {
	g := NewGeoEnricher()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Geo-Country") != "US" {
			t.Errorf("expected preserved US, got %s", r.Header.Get("X-Geo-Country"))
		}
	})
	h := GeoEnrichMiddleware(g)(next)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-Geo-Country", "US") // pre-set by upstream proxy
	r.RemoteAddr = "127.0.0.1:1234"
	h.ServeHTTP(w, r)
}

func TestGeoEnrichMiddleware_XForwardedFor(t *testing.T) {
	g := NewGeoEnricher()
	g.AddRange("203.0.113.", "FR", "Paris")

	var mu sync.Mutex
	var gotCountry string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotCountry = r.Header.Get("X-Geo-Country")
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	})
	h := GeoEnrichMiddleware(g)(next)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-Forwarded-For", "203.0.113.99")
	h.ServeHTTP(w, r)

	mu.Lock()
	defer mu.Unlock()
	if gotCountry != "FR" {
		t.Errorf("expected FR from XFF, got %s", gotCountry)
	}
}
