# GGID OAuth 2.0 Demo (Ruby)
# Run: GGID_URL=... CLIENT_ID=... ruby oauth_demo.rb
require 'webrick'
require 'net/http'
require 'json'
require 'uri'

ggid_url = ENV['GGID_URL'] || 'http://localhost:8080'
client_id = ENV['CLIENT_ID'] || ''
client_secret = ENV['CLIENT_SECRET'] || ''
redirect_uri = ENV['REDIRECT_URI'] || 'http://localhost:3000/callback'

server = WEBrick::HTTPServer.new(Port: 3000)

server.mount_proc '/' do |req, res|
  auth_url = "#{ggid_url}/api/v1/oauth/authorize?" + URI.encode_www_form(
    response_type: 'code', client_id: client_id, redirect_uri: redirect_uri,
    scope: 'openid profile email', state: 'demo'
  )
  res.body = "<h1>GGID OAuth Demo</h1><a href='#{auth_url}'>Login with GGID</a>"
end

server.mount_proc '/callback' do |req, res|
  code = req.query['code']
  uri = URI("#{ggid_url}/api/v1/oauth/token")
  tokens = JSON.parse(Net::HTTP.post_form(uri, {
    grant_type: 'authorization_code', code: code,
    redirect_uri: redirect_uri, client_id: client_id, client_secret: client_secret,
  }).body)

  user_uri = URI("#{ggid_url}/api/v1/oauth/userinfo")
  user_req = Net::HTTP::Get.new(user_uri)
  user_req['Authorization'] = "Bearer #{tokens['access_token']}"
  user = JSON.parse(Net::HTTP.start(user_uri.host, user_uri.port, use_ssl: user_uri.scheme == 'https') { |h| h.request(user_req) }.body)

  res['Content-Type'] = 'application/json'
  res.body = JSON.pretty_generate(tokens: tokens, user: user)
end

trap('INT') { server.shutdown }
server.start
