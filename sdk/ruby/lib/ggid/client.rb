# frozen_string_literal: true

require "httparty"
require "json"

module GGID
  # Main GGID API client.
  #
  # Usage:
  #   ggid = GGID::Client.new(base_url: 'https://ggid.iot2.win', tenant_id: '00000000-...')
  #   tokens = ggid.login('admin', 'Admin@123456')
  #   claims = ggid.verify_token(tokens['access_token'])
  #   allowed = ggid.check_permission(token, 'products', 'read')
  class Client
    include Auth
    include RBAC
    include ABAC

    DEFAULT_TENANT_ID = "00000000-0000-0000-0000-000000000001"
    DEFAULT_TIMEOUT = 30

    attr_reader :base_url, :tenant_id

    # @param base_url [String] GGID gateway URL
    # @param tenant_id [String] tenant UUID
    # @param timeout [Integer] request timeout in seconds
    def initialize(base_url:, tenant_id: DEFAULT_TENANT_ID, timeout: DEFAULT_TIMEOUT)
      @base_url = base_url.to_s.chomp("/")
      @tenant_id = tenant_id
      @timeout = timeout
      @jwks_cache = nil
      @jwks_cached_at = 0
    end

    # ── Authentication ──────────────────────────────────────────

    # Register a new user.
    #
    # @return [Hash]
    def register(username:, email:, password:)
      http_post("/api/v1/auth/register", body: {
        username: username,
        email: email,
        password: password,
      })
    end

    # Login and obtain tokens.
    #
    # @return [Hash] token response with access_token, refresh_token, etc.
    def login(username, password)
      http_post("/api/v1/auth/login", body: {
        username: username,
        password: password,
      })
    end

    # ── User Management ────────────────────────────────────────

    def get_user(token, user_id)
      http_get("/api/v1/users/#{user_id}", token: token)
    end

    def list_users(token, params = {})
      http_get("/api/v1/users", token: token, params: params)
    end

    def create_user(token, data)
      http_post("/api/v1/users", body: data, token: token)
    end

    def update_user(token, user_id, data)
      http_put("/api/v1/users/#{user_id}", body: data, token: token)
    end

    def delete_user(token, user_id)
      http_delete("/api/v1/users/#{user_id}", token: token)
    end

    # ── Roles CRUD ─────────────────────────────────────────────

    def create_role(token, name:, key:, description: "")
      http_post("/api/v1/roles", body: {
        name: name,
        key: key,
        description: description,
      }, token: token)
    end

    def get_role(token, role_id)
      http_get("/api/v1/roles/#{role_id}", token: token)
    end

    def update_role(token, role_id, name: nil, description: nil)
      body = {}
      body[:name] = name if name
      body[:description] = description if description
      http_put("/api/v1/roles/#{role_id}", body: body, token: token)
    end

    def delete_role(token, role_id)
      http_delete("/api/v1/roles/#{role_id}", token: token)
    end

    # ── Audit ──────────────────────────────────────────────────

    def list_audit_events(token, params = {})
      http_get("/api/v1/audit/events", token: token, params: params)
    end

    # ── Internal HTTP methods ──────────────────────────────────

    # Perform an HTTP GET request.
    #
    # @param path [String] API path
    # @param token [String, nil] bearer token
    # @param params [Hash] query parameters
    # @return [Hash] parsed JSON response
    def http_get(path, token: nil, params: {})
      request(:get, path, token: token, query: params)
    end

    # Perform an HTTP POST request.
    def http_post(path, body: nil, token: nil)
      request(:post, path, token: token, body: body)
    end

    # Perform an HTTP PUT request.
    def http_put(path, body: nil, token: nil)
      request(:put, path, token: token, body: body)
    end

    # Perform an HTTP DELETE request.
    def http_delete(path, token: nil)
      request(:delete, path, token: token)
    end

    private

    def request(method, path, token: nil, body: nil, query: {})
      options = {
        headers: build_headers(token),
        timeout: @timeout,
        open_timeout: 10,
      }
      options[:query] = query unless query.empty?
      options[:body] = JSON.generate(body) if body

      url = "#{@base_url}#{path}"
      response = HTTParty.send(method, url, options)
      status = response.code
      resp_body = response.body.to_s

      if status >= 400
        error_body = begin
          JSON.parse(resp_body)
        rescue JSON::ParserError
          resp_body
        end
        raise ApiError.new(
          "API error #{status} for #{method.to_s.upcase} #{path}",
          status_code: status,
          body: error_body,
        )
      end

      return {} if status == 204 || resp_body.nil? || resp_body.empty?

      JSON.parse(resp_body)
    rescue SocketError => e
      raise ApiError.new("Connection failed: #{e.message}")
    end

    def build_headers(token)
      headers = {
        "Content-Type" => "application/json",
        "Accept" => "application/json",
        "X-Tenant-ID" => @tenant_id,
      }
      headers["Authorization"] = "Bearer #{token}" if token
      headers
    end
  end
end
