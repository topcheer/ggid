# frozen_string_literal: true

module GGID
  # Middleware module — Rails ActionController helpers.
  #
  # In your ApplicationController:
  #
  #   class ApplicationController < ActionController::Base
  #     include GGID::Middleware
  #
  #     before_action :require_auth
  #
  #     # For specific controllers:
  #     # before_action -> { require_permission!('products', 'read') }, only: [:index]
  #     # before_action -> { require_role!('admin') }, only: [:admin_panel]
  #
  #     def current_ggid
  #       @ggid ||= GGID::Client.new(
  #         base_url: Rails.configuration.x.ggid_base_url,
  #         tenant_id: Rails.configuration.x.ggid_tenant_id,
  #       )
  #     end
  #   end
  module Middleware
    # Extract the bearer token from the Authorization header.
    #
    # @return [String, nil]
    def ggid_token
      auth_header = request.headers["Authorization"].to_s
      match = auth_header.match(/^Bearer\s+(.+)/i)
      match ? match[1].strip : nil
    end

    # Get the verified claims from the current request's token.
    #
    # @return [Claims, nil]
    def ggid_claims
      return @ggid_claims if defined?(@ggid_claims)
      token = ggid_token
      return @ggid_claims = nil unless token
      begin
        @ggid_claims = current_ggid.verify_token(token)
      rescue InvalidTokenError
        @ggid_claims = nil
      end
    end

    # Require authentication — redirects or returns 401 if no valid token.
    # Use as: before_action :require_auth
    def require_auth
      token = ggid_token
      unless token
        render_unauthorized("Missing or invalid Authorization header")
        return
      end
      begin
        @ggid_claims = current_ggid.verify_token(token)
        @ggid_access_token = token
      rescue InvalidTokenError => e
        render_unauthorized(e.message)
      end
    end

    # Require a specific permission.
    # Use as: before_action -> { require_permission!('products', 'read') }
    #
    # @param resource [String]
    # @param action [String]
    def require_permission!(resource, action)
      token = ggid_token
      unless token
        render_unauthorized("Authentication required")
        return
      end
      result = current_ggid.check_permission(token, resource, action)
      unless result.allowed
        render_forbidden("Permission denied: #{resource}:#{action}")
      end
    end

    # Require a specific role.
    # Use as: before_action -> { require_role!('admin') }
    #
    # @param role [String]
    def require_role!(role)
      token = ggid_token
      unless token
        render_unauthorized("Authentication required")
        return
      end
      begin
        claims = current_ggid.verify_token(token)
      rescue InvalidTokenError => e
        render_unauthorized(e.message)
        return
      end
      unless claims.has_role?(role)
        render_forbidden("Required role: #{role}")
      end
    end

    # Convenience: check if the current user has a permission (non-raising).
    #
    # @param resource [String]
    # @param action [String]
    # @return [Boolean]
    def can?(resource, action)
      token = ggid_token
      return false unless token
      result = current_ggid.check_permission(token, resource, action)
      result.allowed
    rescue StandardError
      false
    end

    # Convenience: check if the current user has a role (non-raising).
    #
    # @param role [String]
    # @return [Boolean]
    def has_role?(role)
      claims = ggid_claims
      claims&.has_role?(role)
    end

    private

    def render_unauthorized(message)
      respond_to do |format|
        format.json { render json: { error: "unauthorized", message: message }, status: :unauthorized }
        format.html { redirect_to "/login", alert: message }
        format.any { render plain: message, status: :unauthorized }
      end
    end

    def render_forbidden(message)
      respond_to do |format|
        format.json { render json: { error: "forbidden", message: message }, status: :forbidden }
        format.html { render plain: "403 Forbidden: #{message}", status: :forbidden }
        format.any { render plain: message, status: :forbidden }
      end
    end
  end
end
