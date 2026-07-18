package server

import (
	"encoding/json"
	"net/http"
	"strings"
)

type EntitlementPermission struct {
	Permission   string `json:"permission"`
	Source       string `json:"source"`
	LastUsed     string `json:"last_used"`
	UsedIn90d    bool   `json:"used_in_90d"`
	Recommendation string `json:"recommendation"`
}

type EntitlementReviewResult struct {
	UserID                string                  `json:"user_id"`
	DirectPermissions     []EntitlementPermission `json:"direct_permissions"`
	InheritedPermissions  []EntitlementPermission `json:"inherited_permissions"`
	Unused90d             []string                `json:"unused_90d"`
	OverPrivilegedResources []string             `json:"over_privileged_resources"`
	TotalPermissions      int                     `json:"total_permissions"`
	UnusedCount           int                     `json:"unused_count"`
	Recommendation        string                  `json:"overall_recommendation"`
}

func (h *HTTPHandler) handleEntitlementReview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/identity/entitlement-review/"), "/")

	// Cross-analysis sub-endpoint: granted vs used comparison across all users.
	if len(parts) >= 1 && parts[0] == "cross-analysis" {
		h.handleEntitlementCrossAnalysis(w, r)
		return
	}

	userID := parts[0]
	if userID == "" {
		userID = "unknown"
	}

	result := EntitlementReviewResult{
		UserID: userID,
		DirectPermissions: []EntitlementPermission{
			{Permission: "doc:read", Source: "direct", LastUsed: "2025-01-14T10:00:00Z", UsedIn90d: true, Recommendation: "keep"},
			{Permission: "doc:write", Source: "direct", LastUsed: "2025-01-10T14:00:00Z", UsedIn90d: true, Recommendation: "keep"},
			{Permission: "admin:config", Source: "direct", LastUsed: "2024-08-01T09:00:00Z", UsedIn90d: false, Recommendation: "revoke"},
		},
		InheritedPermissions: []EntitlementPermission{
			{Permission: "folder:read", Source: "role:engineer", LastUsed: "2025-01-13T11:00:00Z", UsedIn90d: true, Recommendation: "keep"},
			{Permission: "billing:read", Source: "role:viewer", LastUsed: "2024-06-15T08:00:00Z", UsedIn90d: false, Recommendation: "revoke"},
			{Permission: "storage:write", Source: "group:eng-team", LastUsed: "2024-10-20T16:00:00Z", UsedIn90d: false, Recommendation: "reduce"},
		},
		Unused90d:             []string{"admin:config", "billing:read", "storage:write"},
		OverPrivilegedResources: []string{"system:settings", "billing:invoices"},
		TotalPermissions:      6,
		UnusedCount:           3,
		Recommendation:        "reduce",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
