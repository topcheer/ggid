package server

// This file contains swaggo/swag v2 OpenAPI annotations for the identity service's
// top 20 endpoints. The annotations are package-level doc comments above empty type
// declarations; they do NOT change handler logic or existing files.

// ------------------------------------------------------------------------------
// Request / response type definitions referenced by the annotations below.
// ------------------------------------------------------------------------------

// CreateUserRequest is the body for POST /api/v1/users.
type CreateUserRequest struct {
	Username    string `json:"username" example:"alice"`
	Email       string `json:"email" example:"alice@example.com"`
	Password    string `json:"password" example:"s3cret!"`
	Phone       string `json:"phone,omitempty" example:"+1-555-0100"`
	DisplayName string `json:"display_name,omitempty" example:"Alice"`
	Locale      string `json:"locale,omitempty" example:"en-US"`
	Timezone    string `json:"timezone,omitempty" example:"America/New_York"`
}

// UpdateUserRequest is the body for PATCH /api/v1/users/{id}.
type UpdateUserRequest struct {
	Phone       *string `json:"phone,omitempty"`
	DisplayName *string `json:"display_name,omitempty"`
	Locale      *string `json:"locale,omitempty"`
	Timezone    *string `json:"timezone,omitempty"`
}

// UserResponse is the JSON representation of a user returned by the API.
type UserResponse struct {
	ID            string `json:"id"`
	TenantID      string `json:"tenant_id"`
	Username      string `json:"username"`
	Email         string `json:"email"`
	Phone         string `json:"phone"`
	Status        string `json:"status"`
	EmailVerified bool   `json:"email_verified"`
	DisplayName   string `json:"display_name"`
	Locale        string `json:"locale"`
	Timezone      string `json:"timezone"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

// ListUsersResponse is the body for GET /api/v1/users.
type ListUsersResponse struct {
	Users      []UserResponse `json:"users"`
	Total      int            `json:"total"`
	NextOffset int            `json:"next_offset"`
}

// SearchUsersResponse is the body for GET /api/v1/users/search.
type SearchUsersResponse struct {
	Users  []map[string]any `json:"users"`
	Count  int              `json:"count"`
	Total  int              `json:"total"`
	Limit  int              `json:"limit"`
	Offset int              `json:"offset"`
}

// ErrorResponse is the standard error envelope.
type ErrorResponse struct {
	Error string `json:"error"`
}

// AssignRoleRequest is the body for POST /api/v1/users/{id}/roles.
type AssignRoleRequest struct {
	RoleID   string `json:"role_id"`
	RoleName string `json:"role_name,omitempty"`
}

// CreateSCIMTokenRequest is the body for POST /api/v1/identity/scim/tokens.
type CreateSCIMTokenRequest struct {
	Name string `json:"name"`
}

// CreateSCIMTokenResponse is returned once after creating a SCIM token.
type CreateSCIMTokenResponse struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Token     string   `json:"token"`
	Scopes    []string `json:"scopes"`
	CreatedAt string   `json:"created_at"`
	Message   string   `json:"message"`
}

// ListSCIMTokensResponse is the body for GET /api/v1/identity/scim/tokens.
type ListSCIMTokensResponse struct {
	Tokens []map[string]any `json:"tokens"`
	Total  int              `json:"total"`
}

// SCIMPatchRequest is the body for PATCH /api/v1/scim/Groups/{id} (RFC 7644).
type SCIMPatchRequest struct {
	Operations []SCIMPatchOperation `json:"Operations"`
}

// SCIMPatchOperation is a single SCIM PATCH operation.
type SCIMPatchOperation struct {
	Op    string `json:"op" example:"add"`
	Path  string `json:"path" example:"members"`
	Value any    `json:"value"`
}

// SCIMGroupMemberValue is a member reference used in SCIM PATCH add/remove values.
type SCIMGroupMemberValue struct {
	Value   string `json:"value" example:"user-uuid"`
	Display string `json:"display,omitempty" example:"alice"`
	Ref     string `json:"$ref,omitempty" example:"Users/user-uuid"`
	Type    string `json:"type,omitempty" example:"User"`
}

// ------------------------------------------------------------------------------
// 1. Users CRUD (6 endpoints)
// ------------------------------------------------------------------------------

// CreateUser
// @Summary      Create a new user
// @Description  Creates a new user account scoped to the tenant identified by the X-Tenant-ID header.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID  header    string             true  "Tenant identifier"
// @Param        body         body      CreateUserRequest  true  "User creation payload"
// @Success      201  {object}  UserResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/users [post]
type CreateUserDoc struct{}

// GetUser
// @Summary      Get a user by ID
// @Description  Returns the full profile of the user identified by the path ID.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID  header    string  true  "Tenant identifier"
// @Param        id           path      string  true  "User UUID"
// @Success      200  {object}  UserResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/users/{id} [get]
type GetUserDoc struct{}

// ListUsers
// @Summary      List users
// @Description  Returns a paginated list of users with optional multi-criteria filtering and sorting.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID     header   string  true   "Tenant identifier"
// @Param        search          query    string  false  "Free-text search term"
// @Param        page_size       query    int     false  "Page size (default 50)"
// @Param        status          query    string  false  "Filter by user status"
// @Param        created_after   query    string  false  "RFC3339 timestamp lower bound"
// @Param        created_before  query    string  false  "RFC3339 timestamp upper bound"
// @Param        org_id          query    string  false  "Filter by organization UUID"
// @Param        role_id         query    string  false  "Filter by role UUID"
// @Param        sort_by         query    string  false  "Sort field"
// @Param        sort_order      query    string  false  "Sort order: asc | desc"
// @Success      200  {object}  ListUsersResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/users [get]
type ListUsersDoc struct{}

// UpdateUser
// @Summary      Update a user
// @Description  Updates mutable profile fields (phone, display name, locale, timezone) for the given user.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID  header    string            true  "Tenant identifier"
// @Param        id           path      string            true  "User UUID"
// @Param        body         body      UpdateUserRequest true  "Fields to update"
// @Success      200  {object}  UserResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/users/{id} [patch]
type UpdateUserDoc struct{}

// DeleteUser
// @Summary      Delete a user
// @Description  Permanently deletes the user identified by the path ID.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID  header    string  true  "Tenant identifier"
// @Param        id           path      string  true  "User UUID"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/users/{id} [delete]
type DeleteUserDoc struct{}

// SearchUsers
// @Summary      Search users
// @Description  Performs a filtered search across users with pagination and optional status / last-login filters.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        q                  query   string  false  "Search query"
// @Param        status             query   string  false  "User status filter"
// @Param        limit              query   int     false  "Max results (default 20, max 100)"
// @Param        offset             query   int     false  "Result offset for pagination"
// @Param        last_login_before  query   string  false  "RFC3339 timestamp cutoff"
// @Success      200  {object}  SearchUsersResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/users/search [get]
type SearchUsersDoc struct{}

// ------------------------------------------------------------------------------
// 2. Groups (SCIM) (4 endpoints)
// ------------------------------------------------------------------------------

// CreateGroup
// @Summary      Create a group
// @Description  Creates a new SCIM Group resource (groups map to GGID roles). Members are user references.
// @Tags         Groups
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID  header    string       true  "Tenant identifier"
// @Param        body         body      object       true  "SCIM Group create payload"
// @Success      201  {object}  object
// @Failure      400  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/scim/Groups [post]
type CreateGroupDoc struct{}

// ListGroups
// @Summary      List groups
// @Description  Returns a paginated list of SCIM Group resources, optionally filtered by displayName.
// @Tags         Groups
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID  header   string  true   "Tenant identifier"
// @Param        startIndex   query    int     false  "1-based start index (default 1)"
// @Param        count        query    int     false  "Page size (default 100)"
// @Param        filter       query    string  false  "SCIM filter expression e.g. displayName eq \"value\""
// @Success      200  {object}  object
// @Failure      400  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/scim/Groups [get]
type ListGroupsDoc struct{}

// AddUserToGroup
// @Summary      Add user to group
// @Description  Applies a SCIM PATCH with op "add" on the members path to add one or more users to the group.
// @Tags         Groups
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID  header    string           true  "Tenant identifier"
// @Param        id           path      string           true  "Group ID"
// @Param        body         body      SCIMPatchRequest true  "SCIM PATCH add-member operation"
// @Success      200  {object}  object
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/scim/Groups/{id} [patch]
type AddUserToGroupDoc struct{}

// RemoveUserFromGroup
// @Summary      Remove user from group
// @Description  Applies a SCIM PATCH with op "remove" on the members path (or a member filter) to remove users from the group.
// @Tags         Groups
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID  header    string           true  "Tenant identifier"
// @Param        id           path      string           true  "Group ID"
// @Param        body         body      SCIMPatchRequest true  "SCIM PATCH remove-member operation"
// @Success      200  {object}  object
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/scim/Groups/{id} [patch]
type RemoveUserFromGroupDoc struct{}

// ------------------------------------------------------------------------------
// 3. Roles (4 endpoints)
// ------------------------------------------------------------------------------

// AssignRoleToUser
// @Summary      Assign a role to a user
// @Description  Assigns the given role to the user identified by the path ID. Duplicate assignments are rejected.
// @Tags         Roles
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID  header    string             true  "Tenant identifier"
// @Param        id           path      string             true  "User UUID"
// @Param        body         body      AssignRoleRequest  true  "Role to assign"
// @Success      201  {object}  UserRoleAssignment
// @Failure      400  {object}  ErrorResponse
// @Failure      409  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/users/{id}/roles [post]
type AssignRoleToUserDoc struct{}

// ListUserRoles
// @Summary      List roles for a user
// @Description  Returns all roles currently assigned to the user identified by the path ID.
// @Tags         Roles
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID  header    string  true  "Tenant identifier"
// @Param        id           path      string  true  "User UUID"
// @Success      200  {object}  object
// @Failure      400  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/users/{id}/roles [get]
type ListUserRolesDoc struct{}

// RevokeRoleFromUser
// @Summary      Revoke a role from a user
// @Description  Removes the specified role assignment from the user identified by the path ID.
// @Tags         Roles
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID  header    string  true  "Tenant identifier"
// @Param        id           path      string  true  "User UUID"
// @Param        roleId       path      string  true  "Role ID to revoke"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/users/{id}/roles/{roleId} [delete]
type RevokeRoleFromUserDoc struct{}

// CreateRoleViaGroup
// @Summary      Create a role (via SCIM Group)
// @Description  In GGID, SCIM Groups map to roles. Creating a group effectively creates a role with an initial membership set.
// @Tags         Roles
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID  header    string  true  "Tenant identifier"
// @Param        body         body      object  true  "SCIM Group create payload (role definition)"
// @Success      201  {object}  object
// @Failure      400  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/scim/Groups [post]
type CreateRoleViaGroupDoc struct{}

// ------------------------------------------------------------------------------
// 4. SCIM Tokens (3 endpoints)
// ------------------------------------------------------------------------------

// CreateSCIMToken
// @Summary      Create a SCIM token
// @Description  Mints a new SCIM bearer token for tenant SCIM provisioning. The plaintext token is returned only once.
// @Tags         SCIM
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID  header    string                  true  "Tenant identifier"
// @Param        body         body      CreateSCIMTokenRequest  true  "Token name"
// @Success      201  {object}  CreateSCIMTokenResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      503  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/identity/scim/tokens [post]
type CreateSCIMTokenDoc struct{}

// ListSCIMTokens
// @Summary      List SCIM tokens
// @Description  Returns all SCIM bearer tokens for the tenant. Token hashes are not exposed.
// @Tags         SCIM
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID  header    string  true  "Tenant identifier"
// @Success      200  {object}  ListSCIMTokensResponse
// @Failure      500  {object}  ErrorResponse
// @Failure      503  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/identity/scim/tokens [get]
type ListSCIMTokensDoc struct{}

// RevokeSCIMToken
// @Summary      Revoke a SCIM token
// @Description  Permanently revokes the SCIM bearer token identified by the path ID.
// @Tags         SCIM
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID  header    string  true  "Tenant identifier"
// @Param        id           path      string  true  "Token UUID"
// @Success      200  {object}  map[string]bool
// @Failure      400  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/identity/scim/tokens/{id} [delete]
type RevokeSCIMTokenDoc struct{}

// ------------------------------------------------------------------------------
// 5. Additional identity endpoints (3 endpoints to reach 20)
// ------------------------------------------------------------------------------

// GetCurrentUser
// @Summary      Get current user profile
// @Description  Returns the profile of the authenticated user identified by the X-User-ID header (set by the gateway after JWT verification).
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        X-User-ID  header    string  true  "Authenticated user UUID"
// @Success      200  {object}  UserResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/users/me [get]
type GetCurrentUserDoc struct{}

// LockUser
// @Summary      Lock a user
// @Description  Locks the user account, preventing authentication while preserving the account.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID  header    string  true  "Tenant identifier"
// @Param        id           path      string  true  "User UUID"
// @Success      200  {object}  UserResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/users/{id}/lock [post]
type LockUserDoc struct{}

// DeactivateUser
// @Summary      Deactivate a user
// @Description  Deactivates the user account, marking it inactive while retaining the record for audit.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        X-Tenant-ID  header    string  true  "Tenant identifier"
// @Param        id           path      string  true  "User UUID"
// @Success      200  {object}  UserResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/users/{id}/deactivate [post]
type DeactivateUserDoc struct{}
