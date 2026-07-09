// Package middleware implements HTTP middleware for the API Gateway.
package middleware

import (
	"context"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ggid/ggid/pkg/tenant"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// --- Context Keys ---

type contextKey string

const (
	RequestIDKey contextKey = "request_id"
	UserIDKey    contextKey = "user_id"
	TenantIDKey  contextKey = "tenant_id"
)

// --- Request ID ---

// RequestID injects a unique request ID into every request.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		w.Header().Set("X-Request-ID", requestID)
		ctx := context.WithValue(r.Context(), RequestIDKey, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// --- Request Logging ---

type statusRecorder struct {
	http.ResponseWriter
	status int
	size   int
}

func (sr *statusRecorder) WriteHeader(code int) {
	sr.status = code
	sr.ResponseWriter.WriteHeader(code)
}

func (sr *statusRecorder) Write(b []byte) (int, error) {
	n, err := sr.ResponseWriter.Write(b)
	sr.size += n
	return n, err
}

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sr := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sr, r)
		requestID, _ := r.Context().Value(RequestIDKey).(string)
		log.Printf("%s %s %d %d %s req=%s",
			r.Method, r.URL.Path, sr.status, sr.size,
			time.Since(start), requestID)
	})
}

// --- CORS ---

func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Tenant-ID, X-Request-ID, X-API-Key")
		w.Header().Set("Access-Control-Max-Age", "3600")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// --- Tenant Resolution ---

// TenantResolver extracts tenant ID from header or subdomain and injects into context.
func TenantResolver(domainSuffix string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var tenantID uuid.UUID

			// 1. Try X-Tenant-ID header
			if tidStr := r.Header.Get("X-Tenant-ID"); tidStr != "" {
				if id, err := uuid.Parse(tidStr); err == nil {
					tenantID = id
				}
			}

			// 2. Fallback to subdomain
			if tenantID == uuid.Nil && domainSuffix != "" {
				host := r.Host
				if strings.HasSuffix(host, domainSuffix) {
					sub := strings.TrimSuffix(host, domainSuffix)
					sub = strings.SplitN(sub, ".", 2)[0]
					if id, err := uuid.Parse(sub); err == nil {
						tenantID = id
					}
				}
			}

			if tenantID != uuid.Nil {
				tc := &tenant.Context{
					TenantID:       tenantID,
					IsolationLevel: tenant.IsolationShared,
				}
				ctx := tenant.WithContext(r.Context(), tc)
				ctx = context.WithValue(ctx, TenantIDKey, tenantID.String())
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// --- JWT Verification ---

// JWKSClient manages JWT key verification with caching and optional JWKS endpoint refresh.
type JWKSClient struct {
	jwksURL    string
	publicKey  *rsa.PublicKey
	keyID      string
	keys       map[string]*rsa.PublicKey
	mu         sync.RWMutex
	httpClient *http.Client
}

// NewJWKSClient creates a JWKS client. If jwksURL is empty, uses the static public key.
func NewJWKSClient(jwksURL, publicKeyPath string) (*JWKSClient, error) {
	c := &JWKSClient{
		jwksURL:    jwksURL,
		keys:       make(map[string]*rsa.PublicKey),
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}

	if publicKeyPath != "" {
		pub, kid, err := loadPublicKey(publicKeyPath)
		if err != nil {
			return nil, fmt.Errorf("load public key: %w", err)
		}
		c.publicKey = pub
		c.keyID = kid
		c.keys[kid] = pub
	}

	if jwksURL != "" {
		// Try initial JWKS fetch; fall back to static key on error
		if err := c.refreshJWKS(); err != nil {
			log.Printf("warning: initial JWKS fetch failed: %v (using static key)", err)
		}
	}

	return c, nil
}

// StartRefresh starts a background goroutine to periodically refresh the JWKS.
func (c *JWKSClient) StartRefresh(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := c.refreshJWKS(); err != nil {
					log.Printf("JWKS refresh error: %v", err)
				}
			}
		}
	}()
}

func (c *JWKSClient) refreshJWKS() error {
	if c.jwksURL == "" {
		return nil
	}
	resp, err := c.httpClient.Get(c.jwksURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("JWKS endpoint returned %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var jwks struct {
		Keys []struct {
			KTY string `json:"kty"`
			KID string `json:"kid"`
			Use string `json:"use"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := json.Unmarshal(body, &jwks); err != nil {
		return err
	}
	newKeys := make(map[string]*rsa.PublicKey)
	for _, k := range jwks.Keys {
		if k.KTY != "RSA" || k.Use != "sig" {
			continue
		}
		pub, err := jwkToRSAPublicKey(k.N, k.E)
		if err != nil {
			continue
		}
		newKeys[k.KID] = pub
	}
	c.mu.Lock()
	if len(newKeys) > 0 {
		c.keys = newKeys
	}
	c.mu.Unlock()
	return nil
}

// GetKey returns the RSA public key for the given key ID.
func (c *JWKSClient) GetKey(keyID string) (*rsa.PublicKey, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if key, ok := c.keys[keyID]; ok {
		return key, nil
	}
	if c.publicKey != nil {
		return c.publicKey, nil
	}
	return nil, fmt.Errorf("key not found for kid: %s", keyID)
}

// UpdatePublicKey replaces the cached static public key.
func (c *JWKSClient) UpdatePublicKey(pub *rsa.PublicKey, keyID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.publicKey = pub
	c.keyID = keyID
	c.keys[keyID] = pub
}

// KeyID returns the current key identifier.
func (c *JWKSClient) KeyID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.keyID
}

// JWKSHandler returns an http.HandlerFunc that serves the JWKS at /.well-known/jwks.json.
func (c *JWKSClient) JWKSHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c.mu.RLock()
		keys := make([]map[string]any, 0, len(c.keys))
		for kid, pub := range c.keys {
			keys = append(keys, map[string]any{
				"kty": "RSA",
				"use": "sig",
				"alg": "RS256",
				"kid": kid,
				"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
			})
		}
		c.mu.RUnlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"keys": keys})
	}
}

// JWTAuth returns middleware that validates JWT Bearer tokens.
// If required is true, requests without a valid token get 401.
// If required is false, requests without a token pass through.
func JWTAuth(jwks *JWKSClient, required bool, issuer, audience string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				if required {
					writeUnauthorized(w, "missing Authorization header")
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				if required {
					writeUnauthorized(w, "invalid Authorization header format")
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			tokenStr := strings.TrimSpace(parts[1])
			token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
				if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				keyID, _ := token.Header["kid"].(string)
				if keyID == "" {
					keyID = jwks.KeyID()
				}
				return jwks.GetKey(keyID)
			})

			if err != nil || !token.Valid {
				if required {
					writeUnauthorized(w, "invalid or expired token")
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				if required {
					writeUnauthorized(w, "invalid token claims")
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			// Validate issuer
			if issuer != "" {
				if iss, _ := claims["iss"].(string); iss != issuer {
					if required {
						writeUnauthorized(w, "invalid token issuer")
						return
					}
					next.ServeHTTP(w, r)
					return
				}
			}

			// Extract user ID and tenant ID from claims, inject into context
			ctx := r.Context()
			if sub, _ := claims["sub"].(string); sub != "" {
				ctx = context.WithValue(ctx, UserIDKey, sub)
			}
			if tid, _ := claims["tenant_id"].(string); tid != "" {
				ctx = context.WithValue(ctx, TenantIDKey, tid)
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserIDFromRequest extracts the user ID from the request context.
func UserIDFromRequest(r *http.Request) (uuid.UUID, bool) {
	idStr, ok := r.Context().Value(UserIDKey).(string)
	if !ok || idStr == "" {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}

// TenantIDFromRequest extracts the tenant ID from the request context.
func TenantIDFromRequest(r *http.Request) (string, bool) {
	id, ok := r.Context().Value(TenantIDKey).(string)
	return id, ok && id != ""
}

// --- Helpers ---

func writeUnauthorized(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("WWW-Authenticate", `Bearer realm="ggid"`)
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]string{
		"type":   "https://ggid.dev/errors/unauthenticated",
		"title":  "Unauthenticated",
		"detail": msg,
	})
}

func loadPublicKey(path string) (*rsa.PublicKey, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, "", fmt.Errorf("failed to decode PEM block")
	}

	var rsaPub *rsa.PublicKey

	// Try PKIX first (most common for "PUBLIC KEY" PEM blocks)
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err == nil {
		var ok bool
		rsaPub, ok = pub.(*rsa.PublicKey)
		if !ok {
			return nil, "", fmt.Errorf("PKIX key is not RSA")
		}
	} else {
		// Try PKCS1
		rsaPub, err = x509.ParsePKCS1PublicKey(block.Bytes)
		if err != nil {
			return nil, "", fmt.Errorf("parse public key (tried PKIX and PKCS1)")
		}
	}

	kid := keyFingerprint(rsaPub)
	return rsaPub, kid, nil
}

func keyFingerprint(pub *rsa.PublicKey) string {
	data, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return "unknown"
	}
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h[:8])
}

func jwkToRSAPublicKey(nStr, eStr string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(nStr)
	if err != nil {
		return nil, err
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(eStr)
	if err != nil {
		return nil, err
	}
	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)
	return &rsa.PublicKey{N: n, E: int(e.Int64())}, nil
}

// --- JWTVerifier wrapper ---

// JWTVerifier wraps JWKSClient to provide the interface that the router expects.
type JWTVerifier struct {
	client   *JWKSClient
	issuer   string
	audience string
}

// NewJWTVerifier creates a JWTVerifier from a JWKSClient.
func NewJWTVerifier(client *JWKSClient, issuer, audience string) *JWTVerifier {
	return &JWTVerifier{client: client, issuer: issuer, audience: audience}
}

// KeyID delegates to the underlying JWKSClient.
func (v *JWTVerifier) KeyID() string {
	return v.client.KeyID()
}

// HandleJWKS serves the JWKS JSON at /.well-known/jwks.json.
func (v *JWTVerifier) HandleJWKS(w http.ResponseWriter, r *http.Request) {
	v.client.JWKSHandler()(w, r)
}

// Middleware returns HTTP middleware that validates JWT Bearer tokens.
// Paths listed in publicPaths skip JWT verification.
func (v *JWTVerifier) Middleware(publicPaths []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip auth for public paths.
			for _, p := range publicPaths {
				if strings.HasPrefix(r.URL.Path, p) {
					next.ServeHTTP(w, r)
					return
				}
			}

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeUnauthorized(w, "missing Authorization header")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				writeUnauthorized(w, "invalid Authorization header format")
				return
			}

			tokenStr := strings.TrimSpace(parts[1])
			token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
				if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				keyID, _ := token.Header["kid"].(string)
				if keyID == "" {
					keyID = v.client.KeyID()
				}
				return v.client.GetKey(keyID)
			})

			if err != nil || !token.Valid {
				writeUnauthorized(w, "invalid or expired token")
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				writeUnauthorized(w, "invalid token claims")
				return
			}

			// Validate issuer.
			if v.issuer != "" {
				if iss, _ := claims["iss"].(string); iss != v.issuer {
					writeUnauthorized(w, "invalid token issuer")
					return
				}
			}

			// Inject user ID and tenant ID into context.
			ctx := r.Context()
			if sub, _ := claims["sub"].(string); sub != "" {
				ctx = context.WithValue(ctx, UserIDKey, sub)
			}
			if tid, _ := claims["tenant_id"].(string); tid != "" {
				ctx = context.WithValue(ctx, TenantIDKey, tid)
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// --- Context helper functions (for router) ---

// UserIDFromContext extracts the user ID from the request context.
func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	idStr, ok := ctx.Value(UserIDKey).(string)
	if !ok || idStr == "" {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}

// TenantIDFromContext extracts the tenant ID from the request context.
func TenantIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	idStr, ok := ctx.Value(TenantIDKey).(string)
	if !ok || idStr == "" {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}

// --- Tenant middleware (for router) ---

// TenantConfig configures the Tenant middleware.
type TenantConfig struct {
	Header string // header name, e.g. "X-Tenant-ID"
	Domain string // domain suffix for subdomain extraction, e.g. ".ggid.dev"
}

// Tenant returns middleware that extracts tenant ID from header or subdomain.
func Tenant(cfg TenantConfig) func(http.Handler) http.Handler {
	header := cfg.Header
	if header == "" {
		header = "X-Tenant-ID"
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var tenantID uuid.UUID

			// 1. Try header.
			if tidStr := r.Header.Get(header); tidStr != "" {
				if id, err := uuid.Parse(tidStr); err == nil {
					tenantID = id
				}
			}

			// 2. Fallback to subdomain.
			if tenantID == uuid.Nil && cfg.Domain != "" {
				host := r.Host
				if strings.HasSuffix(host, cfg.Domain) {
					sub := strings.TrimSuffix(host, cfg.Domain)
					sub = strings.SplitN(sub, ".", 2)[0]
					if id, err := uuid.Parse(sub); err == nil {
						tenantID = id
					}
				}
			}

			if tenantID != uuid.Nil {
				tc := &tenant.Context{
					TenantID:       tenantID,
					IsolationLevel: tenant.IsolationShared,
				}
				ctx := tenant.WithContext(r.Context(), tc)
				ctx = context.WithValue(ctx, TenantIDKey, tenantID.String())
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
