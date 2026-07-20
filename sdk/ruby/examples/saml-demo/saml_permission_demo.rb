# GGID SAML SSO Demo with Permissions (Ruby)
# Run: GGID_URL=... ruby saml_permission_demo.rb
require 'webrick'
require_relative '../../lib/ggid/saml'

module GGID
  USER = {
    username: 'demo_user',
    roles: ['viewer'],
    permissions: ['dashboard:read', 'orders:read', 'inventory:read']
  }

  def self.has_permission?(perm)
    USER[:permissions].include?('admin') || USER[:permissions].include?(perm)
  end

  def self.render_menu
    items = ['<li><a href="/">Dashboard</a></li>']
    items << '<li><a href="/orders">Orders</a></li>' if has_permission?('orders:read')
    items << '<li><a href="/inventory">Inventory</a></li>' if has_permission?('inventory:read')
    "<aside><h2>Menu</h2><ul>#{items.join}</ul><p>Roles: #{USER[:roles].join(', ')}</p></aside>"
  end

  def self.render_dashboard
    "<html><body>#{render_menu}<main><h1>Dashboard</h1><p>#{USER[:username]}</p></main></body></html>"
  end

  def self.render_page(title, can_write: false)
    btn = can_write ? '<button>New</button>' : '<p>Read-only</p>'
    "<html><body>#{render_menu}<main><h1>#{title}</h1>#{btn}</main></body></html>"
  end

  def self.render_403(perm)
    "<html><body><h1>403 Forbidden</h1><p>Need: #{perm}</p></body></html>"
  end
end

ggid_url = ENV['GGID_URL'] || 'http://localhost:8080'
entity_id = ENV['SP_ENTITY_ID'] || 'http://localhost:3104/saml/metadata'
acs_url = ENV['ACS_URL'] || 'http://localhost:3104/saml/acs'

server = WEBrick::HTTPServer.new(Port: 3104)

server.mount_proc '/' do |req, res|
  res['Content-Type'] = 'text/html'
  res.body = GGID.render_dashboard
end

server.mount_proc '/saml/metadata' do |req, res|
  res['Content-Type'] = 'application/xml'
  res.body = GGID::SAML.generate_sp_metadata(entity_id: entity_id, acs_url: acs_url)
end

server.mount_proc '/login' do |req, res|
  url = GGID::SAML.build_authn_request_url(
    sso_url: "#{ggid_url}/saml/sso", entity_id: entity_id, acs_url: acs_url, relay_state: '/')
  res.set_redirect(WEBrick::HTTPStatus::Found, url)
end

server.mount_proc '/inventory' do |req, res|
  res['Content-Type'] = 'text/html'
  if !GGID.has_permission?('inventory:read')
    res.status = 403
    res.body = GGID.render_403('inventory:read')
  else
    res.body = GGID.render_page('Inventory', can_write: GGID.has_permission?('inventory:write'))
  end
end

server.mount_proc '/orders' do |req, res|
  res['Content-Type'] = 'text/html'
  if !GGID.has_permission?('orders:read')
    res.status = 403
    res.body = GGID.render_403('orders:read')
  else
    res.body = GGID.render_page('Orders', can_write: GGID.has_permission?('orders:write'))
  end
end

trap('INT') { server.shutdown }
server.start
