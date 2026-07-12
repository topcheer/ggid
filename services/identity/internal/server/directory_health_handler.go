package server

import (
	"net/http"
	"sync"
	"time"
)

// directoryHealthIssue represents a data quality issue in the user directory.
type directoryHealthIssue struct {
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Count       int    `json:"count"`
	Description string `json:"description"`
	Examples    []string `json:"examples,omitempty"`
}

var directoryHealthStore = struct {
	sync.RWMutex
	lastChecked string
}{lastChecked: time.Now().UTC().Add(-10 * time.Minute).Format(time.RFC3339)}

// GET /api/v1/identity/directory-health
func (h *HTTPHandler) handleDirectoryHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	now := time.Now().UTC()

	issues := []directoryHealthIssue{
		{
			Type: "orphaned_accounts", Severity: "high", Count: 12,
			Description: "Users with no manager assignment and no active role",
			Examples:    []string{"user-0042", "user-0078", "user-0156"},
		},
		{
			Type: "duplicate_emails", Severity: "medium", Count: 3,
			Description: "Multiple accounts sharing the same email address",
			Examples:    []string{"dup@example.com (2 accounts)", "shared@company.com (2 accounts)"},
		},
		{
			Type: "stale_managers", Severity: "medium", Count: 8,
			Description: "Manager accounts that are inactive or deactivated",
			Examples:    []string{"mgr-002 (deactivated 30d ago)", "mgr-007 (suspended)"},
		},
		{
			Type: "missing_departments", Severity: "low", Count: 45,
			Description: "Users without a department assignment",
		},
		{
			Type: "stale_accounts", Severity: "medium", Count: 28,
			Description: "Active accounts with no login in 90+ days",
		},
		{
			Type: "non_compliant_mfa", Severity: "high", Count: 5800,
			Description: "Users without MFA enrollment",
		},
		{
			Type: "weak_passwords", Severity: "high", Count: 245,
			Description: "Users with passwords not meeting current policy",
		},
	}

	totalIssues := 0
	highSeverity := 0
	for _, iss := range issues {
		totalIssues += iss.Count
		if iss.Severity == "high" {
			highSeverity += iss.Count
		}
	}

	// Overall health score
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

	writeJSON(w, http.StatusOK, map[string]any{
		"overall_status":    overallStatus,
		"health_score":      healthScore,
		"total_users":       15420,
		"total_issues":      totalIssues,
		"high_severity_count": highSeverity,
		"compliance_issues": issues,
		"by_severity": map[int]string{
			1: "high", 2: "medium", 3: "low",
		},
		"severity_counts": func() map[string]int {
			counts := map[string]int{}
			for _, iss := range issues {
				counts[iss.Severity]++
			}
			return counts
		}(),
		"checked_at": now.Format(time.RFC3339),
		"last_full_scan": directoryHealthStore.lastChecked,
	})
}
