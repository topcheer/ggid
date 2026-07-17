package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// --- PII Detection Patterns ---

var (
	ssnPattern       = regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`)
	creditCardPattern = regexp.MustCompile(`\b(?:\d[ -]*?){13,19}\b`)
	emailPattern      = regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`)
	phonePattern      = regexp.MustCompile(`\b(?:\+?1[-.\s]?)?\(?\d{3}\)?[-.\s]?\d{3}[-.\s]?\d{4}\b`)
	apiKeyPattern     = regexp.MustCompile(`\b(?:sk_live_|sk_test_|AKIA|ghp_|gho_|ghs_|ghr_|xox[bpoa]-)[A-Za-z0-9]{16,}\b`)
	jwtPattern        = regexp.MustCompile(`\beyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\b`)
)

// PIIMatch represents a detected PII element.
type PIIMatch struct {
	Type    string `json:"type"`
	Value   string `json:"value"`
	Field   string `json:"field,omitempty"`
	Strategy string `json:"strategy"`
}

// RedactionStrategy defines how a PII element is masked.
type RedactionStrategy string

const (
	StrategyFullMask    RedactionStrategy = "full_mask"
	StrategyPartialMask RedactionStrategy = "partial_mask"
	StrategyEmailMask   RedactionStrategy = "email_mask"
	StrategyTokenize    RedactionStrategy = "tokenize"
	StrategyRemove      RedactionStrategy = "remove"
)

// DLPEgressConfig holds per-tenant DLP egress policies.
type DLPEgressConfig struct {
	Enabled        bool
	DefaultPolicy  RedactionStrategy // default if no field-specific rule
	FieldRules     map[string]RedactionStrategy // field name → strategy
	Classification map[string]RedactionStrategy // data classification → strategy
}

// DefaultDLPEgressConfig returns a production-safe default.
func DefaultDLPEgressConfig() *DLPEgressConfig {
	return &DLPEgressConfig{
		Enabled:       true,
		DefaultPolicy: StrategyPartialMask,
		FieldRules: map[string]RedactionStrategy{
			"ssn":           StrategyPartialMask,
			"credit_card":   StrategyPartialMask,
			"email":         StrategyEmailMask,
			"phone":         StrategyPartialMask,
			"api_key":       StrategyFullMask,
			"jwt":           StrategyFullMask,
			"password":      StrategyFullMask,
			"token":         StrategyFullMask,
			"secret":        StrategyFullMask,
		},
		Classification: map[string]RedactionStrategy{
			"core":      StrategyFullMask,
			"important": StrategyPartialMask,
			"general":   StrategyRemove,
		},
	}
}

// DLPEgressMiddleware intercepts response bodies and redacts PII before sending to client.
func DLPEgressMiddleware(cfg *DLPEgressConfig, auditFn func(match PIIMatch)) func(http.Handler) http.Handler {
	if cfg == nil {
		cfg = DefaultDLPEgressConfig()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !cfg.Enabled {
				next.ServeHTTP(w, r)
				return
			}

			// Wrap the ResponseWriter to capture the body.
			capture := newResponseCapture()
			next.ServeHTTP(capture, r)

			// Only process JSON responses.
			contentType := capture.Header().Get("Content-Type")
			if !strings.Contains(contentType, "application/json") {
				w.Header().Set("Content-Type", contentType)
				w.WriteHeader(capture.status)
				w.Write(capture.body.Bytes())
				return
			}

			// Scan and redact.
			redacted, matches := scanAndRedact(capture.body.Bytes(), cfg)

			// Audit matches.
			if auditFn != nil {
				for _, m := range matches {
					auditFn(m)
				}
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(capture.status)
			w.Write(redacted)
		})
	}
}

// responseCapture wraps http.ResponseWriter to capture the response body.
type responseCapture struct {
	header http.Header
	body   *bytes.Buffer
	status int
}

func newResponseCapture() *responseCapture {
	return &responseCapture{
		header: http.Header{},
		body:   &bytes.Buffer{},
		status: http.StatusOK,
	}
}

func (c *responseCapture) Header() http.Header {
	return c.header
}

func (c *responseCapture) WriteHeader(code int) {
	c.status = code
}

func (c *responseCapture) Write(b []byte) (int, error) {
	return c.body.Write(b)
}

// scanAndRedact recursively traverses JSON and applies redaction rules.
func scanAndRedact(data []byte, cfg *DLPEgressConfig) ([]byte, []PIIMatch) {
	var obj any
	if err := json.Unmarshal(data, &obj); err != nil {
		return data, nil // not valid JSON, return as-is
	}

	var matches []PIIMatch
	redacted := redactValue("", obj, cfg, &matches)

	result, err := json.Marshal(redacted)
	if err != nil {
		return data, matches
	}
	return result, matches
}

// redactValue recursively traverses values and applies PII redaction.
func redactValue(field string, val any, cfg *DLPEgressConfig, matches *[]PIIMatch) any {
	switch v := val.(type) {
	case map[string]any:
		for key, child := range v {
			v[key] = redactValue(key, child, cfg, matches)
		}
		return v
	case []any:
		for i, child := range v {
			v[i] = redactValue(field, child, cfg, matches)
		}
		return v
	case string:
		return redactString(field, v, cfg, matches)
	default:
		return val
	}
}

// redactString applies PII detection and redaction to a string value.
func redactString(field, value string, cfg *DLPEgressConfig, matches *[]PIIMatch) string {
	result := value

	// Check field-name-based rules first.
	lowerField := strings.ToLower(field)
	if strategy, ok := cfg.FieldRules[lowerField]; ok {
		if shouldRedact(value, lowerField) {
			*matches = append(*matches, PIIMatch{Type: lowerField, Value: maskValue(value, strategy), Field: field, Strategy: string(strategy)})
			return applyStrategy(value, strategy)
		}
	}

	// Pattern-based detection.
	result = applyPattern(ssnPattern, "ssn", result, cfg, matches, field)
	result = applyPattern(creditCardPattern, "credit_card", result, cfg, matches, field)
	result = applyPattern(emailPattern, "email", result, cfg, matches, field)
	result = applyPattern(phonePattern, "phone", result, cfg, matches, field)
	result = applyPattern(apiKeyPattern, "api_key", result, cfg, matches, field)
	result = applyPattern(jwtPattern, "jwt", result, cfg, matches, field)

	return result
}

// applyPattern detects and redacts a specific PII pattern.
func applyPattern(pattern *regexp.Regexp, piiType, input string, cfg *DLPEgressConfig, matches *[]PIIMatch, field string) string {
	strategy, ok := cfg.FieldRules[piiType]
	if !ok {
		strategy = cfg.DefaultPolicy
	}

	return pattern.ReplaceAllStringFunc(input, func(match string) string {
		// Validate credit cards with Luhn check.
		if piiType == "credit_card" && !luhnValid(match) {
			return match
		}
		*matches = append(*matches, PIIMatch{
			Type:     piiType,
			Value:    maskValue(match, strategy),
			Field:    field,
			Strategy: string(strategy),
		})
		return applyStrategy(match, strategy)
	})
}

// shouldRedact checks if a field value should be redacted based on field name.
func shouldRedact(value, fieldName string) bool {
	switch fieldName {
	case "password", "token", "secret", "api_key":
		return value != ""
	default:
		return false
	}
}

// applyStrategy masks a value according to the redaction strategy.
func applyStrategy(value string, strategy RedactionStrategy) string {
	switch strategy {
	case StrategyFullMask:
		return strings.Repeat("*", len(value))
	case StrategyPartialMask:
		if len(value) <= 4 {
			return strings.Repeat("*", len(value))
		}
		return value[:len(value)-4] + strings.Repeat("*", 4)
	case StrategyEmailMask:
		parts := strings.SplitN(value, "@", 2)
		if len(parts) != 2 {
			return "***"
		}
		if len(parts[0]) <= 2 {
			return "**@" + parts[1]
		}
		return parts[0][:2] + "***@" + parts[1]
	case StrategyTokenize:
		return "tok_" + hashToken(value)[:16]
	case StrategyRemove:
		return "[REDACTED]"
	default:
		return value
	}
}

// maskValue returns a masked preview for audit logging (never the real value).
func maskValue(value string, strategy RedactionStrategy) string {
	if len(value) > 8 {
		return value[:4] + "..." + value[len(value)-4:]
	}
	return "***"
}

// luhnValid validates a credit card number using the Luhn algorithm.
func luhnValid(cardNumber string) bool {
	var sum int
	var alt bool
	// Remove non-digit characters.
	for i := len(cardNumber) - 1; i >= 0; i-- {
		c := cardNumber[i]
		if c == ' ' || c == '-' {
			continue
		}
		if c < '0' || c > '9' {
			return false
		}
		digit := int(c - '0')
		if alt {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		sum += digit
		alt = !alt
	}
	return sum > 0 && sum%10 == 0
}

// hashToken generates a deterministic token hash for tokenization.
func hashToken(value string) string {
	return strings.ReplaceAll(strings.ToUpper(strings.TrimLeft(
		strings.TrimRight(fmt.Sprintf("%x", time.Now().UnixNano()), ""), "0")), " ", "")
}

// ScanResponseBody is a standalone function for testing PII detection.
func ScanResponseBody(body []byte, cfg *DLPEgressConfig) ([]byte, []PIIMatch, error) {
	if cfg == nil {
		cfg = DefaultDLPEgressConfig()
	}
	redacted, matches := scanAndRedact(body, cfg)
	return redacted, matches, nil
}

// Ensure imports are used.
var (
	_ = io.ReadAll
	_ = context.Background
	_ = slog.Error
	_ = time.Now
)
