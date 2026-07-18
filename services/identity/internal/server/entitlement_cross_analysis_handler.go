package server

import (
	"net/http"

	ggidtenant "github.com/ggid/ggid/pkg/tenant"
)

// CrossAnalysisResponse shows granted vs used entitlements across a tenant.
type CrossAnalysisResponse struct {
	TenantID          string                 `json:"tenant_id"`
	TotalGranted      int                    `json:"total_granted"`
	TotalUsed90d      int                    `json:"total_used_90d"`
	UnusedCount       int                    `json:"unused_count"`
	UnusedPct         int                    `json:"unused_pct"`
	OverPrivilegedUsers []CrossAnalysisUser   `json:"over_privileged_users"`
	Summary           string                 `json:"summary"`
}

type CrossAnalysisUser struct {
	UserID       string   `json:"user_id"`
	GrantedCount int      `json:"granted_count"`
	UsedCount    int      `json:"used_count"`
	UnusedPerms  []string `json:"unused_perms"`
}

// handleEntitlementCrossAnalysis returns granted x used cross-analysis.
// GET /api/v1/identity/entitlement-review/cross-analysis
func (h *HTTPHandler) handleEntitlementCrossAnalysis(w http.ResponseWriter, r *http.Request) {
	tc, err := ggidtenant.FromContext(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "tenant context required")
		return
	}

	// In production, this would query the policy service for all granted
	// permissions and cross-reference with audit logs for usage in the last 90 days.
	// For now, returns a structured response with empty data (ready for DB wiring).

	resp := CrossAnalysisResponse{
		TenantID:            tc.TenantID.String(),
		TotalGranted:        0,
		TotalUsed90d:        0,
		UnusedCount:         0,
		UnusedPct:           0,
		OverPrivilegedUsers: []CrossAnalysisUser{},
		Summary:             "No entitlement usage data available. Wire policy service for granted permissions and audit service for usage tracking.",
	}

	writeJSON(w, http.StatusOK, resp)
}
