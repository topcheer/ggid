# frozen_string_literal: true

require "ggid"
require "json"
require "webmock/rspec"

RSpec.describe GGID::Client do
  let(:base_url) { "https://ggid.test" }
  let(:tenant_id) { "tenant-123" }
  let(:client) { described_class.new(base_url: base_url, tenant_id: tenant_id) }

  describe "#list_webhooks" do
    it "returns array of webhooks" do
      data = [
        {"id" => "wh-1", "url" => "https://example.com/hook", "events" => ["user.created"]},
        {"id" => "wh-2", "url" => "https://example.com/hook2", "events" => ["role.assigned"]},
      ]
      stub_request(:get, %r{.*/api/v1/webhooks}).to_return(
        status: 200, body: JSON.generate(data)
      )

      result = client.list_webhooks("token")
      expect(result.length).to eq(2)
      expect(result[0]["id"]).to eq("wh-1")
      expect(result[0]["url"]).to eq("https://example.com/hook")
    end

    it "returns webhooks from object wrapper" do
      data = {"webhooks" => [{"id" => "wh-1", "url" => "https://example.com/hook"}]}
      stub_request(:get, %r{.*/api/v1/webhooks}).to_return(
        status: 200, body: JSON.generate(data)
      )

      result = client.list_webhooks("token")
      expect(result["webhooks"].length).to eq(1)
    end
  end

  describe "#create_webhook" do
    it "creates a webhook and returns webhook data" do
      data = {"id" => "wh-3", "url" => "https://example.com/hook3", "events" => ["user.created"]}
      stub_request(:post, %r{.*/api/v1/webhooks}).to_return(
        status: 201, body: JSON.generate(data)
      )

      result = client.create_webhook("token", url: "https://example.com/hook3", events: ["user.created"])
      expect(result["id"]).to eq("wh-3")
      expect(result["url"]).to eq("https://example.com/hook3")
    end
  end

  describe "#delete_webhook" do
    it "deletes a webhook without error" do
      stub_request(:delete, %r{.*/api/v1/webhooks/wh-1}).to_return(
        status: 204, body: ""
      )

      expect { client.delete_webhook("token", "wh-1") }.not_to raise_error
    end
  end

  describe "#introspect_token" do
    it "returns active status and claims" do
      data = {"active" => true, "sub" => "user-1", "exp" => 1700000000, "scope" => "openid profile"}
      stub_request(:post, %r{.*/api/v1/oauth/introspect}).to_return(
        status: 200, body: JSON.generate(data)
      )

      result = client.introspect_token(token: "jwt-token", client_id: "cid", client_secret: "sec")
      expect(result["active"]).to be(true)
      expect(result["sub"]).to eq("user-1")
      expect(result["scope"]).to eq("openid profile")
    end

    it "returns inactive for revoked token" do
      data = {"active" => false}
      stub_request(:post, %r{.*/api/v1/oauth/introspect}).to_return(
        status: 200, body: JSON.generate(data)
      )

      result = client.introspect_token(token: "revoked", client_id: "cid", client_secret: "sec")
      expect(result["active"]).to be(false)
    end
  end
end
