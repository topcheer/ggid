// Package middleware implements HTTP middleware for the API Gateway.
package middleware

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log/slog"
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
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", sr.status,
			"size", sr.size,
			"duration_ms", time.Since(start).Milliseconds(),
			"request_id", requestID)
	})
}

// --- CORS ---

// CORSConfig holds CORS configuration.
type CORSConfig struct {
	AllowedOrigins   []string // exact origins to allow; ["*"] for wildcard
	AllowCredentials bool     // allow cookies / Authorization header from browser
}

// DefaultCORSConfig returns a secure-by-default CORS config.
// In production, set AllowedOrigins to your frontend domain(s).
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: false,
	}
}

// CORSWithConfig returns a CORS middleware with the given config.
// When AllowCredentials is true, the Origin header is echoed back instead of
// using a wildcard, and Access-Control-Allow-Credentials is set to true.
func CORSWithConfig(cfg CORSConfig) func(http.Handler) http.Handler {
	allowAll := len(cfg.AllowedOrigins) == 0
	for _, o := range cfg.AllowedOrigins {
		if o == "*" {
			allowAll = true
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			if allowAll {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else {
				// Echo the origin if it's in the allowed list
				for _, allowed := range cfg.AllowedOrigins {
					if subtle.ConstantTimeCompare([]byte(origin), []byte(allowed)) == 1 {
						w.Header().Set("Access-Control-Allow-Origin", origin)
						w.Header().Set("Vary", "Origin")
						break
					}
				}
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Tenant-ID, X-Request-ID, X-API-Key")
			w.Header().Set("Access-Control-Expose-Headers", "X-Request-ID, X-Tenant-ID")
			w.Header().Set("Access-Control-Max-Age", "3600")

			if cfg.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// CORS is the default CORS middleware (backward-compatible with wildcard origin).
func CORS(next http.Handler) http.Handler {
	return CORSWithConfig(DefaultCORSConfig())(next)
}

// --- CSRF Protection (Double-Submit Cookie) ---

// CSRFProtect implements double-submit cookie CSRF protection.
// On safe requests (GET/HEAD/OPTIONS), it sets a csrf_token cookie.
// On unsafe requests (POST/PUT/PATCH/DELETE), it validates that the
// X-CSRF-Token header matches the csrf_token cookie.
func CSRFProtect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Safe methods don't need CSRF check, but we refresh the cookie
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			setCSRFCookie(w)
			next.ServeHTTP(w, r)
			return
		}

		// For unsafe methods, validate the double-submit token
		cookieToken, err := r.Cookie("csrf_token")
		if err != nil || cookieToken.Value == "" {
			writeForbidden(w, "missing CSRF token")
			return
		}

		headerToken := r.Header.Get("X-CSRF-Token")
		if headerToken == "" {
			writeForbidden(w, "missing CSRF header")
			return
		}

		// Constant-time comparison to prevent timing attacks
		if subtle.ConstantTimeCompare([]byte(cookieToken.Value), []byte(headerToken)) != 1 {
			writeForbidden(w, "CSRF token mismatch")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func setCSRFCookie(w http.ResponseWriter) {
	token := generateCSRFToken()
	http.SetCookie(w, &http.Cookie{
		Name:     "csrf_token",
		Value:    token,
		Path:     "/",
		MaxAge:   3600,
		HttpOnly: false, // Must be readable by JavaScript for double-submit
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

func generateCSRFToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// Fallback should never happen with crypto/rand on modern systems,
		// but if it does, fail closed with a panic rather than using weak entropy.
		panic("crypto/rand failed: " + err.Error())
	}
	hash := sha256.Sum256(b)
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

func writeForbidden(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// --- Security Headers ---

// SecurityHeaders adds common security headers to all responses.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		next.ServeHTTP(w, r)
	})
}

// --- Tenant Resolution ---

// TenantResolver extracts tenant ID from multiple sources in priority order:
// 1. X-Tenant-ID header (explicit)
// 2. JWT claim "tenant_id" (parsed without verification — verified later by JWTAuth)
// 3. Subdomain (acme.iam.com → tenant "acme", or UUID subdomain)
// The domainSuffix is used for subdomain extraction (e.g. ".iam.com").
func TenantResolver(domainSuffix string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var tenantID uuid.UUID

			// 1. Try JWT claim tenant_id first (highest priority — authenticated source)
			// X-Tenant-ID header is unauthenticated and must NOT override JWT claims.
			if tidStr := extractTenantFromJWT(r); tidStr != "" {
				if id, err := uuid.Parse(tidStr); err == nil {
					tenantID = id
				}
			}

			// 2. Try X-Tenant-ID header (only for public endpoints without JWT)
			if tenantID == uuid.Nil {
				if tidStr := r.Header.Get("X-Tenant-ID"); tidStr != "" {
					if id, err := uuid.Parse(tidStr); err == nil {
						tenantID = id
					}
				}
			}

			// 3. Fallback to subdomain extraction
			if tenantID == uuid.Nil && domainSuffix != "" {
				tenantID = extractTenantFromSubdomain(r.Host, domainSuffix)
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

// extractTenantFromJWT parses the JWT from the Authorization header
// (without verifying the signature) and extracts the tenant_id claim.
// This is safe because JWTAuth will verify the token later — we're just
// reading metadata for routing context.
func extractTenantFromJWT(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	tokenStr := strings.TrimSpace(parts[1])
	// JWT format: header.payload.signature — we only need the payload
	tokenParts := strings.Split(tokenStr, ".")
	if len(tokenParts) < 2 {
		return ""
	}
	// Decode the payload (base64url, no padding)
	payload, err := base64.RawURLEncoding.DecodeString(tokenParts[1])
	if err != nil {
		return ""
	}
	// Parse claims
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return ""
	}
	if tid, ok := claims["tenant_id"].(string); ok {
		return tid
	}
	return ""
}

// extractTenantFromSubdomain extracts a tenant UUID from the subdomain.
// For example: "acme.iam.com" with domainSuffix ".iam.com" → tries parsing "acme" as UUID.
// This only works if tenants use UUID-format subdomains.
func extractTenantFromSubdomain(host, domainSuffix string) uuid.UUID {
	// Strip port if present
	if idx := strings.LastIndex(host, ":"); idx >= 0 {
		host = host[:idx]
	}
	if !strings.HasSuffix(host, domainSuffix) {
		return uuid.Nil
	}
	sub := strings.TrimSuffix(host, domainSuffix)
	sub = strings.SplitN(sub, ".", 2)[0]
	if sub == "" || sub == "www" {
		return uuid.Nil
	}
	id, err := uuid.Parse(sub)
	if err != nil {
		return uuid.Nil
	}
	return id
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
			slog.Warn("initial JWKS fetch failed, using static key", "err", err)
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
						slog.Error("JWKS refresh error", "err", err)
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
		// Build validation options: jwt/v5 validates exp/nbf/iat by default
		// when claims are present. We add issuer, audience, and method restrictions.
		parseOpts := []jwt.ParserOption{
			jwt.WithValidMethods([]string{"RS256"}),
		}
		if issuer != "" {
			parseOpts = append(parseOpts, jwt.WithIssuer(issuer))
		}
		if audience != "" {
			parseOpts = append(parseOpts, jwt.WithAudience(audience))
		}
		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			keyID, _ := token.Header["kid"].(string)
			_ = keyID // ignore kid from JWT; always use static public key
			return jwks.publicKey, nil
		}, parseOpts...)

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

			// Issuer and audience are validated by jwt.Parse via parseOpts above.

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
