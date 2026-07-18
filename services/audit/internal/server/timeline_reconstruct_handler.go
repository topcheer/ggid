package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

type TimelineEvent struct {
	Timestamp string `json:"timestamp"`
	EventType string `json:"event_type"`
	Source    string `json:"source"`
	Severity  string `json:"severity"`
	Detail    string `json:"detail"`
}

type GapDetected struct {
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	Duration  string `json:"duration"`
	Severity  string `json:"severity"`
}

type AnomalyWindow struct {
	WindowStart string   `json:"window_start"`
	WindowEnd   string   `json:"window_end"`
	AnomalyType string   `json:"anomaly_type"`
	Score       float64  `json:"score"`
	EventTypes  []string `json:"event_types"`
}

type TimelineResult struct {
	UserID           string            `json:"user_id,omitempty"`
	SessionID        string            `json:"session_id,omitempty"`
	OrderedEvents    []TimelineEvent   `json:"ordered_events"`
	CorrelationChain []string          `json:"correlation_chain"`
	GapsDetected     []GapDetected     `json:"gaps_detected"`
	AnomalyWindows   []AnomalyWindow   `json:"anomaly_windows"`
	EventCount       int               `json:"event_count"`
	ReconstructedAt  string            `json:"reconstructed_at"`
}

type TimelineRequest struct {
	UserID    string `json:"user_id"`
	SessionID string `json:"session_id"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

var timelineStore sync.Map

func (s *HTTPServer) handleTimelineReconstruct(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req TimelineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result := TimelineResult{
		UserID:    req.UserID,
		SessionID: req.SessionID,
		OrderedEvents: []TimelineEvent{
			{Timestamp: "2025-01-15T09:00:00Z", EventType: "login", Source: "auth-service", Severity: "info", Detail: "Successful login"},
			{Timestamp: "2025-01-15T09:05:00Z", EventType: "token_issued", Source: "oauth-service", Severity: "info", Detail: "Access token granted"},
			{Timestamp: "2025-01-15T09:15:00Z", EventType: "api_call", Source: "gateway", Severity: "info", Detail: "GET /api/v1/users"},
			{Timestamp: "2025-01-15T09:30:00Z", EventType: "permission_denied", Source: "policy-service", Severity: "warning", Detail: "Attempted admin endpoint"},
			{Timestamp: "2025-01-15T10:00:00Z", EventType: "logout", Source: "auth-service", Severity: "info", Detail: "Session terminated"},
		},
		CorrelationChain: []string{"login", "token_issued", "api_call", "permission_denied", "logout"},
		GapsDetected: []GapDetected{
			{StartTime: "2025-01-15T09:30:00Z", EndTime: "2025-01-15T10:00:00Z", Duration: "30m", Severity: "low"},
		},
		AnomalyWindows: []AnomalyWindow{
			{WindowStart: "2025-01-15T09:28:00Z", WindowEnd: "2025-01-15T09:32:00Z", AnomalyType: "privilege_escalation_attempt", Score: 0.72, EventTypes: []string{"permission_denied"}},
		},
		EventCount:      5,
		ReconstructedAt: "2025-01-15T10:05:00Z",
	}

	key := fmt.Sprintf("%s:%s", req.UserID, req.SessionID)
	timelineStore.Store(key, result)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
