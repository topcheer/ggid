#!/bin/bash
# scripts/gen-swagger.sh — Auto-generate OpenAPI spec from HandleFunc routes.
# Scans all services/*/internal/server/*.go for HandleFunc paths and
# generates a base OpenAPI 3.1 YAML at deploy/openapi.yaml.
#
# Usage: make swagger (or ./scripts/gen-swagger.sh)

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUTPUT="$REPO_ROOT/deploy/openapi.yaml"

echo ">> Scanning service routes..."

# Collect all unique API paths from HandleFunc calls across all services.
PATHS=$(grep -rh 'HandleFunc("' \
  "$REPO_ROOT"/services/*/internal/server/http.go \
  "$REPO_ROOT"/services/*/internal/server/server.go \
  2>/dev/null \
  | sed 's/.*HandleFunc("//; s/".*//' \
  | grep -E '^/api|^/oauth|^/\.well-known|^/health|^/ready|^/graphql' \
  | sort -u)

COUNT=$(echo "$PATHS" | wc -l | tr -d ' ')
echo ">> Found $COUNT unique paths"

# Generate YAML
cat > "$OUTPUT" << 'HEADER'
openapi: 3.1.0
info:
  title: GGID Platform API
  description: |
    GGID — Global Governance & Identity Platform API.
    Auto-generated from service route registration.
  version: 1.0.0
  contact:
    name: GGID Support
    url: https://github.com/topcheer/ggid
servers:
  - url: http://localhost:8080
    description: Local development
security:
  - BearerAuth: []
paths:
HEADER

# Determine tag from path
tag_for() {
  local path="$1"
  case "$path" in
    /api/v1/auth/*)  echo "Auth" ;;
    /api/v1/users*)  echo "Identity" ;;
    /api/v1/groups*) echo "Identity" ;;
    /api/v1/identity/*) echo "Identity" ;;
    /api/v1/orgs*|/api/v1/departments*|/api/v1/teams*) echo "Org" ;;
    /api/v1/oauth/*|/oauth/*|/.well-known/*) echo "OAuth" ;;
    /api/v1/roles*|/api/v1/permissions*|/api/v1/policies*|/api/v1/policy*) echo "Policy" ;;
    /api/v1/audit/*) echo "Audit" ;;
    /api/v1/admin/*) echo "Admin" ;;
    /health*|/ready*) echo "System" ;;
    /graphql*) echo "GraphQL" ;;
    *) echo "Other" ;;
  esac
}

# Generate path entries
echo "$PATHS" | while IFS= read -r path; do
  [ -z "$path" ] && continue
  tag=$(tag_for "$path")
  # Human-readable summary from last path segment
  summary=$(echo "$path" | sed 's/^.*\///' | tr '-' ' ' | awk '{for(i=1;i<=NF;i++) $i=toupper(substr($i,1,1)) substr($i,2)}1')
  [ "$summary" = "$path" ] && summary="API endpoint"

  # Keep original path with braces
  yaml_path="$path"

  echo "  $yaml_path:" >> "$OUTPUT"
  echo "    get:" >> "$OUTPUT"
  echo "      tags: [$tag]" >> "$OUTPUT"
  echo "      summary: $summary" >> "$OUTPUT"
  echo "      security: [{BearerAuth: []}]" >> "$OUTPUT"
  echo "      responses:" >> "$OUTPUT"
  echo "        '200': { description: OK }" >> "$OUTPUT"
done

# Components
cat >> "$OUTPUT" << 'FOOTER'
components:
  securitySchemes:
    BearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT
FOOTER

FINAL_COUNT=$(grep -c "^  /" "$OUTPUT")
echo ">> Generated $OUTPUT with $FINAL_COUNT paths"
