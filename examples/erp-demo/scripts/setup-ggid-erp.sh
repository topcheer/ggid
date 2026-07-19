#!/bin/bash
#
# GGID ERP Setup Script
#
# Creates 6 ERP roles, permissions for 8 modules, and demo users.
# Run this AFTER the GGID system is bootstrapped (admin user exists).
#
# Usage:
#   GGID_URL=https://ggid.iot2.win ADMIN_PASS="q7Rf9Xk2Lm3pW8zBA" bash setup-ggid-erp.sh
#

set -euo pipefail

GGID_URL="${GGID_URL:-https://ggid.iot2.win}"
ADMIN_USER="${ADMIN_USER:-admin}"
ADMIN_PASS="${ADMIN_PASS:-q7Rf9Xk2Lm3pW8zBA}"
TENANT_ID="${TENANT_ID:-00000000-0000-0000-0000-000000000001}"
ERP_PASS="${ERP_PASS:-ErpDemo2024!}"

echo "=========================================="
echo "  GGID ERP Setup"
echo "=========================================="
echo "Gateway:  $GGID_URL"
echo "Tenant:   $TENANT_ID"
echo ""

# --- Step 1: Login as admin ---
echo "[1/5] Logging in as admin..."
LOGIN_RESP=$(curl -sf -X POST "$GGID_URL/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d "{\"username\": \"$ADMIN_USER\", \"password\": \"$ADMIN_PASS\"}")

ADMIN_TOKEN=$(echo "$LOGIN_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin).get('access_token',''))")
if [ -z "$ADMIN_TOKEN" ]; then
  echo "FAILED: Could not get admin token. Check credentials."
  exit 1
fi
echo "  Admin token acquired."

# --- Step 2: Create ERP roles ---
echo ""
echo "[2/5] Creating ERP roles..."

create_role() {
  local key="$1" name="$2" desc="$3"
  echo "  - Creating role: $name ($key)"
  curl -sf -X POST "$GGID_URL/api/v1/roles" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "X-Tenant-ID: $TENANT_ID" \
    -H "Content-Type: application/json" \
    -d "{\"key\": \"$key\", \"name\": \"$name\", \"description\": \"$desc\"}" 2>/dev/null || true
}

create_role "erp:sales_manager"     "Sales Manager"      "Manage orders, customers, view inventory"
create_role "erp:warehouse_manager"  "Warehouse Manager"   "Manage inventory, fulfill orders"
create_role "erp:finance_officer"    "Finance Officer"     "Manage invoices, payments, view orders"
create_role "erp:hr_manager"         "HR Manager"          "Manage employees, view reports"
create_role "erp:production_manager" "Production Manager"  "Manage production, view inventory"
create_role "erp:system_admin"       "ERP System Admin"    "Full access to all ERP modules"

# --- Step 3: Create permissions ---
echo ""
echo "[3/5] Creating ERP permissions..."

create_perm() {
  local resource="$1" action="$2"
  local key="${resource}:${action}"
  echo "  - Creating permission: $key"
  curl -sf -X POST "$GGID_URL/api/v1/permissions" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "X-Tenant-ID: $TENANT_ID" \
    -H "Content-Type: application/json" \
    -d "{\"key\": \"$key\", \"name\": \"$resource $action\", \"resource_type\": \"$resource\", \"action\": \"$action\"}" 2>/dev/null || true
}

MODULES="inventory orders customers invoices payments employees production reports"
ACTIONS="read write delete approve"

for module in $MODULES; do
  for action in $ACTIONS; do
    create_perm "$module" "$action"
  done
done

# --- Step 4: Grant permissions to roles ---
echo ""
echo "[4/5] Granting permissions to roles..."
echo "  (See README.md for the permission matrix)"
echo "  Grant permissions via GGID Console > Roles > [Role] > Permissions"
echo "  Or use the PUT /api/v1/roles/{id}/permissions API"

# --- Step 5: Create demo users ---
echo ""
echo "[5/5] Creating demo users..."

create_user() {
  local username="$1" email="$2" role_key="$3"
  echo "  - Creating user: $username ($role_key)"
  USER_RESP=$(curl -sf -X POST "$GGID_URL/api/v1/users" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "X-Tenant-ID: $TENANT_ID" \
    -H "Content-Type: application/json" \
    -d "{\"username\": \"$username\", \"email\": \"$email\", \"password\": \"$ERP_PASS\"}" 2>/dev/null) || true
  
  USER_ID=$(echo "$USER_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin).get('id',''))" 2>/dev/null || echo "")
  
  if [ -n "$USER_ID" ]; then
    # Find role ID by key
    ROLE_RESP=$(curl -sf -X GET "$GGID_URL/api/v1/roles" \
      -H "Authorization: Bearer $ADMIN_TOKEN" \
      -H "X-Tenant-ID: $TENANT_ID" 2>/dev/null || echo "")
    ROLE_ID=$(echo "$ROLE_RESP" | python3 -c "
import sys,json
data = json.load(sys.stdin)
items = data if isinstance(data, list) else data.get('data', data.get('items', []))
for r in items:
    if r.get('key') == '$role_key':
        print(r.get('id',''))
        break
" 2>/dev/null || echo "")
    
    if [ -n "$ROLE_ID" ]; then
      curl -sf -X POST "$GGID_URL/api/v1/users/$USER_ID/roles" \
        -H "Authorization: Bearer $ADMIN_TOKEN" \
        -H "X-Tenant-ID: $TENANT_ID" \
        -H "Content-Type: application/json" \
        -d "{\"role_id\": \"$ROLE_ID\"}" 2>/dev/null || true
      echo "    Assigned role: $role_key"
    fi
  fi
}

create_user "sales_manager"      "sales@erp-demo.local"     "erp:sales_manager"
create_user "warehouse_manager"   "warehouse@erp-demo.local"  "erp:warehouse_manager"
create_user "finance_officer"     "finance@erp-demo.local"    "erp:finance_officer"
create_user "hr_manager"          "hr@erp-demo.local"         "erp:hr_manager"
create_user "production_manager"  "production@erp-demo.local" "erp:production_manager"
create_user "erp_admin"           "admin@erp-demo.local"      "erp:system_admin"

echo ""
echo "=========================================="
echo "  Setup Complete!"
echo "=========================================="
echo ""
echo "Demo users created with password: $ERP_PASS"
echo ""
echo "Next steps:"
echo "  1. Build the ERP app:  cd examples/erp-demo && mvn spring-boot:run"
echo "  2. Open:              http://localhost:8090"
echo "  3. Login with any demo user above"
echo ""
echo "Permission Matrix (configure in GGID Console):"
echo "  sales_manager:      orders(r/w/a), customers(r/w/d), inventory(r), reports(r)"
echo "  warehouse_manager:  inventory(r/w/d), orders(r/w), reports(r)"
echo "  finance_officer:    invoices(r/w/a/d), payments(r/w/a), orders(r), reports(r)"
echo "  hr_manager:         employees(r/w/d), reports(r)"
echo "  production_manager: production(r/w/a), inventory(r), reports(r)"
echo "  erp_admin:          ALL modules ALL actions"
