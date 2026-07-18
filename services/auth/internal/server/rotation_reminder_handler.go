package server

import (
	"net/http"
	"sync"
	"time"
)

// rotationReminder represents a credential due for rotation.
type rotationReminder struct {
	ID                    string `json:"id"`
	Type                  string `json:"type"` // password, api_key, client_secret
	UserID                string `json:"user_id,omitempty"`
	ClientID              string `json:"client_id,omitempty"`
	Identifier            string `json:"identifier"`
	LastRotated           string `json:"last_rotated"`
	RecommendedRotateDate string `json:"recommended_rotate_date"`
	DaysOverdue           int    `json:"days_overdue"`
	Severity              string `json:"severity"` // info, warning, critical
}

var rotationReminderStore = struct {
	sync.RWMutex
	reminders []rotationReminder
}{reminders: []rotationReminder{
	{
		ID: "rot-1", Type: "password", UserID: "user-001", Identifier: "admin@ggid.dev",
		LastRotated: time.Now().UTC().Add(-95 * 24 * time.Hour).Format(time.RFC3339),
		RecommendedRotateDate: time.Now().UTC().Add(-5 * 24 * time.Hour).Format("2006-01-02"),
		DaysOverdue: 5, Severity: "warning",
	},
	{
		ID: "rot-2", Type: "api_key", UserID: "user-002", Identifier: "service-key-prod",
		LastRotated: time.Now().UTC().Add(-200 * 24 * time.Hour).Format(time.RFC3339),
		RecommendedRotateDate: time.Now().UTC().Add(-20 * 24 * time.Hour).Format("2006-01-02"),
		DaysOverdue: 20, Severity: "critical",
	},
	{
		ID: "rot-3", Type: "client_secret", ClientID: "web-app", Identifier: "web-app-secret",
		LastRotated: time.Now().UTC().Add(-80 * 24 * time.Hour).Format(time.RFC3339),
		RecommendedRotateDate: time.Now().UTC().Add(10 * 24 * time.Hour).Format("2006-01-02"),
		DaysOverdue: 0, Severity: "info",
	},
	{
		ID: "rot-4", Type: "api_key", UserID: "user-003", Identifier: "legacy-integration-key",
		LastRotated: time.Now().UTC().Add(-365 * 24 * time.Hour).Format(time.RFC3339),
		RecommendedRotateDate: time.Now().UTC().Add(-185 * 24 * time.Hour).Format("2006-01-02"),
		DaysOverdue: 185, Severity: "critical",
	},
}}

// GET /api/v1/auth/rotation-reminders?type=password&severity=critical
func (h *Handler) handleRotationReminders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	typeFilter := r.URL.Query().Get("type")
	severityFilter := r.URL.Query().Get("severity")

	rotationReminderStore.RLock()
	result := []rotationReminder{}
	for _, rem := range rotationReminderStore.reminders {
		if typeFilter != "" && rem.Type != typeFilter {
			continue
		}
		if severityFilter != "" && rem.Severity != severityFilter {
			continue
		}
		result = append(result, rem)
	}
	rotationReminderStore.RUnlock()

	// Summary
	byType := map[string]int{}
	bySeverity := map[string]int{}
	totalOverdue := 0
	for _, r := range result {
		byType[r.Type]++
		bySeverity[r.Severity]++
		if r.DaysOverdue > 0 {
			totalOverdue++
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"reminders":     result,
		"total":         len(result),
		"overdue_count": totalOverdue,
		"by_type":       byType,
		"by_severity":   bySeverity,
		"checked_at":    time.Now().UTC().Format(time.RFC3339),
	})
}
