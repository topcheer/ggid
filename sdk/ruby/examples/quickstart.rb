#!/usr/bin/env ruby
# frozen_string_literal: true
#
# GGID Ruby SDK — Quick Start Example (Sinatra)
#
# Run: ruby examples/quickstart.rb
# (Requires: gem install ggid sinatra)
#
# Or as a Sinatra app:
#   ruby examples/quickstart.rb
#   Then visit: http://localhost:4567

require "ggid"
require "json"

# ─── Configuration ──────────────────────────────────────────────────
BASE_URL = ENV.fetch("GGID_BASE_URL", "https://ggid.iot2.win")
TENANT_ID = ENV.fetch("GGID_TENANT_ID", "00000000-0000-0000-0000-000000000001")
USERNAME = ENV.fetch("GGID_USERNAME", "admin")
PASSWORD = ENV.fetch("GGID_PASSWORD", "Admin@123456")

puts "=== GGID Ruby SDK Quick Start ==="
puts

# ─── 1. Initialize ──────────────────────────────────────────────────
ggid = GGID::Client.new(base_url: BASE_URL, tenant_id: TENANT_ID)
puts "1. Client initialized: #{BASE_URL}"

# ─── 2. Login ───────────────────────────────────────────────────────
begin
  tokens = ggid.login(USERNAME, PASSWORD)
  puts "2. Login successful! Access token: #{tokens['access_token'][0..19]}..."
rescue GGID::ApiError => e
  puts "2. Login failed: #{e.message}"
  exit 1
end

access_token = tokens["access_token"]

# ─── 3. Get User Info ───────────────────────────────────────────────
begin
  user_info = ggid.get_user_info(access_token)
  puts "3. User info: #{user_info.sub} (#{user_info.email})"
rescue GGID::Error => e
  puts "3. UserInfo failed: #{e.message}"
end

# ─── 4. Check Permission ────────────────────────────────────────────
begin
  result = ggid.check_permission(access_token, "products", "read")
  puts "4. Permission check (products:read): #{result.allowed ? 'ALLOWED' : 'DENIED'}"
  puts "   Reason: #{result.reason}" unless result.allowed
rescue GGID::Error => e
  puts "4. Permission check failed: #{e.message}"
end

# ─── 5. List Roles ──────────────────────────────────────────────────
begin
  roles = ggid.list_roles(access_token)
  puts "5. Roles found: #{roles.length}"
  roles.first(5).each { |r| puts "   - #{r.name} (key: #{r.key})" }
rescue GGID::Error => e
  puts "5. List roles failed: #{e.message}"
end

# ─── 6. List Users ──────────────────────────────────────────────────
begin
  users = ggid.list_users(access_token)
  puts "6. Users found: #{users.length}"
  users.first(5).each { |u| puts "   - #{u['username'] || u['email'] || 'unknown'}" }
rescue GGID::Error => e
  puts "6. List users failed: #{e.message}"
end

# ─── 7. ABAC Evaluation ─────────────────────────────────────────────
begin
  abac_result = ggid.evaluate_abac(
    access_token,
    action: "transfer",
    resource: "inventory",
    subject: tokens["user_id"] || "user-001",
    conditions: [{ field: "warehouse", operator: "eq", value: "WH-001" }],
  )
  puts "7. ABAC evaluation (inventory:transfer): #{abac_result.allowed ? 'ALLOWED' : 'DENIED'}"
  if abac_result.allowed && abac_result.matched_rules.any?
    puts "   Matched rules: #{abac_result.matched_rules.join(', ')}"
  end
rescue GGID::Error => e
  puts "7. ABAC evaluation failed: #{e.message}"
end

# ─── 8. Audit Events ────────────────────────────────────────────────
begin
  events = ggid.list_audit_events(access_token, limit: 5)
  count = events.is_a?(Array) ? events.length : 0
  puts "8. Audit events: #{count} recent"
rescue GGID::Error => e
  puts "8. Audit query failed: #{e.message}"
end

puts
puts "=== Done! ==="
