package httpserver

import (
	"net/http"
	"sync"
	"time"

	"github.com/ggid/ggid/services/policy/internal/service"
)

// sodViolationRecord tracks a detected SoD violation with detection timestamp.
type sodViolationRecord struct {
	UserID         string    `json:"user_id"`
	ConflictingRoles []string `json:"conflicting_roles"`
	RuleID         string    `json:"rule_id"`
	Reason         string    `json:"reason"`
	DetectedAt     time.Time `json:"detected_at"`
}

// sodViolationStore holds detected violations keyed by a composite key.
var (
	sodViolationsMu     sync.RWMutex
	sodViolationsByUser = make(map[string][]*sodViolationRecord)
)

// RecordSoDViolation stores a violation for later reporting.
func RecordSoDViolation(userID string, roles []string, ruleID, reason string) {
	sodViolationsMu.Lock()
	defer sodViolationsMu.Unlock()
	sodViolationsByUser[userID] = append(sodViolationsByUser[userID], &sodViolationRecord{
		UserID:           userID,
		ConflictingRoles: roles,
		RuleID:           ruleID,
		Reason:           reason,
		DetectedAt:       time.Now().UTC(),
	})
}

// GET /api/v1/policies/sod/violations — list all active SoD violations.
// Query params: user_id (optional filter), tenant_id (optional filter).
func (s *HTTPServer) handleSoDViolations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userFilter := r.URL.Query().Get("user_id")

	// Collect violations from store
	sodViolationsMu.RLock()
	result := []*sodViolationRecord{}
	for uid, violations := range sodViolationsByUser {
		if userFilter != "" && uid != userFilter {
			continue
		}
		result = append(result, violations...)
	}
	sodViolationsMu.RUnlock()

	// Also evaluate known rules against the SoD rule set to report potential conflicts
	// Check the service-layer rules and report them as "policy definitions"
	sodRules := service.GetSoDRules()

	writeJSON(w, http.StatusOK, map[string]any{
		"violations":       result,
		"violation_count":  len(result),
		"active_rules":     sodRules,
		"rule_count":       len(sodRules),
	})
}
