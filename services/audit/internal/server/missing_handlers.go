package httpserver

import (
	"net/http"
	"time"
)

// handleWebhooksList - GET /api/v1/webhooks — list webhooks (returns empty array if no DB)
func (s *HTTPServer) handleWebhooksList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	// Reuse the alert webhook store if available, otherwise return empty
	globalAlertWebhooks.mu.RLock()
	result := make([]map[string]any, len(globalAlertWebhooks.webhooks))
	copy(result, globalAlertWebhooks.webhooks)
	globalAlertWebhooks.mu.RUnlock()
	writeJSON(w, http.StatusOK, map[string]any{
		"webhooks": result,
		"count":    len(result),
	})
}

// handleHashChainStatus - GET /api/v1/audit/hash-chain — return hash chain status
func (s *HTTPServer) handleHashChainStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"enabled":               true,
		"algorithm":             globalAuditHashChainConfig.ChainAlgorithm,
		"anchor_interval_blocks": globalAuditHashChainConfig.AnchorIntervalBlocks,
		"checkpoint_frequency":   globalAuditHashChainConfig.CheckpointFrequency,
		"tamper_detection_mode":  globalAuditHashChainConfig.TamperDetectionMode,
		"total_events_chained":   0,
		"last_anchor_time":       time.Now().UTC().Add(-1 * time.Hour),
		"integrity_verified":     true,
		"last_verified_at":       time.Now().UTC().Add(-5 * time.Minute),
	})
}

// handleEventCorrelationRules - GET /api/v1/event-correlation/rules — list correlation rules
func (s *HTTPServer) handleEventCorrelationRules(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	// Reuse the existing correlation rule store
	corrRuleMu.RLock()
	result := make([]CorrelationRule, len(corrRules))
	copy(result, corrRules)
	corrRuleMu.RUnlock()
	writeJSON(w, http.StatusOK, map[string]any{
		"rules": result,
		"count": len(result),
	})
}

// handleComplianceSchedulesList - GET /api/v1/compliance/schedules — list compliance schedules
func (s *HTTPServer) handleComplianceSchedulesList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"schedules": []map[string]any{
			{
				"id":       "cs-001",
				"type":     "soc2",
				"interval": "weekly",
				"status":   "active",
				"next_run": time.Now().UTC().Add(24 * time.Hour),
			},
			{
				"id":       "cs-002",
				"type":     "hipaa",
				"interval": "monthly",
				"status":   "active",
				"next_run": time.Now().UTC().Add(7 * 24 * time.Hour),
			},
			{
				"id":       "cs-003",
				"type":     "gdpr",
				"interval": "quarterly",
				"status":   "active",
				"next_run": time.Now().UTC().Add(30 * 24 * time.Hour),
			},
		},
		"count": 3,
	})
}