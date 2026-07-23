#!/bin/bash
# GGID Instance Reset — clears all data back to fresh boot state
# Usage: bash deploy/scripts/reset-instance.sh
set -euo pipefail
NAMESPACE="${NAMESPACE:-ggid}"

echo "=== GGID Instance Reset ==="

PGPOD=$(kubectl get pods -n $NAMESPACE | grep postgresql | grep Running | head -1 | awk '{print $1}')
REDIS_POD=$(kubectl get pods -n $NAMESPACE | grep redis | grep Running | head -1 | awk '{print $1}')

if [ -z "$PGPOD" ]; then echo "ERROR: No PG pod found"; exit 1; fi

echo "[1/3] Truncating all business data..."
kubectl exec -n $NAMESPACE $PGPOD -- psql -U ggid -d ggid -c "
DO \$\$
DECLARE r RECORD;
BEGIN
  FOR r IN (SELECT tablename FROM pg_tables WHERE schemaname='public' AND tablename NOT IN ('schema_migrations')) LOOP
    EXECUTE 'TRUNCATE TABLE ' || quote_ident(r.tablename) || ' CASCADE';
  END LOOP;
END \$\$;" 2>&1 | grep -v NOTICE || true

echo "[2/3] Flushing Redis..."
kubectl exec -n $NAMESPACE $REDIS_POD -- redis-cli FLUSHALL 2>&1

echo "[3/3] Restarting gateway to reset in-memory state..."
GW_POD=$(kubectl get pods -n $NAMESPACE | grep ggid-gateway | grep Running | head -1 | awk '{print $1}')
if [ -n "$GW_POD" ]; then
  kubectl delete pod -n $NAMESPACE $GW_POD 2>&1
  echo "  Waiting for new gateway pod..."
  sleep 25
fi

# Verify
echo ""
echo "=== Verify ==="
kubectl exec -n $NAMESPACE $PGPOD -- psql -U ggid -d ggid -t -c "
SELECT 'tenants', count(*) FROM tenants
UNION ALL SELECT 'users', count(*) FROM users
UNION ALL SELECT 'oauth_clients', count(*) FROM oauth_clients;" 2>&1
echo ""
echo "✅ Instance reset complete. Run bootstrap-and-seed.sh to initialize."
