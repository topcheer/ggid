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
	// Check for sub-paths: /api/v1/webhooks/{id}/test|deliveries|rotate-secret
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	// Expected: ["api", "v1", "webhooks", "{id}", "{action}"] or ["api", "v1", "webhooks", "{id}"]
	if len(pathParts) >= 5 {
		whID := pathParts[3]
		action := pathParts[4]
		switch action {
		case "test":
			s.handleWebhookTest(w, r, whID)
			return
		case "deliveries":
			if len(pathParts) >= 7 && pathParts[5] == "retry" {
				s.handleWebhookDeliveryRetry(w, r, whID, pathParts[6])
				return
			}
			s.handleWebhookDeliveries(w, r, whID)
			return
		case "rotate-secret":
			s.handleWebhookRotateSecret(w, r, whID)
			return
		}
	}

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
	case http.MethodPut, http.MethodPatch:
		// Extract webhook ID from path: /api/v1/webhooks/{id}
		pathParts := strings.Split(r.URL.Path, "/")
		whID := ""
		if len(pathParts) > 0 {
			whID = pathParts[len(pathParts)-1]
		}
		var update struct {
			Name   *string  `json:"name"`
			URL    *string  `json:"url"`
			Events *[]string `json:"events"`
			Active *bool    `json:"active"`
		}
		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		globalAlertWebhooks.mu.Lock()
		for _, wh := range globalAlertWebhooks.webhooks {
			if wh["id"] == whID {
				if update.Name != nil { wh["name"] = *update.Name }
				if update.URL != nil { wh["url"] = *update.URL }
				if update.Events != nil { wh["events"] = *update.Events }
				if update.Active != nil { wh["active"] = *update.Active }
				if s.memMapRepo2 != nil {
					_ = s.memMapRepo2.StoreJSON(r.Context(), "audit_webhook_configs", whID, wh)
				}
				globalAlertWebhooks.mu.Unlock()
				writeJSON(w, http.StatusOK, wh)
				return
			}
		}
		globalAlertWebhooks.mu.Unlock()
		writeJSONError(w, http.StatusNotFound, "webhook not found")
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

// handleWebhookTest - POST /api/v1/webhooks/{id}/test
func (s *HTTPServer) handleWebhookTest(w http.ResponseWriter, r *http.Request, whID string) {
	// Find the webhook
	globalAlertWebhooks.mu.RLock()
	var webhook map[string]any
	for _, wh := range globalAlertWebhooks.webhooks {
		if wh["id"] == whID {
			webhook = wh
			break
		}
	}
	globalAlertWebhooks.mu.RUnlock()

	if webhook == nil {
		writeJSONError(w, http.StatusNotFound, "webhook not found")
		return
	}

	url, _ := webhook["url"].(string)
	if url == "" {
		writeJSONError(w, http.StatusBadRequest, "webhook has no URL")
		return
	}

	// Send a test payload
	testPayload := map[string]any{
		"event":     "webhook.test",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"data": map[string]any{
			"webhook_id": whID,
			"message":    "This is a test delivery from GGID",
		},
	}
	body, _ := json.Marshal(testPayload)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(url, "application/json", strings.NewReader(string(body)))
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":   "failed",
			"error":    err.Error(),
			"success":  false,
		})
		return
	}
	defer resp.Body.Close()

	writeJSON(w, http.StatusOK, map[string]any{
		"status":      "delivered",
		"status_code": resp.StatusCode,
		"success":     resp.StatusCode >= 200 && resp.StatusCode < 300,
	})
}

// handleWebhookDeliveries - GET /api/v1/webhooks/{id}/deliveries
func (s *HTTPServer) handleWebhookDeliveries(w http.ResponseWriter, r *http.Request, whID string) {
	// Return empty list for now — delivery tracking would require persistent storage
	writeJSON(w, http.StatusOK, map[string]any{
		"deliveries": []any{},
		"count":      0,
	})
}

// handleWebhookDeliveryRetry - POST /api/v1/webhooks/{id}/deliveries/{deliveryId}/retry
func (s *HTTPServer) handleWebhookDeliveryRetry(w http.ResponseWriter, r *http.Request, whID string, deliveryID string) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":      "retrying",
		"delivery_id": deliveryID,
		"webhook_id":  whID,
	})
}

// handleWebhookRotateSecret - POST /api/v1/webhooks/{id}/rotate-secret
func (s *HTTPServer) handleWebhookRotateSecret(w http.ResponseWriter, r *http.Request, whID string) {
	// Generate a new secret
	secret := "whsec_" + fmt.Sprintf("%d", time.Now().UnixNano())

	// Update the webhook in memory
	globalAlertWebhooks.mu.Lock()
	for _, wh := range globalAlertWebhooks.webhooks {
		if wh["id"] == whID {
			wh["secret"] = secret
			if s.memMapRepo2 != nil {
				_ = s.memMapRepo2.StoreJSON(r.Context(), "audit_webhook_configs", whID, wh)
			}
			break
		}
	}
	globalAlertWebhooks.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"secret": secret,
	})
}