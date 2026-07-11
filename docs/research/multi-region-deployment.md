# Multi-Region Active-Active Deployment

## Status: PROPOSED (P2)

## Problem

GGID currently runs in a single region. For production IAM, multi-region is required for:
- Disaster recovery (RTO < 5 min, RPO < 1 min)
- Geographic latency reduction
- Compliance with data sovereignty laws

## Architecture

### Active-Active with Geo-Replicated PostgreSQL

```
                    ┌─────────────┐
                    │  Global LB  │
                    │ (Route 53)  │
                    └──┬────┬────┘
           ┌──────────┘    └──────────┐
           ▼                          ▼
    ┌──────────────┐          ┌──────────────┐
    │ Region: US   │          │ Region: EU   │
    │ Gateway ×3   │          │ Gateway ×3   │
    │ Auth ×3      │          │ Auth ×3      │
    │ Identity ×3  │          │ Identity ×3  │
    │ OAuth ×3     │          │ OAuth ×3     │
    │ Policy ×3    │          │ Policy ×3    │
    │ Org ×3       │          │ Org ×3       │
    │ Audit ×3     │          │ Audit ×3     │
    └──────┬───────┘          └──────┬───────┘
           │                         │
    ┌──────┴───────┐          ┌──────┴───────┐
    │ PostgreSQL   │◄────────►│ PostgreSQL   │
    │ (Primary)    │  Logical │ (Primary)    │
    │              │  Replication            │
    └──────────────┘          └──────────────┘
```

### Data Layer

| Component | Strategy | RPO |
|-----------|----------|-----|
| PostgreSQL | Bidirectional logical replication (pglogical) | < 1s |
| Redis | Redis CRDT (Active-Active) or region-local cache | 0s |
| NATS | JetStream mirror | < 1s |
| LDAP | Multi-master replication | < 1s |

### Tenant Affinity

- Tenants assigned a "home region" for data residency
- Global Load Balancer routes by tenant: `X-Tenant-ID` → region
- Cross-region reads allowed, writes pinned to home region
- Failover: secondary region becomes primary for affected tenants

### JWT Key Sharing

All regions share the same JWT signing key (via Vault/KMS). JWKS endpoint is globally consistent.

### Audit Log Replication

- Audit events published to NATS JetStream
- Each region consumes all events
- Local audit tables in each region, with global query via gateway fan-out

## Conflict Resolution

- Last-write-wins for user/org profile fields
- CRDT for session state (Redis ACTIVE_ACTIVE)
- UUID primary keys prevent conflicts on inserts
- Sequence-based optimistic concurrency on updates

## Deployment

### Kubernetes

```yaml
# values-us.yaml
region: us-east-1
replicas: 3
database:
  host: pg-us-primary.example.com
  mode: primary

# values-eu.yaml
region: eu-west-1
replicas: 3
database:
  host: pg-eu-primary.example.com
  mode: primary
  replicationPeer: pg-us-primary.example.com
```

### Health Checking

- Gateway health checks include database replication lag
- If lag > 5s, LB drains traffic to affected region
- Automatic failover when primary database unreachable > 30s

## Compliance

- GDPR: EU tenants' data stays in EU region
- CCPA: US tenants can request data deletion across all regions
- Data residency enforced at gateway level via tenant routing
