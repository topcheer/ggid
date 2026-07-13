# frozen_string_literal: true

module GGID
  # ABAC module — attribute-based access control methods.
  #
  # Included into GGID::Client.
  module ABAC
    # Evaluate ABAC policy with structured conditions.
    #
    # @param token [String] access token
    # @param action [String] e.g. "transfer"
    # @param resource [String] e.g. "inventory"
    # @param subject [String] user ID
    # @param conditions [Array<Hash>] array of {field, operator, value}
    # @param tenant_id [String, nil] override tenant ID
    # @return [ABACResult]
    def evaluate_abac(token, action:, resource:, subject:, conditions: [], tenant_id: nil)
      body = {
        action: action,
        resource: resource,
        subject: subject,
      }
      body[:conditions] = conditions unless conditions.empty?
      body[:tenant_id] = tenant_id if tenant_id
      data = http_post("/api/v1/policies/abac/evaluate", body: body, token: token)
      ABACResult.from_hash(data)
    end

    # Full ABAC policy check with subject context.
    #
    # @param token [String] access token
    # @param subject [String] subject identifier
    # @param resource [String] resource being accessed
    # @param action [String] action to evaluate
    # @param context [Hash] additional context attributes
    # @return [ABACResult]
    def check_policy(token, subject:, resource:, action:, context: {})
      body = {
        subject: subject,
        resource: resource,
        action: action,
      }
      body[:context] = context unless context.empty?
      data = http_post("/api/v1/policies/abac/evaluate", body: body, token: token)
      ABACResult.from_hash(data)
    end
  end
end
