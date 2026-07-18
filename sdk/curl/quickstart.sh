#!/bin/bash
# GGID curl Quickstart — 5 minute integration
#
# Prerequisites: GGID running on localhost:8080
#
# Usage: bash quickstart.sh
set -euo pipefail

API="http://localhost:8080"

echo "=== 1. Register a new user ==="
REG=$(curl -s -X POST "$API/api/v1/auth/register" \
  -H "Content-Type: application/json" \
  -d '{"email":"alice@company.com","password":"SecurePass#123","display_name":"Alice"}')
echo "$REG" | head -c 200; echo

echo "=== 2. Login ==="
TOKEN=$(curl -s -X POST "$API/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"email":"alice@company.com","password":"SecurePass#123"}' | jq -r '.access_token')
echo "✓ Token: ${TOKEN:0:20}..."

echo "=== 3. Get profile ==="
curl -s "$API/api/v1/users/me" -H "Authorization: Bearer $TOKEN" | jq '{email, display_name}'

echo "=== 4. List users ==="
curl -s "$API/api/v1/users" -H "Authorization: Bearer $TOKEN" | jq 'length' | xargs -I{} echo "✓ {} users found"

echo "=== 5. Logout ==="
curl -s -X POST "$API/api/v1/auth/logout" -H "Authorization: Bearer $TOKEN" | jq '.'
echo "✓ Done!"
