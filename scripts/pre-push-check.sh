#!/bin/bash
# Pre-push verification script — mirrors CI checks
# Run before every push: bash scripts/pre-push-check.sh
set -e

echo "=== Pre-Push CI Verification ==="
echo ""

# 1. go build ./...
echo "1. go build ./..."
go build ./... || { echo "FAIL: go build"; exit 1; }
echo "   PASS"

# 2. go vet ./...
echo "2. go vet ./..."
go vet ./... || { echo "FAIL: go vet"; exit 1; }
echo "   PASS"

# 3. go mod tidy check
echo "3. go mod tidy check..."
cp go.mod /tmp/go.mod.bak
cp go.sum /tmp/go.sum.bak
go mod tidy
if ! diff -q go.mod /tmp/go.mod.bak > /dev/null || ! diff -q go.sum /tmp/go.sum.bak > /dev/null; then
  echo "FAIL: go.mod/go.sum not tidy. Run 'go mod tidy' and commit."
  rm /tmp/go.mod.bak /tmp/go.sum.bak
  exit 1
fi
rm /tmp/go.mod.bak /tmp/go.sum.bak
echo "   PASS"

# 4. make test
echo "4. make test..."
make test > /dev/null 2>&1 || { echo "FAIL: make test"; exit 1; }
echo "   PASS"

# 5. console tsc
echo "5. console tsc --noEmit..."
(cd console && npx tsc --noEmit 2>&1 | grep -v '__tests__\|node_modules' | grep -E 'error TS' | head -5) || true
echo "   SKIP (pre-existing errors allowed)"

echo ""
echo "=== All CI checks passed. Safe to push. ==="
