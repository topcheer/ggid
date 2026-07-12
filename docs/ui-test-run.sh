#!/bin/bash
# GGID UI Automation Test - Sections 7, 14, 15, 16
TENANT="00000000-0000-0000-0000-000000000001"
BASE="https://ggid.iot2.win"

echo "=========================================="
echo "GGID UI Automation Test Suite"
echo "Sections: 7(Audit), 14(Webhooks), 15(SIEM), 16(SoD)"
echo "=========================================="

# Register
echo ""
echo "=== 1. Register ==="
curl -s -H 'Accept-Encoding: identity' -H 'Content-Type: application/json' -H "X-Tenant-ID: $TENANT" \
  -X POST "$BASE/api/v1/auth/register" \
  -d '{"username":"uitest9901","email":"uitest9901@test.com","password":"TestPass1234!"}'
echo ""

sleep 3

# Login
echo "=== 2. Login ==="
LOGIN=$(curl -s -H 'Accept-Encoding: identity' -H 'Content-Type: application/json' -H "X-Tenant-ID: $TENANT" \
  -X POST "$BASE/api/v1/auth/login" \
  -d '{"username":"uitest9901","password":"TestPass1234!"}')
echo "$LOGIN" | head -c 300
echo ""

TOKEN=$(echo "$LOGIN" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('access_token') or d.get('token',''))" 2>/dev/null)
echo "TOKEN length: ${#TOKEN}"

if [ -z "$TOKEN" ]; then
  echo "FAIL: No token obtained"
  exit 1
fi

echo ""
echo "=== Section 7: Audit Log ==="

echo "--- 7.1 GET /api/v1/audit/events ---"
curl -s -H 'Accept-Encoding: identity' -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT" \
  "$BASE/api/v1/audit/events?limit=5" | head -c 500
echo ""

echo "--- 7.2 GET /api/v1/audit/events (filter event_type=auth) ---"
curl -s -H 'Accept-Encoding: identity' -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT" \
  "$BASE/api/v1/audit/events?event_type=auth&limit=5" | head -c 500
echo ""

echo "--- 7.3 GET /api/v1/audit/hash-chain ---"
curl -s -H 'Accept-Encoding: identity' -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT" \
  "$BASE/api/v1/audit/hash-chain" | head -c 500
echo ""

echo ""
echo "=== Section 15: SIEM & Compliance ==="

echo "--- 15.1 GET /api/v1/siem/health ---"
curl -s -H 'Accept-Encoding: identity' -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT" \
  "$BASE/api/v1/siem/health" | head -c 500
echo ""

echo "--- 15.2 GET /api/v1/compliance/schedules ---"
curl -s -H 'Accept-Encoding: identity' -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT" \
  "$BASE/api/v1/compliance/schedules" | head -c 500
echo ""

echo ""
echo "=== Section 16: SoD ==="

echo "--- 16.1 GET /api/v1/sod/rules ---"
curl -s -H 'Accept-Encoding: identity' -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT" \
  "$BASE/api/v1/sod/rules" | head -c 500
echo ""

echo "--- 16.2 GET /api/v1/sod/conflicts ---"
curl -s -H 'Accept-Encoding: identity' -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT" \
  "$BASE/api/v1/sod/conflicts" | head -c 500
echo ""

echo ""
echo "=== Section 14: Webhooks & Notifications ==="

echo "--- 14.1 GET /api/v1/webhooks ---"
curl -s -H 'Accept-Encoding: identity' -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT" \
  "$BASE/api/v1/webhooks" | head -c 500
echo ""

echo ""
echo "=== DONE ==="
