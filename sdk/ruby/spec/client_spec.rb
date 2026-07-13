# frozen_string_literal: true

require "ggid"
require "json"
require "webmock/rspec"

RSpec.describe GGID::Client do
  let(:base_url) { "https://ggid.test" }
  let(:tenant_id) { "tenant-123" }
  let(:client) { described_class.new(base_url: base_url, tenant_id: tenant_id) }

  # Helper: stub Net::HTTP responses
  def stub_response(status, body, path_pattern)
    stub_request(:any, %r{#{Regexp.escape(base_url)}.+}).to_return do |request|
      if request.uri.path.match?(path_pattern)
        { status: status, body: body.is_a?(String) ? body : JSON.generate(body) }
      else
        { status: 404, body: '{"error":"not found"}' }
      end
    end
  end

  describe "#initialize" do
    it "strips trailing slash from base_url" do
      c = described_class.new(base_url: "https://ggid.test/")
      expect(c.base_url).to eq("https://ggid.test")
    end

    it "sets default tenant ID" do
      c = described_class.new(base_url: "https://ggid.test")
      expect(c.tenant_id).to eq(GGID::Client::DEFAULT_TENANT_ID)
    end
  end

  describe "#login" do
    it "posts credentials and returns token hash" do
      token_data = { "access_token" => "jwt-123", "refresh_token" => "refresh-456", "expires_in" => 3600 }
      stub_request(:post, %r{.*/api/v1/auth/login}).to_return(
        status: 200, body: JSON.generate(token_data)
      )

      result = client.login("admin", "password123")
      expect(result["access_token"]).to eq("jwt-123")
      expect(result["refresh_token"]).to eq("refresh-456")
    end

    it "raises ApiError on 401" do
      stub_request(:post, %r{.*/api/v1/auth/login}).to_return(
        status: 401, body: JSON.generate({ "error" => "invalid_credentials" })
      )

      expect { client.login("bad", "creds") }.to raise_error(GGID::ApiError) do |e|
        expect(e.status_code).to eq(401)
      end
    end
  end

  describe "#register" do
    it "creates a user and returns 201 data" do
      user_data = { "id" => "u1", "username" => "newuser", "email" => "new@test.com" }
      stub_request(:post, %r{.*/api/v1/auth/register}).to_return(
        status: 201, body: JSON.generate(user_data)
      )

      result = client.register(username: "newuser", email: "new@test.com", password: "Pass123!")
      expect(result["id"]).to eq("u1")
    end
  end

  describe "#list_users" do
    it "returns array of users" do
      users = [{ "id" => "u1", "username" => "admin" }, { "id" => "u2", "username" => "user2" }]
      stub_request(:get, %r{.*/api/v1/users}).to_return(
        status: 200, body: JSON.generate(users)
      )

      result = client.list_users("token")
      expect(result.length).to eq(2)
      expect(result[0]["username"]).to eq("admin")
    end
  end

  describe "#create_role" do
    it "creates a role and returns role data" do
      role_data = { "id" => "r1", "name" => "Editor", "key" => "editor" }
      stub_request(:post, %r{.*/api/v1/roles}).to_return(
        status: 201, body: JSON.generate(role_data)
      )

      result = client.create_role("token", name: "Editor", key: "editor")
      expect(result["id"]).to eq("r1")
      expect(result["key"]).to eq("editor")
    end
  end

  describe "#list_roles" do
    it "returns Role objects" do
      roles_data = { "roles" => [
        { "id" => "r1", "name" => "Admin", "key" => "admin" },
        { "id" => "r2", "name" => "User", "key" => "user" },
      ] }
      stub_request(:get, %r{.*/api/v1/roles}).to_return(
        status: 200, body: JSON.generate(roles_data)
      )

      roles = client.list_roles("token")
      expect(roles.length).to eq(2)
      expect(roles[0]).to be_a(GGID::Role)
      expect(roles[0].name).to eq("Admin")
    end
  end

  describe "#check_permission" do
    it "returns PermissionCheckResult with allowed=true" do
      data = { "allowed" => true, "reason" => "matched", "matched_by" => "admin" }
      stub_request(:get, %r{.*/api/v1/policies/check}).to_return(
        status: 200, body: JSON.generate(data)
      )

      result = client.check_permission("token", "products", "read")
      expect(result).to be_a(GGID::PermissionCheckResult)
      expect(result.allowed).to be(true)
      expect(result.matched_by).to eq("admin")
    end

    it "returns denied result" do
      data = { "allowed" => false, "reason" => "no matching policy" }
      stub_request(:get, %r{.*/api/v1/policies/check}).to_return(
        status: 200, body: JSON.generate(data)
      )

      result = client.check_permission("token", "products", "delete")
      expect(result.allowed).to be(false)
    end
  end

  describe "#assign_role" do
    it "assigns role and returns response" do
      stub_request(:post, %r{.*/api/v1/policies/roles/r1/users/u1}).to_return(
        status: 200, body: JSON.generate({ "success" => true })
      )

      result = client.assign_role("token", "u1", "r1")
      expect(result["success"]).to be(true)
    end
  end

  describe "#revoke_role" do
    it "deletes role assignment" do
      stub_request(:delete, %r{.*/api/v1/policies/roles/r1/users/u1}).to_return(
        status: 204, body: ""
      )

      expect { client.revoke_role("token", "u1", "r1") }.not_to raise_error
    end
  end

  describe "#get_user_roles" do
    it "returns Role objects" do
      data = [{ "id" => "r1", "name" => "Admin", "key" => "admin" }]
      stub_request(:get, %r{.*/api/v1/policies/users/u1/roles}).to_return(
        status: 200, body: JSON.generate(data)
      )

      roles = client.get_user_roles("token", "u1")
      expect(roles.length).to eq(1)
      expect(roles[0]).to be_a(GGID::Role)
      expect(roles[0].key).to eq("admin")
    end
  end

  describe "#list_permissions" do
    it "returns Permission objects" do
      data = [{ "id" => "p1", "name" => "Read", "resource" => "products", "action" => "read" }]
      stub_request(:get, %r{.*/api/v1/policies/permissions/tree}).to_return(
        status: 200, body: JSON.generate(data)
      )

      perms = client.list_permissions("token")
      expect(perms.length).to eq(1)
      expect(perms[0]).to be_a(GGID::Permission)
      expect(perms[0].resource).to eq("products")
    end
  end

  describe "#evaluate_abac" do
    it "returns ABACResult" do
      data = { "allowed" => true, "reason" => "matched", "matched_rules" => ["rule-1"] }
      stub_request(:post, %r{.*/api/v1/policies/abac/evaluate}).to_return(
        status: 200, body: JSON.generate(data)
      )

      result = client.evaluate_abac("token",
        action: "transfer",
        resource: "inventory",
        subject: "u1",
        conditions: [{ field: "warehouse", operator: "eq", value: "WH-001" }])

      expect(result).to be_a(GGID::ABACResult)
      expect(result.allowed).to be(true)
      expect(result.matched_rules).to eq(["rule-1"])
    end
  end

  describe "#check_policy" do
    it "returns ABACResult with denied" do
      data = { "allowed" => false, "reason" => "no matching policy" }
      stub_request(:post, %r{.*/api/v1/policies/abac/evaluate}).to_return(
        status: 200, body: JSON.generate(data)
      )

      result = client.check_policy("token", subject: "u1", resource: "inventory", action: "delete", context: { dept: "sales" })
      expect(result.allowed).to be(false)
    end
  end

  describe "#get_authorize_url" do
    it "builds a valid authorize URL" do
      url = client.get_authorize_url(
        client_id: "client-123",
        redirect_uri: "https://app.test/callback",
        state: "xyz",
      )

      expect(url).to include("client_id=client-123")
      expect(url).to include("response_type=code")
      expect(url).to include("state=xyz")
      expect(url).to include("/api/v1/oauth/authorize")
    end
  end

  describe "#get_discovery" do
    it "returns OIDC discovery document" do
      data = {
        "issuer" => "https://ggid.test",
        "jwks_uri" => "https://ggid.test/.well-known/jwks.json",
      }
      stub_request(:get, %r{.*/.well-known/openid-configuration}).to_return(
        status: 200, body: JSON.generate(data)
      )

      result = client.get_discovery
      expect(result["issuer"]).to eq("https://ggid.test")
    end
  end

  describe "Claims" do
    it "parses from JWT payload" do
      payload = {
        "sub" => "u123",
        "tenant_id" => "t1",
        "roles" => ["admin", "editor"],
        "scope" => "read write",
        "exp" => Time.now.to_i + 3600,
        "iat" => Time.now.to_i,
        "iss" => "https://ggid.test",
      }

      claims = GGID::Claims.from_payload(payload)
      expect(claims.user_id).to eq("u123")
      expect(claims.has_role?("admin")).to be(true)
      expect(claims.has_scope?("read")).to be(true)
      expect(claims.expired?).to be(false)
    end

    it "detects expired tokens" do
      claims = GGID::Claims.new(
        user_id: "u1", tenant_id: "t1", roles: [], scope: "",
        exp: 1000, iat: 900, iss: "test",
      )
      expect(claims.expired?(now: 2000)).to be(true)
    end
  end

  describe "TokenResponse" do
    it "parses from hash" do
      data = { "access_token" => "at", "refresh_token" => "rt", "expires_in" => 3600 }
      tr = GGID::TokenResponse.from_hash(data)
      expect(tr.access_token).to eq("at")
      expect(tr.refresh_token).to eq("rt")
      expect(tr.expires_in).to eq(3600)
    end
  end

  describe "error handling" do
    it "raises ApiError on 404" do
      stub_request(:get, %r{.*/api/v1/users/nonexistent}).to_return(
        status: 404, body: JSON.generate({ "error" => "not_found" })
      )

      expect { client.get_user("token", "nonexistent") }.to raise_error(GGID::ApiError) do |e|
        expect(e.status_code).to eq(404)
      end
    end
  end
end
