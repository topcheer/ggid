// Package httpserver OpenAPI annotations for the policy service.
// These comments are consumed by swaggo/swag to generate OpenAPI documentation.
// To regenerate: swag init -g services/policy/internal/server/http.go
package httpserver

// --- Policy: Roles ---

// CreateRole godoc
// @Summary Create a role
// @Description Create a new role with optional parent for role hierarchy. Supports scopes, permissions, and tenant isolation.
// @Tags roles
// @Accept json
// @Produce json
// @Param request body object true "Role creation request {name, description, scopes[], parent_id}"
// @Success 201 {object} map[string]any "Role created"
// @Failure 400 {object} map[string]string "invalid request body"
// @Router /api/v1/roles [post]

// ListRoles godoc
// @Summary List roles
// @Description List all roles for the tenant with optional pagination and scope filtering.
// @Tags roles
// @Produce json
// @Param page query int false "Page number (default 1)"
// @Param page_size query int false "Page size (default 20)"
// @Param scope query string false "Filter by scope"
// @Success 200 {object} map[string]any "Paginated role list"
// @Router /api/v1/roles [get]

// GetRoleByID godoc
// @Summary Get role by ID
// @Description Retrieve a single role with its full permission set and child roles.
// @Tags roles
// @Produce json
// @Param id path string true "Role ID"
// @Success 200 {object} map[string]any "Role detail"
// @Failure 404 {object} map[string]string "Role not found"
// @Router /api/v1/roles/{id} [get]

// UpdateRole godoc
// @Summary Update a role
// @Description Update role name, description, or scopes. Changes propagate to all assigned users.
// @Tags roles
// @Accept json
// @Produce json
// @Param id path string true "Role ID"
// @Param request body object true "Role update fields"
// @Success 200 {object} map[string]any "Role updated"
// @Router /api/v1/roles/{id} [put]

// DeleteRole godoc
// @Summary Delete a role
// @Description Delete a role and unassign it from all users. Child roles are reassigned to the parent.
// @Tags roles
// @Produce json
// @Param id path string true "Role ID"
// @Success 200 {object} map[string]string "Role deleted"
// @Router /api/v1/roles/{id} [delete]

// --- Policy: Permissions ---

// ListPermissions godoc
// @Summary List permissions
// @Description List all defined permissions with optional resource/action filtering.
// @Tags permissions
// @Produce json
// @Param resource query string false "Filter by resource type"
// @Success 200 {object} map[string]any "Permission list"
// @Router /api/v1/permissions [get]

// CreatePermission godoc
// @Summary Create a permission
// @Description Define a new permission with resource, action, and optional conditions.
// @Tags permissions
// @Accept json
// @Produce json
// @Param request body object true "Permission definition {resource, action, conditions}"
// @Success 201 {object} map[string]any "Permission created"
// @Router /api/v1/permissions [post]

// --- Policy: Policies ---

// CreatePolicy godoc
// @Summary Create a policy
// @Description Create an ABAC/RBAC policy with effect (allow/deny), conditions, and resource patterns.
// @Tags policies
// @Accept json
// @Produce json
// @Param request body object true "Policy definition {name, effect, resources[], actions[], conditions}"
// @Success 201 {object} map[string]any "Policy created"
// @Router /api/v1/policies [post]

// ListPolicies godoc
// @Summary List policies
// @Description List all policies for the tenant with optional status filtering.
// @Tags policies
// @Produce json
// @Param status query string false "Filter by enabled/disabled"
// @Success 200 {object} map[string]any "Policy list"
// @Router /api/v1/policies [get]

// --- Policy: Check & Evaluate ---

// Check godoc
// @Summary Check access
// @Description Evaluate whether a principal has permission for a specific resource+action. Returns allow/deny.
// @Tags policies
// @Accept json
// @Produce json
// @Param request body object true "Check request {principal_id, resource, action}"
// @Success 200 {object} map[string]any "{allowed: bool, policy_id: string}"
// @Router /api/v1/policies/check [post]

// Evaluate godoc
// @Summary Evaluate policies
// @Description Evaluate all matching policies for a principal against a resource set. Returns detailed decision trail.
// @Tags policies
// @Accept json
// @Produce json
// @Param request body object true "Evaluate request {principal_id, resources[], actions[]}"
// @Success 200 {object} map[string]any "Evaluation results with decision trail"
// @Router /api/v1/policies/evaluate [post]

// --- Policy: SoD ---

// SoDCheck godoc
// @Summary Check Separation of Duties
// @Description Check whether a role assignment creates a SoD violation against configured SoD rules.
// @Tags sod
// @Accept json
// @Produce json
// @Param request body object true "SoD check request {user_id, role_id}"
// @Success 200 {object} map[string]any "{violated: bool, conflicts: []}"
// @Router /api/v1/policies/sod/check [post]
