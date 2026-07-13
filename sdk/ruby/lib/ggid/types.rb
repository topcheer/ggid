# frozen_string_literal: true

module GGID
  # JWT Claims decoded from a verified access token.
  Claims = Struct.new(:user_id, :tenant_id, :roles, :scope, :exp, :iat, :iss, :sub, :email, :name, keyword_init: true) do
    # Check whether the token has expired.
    def expired?(now: Time.now.to_i)
      now >= exp.to_i
    end

    # Check whether the user has a specific role.
    def has_role?(role)
      roles&.include?(role)
    end

    # Check whether the token grants a specific scope.
    def has_scope?(scope_name)
      scope.to_s.split(" ").include?(scope_name)
    end

    # Build a Claims from a JWT payload hash.
    def self.from_payload(payload)
      new(
        user_id: payload["sub"] || payload["user_id"],
        tenant_id: payload["tenant_id"],
        roles: payload["roles"] || [],
        scope: payload["scope"] || "",
        exp: payload["exp"],
        iat: payload["iat"],
        iss: payload["iss"],
        sub: payload["sub"],
        email: payload["email"],
        name: payload["name"],
      )
    end
  end

  # UserInfo returned by the OIDC /userinfo endpoint.
  UserInfo = Struct.new(:sub, :name, :email, :roles, :picture, keyword_init: true) do
    def self.from_hash(data)
      new(
        sub: data["sub"],
        name: data["name"],
        email: data["email"],
        roles: data["roles"] || [],
        picture: data["picture"],
      )
    end
  end

  # Token response from OAuth token exchange or refresh.
  TokenResponse = Struct.new(:access_token, :refresh_token, :id_token, :expires_in, :token_type, keyword_init: true) do
    def self.from_hash(data)
      new(
        access_token: data["access_token"],
        refresh_token: data["refresh_token"],
        id_token: data["id_token"],
        expires_in: data["expires_in"],
        token_type: data["token_type"] || "Bearer",
      )
    end
  end

  # Role model.
  Role = Struct.new(:id, :name, :key, :description, keyword_init: true) do
    def self.from_hash(data)
      new(
        id: data["id"],
        name: data["name"],
        key: data["key"],
        description: data["description"],
      )
    end
  end

  # Permission model.
  Permission = Struct.new(:id, :name, :resource, :action, :description, :children, keyword_init: true) do
    def self.from_hash(data)
      children = (data["children"] || []).map { |c| from_hash(c) if c.is_a?(Hash) }.compact
      new(
        id: data["id"],
        name: data["name"],
        resource: data["resource"],
        action: data["action"],
        description: data["description"],
        children: children,
      )
    end
  end

  # ABAC evaluation result.
  ABACResult = Struct.new(:allowed, :reason, :matched_rules, :decision, keyword_init: true) do
    def self.from_hash(data)
      new(
        allowed: data["allowed"] || data["matched"] || false,
        reason: data["reason"] || "",
        matched_rules: data["matched_rules"] || [],
        decision: data["decision"],
      )
    end
  end

  # Permission check result.
  PermissionCheckResult = Struct.new(:allowed, :reason, :matched_by, keyword_init: true) do
    def self.from_hash(data)
      new(
        allowed: data["allowed"] || false,
        reason: data["reason"] || "",
        matched_by: data["matched_by"],
      )
    end
  end
end
