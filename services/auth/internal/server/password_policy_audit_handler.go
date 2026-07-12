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
}{violations: []passwordViolation{
	{UserID: "user-001", Email: "bob@ex.com", Rule: "min_length", Detail: "Password is 6 chars, minimum is 12", Severity: "high"},
	{UserID: "user-003", Email: "carol@ex.com", Rule: "complexity", Detail: "Missing uppercase and special character", Severity: "medium"},
	{UserID: "user-005", Email: "eve@ex.com", Rule: "expiry", Detail: "Password age is 180 days, max is 90", Severity: "high"},
	{UserID: "user-007", Email: "grace@ex.com", Rule: "history", Detail: "Password reused from last 5 passwords", Severity: "critical"},
	{UserID: "user-009", Email: "ivan@ex.com", Rule: "reuse", Detail: "Same password as another account", Severity: "critical"},
}}

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

	totalUsers := 15420 // simulated total
	nonCompliant := len(violations)
	compliant := totalUsers - nonCompliant

	writeJSON(w, http.StatusOK, map[string]any{
		"total_users":         totalUsers,
		"compliant_count":     compliant,
		"non_compliant_count": nonCompliant,
		"compliance_rate_pct": float64(compliant) / float64(totalUsers) * 100,
		"violations":          violations,
		"total_violations":    len(violations),
		"by_rule":             byRule,
		"by_severity":         bySeverity,
		"audited_at":          time.Now().UTC().Format(time.RFC3339),
	})
}
