# Blue-Green Deployment

Zero-downtime switch, DB migration handling, session draining, traffic switch, instant rollback, and cost considerations.

## Architecture

```
┌──────────┐
│ Router    │
└────┬─────┘
     │
     ├── Green (active) ← current production
     └── Blue (idle)    ← next version being prepared
```

## Deployment Steps

### Step 1: Deploy Blue

```bash
# Deploy new version to idle environment
kubectl apply -f deploy/blue/ --recursive
# Blue gets 0% traffic, but pods are running and healthy
```

### Step 2: Smoke Test Blue

```bash
# Route test traffic to blue
kubectl patch virtualservice ggid --type=json \
  -p='[{"op":"replace","path":"/spec/http/0/route/0/destination/subset","value":"blue"}]'

# Run smoke tests
curl https://blue.ggid.dev/healthz
bash tests/smoke.sh

# Route back to green (no user impact yet)
```

### Step 3: Switch Traffic

```bash
# Instant switch: 100% green → 100% blue
kubectl patch virtualservice ggid --type=json \
  -p='[{"op":"replace","path":"/spec/http/0/route/0/destination/subset","value":"blue"}]'

# OR gradual (canary-style)
# 10% blue → 50% blue → 100% blue
```

### Step 4: Drain Green

```bash
# Wait for green sessions to complete (graceful)
kubectl scale deploy/ggid-green --replicas=0
# Pods receive SIGTERM, finish in-flight requests, then stop
```

### Rollback (Instant)

```bash
# If blue has issues, switch back to green instantly
kubectl patch virtualservice ggid --type=json \
  -p='[{"op":"replace","path":"/spec/http/0/route/0/destination/subset","value":"green"}]'
# Green is still running (idle), traffic returns immediately
```

## DB Migration Handling

Blue and green share the same database. Use expand-contract:

```
Deploy Blue:
  1. Blue expects new schema (expanded)
  2. Green still works with new columns (backward compatible)
  3. Switch traffic to Blue
  4. Remove old schema (contract) in next deploy
```

## Session Draining

```yaml
graceful_shutdown:
  termination_grace_period: 60s
  pre_stop:
    - "kubectl patch virtualservice ... remove-green-from-lb"
    - "sleep 15"  # Let load balancer update
  # Pod receives SIGTERM → stops accepting new requests → finishes in-flight → exits
```

## Cost Considerations

| Period | Resources | Cost |
|--------|-----------|------|
| Normal | Green only | 1× |
| Deploy window (~10 min) | Green + Blue | 2× |
| Post-deploy | Blue (new green) + idle old green | 1.5× (keep old for rollback) |
| Cleanup (after 1h stable) | New green only | 1× |

**Recommendation:** Keep old green for 1 hour as rollback target, then scale to 0.

## Monitoring

| Metric | Alert |
|--------|-------|
| Blue health check failures | Pre-switch → abort |
| Post-switch error rate | >1% → instant rollback |
| Session drain timeout | Pods not terminating → force kill |
| Deploy duration | >10 min → investigate |

## See Also

- [Canary Deployment Strategy](canary-deployment-strategy.md)
- [Zero-Downtime Deployment](zero-downtime-deployment.md)
- [Database Migration Playbook](database-migration-playbook.md)
- [Feature Flag Architecture](feature-flag-architecture.md)
