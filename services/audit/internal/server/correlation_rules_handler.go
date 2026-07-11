package httpserver

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

type CorrelationRule struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	EventPattern string        `json:"event_pattern"`
	TimeWindow   string        `json:"time_window"` // e.g. "5m"
	Threshold    int           `json:"threshold"`
	AlertAction  string        `json:"alert_action"`
	Enabled      bool          `json:"enabled"`
	CreatedAt    time.Time     `json:"created_at"`
}

var (
	corrRuleMu sync.RWMutex
	corrRules  = []CorrelationRule{
		{ID: "cr-001", Name: "Brute Force Detection", EventPattern: "failed_login", TimeWindow: "5m", Threshold: 10, AlertAction: "block_ip", Enabled: true, CreatedAt: time.Now().UTC().Add(-48 * time.Hour)},
	}
)

// POST /api/v1/audit/correlation/rules — create correlation rule
// GET /api/v1/audit/correlation/rules — list rules
func (s *HTTPServer) handleCorrelationRules(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req struct {
			Name         string `json:"name"`
			EventPattern string `json:"event_pattern"`
			TimeWindow   string `json:"time_window"`
			Threshold    int    `json:"threshold"`
			AlertAction  string `json:"alert_action"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		if req.EventPattern == "" || req.Threshold == 0 {
			writeJSONError(w, http.StatusBadRequest, "event_pattern and threshold required")
			return
		}
		rule := CorrelationRule{
			ID: "cr-" + uuid.New().String()[:8], Name: req.Name, EventPattern: req.EventPattern,
			TimeWindow: req.TimeWindow, Threshold: req.Threshold, AlertAction: req.AlertAction,
			Enabled: true, CreatedAt: time.Now().UTC(),
		}
		corrRuleMu.Lock()
		corrRules = append(corrRules, rule)
		corrRuleMu.Unlock()
		writeJSON(w, http.StatusCreated, rule)
	case http.MethodGet:
		corrRuleMu.RLock()
		result := make([]CorrelationRule, len(corrRules))
		copy(result, corrRules)
		corrRuleMu.RUnlock()
		writeJSON(w, http.StatusOK, map[string]any{"rules": result, "count": len(result)})
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
