package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ── service-a: caller ──────────────────────────────────────────────

// startServiceA runs the calling service that obtains a client_credentials
// token from GGID and calls service-b with it.
func startServiceA(port string, ggidClient *GGIDClient, serviceBURL string) {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{
			"status": "ok",
			"service": "service-a",
		})
	})

	mux.HandleFunc("/call-service-b", func(w http.ResponseWriter, r *http.Request) {
		// 1. Obtain M2M access token from GGID
		token, err := ggidClient.GetToken()
		if err != nil {
			log.Printf("[service-a] failed to get token: %v", err)
			writeJSON(w, http.StatusBadGateway, map[string]string{
				"error":   "token_acquisition_failed",
				"message": err.Error(),
			})
			return
		}
		log.Printf("[service-a] obtained M2M token (%d chars)", len(token))

		// 2. Call service-b with the token
		req, err := http.NewRequest("GET", serviceBURL+"/api/data", nil)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Printf("[service-a] call to service-b failed: %v", err)
			writeJSON(w, http.StatusBadGateway, map[string]string{
				"error":   "upstream_call_failed",
				"message": err.Error(),
			})
			return
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		log.Printf("[service-a] service-b responded: %d", resp.StatusCode)

		// 3. Return combined result
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		w.Write(body)
	})

	mux.HandleFunc("/call-service-b-post", func(w http.ResponseWriter, r *http.Request) {
		token, err := ggidClient.GetToken()
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}

		payload := map[string]any{
			"requested_by": "service-a",
			"timestamp":    time.Now().UTC().Format(time.RFC3339),
			"data":         "hello from service-a",
		}
		payloadBytes, _ := json.Marshal(payload)

		req, err := http.NewRequest("POST", serviceBURL+"/api/data", strings.NewReader(string(payloadBytes)))
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		w.Write(body)
	})

	addr := ":" + port
	log.Printf("[service-a] listening on %s", addr)
	log.Printf("[service-a] will call service-b at %s", serviceBURL)
	log.Fatal(http.ListenAndServe(addr, mux))
}

// ── service-b: callee ──────────────────────────────────────────────

// JWKSKeyCache caches JWKS public keys with periodic refresh.
type JWKSKeyCache struct {
	ggidURL    string
	httpClient *http.Client
	keys       map[string]map[string]any // kid -> key data
	mu         sync.RWMutex
	lastFetch  time.Time
	ttl        time.Duration
}

func NewJWKSKeyCache(ggidURL string) *JWKSKeyCache {
	return &JWKSKeyCache{
		ggidURL:    ggidURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		keys:       make(map[string]map[string]any),
		ttl:        5 * time.Minute,
	}
}

func (c *JWKSKeyCache) GetKey(kid string) (map[string]any, error) {
	c.mu.RLock()
	key, ok := c.keys[kid]
	c.mu.RUnlock()
	if ok {
		return key, nil
	}

	// Refresh JWKS
	if err := c.refresh(); err != nil {
		return nil, err
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	key, ok = c.keys[kid]
	if !ok {
		return nil, fmt.Errorf("key not found in JWKS for kid: %s", kid)
	}
	return key, nil
}

func (c *JWKSKeyCache) refresh() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if cache is still fresh
	if time.Since(c.lastFetch) < c.ttl && len(c.keys) > 0 {
		return nil
	}

	resp, err := c.httpClient.Get(c.ggidURL + "/.well-known/jwks.json")
	if err != nil {
		return fmt.Errorf("fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	var jwks struct {
		Keys []map[string]any `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("decode JWKS: %w", err)
	}

	c.keys = make(map[string]map[string]any)
	for _, key := range jwks.Keys {
		if kid, ok := key["kid"].(string); ok {
			c.keys[kid] = key
		}
	}
	c.lastFetch = time.Now()
	log.Printf("[service-b] JWKS refreshed: %d keys", len(c.keys))
	return nil
}

// startServiceB runs the protected service that validates GGID JWT tokens.
func startServiceB(port string, jwksCache *JWKSKeyCache, tenantID string) {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{
			"status":  "ok",
			"service": "service-b",
		})
	})

	mux.HandleFunc("/api/data", func(w http.ResponseWriter, r *http.Request) {
		// 1. Extract Bearer token
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			writeJSON(w, http.StatusUnauthorized, map[string]string{
				"error":   "missing_token",
				"message": "Authorization: Bearer <token> header required",
			})
			return
		}
		token := strings.TrimPrefix(authHeader, "Bearer ")

		// 2. Verify the JWT token against GGID JWKS
		claims, err := verifyJWT(token, jwksCache)
		if err != nil {
			log.Printf("[service-b] token verification failed: %v", err)
			writeJSON(w, http.StatusUnauthorized, map[string]string{
				"error":   "invalid_token",
				"message": err.Error(),
			})
			return
		}
		log.Printf("[service-b] valid token from client: %s", claims.ClientID)

		// 3. Handle the request
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, map[string]any{
				"service":   "service-b",
				"message":   "data retrieved successfully",
				"timestamp": time.Now().UTC().Format(time.RFC3339),
				"caller":    claims.ClientID,
				"tenant":    claims.TenantID,
				"scopes":    claims.Scope,
			})

		case http.MethodPost:
			body, _ := io.ReadAll(r.Body)
			var payload map[string]any
			if len(body) > 0 {
				json.Unmarshal(body, &payload)
			}
			writeJSON(w, http.StatusCreated, map[string]any{
				"service":   "service-b",
				"message":   "data created successfully",
				"received":  payload,
				"caller":    claims.ClientID,
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			})

		default:
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{
				"error": "method_not_allowed",
			})
		}
	})

	addr := ":" + port
	log.Printf("[service-b] listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

// ── Helpers ────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}
