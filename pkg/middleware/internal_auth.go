// Package middleware provides shared HTTP middleware for GGID services.
package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	// InternalAuthHeaderService is the header carrying the calling service name.
	InternalAuthHeaderService = "X-Internal-Service"
	// InternalAuthHeaderTimestamp is the Unix timestamp for replay prevention.
	InternalAuthHeaderTimestamp = "X-Internal-Timestamp"
	// InternalAuthHeaderSignature is the HMAC-SHA256 signature.
	InternalAuthHeaderSignature = "X-Internal-Signature"

	// defaultReplayWindow is the max time skew allowed (seconds).
	defaultReplayWindow = 120
)

// InternalAuthConfig configures the internal auth middleware.
type InternalAuthConfig struct {
	Secret       []byte
	PrevSecret   []byte // optional, for key rotation
	ReplayWindow int64  // seconds; defaults to 120
	Whitelist    []string // paths that skip internal auth
}

// LoadInternalSecret loads the internal auth secret from env vars.
// In production (GGID_ENV=production), missing secret is fatal.
// In dev, uses a default and logs a warning.
func LoadInternalSecret() []byte {
	secret := os.Getenv("GGID_INTERNAL_SECRET")
	if secret == "" {
		if os.Getenv("GGID_ENV") == "production" {
			log.Fatal("GGID_INTERNAL_SECRET must be set in production")
		}
		secret = "dev-internal-secret"
		log.Println("WARNING: using default internal secret — not for production")
	}
	return []byte(secret)
}

// LoadInternalSecrets loads current + previous secrets for rotation.
func LoadInternalSecrets() (current, prev []byte) {
	current = LoadInternalSecret()
	if s := os.Getenv("GGID_INTERNAL_SECRET_PREV"); s != "" {
		prev = []byte(s)
	}
	return current, prev
}

// InternalAuth returns middleware that validates HMAC-signed internal headers.
func InternalAuth(cfg InternalAuthConfig) func(http.Handler) http.Handler {
	if cfg.ReplayWindow <= 0 {
		cfg.ReplayWindow = defaultReplayWindow
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check whitelist.
			for _, path := range cfg.Whitelist {
				if r.URL.Path == path || strings.HasPrefix(r.URL.Path, path) {
					next.ServeHTTP(w, r)
					return
				}
			}

			// If no secret configured, allow (dev mode without InternalAuth).
			if len(cfg.Secret) == 0 {
				next.ServeHTTP(w, r)
				return
			}

			service := r.Header.Get(InternalAuthHeaderService)
			tsStr := r.Header.Get(InternalAuthHeaderTimestamp)
			sigHex := r.Header.Get(InternalAuthHeaderSignature)

			if service == "" || tsStr == "" || sigHex == "" {
				deny(w, r, "missing internal auth headers")
				return
			}

			ts, err := strconv.ParseInt(tsStr, 10, 64)
			if err != nil {
				deny(w, r, "invalid timestamp")
				return
			}

			now := time.Now().Unix()
			if math.Abs(float64(now-ts)) > float64(cfg.ReplayWindow) {
				deny(w, r, "timestamp outside replay window")
				return
			}

			// Request ID used in signature (reuse X-Request-ID).
			reqID := r.Header.Get("X-Request-ID")
			payload := service + "|" + tsStr + "|" + reqID

			// Try current secret, then prev.
			if !verifyHMAC(cfg.Secret, payload, sigHex) {
				// Current secret failed — try prev if available.
				prevOK := len(cfg.PrevSecret) > 0 && verifyHMAC(cfg.PrevSecret, payload, sigHex)
				if !prevOK {
					deny(w, r, "invalid signature")
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// SignInternalRequest injects HMAC signature headers into an outbound request.
// Used by services that call other services directly (e.g., auth→identity).
func SignInternalRequest(req *http.Request, serviceName string, secret []byte) {
	if len(secret) == 0 || req == nil {
		return
	}
	ts := time.Now().Unix()
	tsStr := strconv.FormatInt(ts, 10)
	reqID := req.Header.Get("X-Request-ID")
	if reqID == "" {
		reqID = ""
	}
	payload := serviceName + "|" + tsStr + "|" + reqID
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(payload))
	sig := hex.EncodeToString(mac.Sum(nil))

	req.Header.Set(InternalAuthHeaderService, serviceName)
	req.Header.Set(InternalAuthHeaderTimestamp, tsStr)
	req.Header.Set(InternalAuthHeaderSignature, sig)
}

// ComputeSignature computes the HMAC-SHA256 signature for the given inputs.
func ComputeSignature(secret []byte, service, tsStr, reqID string) string {
	payload := service + "|" + tsStr + "|" + reqID
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

func verifyHMAC(secret []byte, payload, sigHex string) bool {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(payload))
	expected := mac.Sum(nil)
	got, err := hex.DecodeString(sigHex)
	if err != nil {
		return false
	}
	return hmac.Equal(expected, got)
}

func deny(w http.ResponseWriter, r *http.Request, reason string) {
	log.Printf("internal auth denied: %s remote=%s path=%s", reason, r.RemoteAddr, r.URL.Path)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	w.Write([]byte(`{"error":"internal auth failed"}`))
}

// DefaultWhitelist returns standard paths that skip internal auth.
func DefaultWhitelist() []string {
	return []string{
		"/healthz",
		"/metrics",
		"/readyz",
	}
}

// InternalAuthPathOnly applies InternalAuth only to paths containing "/internal/".
// All other paths (public APIs, healthz, metrics) pass through without any auth check.
// If no secret is configured, it is a complete no-op (dev mode).
func InternalAuthPathOnly(cfg InternalAuthConfig) func(http.Handler) http.Handler {
	if len(cfg.Secret) == 0 {
		return func(next http.Handler) http.Handler { return next }
	}
	authMW := InternalAuth(cfg)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.Contains(r.URL.Path, "/internal/") {
				next.ServeHTTP(w, r)
				return
			}
			authMW(next).ServeHTTP(w, r)
		})
	}
}
