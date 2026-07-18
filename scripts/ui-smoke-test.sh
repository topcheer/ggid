#!/bin/bash
# scripts/ui-smoke-test.sh — Batch HTTP 200 verification for all console pages.
# Usage: bash scripts/ui-smoke-test.sh [CONSOLE_URL]
# Output: docs/test/ui-smoke-report.md + stdout summary

set -euo pipefail

CONSOLE_URL="${1:-https://ggid-console.iot2.win}"
REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PAGES_FILE="$(mktemp)"
REPORT_DIR="${REPO_ROOT}/docs/test"
REPORT_FILE="${REPORT_DIR}/ui-smoke-report.md"

# Extract all page paths from console/src/app
find "${REPO_ROOT}/console/src/app" -name "page.tsx" | \
  sed "s|${REPO_ROOT}/console/src/app||; s|/page.tsx||" | sort > "$PAGES_FILE"

TOTAL=$(wc -l < "$PAGES_FILE" | tr -d ' ')
PASS=0
FAIL=0
BROKEN=0
ERRORS=""

mkdir -p "$REPORT_DIR"

# Start report
cat > "$REPORT_FILE" << 'EOF'
# UI Smoke Test Report

Automated HTTP status verification of all console pages.

EOF
echo "**Console URL:** ${CONSOLE_URL}" >> "$REPORT_FILE"
echo "**Date:** $(date -u '+%Y-%m-%d %H:%M:%S UTC')" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

# Test each page
while IFS= read -r path; do
  # Skip empty paths
  [ -z "$path" ] && continue
  
  HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${CONSOLE_URL}${path}" 2>/dev/null || echo "000")
  
  case "$HTTP_CODE" in
    200|302|307)
      PASS=$((PASS + 1))
      ;;
    404)
      FAIL=$((FAIL + 1))
      ERRORS="${ERRORS}\n| 404 | ${path} |"
      ;;
    500|502|503)
      BROKEN=$((BROKEN + 1))
      ERRORS="${ERRORS}\n| ${HTTP_CODE} | ${path} |"
      ;;
    000)
      BROKEN=$((BROKEN + 1))
      ERRORS="${ERRORS}\n| TIMEOUT | ${path} |"
      ;;
    *)
      BROKEN=$((BROKEN + 1))
      ERRORS="${ERRORS}\n| ${HTTP_CODE} | ${path} |"
      ;;
  esac
done < "$PAGES_FILE"

# Write summary
cat >> "$REPORT_FILE" << EOF
## Summary

| Status | Count | Percentage |
|--------|-------|------------|
| ✅ PASS (200/302) | ${PASS} | $(echo "scale=1; ${PASS} * 100 / ${TOTAL}" | bc)% |
| ❌ FAIL (404) | ${FAIL} | $(echo "scale=1; ${FAIL} * 100 / ${TOTAL}" | bc)% |
| ⚠ BROKEN (5xx/timeout) | ${BROKEN} | $(echo "scale=1; ${BROKEN} * 100 / ${TOTAL}" | bc)% |
| **Total Pages** | **${TOTAL}** | 100% |

## Failed Pages

| Status | Path |
|--------|------|
EOF

echo -e "$ERRORS" >> "$REPORT_FILE"

echo ""
echo "=== UI Smoke Test Complete ==="
echo "Total: ${TOTAL} | PASS: ${PASS} | FAIL(404): ${FAIL} | BROKEN: ${BROKEN}"
echo "Report: ${REPORT_FILE}"

# Cleanup
rm -f "$PAGES_FILE"
