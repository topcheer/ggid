package server

import (
	"encoding/json"
	"net/http"
	"sync"
)

type UnusedPermission struct {
	Permission string  `json:"permission"`
	LastUsed   string  `json:"last_used"`
	UsageRate  float64 `json:"usage_rate"`
}

type OverAssignedRole struct {
	UserID   string   `json:"user_id"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
	Excess   int      `json:"excess_permissions"`
}

type SuggestedConsolidation struct {
	RoleA       string   `json:"role_a"`
	RoleB       string   `json:"role_b"`
	Overlap     float64  `json:"overlap_score"`
	SharedPerms []string `json:"shared_permissions"`
}

type RedundantRole struct {
	RoleID      string  `json:"role_id"`
	RoleName    string  `json:"role_name"`
	Assignees   int     `json:"assignees"`
	Redundancy  float64 `json:"redundancy_score"`
}

type RoleMiningResult struct {
	UnusedPermissions      []UnusedPermission      `json:"unused_permissions"`
	OverAssignedRoles      []OverAssignedRole      `json:"over_assigned_roles"`
	SuggestedConsolidation []SuggestedConsolidation `json:"suggested_consolidation"`
	EntitlementCreepScore  float64                 `json:"entitlement_creep_score"`
	TopRedundantRoles      []RedundantRole          `json:"top_redundant_roles"`
	GeneratedAt            string                  `json:"generated_at"`
}

var (
	roleMiningStore   sync.Map
	roleMiningOnce    sync.Once
)

func initRoleMiningData() {
	roleMiningOnce.Do(func() {
		data := RoleMiningResult{
			UnusedPermissions: []UnusedPermission{
				{Permission: "storage:write", LastUsed: "2024-12-01T00:00:00Z", UsageRate: 0.02},
				{Permission: "billing:read", LastUsed: "2024-11-15T00:00:00Z", UsageRate: 0.05},
				{Permission: "admin:config", LastUsed: "2024-10-20T00:00:00Z", UsageRate: 0.01},
			},
			OverAssignedRoles: []OverAssignedRole{
				{UserID: "u-001", Username: "alice", Roles: []string{"editor", "viewer"}, Excess: 12},
				{UserID: "u-002", Username: "bob", Roles: []string{"admin", "editor", "viewer"}, Excess: 25},
			},
			SuggestedConsolidation: []SuggestedConsolidation{
				{RoleA: "editor", RoleB: "viewer", Overlap: 0.85, SharedPerms: []string{"read:all", "comment:all"}},
				{RoleA: "admin", RoleB: "editor", Overlap: 0.60, SharedPerms: []string{"write:all", "delete:own"}},
			},
			EntitlementCreepScore: 0.34,
			TopRedundantRoles: []RedundantRole{
				{RoleID: "r-001", RoleName: "legacy_editor", Assignees: 3, Redundancy: 0.92},
				{RoleID: "r-005", RoleName: "temp_admin", Assignees: 1, Redundancy: 0.88},
				{RoleID: "r-009", RoleName: "old_viewer", Assignees: 7, Redundancy: 0.75},
			},
			GeneratedAt: "2025-01-15T10:00:00Z",
		}
		roleMiningStore.Store("latest", data)
	})
}

func (h *HTTPHandler) handleRoleMining(w http.ResponseWriter, r *http.Request) {
	initRoleMiningData()
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	val, ok := roleMiningStore.Load("latest")
	if !ok {
		writeJSONError(w, http.StatusNotFound, "no data")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(val)
}
