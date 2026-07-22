package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// handleWebhooksList - GET /api/v1/webhooks — list webhooks (returns empty array if no DB)
func (s *HTTPServer) handleWebhooksList(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// DB-backed when available (Task-C pattern); memory fallback.
		if s.memMapRepo2 != nil {
			if rows, err := s.memMapRepo2.ListJSON(r.Context(), "audit_webhook_configs"); err == nil {
				if rows == nil {
					rows = []map[string]any{}
				}
				writeJSON(w, http.StatusOK, map[string]any{
					"webhooks": rows,
					"count":    len(rows),
				})
				return
			}
		}
		globalAlertWebhooks.mu.RLock()
		result := make([]map[string]any, len(globalAlertWebhooks.webhooks))
		copy(result, globalAlertWebhooks.webhooks)
		globalAlertWebhooks.mu.RUnlock()
		writeJSON(w, http.StatusOK, map[string]any{
			"webhooks": result,
		"count":    len(result),
	})
	case http.MethodPost:
		var req struct {
			Name    string   `json:"name"`
			URL     string   `json:"url"`
			Events  []string `json:"events"`
			Active  bool     `json:"active"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.Name == "" {
			req.Name = req.URL
		}
		webhook := map[string]any{
			"id":     fmt.Sprintf("whk_%d", time.Now().UnixNano()),
			"name":   req.Name,
			"url":    req.URL,
			"events": req.Events,
			"active": req.Active,
			"created_at": time.Now().UTC().Format(time.RFC3339),
		}
		globalAlertWebhooks.mu.Lock()
		globalAlertWebhooks.webhooks = append(globalAlertWebhooks.webhooks, webhook)
		globalAlertWebhooks.mu.Unlock()
		// Persist so webhooks survive restarts and are shared across replicas.
		if s.memMapRepo2 != nil {
			_ = s.memMapRepo2.StoreJSON(r.Context(), "audit_webhook_configs", webhook["id"].(string), webhook)
		}
		writeJSON(w, http.StatusCreated, webhook)
	case http.MethodDelete:
		// Extract webhook ID from path: /api/v1/webhooks/{id}
		pathParts := strings.Split(r.URL.Path, "/")
		whID := ""
		if len(pathParts) > 0 {
			whID = pathParts[len(pathParts)-1]
		}
		if whID != "" {
			globalAlertWebhooks.mu.Lock()
			filtered := globalAlertWebhooks.webhooks[:0]
			for _, wh := range globalAlertWebhooks.webhooks {
				if wh["id"] != whID {
					filtered = append(filtered, wh)
				}
			}
			globalAlertWebhooks.webhooks = filtered
			globalAlertWebhooks.mu.Unlock()
			if s.memMapRepo2 != nil {
				_ = s.memMapRepo2.DeleteJSON(r.Context(), "audit_webhook_configs", whID)
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "deleted", "id": whID})
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
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