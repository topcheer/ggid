package service

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

// breachCheckEnabled returns whether the HIBP breach check should run.
// Controlled by BREACH_CHECK_ENABLED env var (default: true).
// Set BREACH_CHECK_ENABLED=false to skip the check (useful for E2E
// tests with common passwords like "Password123!").
func breachCheckEnabled() bool {
	v := os.Getenv("BREACH_CHECK_ENABLED")
	if v == "false" || v == "0" || v == "no" {
		return false
	}
	return true // default: enabled
}

// breachCheckBlock returns true if breached passwords should block login
// (requires MFA). Default: false (warn only, fail-open for usability).
// Set BREACH_CHECK_BLOCK=true to enforce blocking.
func breachCheckBlock() bool {
	v := os.Getenv("BREACH_CHECK_BLOCK")
	return v == "true" || v == "1" || v == "yes"
}

// --- HIBP Circuit Breaker ---
//
// breachCircuitBreaker implements a simple fail-open circuit breaker for the HIBP API.
// After `breachCircuitThreshold` consecutive failures, the circuit opens and all
// subsequent calls fail-open (return nil) for `breachCircuitCooldown` duration.
// After the cooldown, one trial request is allowed (half-open). If it succeeds,
// the breaker resets; if it fails, the cooldown restarts.

const (
	breachCircuitThreshold = 3                // consecutive failures to open
	breachCircuitCooldown  = 30 * time.Second // open-state duration
)

// breachCircuitState holds the circuit breaker state using atomics.
//   failureCount: int32  — consecutive failures (0 = closed/healthy)
//   openedAt:     int64  — unix-nano when circuit opened (0 = closed)
var breachFailureCount atomic.Int32
var breachOpenedAt atomic.Int64

// breachCircuitIsOpen returns true if the circuit is currently open.
// If the cooldown has elapsed, it transitions to half-open (returns false)
// but does NOT reset the failure count — that only happens on success.
func breachCircuitIsOpen() bool {
	openedAt := breachOpenedAt.Load()
	if openedAt == 0 {
		return false // closed
	}
	if time.Now().UnixNano()-openedAt >= breachCircuitCooldown.Nanoseconds() {
		// Cooldown elapsed — half-open: allow a trial request
		return false
	}
	return true // still open
}

// breachCircuitRecordSuccess resets the circuit breaker to closed/healthy.
func breachCircuitRecordSuccess() {
	breachFailureCount.Store(0)
	breachOpenedAt.Store(0)
}

// breachCircuitRecordFailure increments the failure counter and opens the circuit
// if the threshold is reached.
func breachCircuitRecordFailure() {
	count := breachFailureCount.Add(1)
	if count >= int32(breachCircuitThreshold) {
		breachOpenedAt.Store(time.Now().UnixNano())
	}
}

// resetBreachCircuitForTest resets the circuit breaker state. Test-only.
func resetBreachCircuitForTest() {
	breachFailureCount.Store(0)
	breachOpenedAt.Store(0)
}

// CheckPasswordBreach checks if a password has been found in known data breaches
// using the HIBP k-anonymity model (haveibeenpwned.com API).
// Only the first 5 characters of the SHA-1 hash are sent to the API.
// Returns an error if the password has been breached.
// Circuit breaker: after 3 consecutive HIBP failures, fail-open for 30s.
func (ps *PasswordService) CheckPasswordBreach(ctx context.Context, password string) error {
	// Circuit breaker: if open, fail-open immediately.
	if breachCircuitIsOpen() {
		return nil
	}

	// Compute SHA-1 hash of the password.
	h := sha1.Sum([]byte(password))
	hash := strings.ToUpper(hex.EncodeToString(h[:]))

	// k-anonymity: send only first 5 chars to the API.
	prefix := hash[:5]
	suffix := hash[5:]

	// Query the HIBP API.
	url := "https://api.pwnedpasswords.com/range/" + prefix
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		// Request creation failure — record and fail-open.
		breachCircuitRecordFailure()
		return nil
	}
	req.Header.Set("User-Agent", "GGID-IAM")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		// Network error — record failure and fail-open.
		breachCircuitRecordFailure()
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// API error — record failure and fail-open.
		breachCircuitRecordFailure()
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		// Body read error — record failure and fail-open.
		breachCircuitRecordFailure()
		return nil
	}

	// Success — reset circuit breaker.
	breachCircuitRecordSuccess()

	// Parse the response: each line is "SUFFIX:COUNT".
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 && parts[0] == suffix {
			return fmt.Errorf("password has been found in %s data breaches", strings.TrimSpace(parts[1]))
		}
	}

	return nil
}
