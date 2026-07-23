#!/bin/bash
# GGID Bootstrap + ERP Demo Seed Script
# Usage: bash deploy/scripts/bootstrap-and-seed.sh
# Prerequisites: kubectl access to ggid namespace, GGID services running
set -euo pipefail

BASE_URL="${GGID_URL:-https://ggid-console.iot2.win}"
ADMIN_PASS="${ADMIN_PASS:-SecureAdmin@Pass2026#Xq}"
ERP_PASS="${ERP_PASS:-ErpDemo@2026Sec}"
NAMESPACE="${NAMESPACE:-ggid}"

echo "=== GGID Bootstrap + Seed ==="

# ── 1. Get pods ──
PGPOD=$(kubectl get pods -n $NAMESPACE | grep postgresql | grep Running | head -1 | awk '{print $1}')
IDENTITY_POD=$(kubectl get pods -n $NAMESPACE | grep ggid-identity | grep Running | head -1 | awk '{print $1}')

if [ -z "$PGPOD" ] || [ -z "$IDENTITY_POD" ]; then
  echo "ERROR: Cannot find PG or Identity pod"
  exit 1
fi

# ── 2. Bootstrap via identity service (bypasses gateway cache bug) ──
echo "[1/7] Bootstrapping admin + default tenant..."
RESULT=$(kubectl exec -n $NAMESPACE $IDENTITY_POD -- \
  wget -qO- --post-data="{\"tenant_name\":\"Default\",\"tenant_slug\":\"default\",\"admin_username\":\"admin\",\"admin_email\":\"admin@iot2.win\",\"admin_password\":\"$ADMIN_PASS\"}" \
  --header='Content-Type: application/json' \
  http://localhost:8080/api/v1/system/bootstrap 2>&1 || true)

if echo "$RESULT" | grep -q "success"; then
  echo "  ✅ Bootstrap successful"
elif echo "$RESULT" | grep -q "already"; then
  echo "  ⏭️ Already bootstrapped, skipping"
else
  echo "  ❌ Bootstrap failed: $RESULT"
  exit 1
fi

TENANT_ID=$(kubectl exec -n $NAMESPACE $PGPOD -- psql -U ggid -d ggid -t -c \
  "SELECT id FROM tenants WHERE slug='default' LIMIT 1;" 2>/dev/null | xargs)

# ── 3. Create ERP demo tenants ──
echo "[2/7] Creating ERP demo tenants..."
kubectl exec -n $NAMESPACE $PGPOD -- psql -U ggid -d ggid -c "
INSERT INTO tenants (id, name, slug, plan, status, max_users) VALUES
('1effd2c4-fc5a-4b2e-85b7-307bb4978bad', 'Go ERP Tenant', 'go-erp', 'enterprise', 'active', 10000),
('b1a2329f-223f-43bb-8cd1-4cdfa3d88570', 'Node ERP Tenant', 'node-erp', 'enterprise', 'active', 10000),
('c2bab17d-e3ce-4a6b-bd48-c3be1e62cf8e', 'Python ERP Tenant', 'python-erp', 'enterprise', 'active', 10000),
('536a18c2-dc0b-4889-853e-48f5e39356bd', 'C# ERP Tenant', 'csharp-erp', 'enterprise', 'active', 10000),
('8aa627c3-d760-4976-a7db-3309cdce41b4', 'Java ERP Tenant', 'java-erp', 'enterprise', 'active', 10000),
('a9a252cf-014f-4272-b2d5-5bcbc6b0126e', 'Ruby ERP Tenant', 'ruby-erp', 'enterprise', 'active', 10000),
('d8cc70a0-60dc-4bac-afc6-0c539d95931d', 'Rust ERP Tenant', 'rust-erp', 'enterprise', 'active', 10000)
ON CONFLICT (id) DO NOTHING;" 2>&1 | grep -v "^$" || true

# ── 4. Create OAuth clients ──
echo "[3/7] Creating OAuth clients..."
kubectl exec -n $NAMESPACE $PGPOD -- psql -U ggid -d ggid -c "
INSERT INTO oauth_clients (tenant_id, client_id, client_secret_hash, name, type, grant_types, response_types, redirect_uris, scopes, token_endpoint_auth_method, enabled) VALUES
('1effd2c4-fc5a-4b2e-85b7-307bb4978bad', 'erp-go-demo', '', 'Go ERP Demo', 'public', ARRAY['password','authorization_code','refresh_token'], ARRAY['code','token'], ARRAY[]::text[], ARRAY['erp_admin','offline_access','openid','profile','email'], 'none', true),
('b1a2329f-223f-43bb-8cd1-4cdfa3d88570', 'erp-node-m2m', '', 'Node ERP M2M', 'public', ARRAY['client_credentials'], ARRAY[]::text[], ARRAY[]::text[], ARRAY['erp_admin'], 'none', true),
('c2bab17d-e3ce-4a6b-bd48-c3be1e62cf8e', 'erp-python-demo', '', 'Python ERP Demo', 'public', ARRAY['password','authorization_code','refresh_token'], ARRAY['code','token'], ARRAY[]::text[], ARRAY['erp_admin','offline_access','openid','profile','email'], 'none', true),
('536a18c2-dc0b-4889-853e-48f5e39356bd', 'erp-csharp-demo', '', 'C# ERP Demo', 'public', ARRAY['password','authorization_code','refresh_token'], ARRAY['code','token'], ARRAY[]::text[], ARRAY['erp_admin','offline_access','openid','profile','email'], 'none', true),
('8aa627c3-d760-4976-a7db-3309cdce41b4', 'erp-java-demo', '', 'Java ERP Demo', 'public', ARRAY['password','authorization_code','refresh_token'], ARRAY['code','token'], ARRAY[]::text[], ARRAY['erp_admin','offline_access','openid','profile','email'], 'none', true),
('a9a252cf-014f-4272-b2d5-5bcbc6b0126e', 'erp-ruby-demo', '', 'Ruby ERP Demo', 'public', ARRAY['password','device_code','refresh_token'], ARRAY['code','token'], ARRAY[]::text[], ARRAY['erp_admin','offline_access','openid','profile','email'], 'none', true),
('d8cc70a0-60dc-4bac-afc6-0c539d95931d', 'erp-rust-exchange', '', 'Rust ERP Exchange', 'public', ARRAY['password','urn:ietf:params:oauth:grant-type:token-exchange','refresh_token'], ARRAY['code','token'], ARRAY[]::text[], ARRAY['erp_admin','offline_access','openid','profile','email'], 'none', true)
ON CONFLICT (tenant_id, client_id) DO NOTHING;" 2>&1 | grep -v "^$" || true

# ── 5. Create demo users via auth register API ──
echo "[4/7] Creating demo users..."
for entry in \
  "admin_go:admin_go@erp-demo.local:1effd2c4-fc5a-4b2e-85b7-307bb4978bad" \
  "viewer_go:viewer_go@erp-demo.local:1effd2c4-fc5a-4b2e-85b7-307bb4978bad" \
  "admin_node:admin_node@erp-demo.local:b1a2329f-223f-43bb-8cd1-4cdfa3d88570" \
  "admin_python:admin_python@erp-demo.local:c2bab17d-e3ce-4a6b-bd48-c3be1e62cf8e" \
  "admin_csharp:admin_csharp@erp-demo.local:536a18c2-dc0b-4889-853e-48f5e39356bd" \
  "admin_java:admin_java@erp-demo.local:8aa627c3-d760-4976-a7db-3309cdce41b4" \
  "admin_ruby:admin_ruby@erp-demo.local:a9a252cf-014f-4272-b2d5-5bcbc6b0126e" \
  "admin_rust:admin_rust@erp-demo.local:d8cc70a0-60dc-4bac-afc6-0c539d95931d"; do
  IFS=':' read -r username email tenant <<< "$entry"
  result=$(timeout 5 curl -s -X POST "$BASE_URL/api/v1/auth/register" \
    -H "Content-Type: application/json" -H "X-Tenant-ID: $tenant" \
    -d "{\"username\":\"$username\",\"email\":\"$email\",\"password\":\"$ERP_PASS\"}" 2>&1 || true)
  echo "$result" | grep -q "user_id\|created\|success" && echo "  ✅ $username" || \
  echo "$result" | grep -q "already\|exists\|conflict" && echo "  ⏭️ $username exists" || \
  echo "  ❌ $username: $result"
done

# ── 6. Create roles + permissions ──
echo "[5/7] Creating roles and permissions..."
ADMIN_ID=$(kubectl exec -n $NAMESPACE $PGPOD -- psql -U ggid -d ggid -t -c \
  "SELECT id FROM users WHERE username='admin' LIMIT 1;" 2>/dev/null | xargs)

# Create roles
kubectl exec -n $NAMESPACE $PGPOD -- psql -U ggid -d ggid -c "
INSERT INTO roles (tenant_id, key, name, system_role)
SELECT t.id, 'erp_admin', 'ERP Admin', true FROM tenants t WHERE t.slug LIKE '%-erp'
ON CONFLICT DO NOTHING;
INSERT INTO roles (tenant_id, key, name, system_role)
SELECT t.id, 'erp_viewer', 'ERP Viewer', true FROM tenants t WHERE t.slug LIKE '%-erp'
ON CONFLICT DO NOTHING;" 2>&1 | grep -v "^$" || true

# Create permissions
kubectl exec -n $NAMESPACE $PGPOD -- psql -U ggid -d ggid -c "
INSERT INTO permissions (tenant_id, key, name, resource_type, action, system_perm)
SELECT t.id, k.key, k.name, k.res, k.act, true
FROM tenants t
CROSS JOIN (VALUES
  ('inventory:read','Inventory Read','inventory','read'),
  ('inventory:write','Inventory Write','inventory','write'),
  ('inventory:delete','Inventory Delete','inventory','delete'),
  ('orders:read','Orders Read','orders','read'),
  ('orders:write','Orders Write','orders','write'),
  ('orders:approve','Orders Approve','orders','approve'),
  ('orders:read:all','Orders Read All','orders','read:all'),
  ('audit:read','Audit Read','audit','read'),
  ('dashboard:read','Dashboard Read','dashboard','read')
) AS k(key, name, res, act)
WHERE t.slug LIKE '%-erp'
ON CONFLICT DO NOTHING;" 2>&1 | grep -v "^$" || true

# Assign permissions to roles
kubectl exec -n $NAMESPACE $PGPOD -- psql -U ggid -d ggid -c "
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON p.tenant_id = r.tenant_id
WHERE r.key = 'erp_admin' AND p.tenant_id IN (SELECT id FROM tenants WHERE slug LIKE '%-erp')
ON CONFLICT DO NOTHING;
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r JOIN permissions p ON p.tenant_id = r.tenant_id
WHERE r.key = 'erp_viewer' AND p.key IN ('inventory:read','orders:read','audit:read','dashboard:read')
AND p.tenant_id IN (SELECT id FROM tenants WHERE slug LIKE '%-erp')
ON CONFLICT DO NOTHING;" 2>&1 | grep -v "^$" || true

# ── 7. Assign roles to users ──
echo "[6/7] Assigning roles to users..."
kubectl exec -n $NAMESPACE $PGPOD -- psql -U ggid -d ggid -c "
INSERT INTO user_roles (user_id, role_id, scope_type, scope_id, granted_by)
SELECT u.id, r.id, 'global', u.tenant_id, '$ADMIN_ID'::uuid
FROM users u JOIN roles r ON r.tenant_id = u.tenant_id AND r.key = 'erp_admin'
WHERE u.username LIKE 'admin_%' AND u.tenant_id != '$TENANT_ID'::uuid
ON CONFLICT DO NOTHING;
INSERT INTO user_roles (user_id, role_id, scope_type, scope_id, granted_by)
SELECT u.id, r.id, 'global', u.tenant_id, '$ADMIN_ID'::uuid
FROM users u JOIN roles r ON r.tenant_id = u.tenant_id AND r.key = 'erp_viewer'
WHERE u.username = 'viewer_go'
ON CONFLICT DO NOTHING;" 2>&1 | grep -v "^$" || true

# ── 8. Verify ──
echo "[7/7] Verification..."
echo "  Tenants: $(kubectl exec -n $NAMESPACE $PGPOD -- psql -U ggid -d ggid -t -c "SELECT count(*) FROM tenants;" 2>/dev/null | xargs)"
echo "  Users:   $(kubectl exec -n $NAMESPACE $PGPOD -- psql -U ggid -d ggid -t -c "SELECT count(*) FROM users;" 2>/dev/null | xargs)"
echo "  Clients: $(kubectl exec -n $NAMESPACE $PGPOD -- psql -U ggid -d ggid -t -c "SELECT count(*) FROM oauth_clients;" 2>/dev/null | xargs)"
echo ""
echo "=== Auth Quick Test ==="
TOKEN=$(timeout 5 curl -s -X POST "$BASE_URL/api/v1/oauth/token" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -H "X-Tenant-ID: 1effd2c4-fc5a-4b2e-85b7-307bb4978bad" \
  -d "grant_type=password&username=admin_go&password=$ERP_PASS&client_id=erp-go-demo&scope=erp_admin" 2>&1)
echo "$TOKEN" | grep -q "access_token" && echo "  ✅ admin_go auth OK" || echo "  ❌ admin_go auth failed"

echo ""
echo "=== Done ==="
echo "Admin login:  admin / $ADMIN_PASS"
echo "ERP demo password: $ERP_PASS"
echo "Tenant ID: $TENANT_ID"

# ── Console OAuth client (if identity bootstrap was used) ──
echo "[bonus] Creating console OAuth client..."
kubectl exec -n $NAMESPACE $PGPOD -- psql -U ggid -d ggid -c "
INSERT INTO oauth_clients (tenant_id, client_id, client_secret_hash, name, type, grant_types, response_types, redirect_uris, scopes, token_endpoint_auth_method, enabled)
SELECT '$TENANT_ID', 'gcid-console', '', 'GGID Console', 'public',
  ARRAY['password','authorization_code','refresh_token'],
  ARRAY['code','token'], ARRAY[]::text[],
  ARRAY['admin','openid','profile','email','offline_access'],
  'none', true
WHERE NOT EXISTS (SELECT 1 FROM oauth_clients WHERE tenant_id='$TENANT_ID')
ON CONFLICT DO NOTHING;" 2>&1 | grep -v "^$" || true
