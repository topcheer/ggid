package service

import (
	"context"
	"time"

	"github.com/ggid/ggid/services/policy/internal/domain"
	"github.com/google/uuid"
)

// ClearDecisionLogForTest clears all stored decisions. Test-only helper.
func ClearDecisionLogForTest() {
	decisionMu.Lock()
	defer decisionMu.Unlock()
	decisionLog = nil
}

// AddTestDecisionForTest adds a synthetic decision entry for testing. Test-only helper.
func AddTestDecisionForTest(allowed bool, matchedBy, action string) {
	decisionMu.Lock()
	defer decisionMu.Unlock()

	decisionLog = append(decisionLog, DecisionEntry{
		Timestamp: time.Now().UTC(),
		UserID:    uuid.New(),
		TenantID:  uuid.New(),
		Action:    action,
		Resource:  "test-resource",
		Allowed:   allowed,
		Reason:    "test reason",
		MatchedBy: matchedBy,
	})

	// Also invoke the callback if set
	if decisionLoggerFn != nil {
		decisionLoggerFn(context.Background(), &domain.CheckRequest{
			Action:   action,
			UserID:   uuid.New(),
			TenantID: uuid.New(),
		}, &domain.CheckResult{
			Allowed:   allowed,
			MatchedBy: matchedBy,
		})
	}
}
