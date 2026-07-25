#!/usr/bin/env bash
# GGID MCP stdio proxy — auto-refreshes OAuth2 token via DCR client_credentials.
# Configured as a stdio MCP server in ggcode; forwards JSON-RPC to remote HTTP MCP.
#
# Usage in ggcode config:
#   mcp_servers.ggid = { "name":"ggid", "type":"stdio",
#     "command":"bash", "args":["/Volumes/new/ggai/ggid/scripts/ggid-mcp-proxy.sh"] }

set -euo pipefail

GGID_OAUTH_URL="${GGID_OAUTH_URL:-https://ggid.iot2.win}"
GGID_MCP_URL="${GGID_MCP_URL:-https://mcp.iot2.win/mcp}"
GGID_TENANT_ID="${GGID_TENANT_ID:-fb44ca98-2a8a-498b-a9b2-00fc014524ce}"
GGID_CLIENT_ID="${GGID_CLIENT_ID:-gcid_qQ3Ju0GK8FTuwTW0K_9qAw}"
GGID_CLIENT_SECRET="${GGID_CLIENT_SECRET:-gcs_dWCjHl2KYYNLCW7VK3P51-t8l-FX4DxFLoxMGrQbsUw}"

CACHED_TOKEN=""
TOKEN_EXPIRES_AT=0

refresh_token() {
  local now
  now=$(date +%s)
  if [[ -n "$CACHED_TOKEN" && "$now" -lt "$TOKEN_EXPIRES_AT" ]]; then
    return 0
  fi

  local resp
  resp=$(curl -sf -X POST "${GGID_OAUTH_URL}/oauth/token" \
    -H "Content-Type: application/x-www-form-urlencoded" \
    -H "X-Tenant-ID: ${GGID_TENANT_ID}" \
    -d "grant_type=client_credentials&client_id=${GGID_CLIENT_ID}&client_secret=${GGID_CLIENT_SECRET}&scope=admin" 2>/dev/null)

  CACHED_TOKEN=$(echo "$resp" | python3 -c "import sys,json;print(json.load(sys.stdin).get('access_token',''))" 2>/dev/null || echo "")
  local expires_in
  expires_in=$(echo "$resp" | python3 -c "import sys,json;print(json.load(sys.stdin).get('expires_in',900))" 2>/dev/null || echo "900")

  if [[ -z "$CACHED_TOKEN" ]]; then
    echo '{"jsonrpc":"2.0","error":{"code":-32603,"message":"Failed to refresh OAuth2 token"}}' >&2
    return 1
  fi

  TOKEN_EXPIRES_AT=$((now + expires_in - 60))
}

while IFS= read -r line; do
  refresh_token || continue

  echo "$line" | curl -sf -X POST "${GGID_MCP_URL}" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer ${CACHED_TOKEN}" \
    -d @- 2>/dev/null || echo '{"jsonrpc":"2.0","error":{"code":-32603,"message":"MCP request failed"}}'
done
