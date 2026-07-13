# frozen_string_literal: true

module GGID
  # RBAC module — role-based access control methods.
  #
  # Included into GGID::Client.
  module RBAC
    # Check if the current user has permission for resource+action.
    #
    # @param token [String] access token
    # @param resource [String] e.g. "products"
    # @param action [String] e.g. "read"
    # @return [PermissionCheckResult]
    def check_permission(token, resource, action)
      data = http_get("/api/v1/policies/check", token: token, params: {
        resource: resource,
        action: action,
      })
      PermissionCheckResult.from_hash(data)
    end

    # Assign a role to a user.
    #
    # @param token [String] access token
    # @param user_id [String]
    # @param role_id [String]
    # @return [Hash]
    def assign_role(token, user_id, role_id)
      http_post("/api/v1/policies/roles/#{role_id}/users/#{user_id}",
                body: { user_id: user_id, role_id: role_id }, token: token)
    end

    # Revoke a role from a user.
    #
    # @param token [String] access token
    # @param user_id [String]
    # @param role_id [String]
    def revoke_role(token, user_id, role_id)
      http_delete("/api/v1/policies/roles/#{role_id}/users/#{user_id}", token: token)
    end

    # Get all roles assigned to a user.
    #
    # @param token [String] access token
    # @param user_id [String]
    # @return [Array<Role>]
    def get_user_roles(token, user_id)
      data = http_get("/api/v1/policies/users/#{user_id}/roles", token: token)
      items = data.is_a?(Hash) ? (data["roles"] || data["data"] || []) : data
      items.map { |h| Role.from_hash(h) if h.is_a?(Hash) }.compact
    end

    # List all roles in the tenant.
    #
    # @param token [String] access token
    # @return [Array<Role>]
    def list_roles(token)
      data = http_get("/api/v1/roles", token: token)
      items = data.is_a?(Hash) ? (data["roles"] || data["data"] || []) : data
      items.map { |h| Role.from_hash(h) if h.is_a?(Hash) }.compact
    end

    # List all permissions (permission tree).
    #
    # @param token [String] access token
    # @return [Array<Permission>]
    def list_permissions(token)
      data = http_get("/api/v1/policies/permissions/tree", token: token)
      items = data.is_a?(Hash) ? (data["permissions"] || data["data"] || []) : data
      items.map { |h| Permission.from_hash(h) if h.is_a?(Hash) }.compact
    end
  end
end
