// Package httpserver provides REST API endpoints for the Org Service.
// These endpoints allow the Admin Console to manage organizations, departments,
// teams, and memberships via HTTP through the API Gateway.
package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

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
	mux.HandleFunc("/api/v1/orgs/", s.handleOrgByID)
	// Departments
	mux.HandleFunc("/api/v1/departments", s.handleDepartments)
	mux.HandleFunc("/api/v1/departments/", s.handleDepartmentByID)
	// Teams
	mux.HandleFunc("/api/v1/teams", s.handleTeams)
	mux.HandleFunc("/api/v1/teams/", s.handleTeamByID)
	// Memberships
	mux.HandleFunc("/api/v1/memberships", s.handleMemberships)
}

// ======================= Organizations =======================

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
	if idStr == "" {
		writeJSONError(w, http.StatusBadRequest, "organization ID is required")
		return
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid organization ID")
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
	case http.MethodPut, http.MethodPatch:
		var req struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		org := &domain.Organization{ID: id, Name: req.Name}
		updated, err := s.orgSvc.Update(r.Context(), org)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, orgToJSON(updated))
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
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

	orgs, err := s.orgSvc.List(r.Context(), tenantID, 1, 100)
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

// ======================= Departments =======================

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
	for i, dept := range depts {
		result[i] = deptToJSON(dept)
	}
	writeJSON(w, http.StatusOK, map[string]any{"departments": result})
}

// ======================= Teams =======================

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

	team := &domain.Team{
		OrgID:       orgID,
		Name:        req.Name,
		Description: req.Description,
		CreatedBy:   createdBy,
	}
	created, err := s.teamSvc.Create(r.Context(), team)
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
	for i, team := range teams {
		result[i] = teamToJSON(team)
	}
	writeJSON(w, http.StatusOK, map[string]any{"teams": result})
}

// ======================= Memberships =======================

func (s *HTTPServer) handleMemberships(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.inviteMember(w, r)
	case http.MethodGet:
		s.listMembers(w, r)
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *HTTPServer) inviteMember(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID   string `json:"user_id"`
		TenantID string `json:"tenant_id"`
		OrgID    string `json:"org_id"`
		DeptID   string `json:"dept_id"`
		TeamID   string `json:"team_id"`
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
	orgID, err := uuid.Parse(req.OrgID)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid org_id")
		return
	}

	m := &domain.Membership{
		UserID:   userID,
		TenantID: tenantID,
		OrgID:    orgID,
		Title:    req.Title,
	}
	if req.DeptID != "" {
		did, err := uuid.Parse(req.DeptID)
		if err == nil {
			m.DeptID = &did
		}
	}
	if req.TeamID != "" {
		tid, err := uuid.Parse(req.TeamID)
		if err == nil {
			m.TeamID = &tid
		}
	}

	created, err := s.memberSvc.Invite(r.Context(), m)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, memberToJSON(created))
}

func (s *HTTPServer) listMembers(w http.ResponseWriter, r *http.Request) {
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

	filter := repository.ListMembersFilter{TenantID: tenantID}
	if orgIDStr := r.URL.Query().Get("org_id"); orgIDStr != "" {
		orgID, err := uuid.Parse(orgIDStr)
		if err == nil {
			filter.OrgID = &orgID
		}
	}
	if status := r.URL.Query().Get("status"); status != "" {
		filter.Status = domain.MembershipStatus(status)
	}

	members, err := s.memberSvc.List(r.Context(), filter, 1, 100)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	result := make([]map[string]any, len(members))
	for i, m := range members {
		result[i] = memberToJSON(m)
	}
	writeJSON(w, http.StatusOK, map[string]any{"memberships": result})
}

// ======================= JSON Helpers =======================

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

func memberToJSON(m *domain.Membership) map[string]any {
	result := map[string]any{
		"id":        m.ID.String(),
		"user_id":   m.UserID.String(),
		"tenant_id": m.TenantID.String(),
		"org_id":    m.OrgID.String(),
		"title":     m.Title,
		"status":    string(m.Status),
	}
	if m.DeptID != nil {
		result["dept_id"] = m.DeptID.String()
	}
	if m.TeamID != nil {
		result["team_id"] = m.TeamID.String()
	}
	return result
}

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
		default:
			writeJSONError(w, http.StatusInternalServerError, ge.Message)
		}
		return
	}
	writeJSONError(w, http.StatusInternalServerError, err.Error())
}
