package middleware

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// --- JWKS Cache Tests ---

// genTestRSAKey generates a real RSA key and returns its JWK components.
func genTestRSAKey(t *testing.T) (nB64, eB64 string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	nB64 = base64.RawURLEncoding.EncodeToString(key.N.Bytes())
	eB64 = base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes())
	return
}

// genTestJWKS creates a test server that serves JWKS keys.
// It tracks how many times the JWKS endpoint was hit.
func genTestJWKS(t *testing.T, kid, nB64, eB64 string) (*httptest.Server, *int32) {
	t.Helper()
	var hitCount int32

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/jwks.json", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hitCount, 1)
		w.Header().Set("Content-Type", "application/json")
		keySet := map[string]any{
			"keys": []map[string]any{
				{
					"kty": "RSA",
					"kid": kid,
					"use": "sig",
					"alg": "RS256",
					"n":   nB64,
					"e":   eB64,
				},
			},
		}
		json.NewEncoder(w).Encode(keySet)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, &hitCount
}

// TestJWKSCache_CacheHit verifies that the cache doesn't re-fetch within the TTL.
func TestJWKSCache_CacheHit(t *testing.T) {
	nB64, eB64 := genTestRSAKey(t)
	srv, hitCount := genTestJWKS(t, "test-key-1", nB64, eB64)
	cache := newJWKSCache(srv.URL + "/.well-known/jwks.json")

	// First call — should fetch from server
	keys1, err := cache.getKeys()
	if err != nil {
		t.Fatalf("first getKeys failed: %v", err)
	}
	if len(keys1) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys1))
	}
	if atomic.LoadInt32(hitCount) != 1 {
		t.Fatalf("expected 1 JWKS fetch, got %d", atomic.LoadInt32(hitCount))
	}

	// Second call within TTL — should use cache
	keys2, err := cache.getKeys()
	if err != nil {
		t.Fatalf("second getKeys failed: %v", err)
	}
	if len(keys2) != 1 {
		t.Fatalf("expected 1 key from cache, got %d", len(keys2))
	}
	if atomic.LoadInt32(hitCount) != 1 {
		t.Fatalf("expected cache hit (1 fetch), got %d fetches", atomic.LoadInt32(hitCount))
	}
}

// TestJWKSCache_TTLExpiry verifies that after the TTL expires, keys are re-fetched.
func TestJWKSCache_TTLExpiry(t *testing.T) {
	nB64, eB64 := genTestRSAKey(t)
	srv, hitCount := genTestJWKS(t, "test-key-2", nB64, eB64)
	cache := newJWKSCache(srv.URL + "/.well-known/jwks.json")

	// First fetch
	_, err := cache.getKeys()
	if err != nil {
		t.Fatalf("first getKeys failed: %v", err)
	}
	if atomic.LoadInt32(hitCount) != 1 {
		t.Fatalf("expected 1 fetch, got %d", atomic.LoadInt32(hitCount))
	}

	// Force cache expiry by setting updated time to 16 minutes ago
	cache.mu.Lock()
	cache.updated = time.Now().Add(-16 * time.Minute)
	cache.mu.Unlock()

	// Second fetch — should re-fetch because cache is stale (> 15 min)
	_, err = cache.getKeys()
	if err != nil {
		t.Fatalf("second getKeys after TTL failed: %v", err)
	}
	if atomic.LoadInt32(hitCount) != 2 {
		t.Fatalf("expected 2 fetches after TTL expiry, got %d", atomic.LoadInt32(hitCount))
	}
}

// TestJWKSCache_KeyRotation verifies that when the JWKS endpoint serves a new key,
// the cache picks it up after TTL expiry.
func TestJWKSCache_KeyRotation(t *testing.T) {
	nB64, eB64 := genTestRSAKey(t)
	var currentKid = "old-key"
	var hitCount int32

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/jwks.json", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hitCount, 1)
		if atomic.LoadInt32(&hitCount) >= 2 {
			currentKid = "new-rotated-key"
		}
		w.Header().Set("Content-Type", "application/json")
		keySet := map[string]any{
			"keys": []map[string]any{
				{
					"kty": "RSA",
					"kid": currentKid,
					"use": "sig",
					"alg": "RS256",
					"n":   nB64,
					"e":   eB64,
				},
			},
		}
		json.NewEncoder(w).Encode(keySet)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	cache := newJWKSCache(srv.URL + "/.well-known/jwks.json")

	// First fetch — should get old key
	keys1, err := cache.getKeys()
	if err != nil {
		t.Fatalf("first getKeys failed: %v", err)
	}
	if _, ok := keys1["old-key"]; !ok {
		t.Fatalf("expected old-key in cache, got keys: %v", keys1)
	}

	// Force cache expiry
	cache.mu.Lock()
	cache.updated = time.Now().Add(-16 * time.Minute)
	cache.mu.Unlock()

	// Second fetch — should get new rotated key
	keys2, err := cache.getKeys()
	if err != nil {
		t.Fatalf("second getKeys failed: %v", err)
	}
	if _, ok := keys2["new-rotated-key"]; !ok {
		t.Fatalf("expected new-rotated-key after rotation, got keys: %v", keys2)
	}
}

// TestJWKSCache_ConcurrentAccess verifies thread safety of the cache.
func TestJWKSCache_ConcurrentAccess(t *testing.T) {
	nB64, eB64 := genTestRSAKey(t)
	srv, hitCount := genTestJWKS(t, "concurrent-key", nB64, eB64)
	cache := newJWKSCache(srv.URL + "/.well-known/jwks.json")

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := cache.getKeys()
			if err != nil {
				t.Errorf("concurrent getKeys failed: %v", err)
			}
		}()
	}
	wg.Wait()

	// Should only have fetched once (all goroutines share the cache)
	if atomic.LoadInt32(hitCount) != 1 {
		t.Fatalf("expected 1 fetch for 20 concurrent calls, got %d", atomic.LoadInt32(hitCount))
	}
}

// TestJWKSCache_EmptyResponse verifies that the cache handles empty JWKS gracefully.
func TestJWKSCache_EmptyResponse(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/jwks.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"keys": []any{}})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	cache := newJWKSCache(srv.URL + "/.well-known/jwks.json")
	keys, err := cache.getKeys()
	if err != nil {
		t.Fatalf("getKeys with empty response failed: %v", err)
	}
	if len(keys) != 0 {
		t.Fatalf("expected 0 keys, got %d", len(keys))
	}
}

// TestJWKSCache_ServerError verifies that the cache returns an error when the server fails.
func TestJWKSCache_ServerError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/jwks.json", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	cache := newJWKSCache(srv.URL + "/.well-known/jwks.json")
	_, err := cache.getKeys()
	if err == nil {
		t.Fatal("expected error for 500 response, got nil")
	}
}

// TestNewJWKSCache verifies the constructor.
func TestNewJWKSCache(t *testing.T) {
	cache := newJWKSCache("http://example.com/.well-known/jwks.json")
	if cache.url != "http://example.com/.well-known/jwks.json" {
		t.Fatalf("unexpected URL: %s", cache.url)
	}
	if cache.client == nil {
		t.Fatal("expected non-nil HTTP client")
	}
	if cache.keys != nil {
		t.Fatal("expected nil keys before first fetch")
	}
}
