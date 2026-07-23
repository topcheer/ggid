# GGID Operations Runbook

## Deployment

### Install (first time)
```bash
# Development
helm install ggid ./deploy/helm/ggid -f deploy/helm/ggid/values-dev.yaml

# Staging
helm install ggid ./deploy/helm/ggid -f deploy/helm/ggid/values-staging.yaml

# Production (set secrets!)
helm install ggid ./deploy/helm/ggid -f deploy/helm/ggid/values-prod.yaml \
  --set jwt.secret=$(openssl rand -hex 32) \
  --set postgresql.auth.password=$(openssl rand -hex 16) \
  --set config.passwordPepper=$(openssl rand -hex 32) \
  --set config.auditHashSecret=$(openssl rand -hex 32)
```

### Upgrade
```bash
# Pull latest, build new images, then:
helm upgrade ggid ./deploy/helm/ggid -f deploy/helm/ggid/values-prod.yaml

# Verify rollout:
kubectl rollout status deployment/ggid-gateway -n ggid
kubectl rollout status deployment/ggid-auth -n ggid
```

## Rollback

### Helm Rollback
```bash
# List revisions
helm history ggid -n ggid

# Rollback to previous revision
helm rollback ggid 0 -n ggid

# Rollback to specific revision
helm rollback ggid <REVISION> -n ggid
```

### Image Rollback (per-service)
```bash
# Pin a specific image tag for rollback
kubectl set image deployment/ggid-gateway gateway=registry.iot2.win/ggid/gateway:<PREVIOUS_TAG> -n ggid
kubectl rollout status deployment/ggid-gateway -n ggid

# OAuth uses :v2 tag (not :latest):
kubectl set image deployment/ggid-oauth oauth=registry.iot2.win/ggid/oauth:v<PREVIOUS> -n ggid
```

### Database Rollback (migrations)
```bash
# Apply down migration
kubectl exec -n ggid deploy/ggid-postgresql -- psql -U ggid -d ggid -f /app/migrations/NNN_down.sql

# WARNING: Down migrations can cause data loss. Always backup first:
kubectl exec -n ggid deploy/ggid-postgresql -- pg_dump -U ggid ggid > backup_$(date +%Y%m%d).sql
```

## Scaling

### Manual Scaling
```bash
# Scale a single service
kubectl scale deployment ggid-gateway --replicas=5 -n ggid
kubectl scale deployment ggid-auth --replicas=4 -n ggid

# Scale all services at once
for svc in gateway auth identity oauth policy audit; do
  kubectl scale deployment ggid-$svc --replicas=4 -n ggid
done
```

### Autoscaling (HPA)
```bash
# Check HPA status
kubectl get hpa -n ggid

# Adjust HPA limits
kubectl autoscale deployment ggid-gateway --min=3 --max=20 --cpu-percent=60 -n ggid
```

### Database Connection Scaling
```bash
# Check PostgreSQL connection count
kubectl exec -n ggid deploy/ggid-postgresql -- psql -U ggid -d ggid -c \
  "SELECT count(*), state FROM pg_stat_activity GROUP BY state;"

# Increase max_connections (requires restart)
kubectl exec -n ggid deploy/ggid-postgresql -- psql -U ggid -d ggid -c \
  "ALTER SYSTEM SET max_connections = '500';"
kubectl rollout restart statefulset/ggid-postgresql -n ggid
```

## Troubleshooting

### Service Down
```bash
# 1. Check pod status
kubectl get pods -n ggid --no-headers | grep -v Running

# 2. Check pod logs
kubectl logs -n ggid <POD_NAME> -c <CONTAINER> --tail=50

# 3. Check service connectivity
kubectl exec -n ggid deploy/ggid-gateway -- curl -s http://ggid-auth:9001/healthz/live

# 4. Check database connectivity
kubectl exec -n ggid deploy/ggid-auth -- pg_isready -h ggid-postgresql -U ggid
```

### Login Failures (Credential Sync)
```bash
# Symptom: admin login returns invalid_grant after auth pod restart
# Cause: bootstrap credential hash race condition
# Fix: reset password hash
HASH=$(go run -mod=mod ./scripts/gen_hash.go)
kubectl exec -n ggid deploy/ggid-postgresql -- psql -U ggid -d ggid -c \
  "UPDATE credentials SET secret = '$HASH' WHERE identifier = 'admin';"
```

### RBAC Permission Issues
```bash
# Check if RBAC resolver is loaded
kubectl logs -n ggid -l app=ggid-gateway -c gateway | grep "RBAC resolver"

# Flush RBAC cache in Redis
kubectl exec -n ggid deploy/ggid-redis -- redis-cli DEL ggid:rbac:snapshot

# Restart gateway to reload RBAC snapshot
kubectl rollout restart deployment/ggid-gateway -n ggid
```

## Backup & Restore

### Database Backup
```bash
# Full backup
kubectl exec -n ggid deploy/ggid-postgresql -- pg_dump -U ggid ggid | gzip > ggid_$(date +%Y%m%d_%H%M).sql.gz

# Restore
gunzip -c ggid_20260723_1200.sql.gz | kubectl exec -n ggid -i deploy/ggid-postgresql -- psql -U ggid -d ggid
```

### Redis Backup
```bash
kubectl exec -n ggid deploy/ggid-redis -- redis-cli SAVE
kubectl cp ggid/<REDIS_POD>:/data/dump.rdb ./redis_backup_$(date +%Y%m%d).rdb
```

## Health Checks

```bash
# All services
for svc in gateway auth identity oauth policy audit console; do
  echo -n "$svc: "
  kubectl exec -n ggid deploy/ggid-$svc -- curl -s http://localhost:8080/healthz/live 2>/dev/null || echo "(check port)"
done

# Deep health check
curl -s https://ggid.iot2.win/healthz/deep | jq .
```
