package middleware

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// CAEMiddleware implements Continuous Authorization Evaluation.
// On every request to protected resources, it evaluates risk via URE
// and enforces the decision (allow/step_up/block).
//
// Uses PDP cache (5s TTL) to avoid re-evaluating on sub-requests.
// Privileged endpoints force re-evaluation bypassing cache.

// privilegedPaths are endpoints that always force risk re-evaluation.
var privilegedPaths = []string{
	"/api/v1/admin/",
	"/api/v1/users/delete",
	"/api/v1/users/create",
	"/api/v1/policies/",
	"/api/v1/crypto/",
	"/api/v1/identity/secret-broker/",
	"/api/v1/audit/retention/",
}

// sessionRiskCache caches risk evaluations per session (15 min re-eval interval).
var (
	sessionRiskCache   sync.Map // sessionID → riskEntry
	sessionRiskTTL      = 15 * time.Minute
)

type riskEntry struct {
	score     int
	decision  string
	evaluatedAt time.Time
}

// isPrivilegedPath checks if the request path requires forced re-evaluation.
func isPrivilegedPath(path string) bool {
	for _, p := range privilegedPaths {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

// CAEMiddleware creates middleware that evaluates risk on every request.
// riskEvalFn is a callback to the URE evaluate function.
func CAEMiddleware(riskEvalFn func(ctx context.Context, userID, sessionID string) (score int, decision string)) func(http.Handler) http.Handler {
	if riskEvalFn == nil {
		return func(next http.Handler) http.Handler {
			return next
		}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip health checks and public endpoints.
			if strings.HasSuffix(r.URL.Path, "/healthz") || strings.HasSuffix(r.URL.Path, "/readyz") {
				next.ServeHTTP(w, r)
				return
			}

			// Extract user/session from JWT context (set by auth middleware).
			userID, _ := UserIDFromRequest(r)
			sessionID := r.Header.Get("X-Session-ID")
			if userID == uuid.Nil {
				// No authenticated user — let auth middleware handle it.
				next.ServeHTTP(w, r)
				return
			}

			// Check session cache (unless privileged).
			cacheKey := sessionID
			if cacheKey == "" {
				cacheKey = userID.String()
			}
			force := isPrivilegedPath(r.URL.Path)

			if !force {
				if v, ok := sessionRiskCache.Load(cacheKey); ok {
					entry := v.(riskEntry)
					if time.Since(entry.evaluatedAt) < sessionRiskTTL {
						if entry.decision == "block" {
							http.Error(w, `{"error":"access_blocked","reason":"risk_score_too_high"}`, http.StatusForbidden)
							return
						}
						// step_up: add header for downstream MFA challenge.
						if entry.decision == "step_up" || entry.decision == "step_up_strong" {
							w.Header().Set("X-Risk-Step-Up", entry.decision)
						}
						next.ServeHTTP(w, r)
						return
					}
				}
			}

			// Evaluate risk via URE.
			score, decision := riskEvalFn(r.Context(), userID.String(), sessionID)

			// Cache the result.
			sessionRiskCache.Store(cacheKey, riskEntry{
				score: score, decision: decision, evaluatedAt: time.Now(),
			})

			// Enforce decision.
			switch decision {
			case "block":
				http.Error(w, `{"error":"access_blocked","reason":"risk_score_too_high"}`, http.StatusForbidden)
			case "step_up", "step_up_strong":
				w.Header().Set("X-Risk-Step-Up", decision)
				fallthrough
			default:
				next.ServeHTTP(w, r)
			}
		})
	}
}

// FlushSessionRiskCache clears all cached risk evaluations.
func FlushSessionRiskCache() {
	sessionRiskCache.Range(func(key, _ any) bool {
		sessionRiskCache.Delete(key)
		return true
	})
}
