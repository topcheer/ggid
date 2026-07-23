#!/bin/bash
# GGID Security Scan Runner
# Runs a suite of automated security checks for CI and manual use.
# Usage: ./deploy/scripts/security-scan.sh [mode]
# Modes: quick (SAST only), full (SAST + deps + config), ci (non-interactive CI mode)

set -euo pipefail

MODE="${1:-quick}"
ROOT_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
REPORT_DIR="${ROOT_DIR}/.security-reports"
mkdir -p "$REPORT_DIR"

echo "=== GGID Security Scan ($MODE mode) ==="
echo ""

# 1. Go vet + static analysis
echo "[1] Go vet..."
cd "$ROOT_DIR"
go vet ./services/... 2>&1 | tee "$REPORT_DIR/govet.txt" || true
VET_ISSUES=$(grep -c "vet:" "$REPORT_DIR/govet.txt" 2>/dev/null || echo "0")
echo "    Go vet: $VET_ISSUES issues"

# 2. Gosec (Go Security) — SAST
if command -v gosec &>/dev/null || go install github.com/securego/gosec/v2/cmd/gosec@latest 2>/dev/null; then
  echo "[2] Gosec SAST..."
  gosec -quiet -fmt=json -out="$REPORT_DIR/gosec.json" ./services/... ./pkg/... 2>/dev/null || true
  SECREC=$(python3 -c "
import json
try:
    with open('$REPORT_DIR/gosec.json') as f:
        d = json.load(f)
    stats = d.get('Stats', {})
    print(f\"HIGH:{stats.get('High', 0)} MEDIUM:{stats.get('Medium', 0)} LOW:{stats.get('Low', 0)}\")
except: print('parse-error')
" 2>/dev/null || echo "scan-error")
  echo "    Gosec: $SECREC"
else
  echo "[2] Gosec: skipped (install failed)"
fi

# 3. Dependency vulnerabilities (govulncheck)
if command -v govulncheck &>/dev/null || go install golang.org/x/vuln/cmd/govulncheck@latest 2>/dev/null; then
  echo "[3] Govulncheck..."
  govulncheck ./... 2>&1 | tee "$REPORT_DIR/govulncheck.txt" || true
  VULNS=$(grep -c "Vulnerability" "$REPORT_DIR/govulncheck.txt" 2>/dev/null || echo "0")
  echo "    Vulnerabilities: $VULNS"
else
  echo "[3] Govulncheck: skipped (install failed)"
fi

# 4. Secret scanning (basic pattern match)
echo "[4] Secret scan..."
SECRET_PATTERNS="api_key\s*=\s*['\"][a-zA-Z0-9]{20,}|password\s*=\s*['\"][^'\"]{6,}|secret\s*=\s*['\"][a-zA-Z0-9]{20,}|BEGIN\s+(RSA|OPENSSH|EC)\s+PRIVATE\s+KEY"
SECRETS_FOUND=$(grep -rE "$SECRET_PATTERNS" "$ROOT_DIR/services/" "$ROOT_DIR/pkg/" --include="*.go" -l 2>/dev/null | grep -v _test.go | wc -l | xargs)
echo "    Potential hardcoded secrets: $SECRETS_FOUND"

if [ "$SECRETS_FOUND" -gt 0 ]; then
  echo "    Files with potential secrets:"
  grep -rE "$SECRET_PATTERNS" "$ROOT_DIR/services/" "$ROOT_DIR/pkg/" --include="*.go" -l 2>/dev/null | grep -v _test.go | head -5
fi

# 5. Full mode: dependency audit + config check
if [ "$MODE" = "full" ] || [ "$MODE" = "ci" ]; then
  echo "[5] Dependency audit..."
  go list -json -m all 2>/dev/null | python3 -c "
import sys, json
deps = []
buf = ''
for line in sys.stdin:
    buf += line
    if line.strip() == '}':
        try:
            d = json.loads(buf)
            if d.get('Path') and d.get('Version'):
                deps.append(f\"{d['Path']}@{d['Version']}\")
        except: pass
        buf = ''
print(f'    Total dependencies: {len(deps)}')
" 2>/dev/null || echo "    Dependency count: error"

  echo "[6] Container config check..."
  for dockerfile in "$ROOT_DIR"/deploy/docker/*/Dockerfile "$ROOT_DIR"/Dockerfile; do
    if [ -f "$dockerfile" ]; then
      if grep -q "USER root" "$dockerfile" 2>/dev/null; then
        echo "    WARNING: $dockerfile runs as root"
      fi
      if ! grep -q "USER " "$dockerfile" 2>/dev/null; then
        echo "    WARNING: $dockerfile has no USER directive"
      fi
    fi
  done

  echo "[7] Helm security check..."
  for values in "$ROOT_DIR"/deploy/helm/ggid/values*.yaml; do
    if grep -q "password.*:" "$values" 2>/dev/null; then
      # Check for hardcoded passwords (not referencing secrets)
      HARDCODED=$(grep -E "password:\s+[^$<{]" "$values" | grep -v "change\|dev\|secure\|MUST\|set\|production\|password" | wc -l | xargs)
      if [ "$HARDCODED" -gt 0 ]; then
        echo "    INFO: $values has $HARDCODED password-like values (review for prod)"
      fi
    fi
  done
fi

# Summary
echo ""
echo "=== Summary ==="
echo "Go vet issues: $VET_ISSUES"
echo "Potential secrets: $SECRETS_FOUND"
if [ -n "${VULNS:-}" ]; then echo "Go vulnerabilities: $VULNS"; fi
echo "Reports saved to: $REPORT_DIR/"

# Exit code for CI
if [ "$MODE" = "ci" ]; then
  [ "$SECRETS_FOUND" -gt 0 ] && exit 1
  [ "${VULNS:-0}" -gt 0 ] && exit 1
fi
exit 0
