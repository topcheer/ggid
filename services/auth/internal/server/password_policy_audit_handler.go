package server

import (
	"net/http"
	"sync"
	"time"
)

// passwordViolation represents a single user's password policy violation.
type passwordViolation struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Rule   string `json:"rule"`   // min_length, complexity, history, expiry, reuse
	Detail string `json:"detail"`
	Severity string `json:"severity"`
}

var passwordAuditStore = struct {
	sync.RWMutex
	violations []passwordViolation
}{violations: []passwordViolation{}}

// GET /api/v1/auth/password-policy/audit
func (h *Handler) handlePasswordPolicyAudit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	passwordAuditStore.RLock()
	violations := make([]passwordViolation, len(passwordAuditStore.violations))
	copy(violations, passwordAuditStore.violations)
	passwordAuditStore.RUnlock()

	// Compute summary
	byRule := map[string]int{}
	bySeverity := map[string]int{}
	for _, v := range violations {
		byRule[v.Rule]++
		bySeverity[v.Severity]++
	}

	nonCompliant := len(violations)

	writeJSON(w, http.StatusOK, map[string]any{
		"total_users":         0, // requires DB aggregate query
		"compliant_count":     0,
		"non_compliant_count": nonCompliant,
		"compliance_rate_pct": 0.0,
		"violations":          violations,
		"total_violations":    len(violations),
		"by_rule":             byRule,
		"by_severity":         bySeverity,
		"audited_at":          time.Now().UTC().Format(time.RFC3339),
	})
}
