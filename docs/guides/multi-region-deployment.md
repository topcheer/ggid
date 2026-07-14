# Multi-Region Deployment Guide

Guide for active-active vs active-passive deployment, failover, and data residency in GGID.

## Deployment Topologies

### Active-Active

```
         Global Load Balancer (GeoDNS)
          ┌──────┴──────┐
          ▼             ▼
     US-East-1     EU-West-1
     (Full stack)  (Full stack)
          │             │
          └──────┬──────┘
                 ▼
          Cross-Region Sync
          (async replication)
```

| Advantage | Detail |
|-----------|--------|
| Zero RTO | Both regions serve traffic |
| Load distribution | Users route to nearest region |
| Graceful degradation | If one region fails, other absorbs |

| Challenge | Mitigation |
|-----------|-----------|
| Data conflicts | Last-write-wins + CRDTs for counters |
| Latency | Async replication, regional read replicas |
| Complexity | Automated failover orchestration |

### Active-Passive

```
         Global Load Balancer (GeoDNS)
          ┌──────┴──────┐
          ▼             ▼
     US-East-1     EU-West-1
     (Active)      (Standby)
     100% traffic  0% (hot standby)
```

| Advantage | Detail |
|-----------|--------|
| Simplicity | No write conflicts |
| Cost | Lower (standby scaled down) |
| Predictability | Clear primary |

| Disadvantage | Detail |
|--------------|--------|
| RTO > 0 | Minutes to promote standby |
| Waste | Standby resources idle |

## Data Residency

### Region-Affinity Rules

```yaml
data_residency:
  eu_users:
    storage_region: "eu-west-1"
    replication: "eu-only"  # GDPR: EU data stays in EU
    condition: "user.country in EU_REGIONS"

  us_users:
    storage_region: "us-east-1"
    replication: "us-only"
    condition: "user.country == 'US'"

  default:
    storage_region: "us-east-1"
    replication: "cross-region"
```

Users are pinned to their home region at registration. Cross-region reads use read-only replicas within legal boundaries.

### Audit Log Residency

```
EU audit data → EU PostgreSQL (7-year retention)
US audit data → US PostgreSQL (7-year retention)
Cross-region: NEVER (regulatory requirement)
```

## RPO / RTO Targets

| Scenario | RPO | RTO |
|----------|-----|-----|
| Single AZ failure | 0 | <1min |
| Single region failure | <1min | <5min |
| Active-Active failover | 0 | 0 |
| Active-Passive promotion | <30s | <5min |
| Disaster (region lost) | <5min | <15min |

RPO = data loss tolerance. RTO = recovery time.

## Cross-Region Sync

### PostgreSQL (Logical Replication)

```sql
-- Publisher (primary region)
CREATE PUBLICATION ggid_pub FOR TABLE users, roles, orgs;

-- Subscriber (secondary region)
CREATE SUBSCRIPTION ggid_sub
  CONNECTION 'host=us-east-1 ...'
  PUBLICATION ggid_pub
  WITH (copy_data = true, synchronous_commit = off);
```

Replication is async for performance. Critical tables can use sync commit.

### NATS JetStream (Event Mirroring)

```yaml
# Mirror events to secondary region
mirror:
  source: "nats://primary-nats:4222"
  stream: "EVENTS"
  destination: "EVENTS_MIRROR"
  mode: async
```

### Redis (Replication)

```
Primary (us-east-1) → Replica (eu-west-1)
# Async replication, sub-second lag
```

## Failover DNS

```yaml
# Route53 health check + failover
record_sets:
  - name: "auth.ggid.dev"
    type: "A"
    set_identifier: "primary"
    failover: "PRIMARY"
    health_check_id: "hc-auth-primary"
    ttl: 30
    records: ["1.2.3.4"]

  - name: "auth.ggid.dev"
    type: "A"
    set_identifier: "secondary"
    failover: "SECONDARY"
    ttl: 30
    records: ["5.6.7.8"]
```

DNS TTL is 30 seconds — failover completes in under 1 minute.

## Failover Procedure

### Automated (Active-Active)

1. Health check fails in region A (3 consecutive failures)
2. DNS shifts traffic to region B
3. Region B absorbs full load
4. Region A isolated for investigation
5. Alert security + ops teams

### Manual (Active-Passive)

```bash
# 1. Verify primary is truly down
ggid region status --region us-east-1

# 2. Promote secondary
ggid region promote --region eu-west-1 --force

# 3. Update DNS
ggid dns failover --from us-east-1 --to eu-west-1

# 4. Verify
ggid health check --region eu-west-1

# Total RTO: <5 minutes
```

## Latency Optimization

| Technique | Implementation | Savings |
|-----------|---------------|---------|
| GeoDNS routing | Route users to nearest region | -50ms RTT |
| Regional read replicas | Read from local replica | -40ms |
| CDN for static assets | Cloudflare edge caching | -100ms |
| Connection pooling | Keep gRPC channels warm | -20ms |
| Async writes | NATS for non-critical writes | -30ms |

## Monitoring

| Metric | Alert |
|--------|-------|
| Cross-region replication lag | >5s (PostgreSQL), >1s (Redis) |
| Region health check failures | >3 consecutive |
| Failover events | Any → page ops |
| Regional traffic imbalance | >70/30 split → investigate |
| Data residency violations | Any → critical alert |

## See Also

- [Disaster Recovery](disaster-recovery.md)
- [High Availability](high-availability.md)
- [Database Security](database-security.md)
- [Secrets Rotation Automation](secrets-rotation-automation.md)
