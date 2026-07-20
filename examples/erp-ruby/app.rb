# Cross-Board ERP Demo — Ruby implementation
# Tests all GGID core features via Ruby SDK
# Run: GGID_URL=http://localhost:8080 ruby app.rb

require 'sinatra'
require 'json'
require 'securerandom'
require_relative '../../sdk/ruby/lib/ggid'

set :port, ENV.fetch('ERP_LISTEN', '9091').to_i
set :bind, '0.0.0.0'
set :server, 'puma'
set :protection, :except => [:host_authorization]

GGID_URL = ENV.fetch('GGID_URL', 'http://localhost:8080')
TENANT_ID = ENV.fetch('GGID_TENANT_ID', '00000000-0000-0000-0000-000000000001')

$ggid = GGID::Client.new(base_url: GGID_URL, tenant_id: TENANT_ID)

# In-memory stores
$products = {}
$orders = {}
$audit_log = []
$product_seq = 0
$order_seq = 0

before '/api/*' do
  content_type :json
end

# Skip auth for public endpoints
before %r{(?!/api/auth|/health)} do
  pass unless request.path_info.start_with?('/api/')
  token = request.env['HTTP_AUTHORIZATION'].to_s.sub('Bearer ', '')
  halt 401, { error: 'Bearer token required' }.to_json if token.empty?
  begin
    @claims = $ggid.verify_token(token)
  rescue => e
    halt 401, { error: 'invalid token' }.to_json
  end
end

# --- Permission helper ---
def require_perm!(perm)
  return if @claims&.has_permission?(perm)
  halt 403, { error: "missing permission: #{perm}" }.to_json
end

def current_user_id
  @claims&.user_id
end

def audit(action, resource, result = 'success')
  $audit_log << { id: "AUD-#{$audit_log.length + 1}", action: action, resource: resource, result: result, actor_id: current_user_id, timestamp: Time.now.utc.iso8601 }
end

# === Auth (public) ===
post '/api/auth/login' do
  data = JSON.parse(request.body.read)
  tokens = $ggid.login(data['username'], data['password'])
  tokens.to_json
end

post '/api/auth/refresh' do
  data = JSON.parse(request.body.read)
  tokens = $ggid.refresh_token(data['refresh_token'])
  tokens.to_json
end

post '/api/auth/verify' do
  data = JSON.parse(request.body.read)
  claims = $ggid.verify_token(data['token'])
  { user_id: claims.user_id, roles: claims.roles, permissions: claims.permissions, scope: claims.scope }.to_json
end

get '/health' do
  { status: 'ok' }.to_json
end

# === Users (via GGID SDK) ===
get '/api/users' do
  require_perm!('users:read')
  $ggid.list_users(request.env['HTTP_AUTHORIZATION'].sub('Bearer ', '')).to_json
end

post '/api/users' do
  require_perm!('users:write')
  data = JSON.parse(request.body.read)
  user = $ggid.create_user(request.env['HTTP_AUTHORIZATION'].sub('Bearer ', ''), data)
  audit('users.create', 'user')
  [201, user.to_json]
end

# === Roles ===
get '/api/roles' do
  require_perm!('roles:read')
  token = request.env['HTTP_AUTHORIZATION'].sub('Bearer ', '')
  begin
    $ggid.list_roles(token).to_json
  rescue
    { items: [] }.to_json
  end
end

post '/api/roles' do
  require_perm!('roles:write')
  data = JSON.parse(request.body.read)
  token = request.env['HTTP_AUTHORIZATION'].sub('Bearer ', '')
  role = $ggid.create_role(token, name: data['name'], key: data['key'], description: data['description'])
  audit('roles.create', 'role')
  [201, role.to_json]
end

# === Organizations ===
get '/api/orgs' do
  require_perm!('orgs:read')
  { items: [], note: 'Query GGID orgs API' }.to_json
end

post '/api/orgs' do
  require_perm!('orgs:write')
  data = JSON.parse(request.body.read)
  audit('orgs.create', 'org')
  [201, { id: SecureRandom.uuid, name: data['name'] }.to_json]
end

# === Inventory ===
get '/api/inventory' do
  require_perm!('inventory:read')
  { items: $products.values, total: $products.size }.to_json
end

post '/api/inventory' do
  require_perm!('inventory:write')
  data = JSON.parse(request.body.read)
  $product_seq += 1
  product = data.merge('id' => "PROD-#{$product_seq.to_s.rjust(4, '0')}", 'created_at' => Time.now.utc.iso8601)
  $products[product['id']] = product
  audit('inventory.create', 'product')
  [201, product.to_json]
end

get '/api/inventory/:id' do
  require_perm!('inventory:read')
  product = $products[params[:id]]
  halt 404, { error: 'not found' }.to_json unless product
  product.to_json
end

put '/api/inventory/:id' do
  require_perm!('inventory:write')
  product = $products[params[:id]]
  halt 404, { error: 'not found' }.to_json unless product
  data = JSON.parse(request.body.read)
  product.merge!(data)
  audit('inventory.update', 'product')
  product.to_json
end
delete '/api/inventory/:id' do
  require_perm!('inventory:delete')
  $products.delete(params[:id])
  audit('inventory.delete', 'product')
  { deleted: true }.to_json
end

# === Orders ===
get '/api/orders' do
  require_perm!('orders:read')
  show_all = @claims&.has_permission?('orders:read:all')
  uid = current_user_id
  list = $orders.values.select { |o| show_all || o['created_by'] == uid }
  { items: list, total: list.size }.to_json
end

post '/api/orders' do
  require_perm!('orders:write')
  data = JSON.parse(request.body.read)
  $order_seq += 1
  order = data.merge(
    'id' => "ORD-#{$order_seq.to_s.rjust(4, '0')}",
    'status' => 'pending',
    'created_by' => current_user_id,
    'created_at' => Time.now.utc.iso8601
  )
  $orders[order['id']] = order
  audit('orders.create', 'order')
  [201, order.to_json]
end

put '/api/orders/:id/approve' do
  require_perm!('orders:approve')
  order = $orders[params[:id]]
  halt 404, { error: 'not found' }.to_json unless order
  order['status'] = 'approved'
  audit('orders.approve', 'order')
  order.to_json
end

delete '/api/orders/:id' do
  require_perm!('orders:write')
  $orders.delete(params[:id])
  audit('orders.delete', 'order')
  { deleted: true }.to_json
end

# === Audit ===
get '/api/audit' do
  require_perm!('audit:read')
  { items: $audit_log, total: $audit_log.size }.to_json
end

# === Dashboard ===
get '/api/dashboard' do
  require_perm!('dashboard:read')
  pending = $orders.values.count { |o| o['status'] == 'pending' }
  approved = $orders.values.count { |o| o['status'] == 'approved' }
  { products: $products.size, orders: $orders.size, pending: pending, approved: approved, audit_entries: $audit_log.size }.to_json
end
