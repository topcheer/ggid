# GGID Ruby SDK

Ruby SDK for the GGID IAM Platform — JWT verification, OAuth/OIDC, RBAC, ABAC, and Rails middleware.

## Quick Start (5 Minutes)

### 1. Install

Add to your Gemfile:

```ruby
gem 'ggid'
```

Or install directly:

```bash
gem install ggid
```

### 2. Initialize

```ruby
require 'ggid'

ggid = GGID::Client.new(
  base_url: 'https://ggid.iot2.win',
  tenant_id: '00000000-0000-0000-0000-000000000001',
)
```

### 3. Authenticate

```ruby
# Login
tokens = ggid.login('admin', 'Admin@123456')

# Verify JWT
claims = ggid.verify_token(tokens['access_token'])
puts claims.user_id   # => "user-001"
puts claims.email     # => "admin@example.com"

# Check permission
result = ggid.check_permission(tokens['access_token'], 'products', 'read')
puts result.allowed   # => true
```

## API Reference

### Client

```ruby
ggid = GGID::Client.new(
  base_url: 'https://ggid.iot2.win',
  tenant_id: '00000000-0000-0000-0000-000000000001',
  timeout: 30,
)
```

### Authentication

| Method | Description |
|--------|-------------|
| `register(username:, email:, password:)` | Register a new user |
| `login(username, password)` | Login and get tokens |
| `verify_token(token)` | Verify JWT via JWKS |
| `get_user_info(access_token)` | OIDC userinfo endpoint |

### OAuth/OIDC

| Method | Description |
|--------|-------------|
| `get_discovery` | OIDC discovery document |
| `get_jwks` | JWKS public keys |
| `get_authorize_url(client_id:, redirect_uri:, scope:, state:)` | Build authorize URL |
| `exchange_code(code:, redirect_uri:, client_id:, client_secret:)` | Exchange auth code for tokens |
| `refresh_token(refresh_token:, client_id:, client_secret:)` | Refresh access token |
| `revoke_token(token:, client_id:, client_secret:)` | Revoke token (RFC 7009) |
| `introspect_token(token:, client_id:, client_secret:)` | Introspect token (RFC 7662) |

### RBAC

| Method | Description |
|--------|-------------|
| `check_permission(token, resource, action)` | Check user permission |
| `assign_role(token, user_id, role_id)` | Assign role to user |
| `revoke_role(token, user_id, role_id)` | Revoke role from user |
| `get_user_roles(token, user_id)` | Get user's roles |
| `list_roles(token)` | List all roles |
| `list_permissions(token)` | List permission tree |

### ABAC

| Method | Description |
|--------|-------------|
| `evaluate_abac(token, action:, resource:, subject:, conditions:, tenant_id:)` | Evaluate ABAC policy |
| `check_policy(token, subject:, resource:, action:, context:)` | Full policy check |

### User Management

| Method | Description |
|--------|-------------|
| `get_user(token, user_id)` | Get user by ID |
| `list_users(token, params)` | List users |
| `create_user(token, data)` | Create user |
| `update_user(token, user_id, data)` | Update user |
| `delete_user(token, user_id)` | Delete user |

## Rails Integration

### ApplicationController

```ruby
class ApplicationController < ActionController::Base
  include GGID::Middleware

  before_action :require_auth

  private

  def current_ggid
    @ggid ||= GGID::Client.new(
      base_url: Rails.configuration.x.ggid_base_url,
      tenant_id: Rails.configuration.x.ggid_tenant_id,
    )
  end
end
```

### Controllers with Permission/Role Guards

```ruby
class ProductsController < ApplicationController
  # Require specific permission
  before_action -> { require_permission!('products', 'read') }, only: [:index, :show]
  before_action -> { require_permission!('products', 'write') }, only: [:create, :update]

  # Require admin role
  before_action -> { require_role!('admin') }, only: [:destroy]

  def index
    # Access verified claims
    render json: { user: ggid_claims.user_id }
  end

  def show
    # Non-raising permission check
    if can?('products', 'read')
      # ...
    end
  end
end
```

### Claims Object

```ruby
claims = ggid.verify_token(jwt)

claims.user_id      # => String
claims.tenant_id    # => String
claims.roles        # => Array<String>
claims.scope        # => String (space-separated)
claims.exp          # => Integer (Unix timestamp)
claims.email        # => String | nil

claims.has_role?('admin')   # => true/false
claims.has_scope?('read')   # => true/false
claims.expired?             # => true/false
```

## Dependencies

- Ruby 3.0+
- httparty ~> 0.22
- jwt ~> 2.8

## License

Apache-2.0
