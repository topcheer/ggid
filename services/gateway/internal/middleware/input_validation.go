package middleware

import (
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
)

// InputValidationConfig controls which endpoints get input validation.
type InputValidationConfig struct {
	Enabled        bool
	ExemptPaths    map[string]bool
	MaxBodySize    int64 // bytes
}

var (
	sqliPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)'\s*OR\s*1\s*=\s*1`),
		regexp.MustCompile(`(?i)'\s*OR\s*'1'='1`),
		regexp.MustCompile(`(?i)UNION\s+SELECT`),
		regexp.MustCompile(`(?i);\s*DROP\s+TABLE`),
		regexp.MustCompile(`(?i);\s*DELETE\s+FROM`),
		regexp.MustCompile(`(?i)INSERT\s+INTO.*VALUES`),
		regexp.MustCompile(`(?i)EXEC\s*\(`),
		regexp.MustCompile(`(?i)xp_cmdshell`),
		regexp.MustCompile(`(?i)WAITFOR\s+DELAY`),
	}
	xssPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)<script[^>]*>`),
		regexp.MustCompile(`(?i)</script>`),
		regexp.MustCompile(`(?i)javascript:`),
		regexp.MustCompile(`(?i)onerror\s*=`),
		regexp.MustCompile(`(?i)onload\s*=`),
		regexp.MustCompile(`(?i)onclick\s*=`),
		regexp.MustCompile(`(?i)<iframe`),
		regexp.MustCompile(`(?i)<object`),
		regexp.MustCompile(`(?i)<embed`),
	}
	defaultValidationConfig = InputValidationConfig{
		Enabled: true,
		ExemptPaths: map[string]bool{
			"/api/v1/audit/pii-scan":     true, // needs raw input
			"/api/v1/crypto/fields":      true, // may contain encoded data
			"/api/v1/dlp/scan":           true, // DLP scanner needs raw input
			"/graphql":                   true, // GraphQL has its own validation
		},
		MaxBodySize: 10 * 1024 * 1024, // 10MB
	}
	validationConfigMu sync.RWMutex
)

// SetInputValidationConfig updates the global validation config.
func SetInputValidationConfig(cfg InputValidationConfig) {
	validationConfigMu.Lock()
	defer validationConfigMu.Unlock()
	defaultValidationConfig = cfg
}

// DetectMaliciousInput checks a string for SQLi/XSS patterns.
// Returns (pattern type, matched pattern) if detected, ("", "") if clean.
func DetectMaliciousInput(input string) (string, string) {
	for _, p := range sqliPatterns {
		if p.MatchString(input) {
			return "sql_injection", p.String()
		}
	}
	for _, p := range xssPatterns {
		if p.MatchString(input) {
			return "xss", p.String()
		}
	}
	return "", ""
}

// InputValidationMiddleware validates request body for SQLi/XSS patterns.
func InputValidationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		validationConfigMu.RLock()
		cfg := defaultValidationConfig
		validationConfigMu.RUnlock()

		if !cfg.Enabled || cfg.ExemptPaths[r.URL.Path] {
			next.ServeHTTP(w, r)
			return
		}

		// Check query params.
		for key, values := range r.URL.Query() {
			for _, val := range values {
				if typ, pattern := DetectMaliciousInput(val); typ != "" {
					rejectInput(w, typ, pattern, key)
					return
				}
			}
		}

		// Check request body (for POST/PUT/PATCH with JSON).
		if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
			if r.Header.Get("Content-Type") == "application/json" && r.ContentLength > 0 && r.ContentLength < cfg.MaxBodySize {
				body, err := io.ReadAll(r.Body)
				if err != nil {
					next.ServeHTTP(w, r)
					return
				}
				r.Body.Close()

				// Scan body content for patterns.
				bodyStr := string(body)
				if typ, pattern := DetectMaliciousInput(bodyStr); typ != "" {
					rejectInput(w, typ, pattern, "body")
					return
				}

				// Restore body for downstream handlers.
				r.Body = io.NopCloser(strings.NewReader(bodyStr))
				r.ContentLength = int64(len(body))
			}
		}

		next.ServeHTTP(w, r)
	})
}

func rejectInput(w http.ResponseWriter, typ, pattern, field string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(map[string]any{
		"error":      "malicious input detected",
		"type":       typ,
		"field":      field,
		"pattern":    pattern,
		"request_id": "input-validation",
	})
}
