# frozen_string_literal: true

require_relative "ggid/version"
require_relative "ggid/types"
require_relative "ggid/auth"
require_relative "ggid/rbac"
require_relative "ggid/abac"
require_relative "ggid/client"
require_relative "ggid/middleware"

# GGID IAM Platform Ruby SDK
#
# Quick start:
#   ggid = GGID::Client.new(base_url: 'https://ggid.iot2.win', tenant_id: '00000000-...')
#   claims = ggid.verify_token(jwt_string)
#   allowed = ggid.check_permission(token, 'products', 'read')
module GGID
  class Error < StandardError; end
  class InvalidTokenError < Error; end
  class ApiError < Error
    attr_reader :status_code, :body

    def initialize(message, status_code: 0, body: nil)
      super(message)
      @status_code = status_code
      @body = body
    end
  end
end
