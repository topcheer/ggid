// Package httpserver provides REST API endpoints for the Org Service.
// These endpoints allow the Admin Console to manage organizations,
// departments, teams, and memberships via HTTP through the API Gateway.
package httpserver

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/ggid/ggid/pkg/audit"
	"github.com/ggid/ggid/pkg/errors"
	"github.com/ggid/ggid/services/org/internal/domain"
	"github.com/ggid/ggid/services/org/internal/repository"
	"github.com/ggid/ggid/services/org/internal/service"
	"github.com/google/uuid"
)

// HTTPServer exposes the Org Service as a REST API.
type HTTPServer struct {
	orgSvc         *service.OrgService
	deptSvc        *service.DeptService
	teamSvc        *service.TeamService
	memberSvc      *service.MembershipService
	tenantSvc      *service.TenantService
	auditPublisher *audit.Publisher
}

// NewHTTPServer creates a new Org Service HTTP server.
func NewHTTPServer(orgSvc *service.OrgService, deptSvc *service.DeptService, teamSvc *service.TeamService, memberSvc *service.MembershipService, tenantSvc *service.TenantService) *HTTPServer {
	s := &HTTPServer{orgSvc: orgSvc, deptSvc: deptSvc, teamSvc: teamSvc, memberSvc: memberSvc, tenantSvc: tenantSvc}
	if natsURL := os.Getenv("NATS_URL"); natsURL != "" {
		if pub, err := audit.NewPublisher(context.Background(), natsURL); err == nil {
			s.auditPublisher = pub
			log.Println("Org: audit publisher connected to NATS")
		} else {
			log.Printf("Org: audit publisher disabled (%v)", err)
		}
	}
	return s
}

// RegisterRoutes registers all Org Service HTTP routes on the given mux.
func (s *HTTPServer) RegisterRoutes(mux *http.ServeMux) {
	// Organizations
	mux.HandleFunc("/api/v1/orgs", s.handleOrgs)
	mux.HandleFunc("/api/v1/organizations", s.handleOrgs)
	mux.HandleFunc("/api/v1/orgs/tree", s.handleFullTree)
	mux.HandleFunc("/api/v1/orgs/tree-with-members", s.handleOrgTreeWithMembers)
	mux.HandleFunc("/api/v1/organizations/", s.handleOrgRoleBindings)
	mux.HandleFunc("/api/v1/orgs/", s.handleOrgByID)
	// Departments
	mux.HandleFunc("/api/v1/departments", s.handleDepartments)
	mux.HandleFunc("/api/v1/departments/", s.handleDepartmentByID)
	// Teams
	mux.HandleFunc("/api/v1/teams", s.handleTeams)
	mux.HandleFunc("/api/v1/teams/", s.handleTeamByID)
	mux.HandleFunc("/api/v1/org/cost-centers", s.handleCostCenters)
	mux.HandleFunc("/api/v1/org/budget-tracking", s.handleBudgetTracking)
	mux.HandleFunc("/api/v1/org/reporting-structure", s.handleReportingStructure)
	mux.HandleFunc("/api/v1/org/team-insights", s.handleTeamInsights)
	mux.HandleFunc("/api/v1/org/vendors", s.handleVendors)
	mux.HandleFunc("/api/v1/org/department-analytics", s.handleDepartmentAnalytics)
	mux.HandleFunc("/api/v1/org/tenants/migrate", s.handleTenantMigrate)
	mux.HandleFunc("/api/v1/org/tenants/suspend", s.handleSuspendTenant)
	mux.HandleFunc("/api/v1/org/tenants/activate", s.handleActivateTenant)
	mux.HandleFunc("/api/v1/org/stats/membership-trends", s.handleMembershipTrends)
}

// GET /api/v1/orgs/tree?tenant_id=X&depth=N — returns full org tree as nested structure
func (s *HTTPServer) handleFullTree(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	tenantIDStr := r.URL.Query().Get("tenant_id")
	if tenantIDStr == "" {
		writeJSONError(w, http.StatusBadRequest, "tenant_id query parameter is required")
		return
	}
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid tenant_id")
		return
	}

	depth := 0 // 0 = unlimited
	if d := r.URL.Query().Get("depth"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 {
			depth = parsed
		}
	}

	orgs, err := s.orgSvc.List(r.Context(), tenantID, 1, 500)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to load organizations")
		return
	}

	// Build nested tree structure
	tree := buildOrgTree(orgs, depth)
	writeJSON(w, http.StatusOK, map[string]any{
		"tree":     tree,
		"count":    len(orgs),
		"depth":    depth,
	})
}

// orgTreeNode represents a node in the org tree response.
type orgTreeNode struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Path     string         `json:"path"`
	ParentID *string        `json:"parent_id,omitempty"`
	Children []*orgTreeNode `json:"children,omitempty"`
}

func buildOrgTree(orgs []*domain.Organization, maxDepth int) []*orgTreeNode {
	nodeMap := make(map[uuid.UUID]*orgTreeNode)
	var roots []*orgTreeNode

	// Create all nodes
	for _, org := range orgs {
		node := &orgTreeNode{
			ID:       org.ID.String(),
			Name:     org.Name,
			Path:     org.Path,
			Children: []*orgTreeNode{},
		}
		if org.ParentID != nil {
			pid := org.ParentID.String()
			node.ParentID = &pid
		}
		nodeMap[org.ID] = node
	}

	// Link children to parents
	for _, org := range orgs {
		if org.ParentID == nil {
			roots = append(roots, nodeMap[org.ID])
		} else if parent, ok := nodeMap[*org.ParentID]; ok {
			parent.Children = append(parent.Children, nodeMap[org.ID])
		} else {
			// Orphan node (parent not in result set) — treat as root
			roots = append(roots, nodeMap[org.ID])
		}
	}

	// Apply depth limit
	if maxDepth > 0 {
		for _, root := range roots {
			pruneTree(root, maxDepth, 1)
		}
	}

	return roots
}

func pruneTree(node *orgTreeNode, maxDepth, currentDepth int) {
	if currentDepth >= maxDepth {
		node.Children = nil
		return
	}
	for _, child := range node.Children {
		pruneTree(child, maxDepth, currentDepth+1)
	}
}

// ===== Organizations =====

func (s *HTTPServer) handleOrgs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.createOrg(w, r)
	case http.MethodGet:
		s.listOrgs(w, r)
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *HTTPServer) handleOrgByID(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/orgs/")
	// Sub-paths: /api/v1/orgs/{id}/members, /api/v1/orgs/{id}/tree
	parts := strings.SplitN(idStr, "/", 2)
	orgIDStr := parts[0]
	if orgIDStr == "" {
		writeJSONError(w, http.StatusBadRequest, "organization ID is required")
		return
	}
	id, err := uuid.Parse(orgIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid organization ID")
		return
	}

	// Handle sub-paths
	if len(parts) == 2 {
		subPath := parts[1]
		if subPath == "members" {
			s.handleOrgMembers(w, r, id)
			return
		}
		if subPath == "members/import" {
			s.handleMemberImport(w, r, id)
			return
		}
		if subPath == "members/export" {
			s.handleMemberExport(w, r, id)
			return
		}
		if subPath == "roles" {
			s.handleOrgRoles(w, r, id)
			return
		}
		if subPath == "departments/tree" {
			s.handleDeptTree(w, r)
			return
		}
		// Handle members/{userId} and roles/{roleId} sub-paths
		subParts := strings.SplitN(subPath, "/", 2)
		if len(subParts) == 2 {
			if subParts[0] == "members" {
				s.handleOrgMemberByID(w, r, id, subParts[1])
				return
			}
			if subParts[0] == "roles" {
				s.handleOrgRoleByID(w, r, id, subParts[1])
				return
			}
			if subParts[0] == "inherit" {
				s.handleOrgInherit(w, r, id, subParts[1])
				return
			}
			if subParts[0] == "stats" {
				s.handleOrgStats(w, r, id)
				return
			}
			if subParts[0] == "members" && subParts[1] == "bulk" {
				s.handleBulkAddMembers(w, r, id)
				return
			}
			if subParts[0] == "members" && strings.HasPrefix(subParts[1], "bulk-remove") {
				s.handleBulkRemoveMembers(w, r, id)
				return
			}
		}
		if subPath == "tree" {
			s.handleOrgTree(w, r, id)
			return
		}
		if subPath == "subtree" {
			s.handleOrgSubtree(w, r, id)
			return
		}
		if subPath == "restructure" {
			s.handleOrgRestructure(w, r)
			return
		}
		if subPath == "access-matrix" {
			s.handleOrgAccessMatrix(w, r, id)
			return
		}
		writeJSONError(w, http.StatusNotFound, "unknown sub-path")
		return
	}

	switch r.Method {
	case http.MethodGet:
		org, err := s.orgSvc.Get(r.Context(), id)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, orgToJSON(org))
	case http.MethodDelete:
		if err := s.orgSvc.Delete(r.Context(), id); err != nil {
			writeServiceError(w, err)
			return
		}
		s.publishAuditEvent("org.delete", "success", "organization", id, uuid.Nil)
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	case http.MethodPut:
		s.updateOrg(w, r, id)
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *HTTPServer) updateOrg(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	org, err := s.orgSvc.Get(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	if req.Name != "" {
		org.Name = req.Name
	}
	updated, err := s.orgSvc.Update(r.Context(), org)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, orgToJSON(updated))
}

func (s *HTTPServer) handleOrgMembers(w http.ResponseWriter, r *http.Request, orgID uuid.UUID) {
	switch r.Method {
	case http.MethodPost:
		var req struct {
			UserID   string `json:"user_id"`
			TenantID string `json:"tenant_id"`
			Title    string `json:"title"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		userID, err := uuid.Parse(req.UserID)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid user_id")
			return
		}
		tenantID, err := uuid.Parse(req.TenantID)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid tenant_id")
			return
		}
		mem, err := s.memberSvc.Invite(r.Context(), &domain.Membership{
			UserID:   userID,
			TenantID: tenantID,
			OrgID:    orgID,
			Title:    req.Title,
		})
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, membershipToJSON(mem))
	case http.MethodGet:
		tenantIDStr := r.URL.Query().Get("tenant_id")
		if tenantIDStr == "" {
			writeJSONError(w, http.StatusBadRequest, "tenant_id required")
			return
		}
		tid, _ := uuid.Parse(tenantIDStr)
		members, err := s.memberSvc.List(r.Context(), repository.ListMembersFilter{
			TenantID: tid, OrgID: &orgID,
		}, 1, 100)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		result := make([]map[string]any, len(members))
		for i, m := range members {
			result[i] = membershipToJSON(m)
		}
		writeJSON(w, http.StatusOK, map[string]any{"members": result})
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *HTTPServer) handleOrgTree(w http.ResponseWriter, r *http.Request, orgID uuid.UUID) {
	tenantIDStr := r.URL.Query().Get("tenant_id")
	if tenantIDStr == "" {
		writeJSONError(w, http.StatusBadRequest, "tenant_id required")
		return
	}
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid tenant_id")
		return
	}
	subTree, err := s.orgSvc.GetSubTree(r.Context(), tenantID, orgID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	depts, _ := s.deptSvc.ListByOrg(r.Context(), orgID)

	orgs := make([]map[string]any, len(subTree))
	for i, o := range subTree {
		orgs[i] = orgToJSON(o)
	}
	departments := make([]map[string]any, len(depts))
	for i, d := range depts {
		departments[i] = deptToJSON(d)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"organizations": orgs,
		"departments":   departments,
	})
}

// handleOrgSubtree returns all descendants of an org node.
// GET /api/v1/orgs/{id}/subtree
func (s *HTTPServer) handleOrgSubtree(w http.ResponseWriter, r *http.Request, orgID uuid.UUID) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	tenantIDStr := r.URL.Query().Get("tenant_id")
	if tenantIDStr == "" {
		writeJSONError(w, http.StatusBadRequest, "tenant_id required")
		return
	}
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid tenant_id")
		return
	}
	subTree, err := s.orgSvc.GetSubTree(r.Context(), tenantID, orgID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	orgs := make([]map[string]any, len(subTree))
	for i, o := range subTree {
		orgs[i] = orgToJSON(o)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"root_id":       orgID.String(),
		"organizations": orgs,
		"count":         len(orgs),
	})
}

// handleOrgRestructure moved an org node to a new parent.
// Uses existing implementation in restructure_handler.go.

// handleOrgAccessMatrix returns the access matrix for an org node.
// GET /api/v1/orgs/{id}/access-matrix
func (s *HTTPServer) handleOrgAccessMatrix(w http.ResponseWriter, r *http.Request, orgID uuid.UUID) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	org, err := s.orgSvc.Get(r.Context(), orgID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	// Return org info + member count for now. Full matrix needs org_members table.
	orgs, err := s.orgSvc.GetSubTree(r.Context(), org.TenantID, orgID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("access matrix failed: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"org_id":         orgID.String(),
		"tenant_id":      org.TenantID.String(),
		"org_name":       org.Name,
		"subtree_count":  len(orgs),
		"subtree":        orgs,
	})
}

func (s *HTTPServer) createOrg(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TenantID string `json:"tenant_id"`
		ParentID string `json:"parent_id"`
		Name     string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid tenant_id")
		return
	}

	org := &domain.Organization{TenantID: tenantID, Name: req.Name}
	if req.ParentID != "" {
		pid, err := uuid.Parse(req.ParentID)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid parent_id")
			return
		}
		org.ParentID = &pid
	}

	created, err := s.orgSvc.Create(r.Context(), org)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	s.publishAuditEvent("org.create", "success", "organization", created.ID, created.TenantID)
	writeJSON(w, http.StatusCreated, orgToJSON(created))
}

func (s *HTTPServer) listOrgs(w http.ResponseWriter, r *http.Request) {
	tenantIDStr := r.URL.Query().Get("tenant_id")
	if tenantIDStr == "" {
		writeJSONError(w, http.StatusBadRequest, "tenant_id query parameter is required")
		return
	}
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid tenant_id")
		return
	}

	orgs, err := s.orgSvc.List(r.Context(), tenantID, 1, 200)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	result := make([]map[string]any, len(orgs))
	for i, org := range orgs {
		result[i] = orgToJSON(org)
	}
	writeJSON(w, http.StatusOK, map[string]any{"organizations": result})
}

// ===== Departments =====

func (s *HTTPServer) handleDepartments(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.createDept(w, r)
	case http.MethodGet:
		s.listDepts(w, r)
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *HTTPServer) handleDepartmentByID(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/departments/")
	if idStr == "" {
		writeJSONError(w, http.StatusBadRequest, "department ID is required")
		return
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid department ID")
		return
	}

	switch r.Method {
	case http.MethodGet:
		dept, err := s.deptSvc.Get(r.Context(), id)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, deptToJSON(dept))
	case http.MethodPut:
		s.updateDept(w, r, id)
	case http.MethodDelete:
		if err := s.deptSvc.Delete(r.Context(), id); err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *HTTPServer) createDept(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OrgID     string `json:"org_id"`
		ParentID  string `json:"parent_id"`
		Name      string `json:"name"`
		ManagerID string `json:"manager_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	orgID, err := uuid.Parse(req.OrgID)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid org_id")
		return
	}

	dept := &domain.Department{OrgID: orgID, Name: req.Name}
	if req.ParentID != "" {
		pid, err := uuid.Parse(req.ParentID)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid parent_id")
			return
		}
		dept.ParentID = &pid
	}
	if req.ManagerID != "" {
		mid, err := uuid.Parse(req.ManagerID)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid manager_id")
			return
		}
		dept.ManagerID = &mid
	}

	created, err := s.deptSvc.Create(r.Context(), dept)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, deptToJSON(created))
}

func (s *HTTPServer) listDepts(w http.ResponseWriter, r *http.Request) {
	orgIDStr := r.URL.Query().Get("org_id")
	if orgIDStr == "" {
		writeJSONError(w, http.StatusBadRequest, "org_id query parameter is required")
		return
	}
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid org_id")
		return
	}

	depts, err := s.deptSvc.ListByOrg(r.Context(), orgID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	result := make([]map[string]any, len(depts))
	for i, d := range depts {
		result[i] = deptToJSON(d)
	}
	writeJSON(w, http.StatusOK, map[string]any{"departments": result})
}

// ===== Teams =====

func (s *HTTPServer) handleTeams(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.createTeam(w, r)
	case http.MethodGet:
		s.listTeams(w, r)
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *HTTPServer) handleTeamByID(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/teams/")
	if idStr == "" {
		writeJSONError(w, http.StatusBadRequest, "team ID is required")
		return
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid team ID")
		return
	}

	switch r.Method {
	case http.MethodGet:
		team, err := s.teamSvc.Get(r.Context(), id)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, teamToJSON(team))
	case http.MethodPut:
		s.updateTeam(w, r, id)
	case http.MethodDelete:
		if err := s.teamSvc.Delete(r.Context(), id); err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *HTTPServer) createTeam(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OrgID       string `json:"org_id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		CreatedBy   string `json:"created_by"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	orgID, err := uuid.Parse(req.OrgID)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid org_id")
		return
	}
	createdBy, err := uuid.Parse(req.CreatedBy)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid created_by")
		return
	}

	created, err := s.teamSvc.Create(r.Context(), &domain.Team{
		OrgID:       orgID,
		Name:        req.Name,
		Description: req.Description,
		CreatedBy:   createdBy,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, teamToJSON(created))
}

func (s *HTTPServer) listTeams(w http.ResponseWriter, r *http.Request) {
	orgIDStr := r.URL.Query().Get("org_id")
	if orgIDStr == "" {
		writeJSONError(w, http.StatusBadRequest, "org_id query parameter is required")
		return
	}
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid org_id")
		return
	}

	teams, err := s.teamSvc.List(r.Context(), orgID, 1, 100)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	result := make([]map[string]any, len(teams))
	for i, t := range teams {
		result[i] = teamToJSON(t)
	}
	writeJSON(w, http.StatusOK, map[string]any{"teams": result})
}

func (s *HTTPServer) updateDept(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	var req struct {
		Name      string `json:"name"`
		ManagerID string `json:"manager_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	dept, err := s.deptSvc.Get(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	if req.Name != "" {
		dept.Name = req.Name
	}
	if req.ManagerID != "" {
		mid, err := uuid.Parse(req.ManagerID)
		if err == nil {
			dept.ManagerID = &mid
		}
	}
	updated, err := s.deptSvc.Update(r.Context(), dept)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, deptToJSON(updated))
}

func (s *HTTPServer) updateTeam(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	team, err := s.teamSvc.Get(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	if req.Name != "" {
		team.Name = req.Name
	}
	if req.Description != "" {
		team.Description = req.Description
	}
	updated, err := s.teamSvc.Update(r.Context(), team)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, teamToJSON(updated))
}

func membershipToJSON(m *domain.Membership) map[string]any {
	result := map[string]any{
		"id":        m.ID.String(),
		"user_id":   m.UserID.String(),
		"tenant_id": m.TenantID.String(),
		"org_id":    m.OrgID.String(),
		"status":    string(m.Status),
		"title":     m.Title,
	}
	if m.DeptID != nil {
		result["dept_id"] = m.DeptID.String()
	}
	if m.TeamID != nil {
		result["team_id"] = m.TeamID.String()
	}
	return result
}

// ===== JSON converters =====

func orgToJSON(o *domain.Organization) map[string]any {
	m := map[string]any{
		"id":        o.ID.String(),
		"tenant_id": o.TenantID.String(),
		"name":      o.Name,
		"path":      o.Path,
	}
	if o.ParentID != nil {
		m["parent_id"] = o.ParentID.String()
	}
	return m
}

func deptToJSON(d *domain.Department) map[string]any {
	m := map[string]any{
		"id":     d.ID.String(),
		"org_id": d.OrgID.String(),
		"name":   d.Name,
		"path":   d.Path,
	}
	if d.ParentID != nil {
		m["parent_id"] = d.ParentID.String()
	}
	if d.ManagerID != nil {
		m["manager_id"] = d.ManagerID.String()
	}
	return m
}

func teamToJSON(t *domain.Team) map[string]any {
	return map[string]any{
		"id":          t.ID.String(),
		"org_id":      t.OrgID.String(),
		"name":        t.Name,
		"description": t.Description,
		"created_by":  t.CreatedBy.String(),
	}
}

// ===== Org Role Assignment (in-memory) =====
//
// POST   /api/v1/orgs/{id}/roles        — assign role to org
// GET    /api/v1/orgs/{id}/roles        — list org roles
// DELETE /api/v1/orgs/{id}/roles/{roleId} — remove role from org

var orgRoles = struct {
	sync.RWMutex
	data map[uuid.UUID][]string // orgID -> []roleID
}{data: make(map[uuid.UUID][]string)}

func (s *HTTPServer) handleOrgRoles(w http.ResponseWriter, r *http.Request, orgID uuid.UUID) {
	switch r.Method {
	case http.MethodPost:
		var req struct {
			RoleID string `json:"role_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if req.RoleID == "" {
			writeJSONError(w, http.StatusBadRequest, "role_id is required")
			return
		}

		orgRoles.Lock()
		// Check for duplicate
		for _, existing := range orgRoles.data[orgID] {
			if existing == req.RoleID {
				orgRoles.Unlock()
				writeJSONError(w, http.StatusConflict, "role already assigned to this organization")
				return
			}
		}
		orgRoles.data[orgID] = append(orgRoles.data[orgID], req.RoleID)
		orgRoles.Unlock()

		writeJSON(w, http.StatusCreated, map[string]any{
			"status":  "assigned",
			"role_id": req.RoleID,
			"org_id":  orgID.String(),
		})

	case http.MethodGet:
		orgRoles.RLock()
		roleIDs := orgRoles.data[orgID]
		orgRoles.RUnlock()

		// Return a copy
		result := make([]string, len(roleIDs))
		copy(result, roleIDs)

		writeJSON(w, http.StatusOK, map[string]any{
			"org_id":   orgID.String(),
			"roles":    result,
			"count":    len(result),
		})

	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *HTTPServer) handleOrgRoleByID(w http.ResponseWriter, r *http.Request, orgID uuid.UUID, roleIDStr string) {
	if r.Method != http.MethodDelete {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	orgRoles.Lock()
	roles := orgRoles.data[orgID]
	found := false
	for i, rid := range roles {
		if rid == roleIDStr {
			orgRoles.data[orgID] = append(roles[:i], roles[i+1:]...)
			found = true
			break
		}
	}
	orgRoles.Unlock()

	if !found {
		writeJSONError(w, http.StatusNotFound, "role not assigned to this organization")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "removed",
		"role_id": roleIDStr,
		"org_id":  orgID.String(),
	})
}

// handleOrgMemberByID handles DELETE /api/v1/orgs/{id}/members/{userId}
func (s *HTTPServer) handleOrgMemberByID(w http.ResponseWriter, r *http.Request, orgID uuid.UUID, userIDStr string) {
	if r.Method != http.MethodDelete {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	tenantIDStr := r.URL.Query().Get("tenant_id")
	tenantID, _ := uuid.Parse(tenantIDStr)

	// Find and remove the membership
	members, err := s.memberSvc.List(r.Context(), repository.ListMembersFilter{
		TenantID: tenantID, OrgID: &orgID,
	}, 1, 500)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	for _, m := range members {
		if m.UserID == userID {
			if err := s.memberSvc.Remove(r.Context(), m.ID); err != nil {
				writeServiceError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{
				"status":  "removed",
				"user_id": userIDStr,
				"org_id":  orgID.String(),
			})
			return
		}
	}

	writeJSONError(w, http.StatusNotFound, "member not found in this organization")
}

// GET /api/v1/orgs/{id}/stats — returns org statistics
func (s *HTTPServer) handleOrgStats(w http.ResponseWriter, r *http.Request, orgID uuid.UUID) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	tenantIDStr := r.URL.Query().Get("tenant_id")
	tenantID, _ := uuid.Parse(tenantIDStr)

	// Get org
	org, err := s.orgSvc.Get(r.Context(), orgID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	// Count members
	members, _ := s.memberSvc.List(r.Context(), repository.ListMembersFilter{
		TenantID: tenantID, OrgID: &orgID,
	}, 1, 1000)

	// Count child orgs
	allOrgs, _ := s.orgSvc.List(r.Context(), tenantID, 1, 500)
	childCount := 0
	for _, o := range allOrgs {
		if o.ParentID != nil && *o.ParentID == orgID {
			childCount++
		}
	}

	// Get role count from in-memory store
	orgRoles.RLock()
	roleCount := len(orgRoles.data[orgID])
	orgRoles.RUnlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"org_id":           orgID.String(),
		"org_name":         org.Name,
		"member_count":     len(members),
		"child_org_count":  childCount,
		"role_count":       roleCount,
		"path":             org.Path,
		"parent_id":        org.ParentID,
	})
}

// POST /api/v1/orgs/{id}/members/bulk — bulk add members
func (s *HTTPServer) handleBulkAddMembers(w http.ResponseWriter, r *http.Request, orgID uuid.UUID) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		TenantID string   `json:"tenant_id"`
		UserIDs  []string `json:"user_ids"`
		Role     string   `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if len(req.UserIDs) == 0 {
		writeJSONError(w, http.StatusBadRequest, "user_ids is required")
		return
	}

	tenantID, _ := uuid.Parse(req.TenantID)
	if req.Role == "" {
		req.Role = "member"
	}

	added := 0
	errors := []map[string]any{}
	for _, uidStr := range req.UserIDs {
		uid, err := uuid.Parse(uidStr)
		if err != nil {
			errors = append(errors, map[string]any{"user_id": uidStr, "error": "invalid UUID"})
			continue
		}
		membership := &domain.Membership{
			ID:       uuid.New(),
			TenantID: tenantID,
			OrgID:    orgID,
			UserID:   uid,
			Title:    req.Role,
			Status:   domain.MembershipActive,
		}
		if _, err := s.memberSvc.Invite(r.Context(), membership); err != nil {
			errors = append(errors, map[string]any{"user_id": uidStr, "error": err.Error()})
			continue
		}
		added++
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"status":       "completed",
		"added":        added,
		"failed":       len(errors),
		"errors":       errors,
		"total_requested": len(req.UserIDs),
	})
}

// POST /api/v1/orgs/{id}/members/bulk-remove — bulk remove members
func (s *HTTPServer) handleBulkRemoveMembers(w http.ResponseWriter, r *http.Request, orgID uuid.UUID) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		TenantID string   `json:"tenant_id"`
		UserIDs  []string `json:"user_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if len(req.UserIDs) == 0 {
		writeJSONError(w, http.StatusBadRequest, "user_ids is required")
		return
	}

	tenantID, _ := uuid.Parse(req.TenantID)

	// Get all members for this org
	members, _ := s.memberSvc.List(r.Context(), repository.ListMembersFilter{
		TenantID: tenantID, OrgID: &orgID,
	}, 1, 1000)

	userIDSet := map[string]bool{}
	for _, uid := range req.UserIDs {
		userIDSet[uid] = true
	}

	removed := 0
	for _, m := range members {
		if userIDSet[m.UserID.String()] {
			if err := s.memberSvc.Remove(r.Context(), m.ID); err == nil {
				removed++
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":         "completed",
		"removed":        removed,
		"total_requested": len(req.UserIDs),
	})
}

// ===== Org Role Inheritance =====
//
// POST /api/v1/orgs/{id}/inherit/{parentId} — set parent org for role inheritance
// GET  /api/v1/orgs/{id}/inherit  — get inheritance config

var orgInheritance = struct {
	sync.RWMutex
	data map[uuid.UUID]uuid.UUID // childOrgID -> parentOrgID
}{data: make(map[uuid.UUID]uuid.UUID)}

func (s *HTTPServer) handleOrgInherit(w http.ResponseWriter, r *http.Request, orgID uuid.UUID, parentIDStr string) {
	switch r.Method {
	case http.MethodPost:
		parentID, err := uuid.Parse(parentIDStr)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid parent org ID")
			return
		}
		if parentID == orgID {
			writeJSONError(w, http.StatusBadRequest, "cannot inherit from self")
			return
		}

		// Check for cycles: walk up the chain
		orgInheritance.RLock()
		visited := map[uuid.UUID]bool{orgID: true}
		cur := parentID
		for i := 0; i < 100; i++ {
			if visited[cur] {
				orgInheritance.RUnlock()
				writeJSONError(w, http.StatusBadRequest, "inheritance cycle detected")
				return
			}
			visited[cur] = true
			next, ok := orgInheritance.data[cur]
			if !ok {
				break
			}
			cur = next
		}
		orgInheritance.RUnlock()

		// Set inheritance
		orgInheritance.Lock()
		orgInheritance.data[orgID] = parentID
		orgInheritance.Unlock()

		// Merge parent roles into child
		orgRoles.Lock()
		childRoles := orgRoles.data[orgID]
		parentRoles := orgRoles.data[parentID]
		for _, prid := range parentRoles {
			found := false
			for _, crid := range childRoles {
				if crid == prid {
					found = true
					break
				}
			}
			if !found {
				childRoles = append(childRoles, prid)
			}
		}
		orgRoles.data[orgID] = childRoles
		orgRoles.Unlock()

		writeJSON(w, http.StatusOK, map[string]any{
			"status":     "inheriting",
			"org_id":     orgID.String(),
			"parent_id":  parentID.String(),
			"merged_roles": len(parentRoles),
		})

	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// ===== Membership CSV Import/Export =====
//
// POST /api/v1/orgs/{id}/members/import  — bulk import from CSV
// GET  /api/v1/orgs/{id}/members/export  — export as CSV or JSON

func (s *HTTPServer) handleMemberImport(w http.ResponseWriter, r *http.Request, orgID uuid.UUID) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	tenantIDStr := r.URL.Query().Get("tenant_id")
	if tenantIDStr == "" {
		writeJSONError(w, http.StatusBadRequest, "tenant_id query parameter is required")
		return
	}
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid tenant_id")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	reader := csv.NewReader(strings.NewReader(string(body)))
	records, err := reader.ReadAll()
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "failed to parse CSV: "+err.Error())
		return
	}

	if len(records) == 0 {
		writeJSONError(w, http.StatusBadRequest, "CSV is empty")
		return
	}

	// First row is header: user_id, title (optional), dept_id (optional), team_id (optional)
	header := records[0]
	headerMap := make(map[string]int)
	for i, col := range header {
		headerMap[strings.ToLower(strings.TrimSpace(col))] = i
	}

	userIdx, hasUser := headerMap["user_id"]
	if !hasUser {
		writeJSONError(w, http.StatusBadRequest, "CSV must have a user_id column")
		return
	}

	titleIdx, hasTitle := headerMap["title"]
	_ = hasTitle

	imported := 0
	var importErrors []map[string]any
	for lineNo, record := range records[1:] {
		if len(record) <= userIdx {
			importErrors = append(importErrors, map[string]any{"line": lineNo + 2, "error": "missing user_id"})
			continue
		}
		uidStr := strings.TrimSpace(record[userIdx])
		if uidStr == "" {
			continue
		}
		uid, err := uuid.Parse(uidStr)
		if err != nil {
			importErrors = append(importErrors, map[string]any{"line": lineNo + 2, "user_id": uidStr, "error": "invalid UUID"})
			continue
		}
		title := "member"
		if hasTitle && titleIdx < len(record) {
			t := strings.TrimSpace(record[titleIdx])
			if t != "" {
				title = t
			}
		}
		membership := &domain.Membership{
			ID:       uuid.New(),
			TenantID: tenantID,
			OrgID:    orgID,
			UserID:   uid,
			Title:    title,
			Status:   domain.MembershipActive,
		}
		if _, err := s.memberSvc.Invite(r.Context(), membership); err != nil {
			importErrors = append(importErrors, map[string]any{"line": lineNo + 2, "user_id": uidStr, "error": err.Error()})
			continue
		}
		imported++
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":   "completed",
		"imported": imported,
		"failed":   len(importErrors),
		"errors":   importErrors,
		"total":    len(records) - 1,
	})
}

func (s *HTTPServer) handleMemberExport(w http.ResponseWriter, r *http.Request, orgID uuid.UUID) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	tenantIDStr := r.URL.Query().Get("tenant_id")
	if tenantIDStr == "" {
		writeJSONError(w, http.StatusBadRequest, "tenant_id query parameter is required")
		return
	}
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid tenant_id")
		return
	}

	members, err := s.memberSvc.List(r.Context(), repository.ListMembersFilter{
		TenantID: tenantID, OrgID: &orgID,
	}, 1, 10000)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	format := r.URL.Query().Get("format")
	if format == "" {
		format = "csv"
	}

	if format == "json" {
		result := make([]map[string]any, len(members))
		for i, m := range members {
			result[i] = membershipToJSON(m)
		}
		writeJSON(w, http.StatusOK, map[string]any{"members": result})
		return
	}

	// CSV export
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="org-%s-members.csv"`, orgID.String()))
	writer := csv.NewWriter(w)
	writer.Write([]string{"user_id", "tenant_id", "org_id", "title", "status"})
	for _, m := range members {
		writer.Write([]string{
			m.UserID.String(),
			m.TenantID.String(),
			m.OrgID.String(),
			m.Title,
			string(m.Status),
		})
	}
	writer.Flush()
}

// ===== Helpers =====

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	errors.WriteSimpleAPIError(w, status, httpStatusToCode(status), msg)
}

func writeServiceError(w http.ResponseWriter, err error) {
	errors.WriteAPIError(w, err, "")
}

// httpStatusToCode maps an HTTP status code to a GGID error code string.
func httpStatusToCode(status int) string {
	switch status {
	case http.StatusBadRequest:
		return string(errors.ErrInvalidArgument)
	case http.StatusUnauthorized:
		return string(errors.ErrUnauthenticated)
	case http.StatusForbidden:
		return string(errors.ErrPermissionDenied)
	case http.StatusNotFound:
		return string(errors.ErrNotFound)
	case http.StatusConflict:
		return string(errors.ErrAlreadyExists)
	case http.StatusTooManyRequests:
		return string(errors.ErrResourceExhausted)
	default:
		return string(errors.ErrInternal)
	}
}
