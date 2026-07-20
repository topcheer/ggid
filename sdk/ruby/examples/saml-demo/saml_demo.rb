# GGID SAML SSO Demo (Ruby)
# Run: GGID_URL=... ruby saml_demo.rb
require 'webrick'
require_relative '../../lib/ggid/saml'

ggid_url = ENV['GGID_URL'] || 'http://localhost:8080'
entity_id = ENV['SP_ENTITY_ID'] || 'http://localhost:3001/saml/metadata'
acs_url = ENV['ACS_URL'] || 'http://localhost:3001/saml/acs'

server = WEBrick::HTTPServer.new(Port: 3001)

server.mount_proc '/' do |req, res|
  res.body = '<h1>GGID SAML Demo</h1><a href="/login">Login with SAML SSO</a>'
end

server.mount_proc '/saml/metadata' do |req, res|
  res['Content-Type'] = 'application/xml'
  res.body = GGID::SAML.generate_sp_metadata(entity_id: entity_id, acs_url: acs_url)
end

server.mount_proc '/login' do |req, res|
  sso_url = "#{ggid_url}/saml/sso"
  url = GGID::SAML.build_authn_request_url(sso_url: sso_url, entity_id: entity_id, acs_url: acs_url, relay_state: '/profile')
  res.set_redirect(WEBrick::HTTPStatus::Found, url)
end

server.mount_proc '/saml/acs' do |req, res|
  saml_response = req.query['SAMLResponse']
  decoded = Base64.decode64(saml_response)
  res.body = "<h1>SAML ACS</h1><pre>#{decoded}</pre><a href=\"/profile\">Continue</a>"
end

server.mount_proc '/profile' do |req, res|
  res.body = '<h1>Profile</h1><p>Authenticated via SAML SSO</p>'
end

trap('INT') { server.shutdown }
server.start
