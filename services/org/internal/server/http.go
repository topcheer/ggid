// Package httpserver provides REST API endpoints for the Org Service.
// These endpoints allow the Admin Console to manage organizations,
// departments, teams, and memberships via HTTP through the API Gateway.
package httpserver

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/ggid/ggid/pkg/errors"
	"github.com/ggid/ggid/services/org/internal/domain"
	"github.com/ggid/ggid/services/org/internal/repository"
	"github.com/ggid/ggid/services/org/internal/service"
	"github.com/google/uuid"
)

// HTTPServer exposes the Org Service as a REST API.
type HTTPServer struct {
	orgSvc    *service.OrgService
	deptSvc   *service.DeptService
	teamSvc   *service.TeamService
	memberSvc *service.MembershipService
}

// NewHTTPServer creates a new Org Service HTTP server.
func NewHTTPServer(orgSvc *service.OrgService, deptSvc *service.DeptService, teamSvc *service.TeamService, memberSvc *service.MembershipService) *HTTPServer {
	return &HTTPServer{orgSvc: orgSvc, deptSvc: deptSvc, teamSvc: teamSvc, memberSvc: memberSvc}
}

// RegisterRoutes registers all Org Service HTTP routes on the given mux.
func (s *HTTPServer) RegisterRoutes(mux *http.ServeMux) {
	// Organizations
	mux.HandleFunc("/api/v1/orgs", s.handleOrgs)
	mux.HandleFunc("/api/v1/orgs/tree", s.handleFullTree)
	mux.HandleFunc("/api/v1/orgs/", s.handleOrgByID)
	// Departments
	mux.HandleFunc("/api/v1/departments", s.handleDepartments)
	mux.HandleFunc("/api/v1/departments/", s.handleDepartmentByID)
	// Teams
	mux.HandleFunc("/api/v1/teams", s.handleTeams)
	mux.HandleFunc("/api/v1/teams/", s.handleTeamByID)
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
		if subPath == "roles" {
			s.handleOrgRoles(w, r, id)
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
		}
		if subPath == "tree" {
			s.handleOrgTree(w, r, id)
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

// ===== Helpers =====

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func writeServiceError(w http.ResponseWriter, err error) {
	if ge, ok := errors.AsGGIDError(err); ok {
		switch ge.Code {
		case errors.ErrNotFound:
			writeJSONError(w, http.StatusNotFound, ge.Message)
		case errors.ErrAlreadyExists:
			writeJSONError(w, http.StatusConflict, ge.Message)
		case errors.ErrInvalidArgument:
			writeJSONError(w, http.StatusBadRequest, ge.Message)
		case errors.ErrPermissionDenied:
			writeJSONError(w, http.StatusForbidden, ge.Message)
		default:
			writeJSONError(w, http.StatusInternalServerError, ge.Message)
		}
		return
	}
	writeJSONError(w, http.StatusInternalServerError, err.Error())
}
