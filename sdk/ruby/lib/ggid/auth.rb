# frozen_string_literal: true

require "jwt"
require "net/http"
require "json"

module GGID
  # Auth module — JWT verification via JWKS and OAuth flows.
  #
  # Included into GGID::Client.
  module Auth
    # Verify a JWT access token using JWKS.
    #
    # @param token [String] JWT access token
    # @return [Claims]
    # @raise [InvalidTokenError] on invalid or expired token
    def verify_token(token)
      segments = token.split(".")
      raise InvalidTokenError, "Invalid token format: expected 3 segments" unless segments.length == 3

      header = decode_json_base64(segments[0])
      kid = header["kid"]
      raise InvalidTokenError, "Token header missing key ID (kid)" unless kid

      key = find_jwk_key(kid)

      begin
        payload, = JWT.decode(token, key, true, algorithm: header["alg"] || "RS256")
      rescue JWT::ExpiredSignature
        raise InvalidTokenError, "Token has expired"
      rescue JWT::VerificationError => e
        raise InvalidTokenError, "Token signature verification failed: #{e.message}"
      rescue JWT::DecodeError => e
        raise InvalidTokenError, "Token decode failed: #{e.message}"
      end

      Claims.from_payload(payload)
    end

    # Get OIDC discovery document.
    #
    # @return [Hash]
    def get_discovery
      http_get("/.well-known/openid-configuration")
    end

    # Fetch JWKS from the GGID gateway.
    #
    # @return [Hash]
    def get_jwks
      @jwks_cache ||= nil
      @jwks_cached_at ||= 0
      now = Time.now.to_i
      if @jwks_cache && (now - @jwks_cached_at) < 300
        return @jwks_cache
      end

      @jwks_cache = http_get("/.well-known/jwks.json")
      @jwks_cached_at = now
      @jwks_cache
    end

    # Build a full authorize URL for browser redirect.
    #
    # @param client_id [String]
    # @param redirect_uri [String]
    # @param scope [String] default "openid profile email"
    # @param state [String]
    # @return [String] full authorize URL
    def get_authorize_url(client_id:, redirect_uri:, scope: "openid profile email", state: "")
      params = {
        response_type: "code",
        client_id: client_id,
        redirect_uri: redirect_uri,
        scope: scope,
      }
      params[:state] = state unless state.empty?
      query = URI.encode_www_form(params)
      "#{@base_url}/api/v1/oauth/authorize?#{query}"
    end

    # Exchange an authorization code for tokens.
    #
    # @return [TokenResponse]
    def exchange_code(code:, redirect_uri:, client_id:, client_secret:, code_verifier: nil)
      body = {
        grant_type: "authorization_code",
        code: code,
        redirect_uri: redirect_uri,
        client_id: client_id,
        client_secret: client_secret,
      }
      body[:code_verifier] = code_verifier if code_verifier
      data = http_post("/api/v1/oauth/token", body: body)
      TokenResponse.from_hash(data)
    end

    # Refresh an access token.
    #
    # @return [TokenResponse]
    def refresh_token(refresh_token:, client_id:, client_secret:)
      data = http_post("/api/v1/oauth/token", body: {
        grant_type: "refresh_token",
        refresh_token: refresh_token,
        client_id: client_id,
        client_secret: client_secret,
      })
      TokenResponse.from_hash(data)
    end

    # Get user info from the OIDC userinfo endpoint.
    #
    # @param access_token [String]
    # @return [UserInfo]
    def get_user_info(access_token)
      data = http_get("/api/v1/oauth/userinfo", token: access_token)
      UserInfo.from_hash(data)
    end

    # Revoke a token (RFC 7009).
    def revoke_token(token:, client_id:, client_secret:)
      http_post("/api/v1/oauth/revoke", body: {
        token: token,
        client_id: client_id,
        client_secret: client_secret,
      })
    end

    # Introspect a token (RFC 7662).
    #
    # @return [Hash]
    def introspect_token(token:, client_id:, client_secret:)
      http_post("/api/v1/oauth/introspect", body: {
        token: token,
        client_id: client_id,
        client_secret: client_secret,
      })
    end

    private

    def find_jwk_key(kid)
      jwks = get_jwks
      keys = jwks["keys"] || []
      keys.each do |key_data|
        next unless key_data["kid"] == kid
        return jwk_to_key(key_data)
      end
      # Refresh once in case keys rotated
      @jwks_cache = nil
      jwks = get_jwks
      (jwks["keys"] || []).each do |key_data|
        next unless key_data["kid"] == kid
        return jwk_to_key(key_data)
      end
      raise InvalidTokenError, "No matching key found for kid: #{kid}"
    end

    def jwk_to_key(key_data)
      if key_data["kty"] == "RSA"
        # JWT gem can parse JWK directly
        JWT::JWK.import(key_data).public_key
      elsif key_data["x5c"]
        cert_b64 = key_data["x5c"].first
        cert_der = Base64.decode64(cert_b64)
        OpenSSL::X509::Certificate.new(cert_der).public_key
      else
        raise InvalidTokenError, "Unsupported JWK format"
      end
    end

    def decode_json_base64(segment)
      padded = segment.tr("-_", "+/")
      pad = padded.length % 4
      padded += "=" * (4 - pad) if pad > 0
      JSON.parse(Base64.decode64(padded))
    rescue JSON::ParserError
      raise InvalidTokenError, "Invalid token header encoding"
    end

    # OAuth2 Client Credentials grant (M2M).
    def client_credentials(client_id:, client_secret:, scope: "")
      form_post("/api/v1/oauth/token",
                grant_type: "client_credentials",
                client_id: client_id,
                client_secret: client_secret,
                scope: scope)
    end

    # OAuth2 Device Code Flow (RFC 8628) — Step 1: Request device authorization.
    #
    # @param client_id [String] OAuth2 client ID
    # @param scope [String] Space-delimited scopes
    # @return [Hash] { device_code, user_code, verification_uri, ... }
    def start_device_flow(client_id:, scope: "openid profile email")
      form_post("/api/v1/oauth/device_authorize",
                client_id: client_id,
                scope: scope,
                tenant_id: tenant_id)
    end

    # OAuth2 Device Code Flow (RFC 8628) — Step 2: Poll for token.
    #
    # @param device_code [String] Device code from start_device_flow
    # @param client_id [String] OAuth2 client ID
    # @return [Hash] { access_token, ... } on success,
    #   or { error: "authorization_pending" } / { error: "slow_down" } while waiting
    def poll_device_token(device_code:, client_id:)
      form_post("/api/v1/oauth/token",
                grant_type: "urn:ietf:params:oauth:grant-type:device_code",
                device_code: device_code,
                client_id: client_id)
    end

    # POST with form-urlencoded body (for OAuth2 token endpoints).
    def form_post(path, **params)
      require "httparty"
      url = "#{base_url}#{path}"
      options = {
        headers: { "Content-Type" => "application/x-www-form-urlencoded", "X-Tenant-ID" => tenant_id },
        body: URI.encode_www_form(params),
        timeout: @timeout,
      }
      resp = HTTParty.post(url, options)
      JSON.parse(resp.body || "{}")
    end
  end
end
