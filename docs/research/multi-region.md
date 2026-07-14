# Multi-Region Deployment Architecture

## Overview

This document analyzes multi-region deployment patterns for GGID — active-active, active-passive, read replicas, conflict resolution, and latency-based routing.

> **Related**: zero-downtime deployment guide

## Region Topologies

### Option A: Single Region (Current Default)

```
                ┌─────────────────────┐
                │   Region: us-east-1 │
                │  ┌───────────────┐  │
                │  │  GGID Stack   │  │
                │  │  (7 services) │  │
                │  └───────────────┘  │
                │  ┌───────────────┐  │
                │  │  PostgreSQL   │  │
                │  │  Redis  NATS  │  │
                │  └───────────────┘  │
                └─────────────────────┘
```

**RTO**: N/A (single point of failure)
**RPO**: N/A
**Best for**: Development, staging, small deployments

### Option B: Active-Passive (Warm Standby)

```
Region A (Primary)              Region B (Standby)
┌──────────────────┐          ┌──────────────────┐
│ GGID (Active)    │          │ GGID (Stopped)   │
│ PostgreSQL (RW)  │── WAL ──→│ PostgreSQL (RO)  │
│ Redis (Primary)  │── Repl ─→│ Redis (Replica)  │
│ NATS (Primary)   │──────────│ NATS (Standby)   │
└──────────────────┘          └──────────────────┘
```

**RTO**: 10-30 min (DNS failover + promote replica)
**RPO**: Near-zero (async streaming replication)
**Best for**: Standard production, cost-conscious HA

**Failover procedure**:
1. Stop traffic to Region A
2. Promote Region B PostgreSQL replica to primary
3. Start GGID services in Region B
4. DNS failover to Region B
5. Verify health endpoints

### Option C: Active-Active

```
Region A                        Region B
┌──────────────────┐          ┌──────────────────┐
│ GGID (Active)    │          │ GGID (Active)    │
│ PostgreSQL (RW)  │← Logical→│ PostgreSQL (RW)  │
│ Redis (Local)    │  Repl    │ Redis (Local)    │
│ NATS (Local)     │          │ NATS (Local)     │
└──────────────────┘          └──────────────────┘
```

**RTO**: < 5 min (traffic shift only)
**RPO**: 0 (both regions accept writes)
**Best for**: Global applications requiring low latency worldwide

**Challenges**:
- Conflict resolution for concurrent writes
- JWT key distribution
- Session sharing across regions

## Conflict Resolution

### Write Conflicts

With active-active, two regions may modify the same record simultaneously:

| Strategy | Description | GGID Suitability |
|----------|-------------|------------------|
| Last-write-wins | Higher timestamp wins | User profile fields |
| Versioned (optimistic) | Client sends version, reject if stale | API updates |
| CRDT (merge) | Automatic merge for compatible types | Session counters |
| Tenant affinity | Each tenant has a home region | Best approach |

### Recommended: Tenant Affinity

Each tenant has a "home" region. All writes go to the home region. Other regions serve reads from replicas:

```
Tenant-A (home: EU) → EU Gateway → EU Database (RW)
Tenant-B (home: US) → US Gateway → US Database (RW)

Read in non-home region → served from replica (eventual consistency)
```

This eliminates write conflicts entirely.

## Read Replicas

### PostgreSQL Streaming Replication

```ini
# Primary (postgresql.conf)
wal_level = logical
max_wal_senders = 10
hot_standby = on

# Replica (postgresql.auto.conf)
primary_conninfo = 'host=primary-host port=5432 user=repl'
```

### Read Routing in GGID

```go
func (s *UserService) GetUser(ctx context.Context, id uuid.UUID) (*User, error) {
    // Reads go to replica (eventual consistency OK)
    return s.replicaRepo.GetByID(ctx, id)
}

func (s *UserService) CreateUser(ctx context.Context, u *User) error {
    // Writes go to primary (strong consistency)
    return s.primaryRepo.Create(ctx, u)
}
```

### Replication Lag Monitoring

```sql
-- On primary
SELECT client_addr, state,
       (sent_lsn - replay_lsn) AS lag_bytes
FROM pg_stat_replication;

-- On replica
SELECT now() - pg_last_xact_replay_timestamp() AS lag_seconds;
```

**Alert**: Lag > 5 seconds indicates replication bottleneck.

## Latency-Based Routing

### Global Load Balancer

```
User Request → Global Load Balancer (Route53 / Cloudflare / GSLB)
                  ↓
            Health check both regions
                  ↓
         Route to lowest-latency healthy region
```

### DNS Configuration (Route53)

```json
{
  "RoutingPolicy": "latency",
  "Records": [
    {"Region": "us-east-1", "Value": "us.ggid.example.com"},
    {"Region": "eu-west-1", "Value": "eu.ggid.example.com"},
    {"Region": "ap-southeast-1", "Value": "ap.ggid.example.com"}
  ],
  "HealthCheck": {
    "Type": "HTTPS",
    "ResourcePath": "/healthz"
  }
}
```

## Session Management Across Regions

### Problem

Sessions stored in Redis are region-local. A user in EU has a session in EU Redis. If routed to US, the session doesn't exist.

### Solutions

| Approach | Description | Trade-off |
|----------|-------------|-----------|
| Sticky sessions | Route user to same region | Simple, but breaks if region fails |
| Redis replication | Replicate session keys globally | High latency, complex |
| Stateless JWT | No server-side session needed | GGID already does this! |
| Session in DB | Store sessions in PostgreSQL (replicated) | Higher latency per request |

**GGID approach**: JWT is stateless — no server-side session lookup needed. jti validation uses Redis but is best-effort (token works for its lifetime even if Redis is unavailable).

## JWT Key Distribution

### Problem

Each region's auth service signs JWTs. The gateway in each region must verify tokens from ALL regions.

### Solution: Aggregated JWKS

```
Region A Auth → publishes JWKS-A
Region B Auth → publishes JWKS-B

Global JWKS Aggregator:
  GET /.well-known/jwks.json
  → Returns merged keys from all regions

Each Gateway:
  Fetches global JWKS
  Can verify tokens from any region
```

## NATS JetStream Multi-Region

### Leaf Node Federation

```
Region A NATS          Region B NATS
  (Leaf Node) ←──────→  (Leaf Node)
       │                      │
   AUDIT_EVENTS          AUDIT_EVENTS
   (local stream)        (local stream)
```

Audit events stay in-region but can be replicated to a central analytics cluster via NATS leaf nodes.

## Cost Analysis (2-Region)

| Component | Single Region | Active-Passive | Active-Active |
|-----------|--------------|----------------|---------------|
| Compute (7 services) | $2,000/mo | $2,500/mo | $4,000/mo |
| PostgreSQL | $500/mo | $700/mo | $1,200/mo |
| Redis | $200/mo | $300/mo | $400/mo |
| NATS | $300/mo | $400/mo | $600/mo |
| Cross-region bandwidth | $0 | $100/mo | $300/mo |
| Global DNS/load balancer | $0 | $50/mo | $100/mo |
| **Total** | **$3,000/mo** | **$4,050/mo** | **$6,600/mo** |

## Deployment Checklist

- [ ] Region deployment automated (Terraform/Pulumi)
- [ ] Database replication configured and tested
- [ ] DNS latency routing configured
- [ ] Health checks per region
- [ ] JWT key distribution working
- [ ] Audit events routed correctly
- [ ] Failover tested (quarterly drill)
- [ ] Replication lag monitored

## See Also

- [Disaster Recovery](disaster-recovery.md)
- [Data Sovereignty](data-sovereignty.md)
- zero-downtime deployment guide
- [Performance Tuning](../guides/performance-tuning.md)
