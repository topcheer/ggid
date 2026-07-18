// Package httpserver OpenAPI annotations for the org service.
// These comments are consumed by swaggo/swag to generate OpenAPI documentation.
// To regenerate: swag init -g services/org/internal/server/http.go
package httpserver

// --- Org: Organizations ---

// CreateOrg godoc
// @Summary Create an organization
// @Description Create a new organization (top-level entity) with name, description, and settings.
// @Tags organizations
// @Accept json
// @Produce json
// @Param request body object true "Org creation request {name, description, settings}"
// @Success 201 {object} map[string]any "Organization created"
// @Failure 400 {object} map[string]string "invalid request body"
// @Router /api/v1/orgs [post]

// ListOrgs godoc
// @Summary List organizations
// @Description List all organizations accessible to the caller with optional filtering.
// @Tags organizations
// @Produce json
// @Success 200 {object} map[string]any "Organization list"
// @Router /api/v1/orgs [get]

// GetOrgByID godoc
// @Summary Get organization by ID
// @Description Retrieve a single organization with its full tree structure.
// @Tags organizations
// @Produce json
// @Param id path string true "Organization ID"
// @Success 200 {object} map[string]any "Organization detail with tree"
// @Failure 404 {object} map[string]string "Organization not found"
// @Router /api/v1/orgs/{id} [get]

// UpdateOrg godoc
// @Summary Update an organization
// @Description Update organization name, description, or settings.
// @Tags organizations
// @Accept json
// @Produce json
// @Param id path string true "Organization ID"
// @Param request body object true "Update fields"
// @Success 200 {object} map[string]any "Organization updated"
// @Router /api/v1/orgs/{id} [put]

// DeleteOrg godoc
// @Summary Delete an organization
// @Description Delete an organization and cascade to all child departments and teams.
// @Tags organizations
// @Produce json
// @Param id path string true "Organization ID"
// @Success 200 {object} map[string]string "Organization deleted"
// @Router /api/v1/orgs/{id} [delete]

// --- Org: Departments ---

// CreateDepartment godoc
// @Summary Create a department
// @Description Create a department under a parent organization or department.
// @Tags departments
// @Accept json
// @Produce json
// @Param request body object true "Department creation {name, parent_id, manager_id}"
// @Success 201 {object} map[string]any "Department created"
// @Router /api/v1/departments [post]

// ListDepartments godoc
// @Summary List departments
// @Description List departments with optional filtering by parent organization.
// @Tags departments
// @Produce json
// @Param org_id query string false "Filter by organization ID"
// @Success 200 {object} map[string]any "Department list"
// @Router /api/v1/departments [get]

// --- Org: Teams ---

// CreateTeam godoc
// @Summary Create a team
// @Description Create a team within a department or organization. Teams group users for collaboration.
// @Tags teams
// @Accept json
// @Produce json
// @Param request body object true "Team creation {name, department_id, description}"
// @Success 201 {object} map[string]any "Team created"
// @Router /api/v1/teams [post]

// ListTeams godoc
// @Summary List teams
// @Description List all teams with optional filtering by department or organization.
// @Tags teams
// @Produce json
// @Param department_id query string false "Filter by department ID"
// @Success 200 {object} map[string]any "Team list"
// @Router /api/v1/teams [get]

// --- Org: Members ---

// ListMembers godoc
// @Summary List organization members
// @Description List all members across an organization with role and department info.
// @Tags members
// @Produce json
// @Param org_id path string true "Organization ID"
// @Param department_id query string false "Filter by department"
// @Success 200 {object} map[string]any "Member list with roles"
// @Router /api/v1/orgs/{id}/members [get]
