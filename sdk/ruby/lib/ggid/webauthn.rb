# GGID SDK - WebAuthn / Passkey utilities (Ruby)
module GGID
  module WebAuthn
    module_function

    # Encode bytes to base64url string
    def buffer_to_base64url(data)
      [data].pack('m0').tr('+/', '-_').gsub(/=+$/, '')
    end

    # Decode base64url string to bytes
    def base64url_to_buffer(b64url)
      padded = b64url.tr('-_', '+/')
      padded += '=' * ((4 - padded.length % 4) % 4)
      padded.unpack1('m0')
    end

    # Register a passkey via GGID API
    # Note: browser-side navigator.credentials.create() must happen on client
    def register_passkey(api_base_url:, auth_token:, user_id:, tenant_id: nil)
      require 'net/http'
      require 'uri'
      uri = URI("#{api_base_url}/api/v1/auth/webauthn/register/begin")
      http = Net::HTTP.new(uri.host, uri.port)
      http.use_ssl = uri.scheme == 'https'
      req = Net::HTTP::Post.new(uri, 'Content-Type' => 'application/json')
      req['Authorization'] = "Bearer #{auth_token}"
      req['X-Tenant-ID'] = tenant_id if tenant_id
      req.body = { user_id: user_id }.to_json
      response = http.request(req)
      response.is_a?(Net::HTTPSuccess)
    end
  end

  # User Management CRUD
  class UserManagement
    def initialize(api_base_url:, auth_token:, tenant_id: nil)
      @base = api_base_url
      @token = auth_token
      @tenant = tenant_id
    end

    def create_user(username:, email:, password: nil)
      request('POST', '/api/v1/users', username: username, email: email, password: password)
    end

    def get_user(user_id)
      request('GET', "/api/v1/users/#{user_id}")
    end

    def list_users(page: 1, page_size: 20)
      request('GET', "/api/v1/users?page=#{page}&page_size=#{page_size}")
    end

    def update_user(user_id, **updates)
      request('PATCH', "/api/v1/users/#{user_id}", updates)
    end

    def delete_user(user_id)
      result = request('DELETE', "/api/v1/users/#{user_id}")
      result['status'] == 'deleted' || result.key?('id')
    end

    private

    def request(method, path, body = nil)
      require 'net/http'
      require 'uri'
      require 'json'
      uri = URI("#{@base}#{path}")
      http = Net::HTTP.new(uri.host, uri.port)
      http.use_ssl = uri.scheme == 'https'
      req_class = Net::HTTP.const_get(method.capitalize)
      req = req_class.new(uri)
      req['Authorization'] = "Bearer #{@token}"
      req['X-Tenant-ID'] = @tenant if @tenant
      req['Content-Type'] = 'application/json'
      req.body = body.to_json if body
      response = http.request(req)
      JSON.parse(response.body)
    end
  end

  # ABAC (Attribute-Based Access Control) condition evaluation
  module ABAC
    module_function

    # Evaluate an ABAC condition against an attribute context
    # condition example: { field: 'department', op: 'eq', value: 'engineering' }
    def evaluate(condition, attributes)
      return true unless condition
      case condition[:op] || condition['op']
      when 'eq', '=='
        attributes[condition[:field] || condition['field']] == (condition[:value] || condition['value'])
      when 'ne', '!='
        attributes[condition[:field] || condition['field']] != (condition[:value] || condition['value'])
      when 'in'
        val = condition[:value] || condition['value']
        attr_val = attributes[condition[:field] || condition['field']]
        val.is_a?(Array) && val.include?(attr_val)
      when 'gt', '>'
        (attributes[condition[:field] || condition['field']] || 0) > (condition[:value] || condition['value'])
      when 'lt', '<'
        (attributes[condition[:field] || condition['field']] || 0) < (condition[:value] || condition['value'])
      when 'contains'
        attr_val = attributes[condition[:field] || condition['field']]
        attr_val.is_a?(String) && attr_val.include?(condition[:value] || condition['value'])
      when 'and', '&&'
        (condition[:conditions] || condition['conditions']).all? { |c| evaluate(c, attributes) }
      when 'or', '||'
        (condition[:conditions] || condition['conditions']).any? { |c| evaluate(c, attributes) }
      else
        false
      end
    end

    # Check if any policy in the set matches
    def check(policies, attributes)
      policies.any? { |p| evaluate(p, attributes) }
    end
  end

  # Rack middleware for request authentication
  if defined?(Rack)
    class Middleware
      def initialize(app, api_base_url:, public_paths: [])
        @app = app
        @api_base_url = api_base_url
        @public_paths = public_paths
      end

      def call(env)
        path = env['PATH_INFO']
        return @app.call(env) if @public_paths.any? { |p| path.start_with?(p) }

        token = env['HTTP_AUTHORIZATION']&.gsub(/^Bearer\s+/i, '')
        return [401, { 'Content-Type' => 'application/json' }, ['{"error":"missing token"}']] unless token

        # Verify token with GGID (simplified - in production, verify JWT locally)
        # For now, pass through to app
        env['ggid.token'] = token
        @app.call(env)
      end
    end
  end
end
