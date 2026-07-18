package server

import (
	"net/http"
	"sync"
	"time"
)

// terminationEntry records a session termination event.
type terminationEntry struct {
	SessionID string `json:"session_id"`
	UserID    string `json:"user_id"`
	Reason    string `json:"reason"` // timeout, revocation, hijack_detection, user_logout, admin_force
	Timestamp string `json:"timestamp"`
	IPAddress string `json:"ip_address"`
}

var terminationStore = struct {
	sync.RWMutex
	entries []terminationEntry
}{entries: []terminationEntry{
	{SessionID: "sess-001", UserID: "user-001", Reason: "timeout", Timestamp: time.Now().UTC().Add(-30 * time.Minute).Format(time.RFC3339), IPAddress: "192.168.1.10"},
	{SessionID: "sess-002", UserID: "user-003", Reason: "user_logout", Timestamp: time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339), IPAddress: "10.0.0.5"},
	{SessionID: "sess-003", UserID: "user-005", Reason: "hijack_detection", Timestamp: time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339), IPAddress: "203.0.113.5"},
	{SessionID: "sess-004", UserID: "user-007", Reason: "revocation", Timestamp: time.Now().UTC().Add(-3 * time.Hour).Format(time.RFC3339), IPAddress: "172.16.0.8"},
	{SessionID: "sess-005", UserID: "user-009", Reason: "admin_force", Timestamp: time.Now().UTC().Add(-4 * time.Hour).Format(time.RFC3339), IPAddress: "192.168.1.22"},
	{SessionID: "sess-006", UserID: "user-011", Reason: "timeout", Timestamp: time.Now().UTC().Add(-5 * time.Hour).Format(time.RFC3339), IPAddress: "10.0.0.12"},
	{SessionID: "sess-007", UserID: "user-013", Reason: "user_logout", Timestamp: time.Now().UTC().Add(-6 * time.Hour).Format(time.RFC3339), IPAddress: "192.168.1.30"},
	{SessionID: "sess-008", UserID: "user-015", Reason: "timeout", Timestamp: time.Now().UTC().Add(-7 * time.Hour).Format(time.RFC3339), IPAddress: "172.16.0.15"},
	{SessionID: "sess-009", UserID: "user-002", Reason: "hijack_detection", Timestamp: time.Now().UTC().Add(-8 * time.Hour).Format(time.RFC3339), IPAddress: "198.51.100.2"},
	{SessionID: "sess-010", UserID: "user-004", Reason: "user_logout", Timestamp: time.Now().UTC().Add(-9 * time.Hour).Format(time.RFC3339), IPAddress: "10.0.0.20"},
}}

// GET /api/v1/auth/sessions/termination-reasons?from=X&to=Y
func (h *Handler) handleTerminationReasons(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	var fromTime, toTime time.Time
	if fromStr != "" {
		fromTime, _ = time.Parse(time.RFC3339, fromStr)
	}
	if toStr != "" {
		toTime, _ = time.Parse(time.RFC3339, toStr)
	}

	terminationStore.RLock()
	defer terminationStore.RUnlock()

	reasonCounts := map[string]int{}
	reasonPct := map[string]float64{}
	var filtered []terminationEntry

	for _, e := range terminationStore.entries {
		ts, _ := time.Parse(time.RFC3339, e.Timestamp)
		if !fromTime.IsZero() && ts.Before(fromTime) {
			continue
		}
		if !toTime.IsZero() && ts.After(toTime) {
			continue
		}
		filtered = append(filtered, e)
		reasonCounts[e.Reason]++
	}

	total := len(filtered)
	for reason, count := range reasonCounts {
		if total > 0 {
			reasonPct[reason] = float64(count) / float64(total) * 100
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"total_terminations":   total,
		"reason_distribution":  reasonCounts,
		"reason_percentage":    reasonPct,
		"entries":              filtered,
		"pie_chart_data": func() []map[string]any {
			data := []map[string]any{}
			for reason, count := range reasonCounts {
				data = append(data, map[string]any{
					"label": reason,
					"value": count,
					"pct":   reasonPct[reason],
				})
			}
			return data
		}(),
		"checked_at": time.Now().UTC().Format(time.RFC3339),
	})
}
