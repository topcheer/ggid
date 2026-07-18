package server

import (
	"net/http"
	"sync"
	"time"

	"github.com/ggid/ggid/services/identity/internal/domain"
)

// directoryHealthIssue represents a data quality issue in the user directory.
type directoryHealthIssue struct {
	Type        string   `json:"type"`
	Severity    string   `json:"severity"`
	Count       int      `json:"count"`
	Description string   `json:"description"`
	Examples    []string `json:"examples,omitempty"`
}

var directoryHealthStore = struct {
	sync.RWMutex
	lastChecked string
}{lastChecked: time.Now().UTC().Add(-10 * time.Minute).Format(time.RFC3339)}

// GET /api/v1/identity/directory-health
func (h *HTTPHandler) handleDirectoryHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	now := time.Now().UTC()

	// Query real user data from the identity service.
	ctx := r.Context()
	result, err := h.svc.ListUsers(ctx, &domain.ListUsersFilter{PageSize: 1000})
	if err != nil {
		// Fall back to empty data if service is unavailable.
		writeJSON(w, http.StatusOK, map[string]any{
			"overall_status":      "unknown",
			"health_score":        0,
			"total_users":         0,
			"total_issues":        0,
			"high_severity_count": 0,
			"compliance_issues":   []directoryHealthIssue{},
			"checked_at":          now.Format(time.RFC3339),
			"last_full_scan":      directoryHealthStore.lastChecked,
			"error":               "unable to query user directory",
		})
		return
	}

	totalUsers := 0
	staleAccounts := 0
	if result != nil {
		totalUsers = result.Total
		for _, u := range result.Users {
			// Stale: no login in 90+ days.
			if u.LastLoginAt != nil && now.Sub(*u.LastLoginAt) > 90*24*time.Hour {
				staleAccounts++
			}
		}
	}

	// Build issues from real data.
	issues := []directoryHealthIssue{}
	if staleAccounts > 0 {
		issues = append(issues, directoryHealthIssue{
			Type: "stale_accounts", Severity: "medium", Count: staleAccounts,
			Description: "Active accounts with no login in 90+ days",
		})
	}

	// Compute health score from issues.
	totalIssues := 0
	highSeverity := 0
	for _, iss := range issues {
		totalIssues += iss.Count
		if iss.Severity == "high" {
			highSeverity += iss.Count
		}
	}

	healthScore := 100
	for _, iss := range issues {
		switch iss.Severity {
		case "high":
			healthScore -= iss.Count / 100
		case "medium":
			healthScore -= iss.Count / 200
		case "low":
			healthScore -= iss.Count / 500
		}
	}
	if healthScore < 0 {
		healthScore = 0
	}

	overallStatus := "healthy"
	if healthScore < 50 {
		overallStatus = "critical"
	} else if healthScore < 70 {
		overallStatus = "warning"
	} else if healthScore < 90 {
		overallStatus = "fair"
	}

	// Update last checked timestamp.
	directoryHealthStore.Lock()
	directoryHealthStore.lastChecked = now.Format(time.RFC3339)
	directoryHealthStore.Unlock()

	severityCounts := map[string]int{}
	for _, iss := range issues {
		severityCounts[iss.Severity]++
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"overall_status":      overallStatus,
		"health_score":        healthScore,
		"total_users":         totalUsers,
		"total_issues":        totalIssues,
		"high_severity_count": highSeverity,
		"compliance_issues":   issues,
		"severity_counts":     severityCounts,
		"checked_at":          now.Format(time.RFC3339),
		"last_full_scan":      directoryHealthStore.lastChecked,
	})
}