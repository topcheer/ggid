# ERP Ruby Demo — using Sinatra::Base to avoid Rack::Protection::HostAuthorization
require 'sinatra/base'
require 'json'
require 'securerandom'
require_relative '../../sdk/ruby/lib/ggid'

GGID_URL_VAL = ENV.fetch('GGID_URL', 'http://localhost:8080')
TENANT_ID = ENV.fetch('GGID_TENANT_ID', '00000007-0000-0000-0000-000000000001')
GGID_DEVICE_AUTH = ENV.fetch('GGID_DEVICE_AUTH_URL', "#{GGID_URL_VAL}/api/v1/oauth/device_authorize")
GGID_TOKEN_URL = ENV.fetch('GGID_TOKEN_URL', "#{GGID_URL_VAL}/api/v1/oauth/token")
CLIENT_ID = ENV.fetch('OAUTH_CLIENT_ID', 'erp-ruby-demo')

$ggid = GGID::Client.new(base_url: GGID_URL_VAL, tenant_id: TENANT_ID)
$products = {}
$orders = {}
$audit_log = []
$product_seq = 0
$order_seq = 0

class ERPApp < Sinatra::Base
  # Sinatra 4.x adds Rack::Protection::HostAuthorization as separate middleware
  # (not controlled by :protection setting) — must disable explicitly
  set :host_authorization, permitted_hosts: ['.']
  set :port, ENV.fetch('ERP_LISTEN', '9091').to_i
  set :bind, '0.0.0.0'
  set :server, 'puma'

  before '/api/*' do
    content_type :json
  end

  before '/api/*' do
    pass if request.path_info.start_with?('/api/auth/')
    token = request.env['HTTP_AUTHORIZATION'].to_s.sub('Bearer ', '')
    halt 401, { error: 'Bearer token required' }.to_json if token.empty?
    begin
      @claims = $ggid.verify_token(token)
      $stderr.puts "[ERP] verify OK: user=#{@claims.user_id} perms=#{@claims.permissions.inspect}"
    rescue => e
      $stderr.puts "[ERP] verify FAILED: #{e.class}: #{e.message}"
      halt 401, { error: 'invalid token' }.to_json
    end
  end

  def require_perm!(perm)
    return if @claims&.has_permission?(perm)
    halt 403, { error: "missing permission: #{perm}" }.to_json
  end

  def current_user_id; @claims&.user_id; end

  def audit(action, resource, result = 'success')
    $audit_log << { id: "AUD-#{$audit_log.length + 1}", action: action, resource: resource, result: result, actor_id: current_user_id, timestamp: Time.now.utc.iso8601 }
  end

  get '/health' do
    { status: 'ok', lang: 'ruby', tenant: TENANT_ID }.to_json
  end

  # OAuth2 Device Code Flow — start device authorization
  post '/api/auth/device/start' do
    require 'net/http'
    require 'uri'
    uri = URI(GGID_DEVICE_AUTH)
    req = Net::HTTP::Post.new(uri)
    req.set_form_data(
      'client_id' => CLIENT_ID,
      'tenant_id' => TENANT_ID,
      'scope' => 'openid profile email'
    )
    req['X-Tenant-ID'] = TENANT_ID
    resp = Net::HTTP.start(uri.hostname, uri.port, use_ssl: uri.scheme == 'https') { |http| http.request(req) }
    body = JSON.parse(resp.body || '{}')
    {
      device_code: body['device_code'],
      user_code: body['user_code'],
      verification_uri: body['verification_uri'] || "#{GGID_URL_VAL}/device",
      verification_uri_complete: body['verification_uri_complete'],
      expires_in: body['expires_in'] || 300,
      interval: body['interval'] || 5
    }.to_json
  end

  # OAuth2 Device Code Flow — poll for token
  post '/api/auth/device/poll' do
    require 'net/http'
    require 'uri'
    data = JSON.parse(request.body.read)
    uri = URI(GGID_TOKEN_URL)
    req = Net::HTTP::Post.new(uri)
    req.set_form_data(
      'grant_type' => 'urn:ietf:params:oauth:grant-type:device_code',
      'device_code' => data['device_code'],
      'client_id' => CLIENT_ID,
      'tenant_id' => TENANT_ID
    )
    req['X-Tenant-ID'] = TENANT_ID
    resp = Net::HTTP.start(uri.hostname, uri.port, use_ssl: uri.scheme == 'https') { |http| http.request(req) }
    body = JSON.parse(resp.body || '{}')
    if resp.code == '200'
      { access_token: body['access_token'], token_type: body['token_type'], expires_in: body['expires_in'] }.to_json
    elsif body['error'] == 'authorization_pending'
      { error: 'authorization_pending', error_description: body['error_description'] }.to_json
    elsif body['error'] == 'slow_down'
      { error: 'slow_down', error_description: body['error_description'], interval: 10 }.to_json
    else
      status resp.code.to_i
      body.to_json
    end
  end

  post '/api/auth/verify' do
    data = JSON.parse(request.body.read)
    claims = $ggid.verify_token(data['token'])
    { user_id: claims.user_id, roles: claims.roles, permissions: claims.permissions }.to_json
  end

  # Inventory
  get '/api/inventory' do
    require_perm!('inventory:read')
    { items: $products.values, total: $products.size }.to_json
  end

  post '/api/inventory' do
    require_perm!('inventory:write')
    data = JSON.parse(request.body.read)
    $product_seq += 1
    id = "PROD-#{$product_seq.to_s.rjust(4, '0')}"
    product = data.merge('id' => id, 'created_at' => Time.now.utc.iso8601)
    $products[id] = product
    audit('inventory.create', 'product')
    [201, product.to_json]
  end

  delete '/api/inventory/:id' do
    require_perm!('inventory:delete')
    $products.delete(params[:id])
    audit('inventory.delete', 'product')
    { deleted: true }.to_json
  end

  # Orders
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
    id = "ORD-#{$order_seq.to_s.rjust(4, '0')}"
    order = data.merge('id' => id, 'status' => 'pending', 'created_by' => current_user_id, 'created_at' => Time.now.utc.iso8601)
    $orders[id] = order
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

  # Audit
  get '/api/audit' do
    require_perm!('audit:read')
    { items: $audit_log, total: $audit_log.size }.to_json
  end

  # Dashboard
  get '/api/dashboard' do
    require_perm!('dashboard:read')
    { products: $products.size, orders: $orders.size, audit: $audit_log.size }.to_json
  end
end

# run! called by start.rb after middleware config
