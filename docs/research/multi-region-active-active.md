# Multi-Region Active-Active Deployment & Geo-Distribution for GGID

> **Focus**: Production architecture for multi-region active-active deployment — PostgreSQL logical replication, Redis cross-region sync, CRDT conflict resolution, geo-routing, and data residency compliance.
>
> **Author**: ggcxf (researcher) | **Date**: 2026-07-17 | **Status**: Research Complete
>
> **Related**: `multi-region.md` (252 lines, theory), `multi-region-deployment.md` (105 lines, basic design). This document covers the **production implementation**.
>
> **Checklist Compliance**: Endpoint precondition check (§7), DoD per backlog item (§8).

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Why Multi-Region](#2-why-multi-region)
3. [GGID Current State: Single Region](#3-ggid-current-state-single-region)
4. [Gap Analysis](#4-gap-analysis)
5. [Active-Active Architecture](#5-active-active-architecture)
6. [PostgreSQL Multi-Region Replication](#6-postgresql-multi-region-replication)
7. [Redis Cross-Region Strategy](#7-redis-cross-region-strategy)
8. [Geo-Routing & Traffic Management](#8-geo-routing--traffic-management)
9. [Conflict Resolution](#9-conflict-resolution)
10. [Data Residency](#10-data-residency)
11. [Endpoint Precondition Check](#11-endpoint-precondition-check)
12. [API Design + Curl Commands](#12-api-design--curl-commands)
13. [Database Schema Changes](#13-database-schema-changes)
14. [Performance Analysis](#14-performance-analysis)
15. [Implementation Backlog with DoD](#15-implementation-backlog-with-dod)
16. [Competitive Differentiation](#16-competitive-differentiation)
17. [Security Considerations](#17-security-considerations)

---

## 1. Executive Summary

Multi-region active-active deployment means GGID runs simultaneously in multiple geographic regions (e.g., US-East + EU-West + AP-Southeast), with all regions serving traffic and replicating data. This provides:
- **Zero-downtime region failover** (< 30 seconds)
- **Geo-latency reduction** (users hit nearest region)
- **Data residency compliance** (EU data stays in EU)
- **Disaster recovery** (entire region loss = no data loss)

GGID currently runs **single-region only**:
- Single PostgreSQL instance (no replication) ❌
- Single Redis instance (no cluster) ❌
- Single NATS instance ❌
- Single k3s cluster ❌
- No geo-routing ❌
- No cross-region data sync ❌

**Recommendation**: Implement phased multi-region starting with **active-passive** (read replica + failover), then advance to **active-active** (logical replication + conflict resolution). The PostgreSQL RLS tenant isolation (already implemented on 27 tables) is a prerequisite — it enables per-tenant data placement.

**Estimated effort**: 4 sprints for active-passive, 4 more for active-active.

---

## 2. Why Multi-Region

### Business Drivers

| Driver | Impact | Urgency |
|--------|--------|---------|
| **EU GDPR data residency** | EU user data must stay in EU region | High (compliance) |
| **China PIPL compliance** | China data must stay in China region | High (compliance) |
| **Global latency** | US users <100ms, APAC users <300ms | Medium |
| **DR** | Entire region failure = 0 downtime | Medium |
| **Enterprise SLA** | 99.99% uptime requires multi-region | High (enterprise sales) |

### Latency by Geography (single region in US-East)

```
User Location      → US-East    → EU-West    → AP-Southeast
US West Coast       80ms         170ms        200ms
EU (Frankfurt)     90ms          20ms        180ms
China (Shanghai)   220ms        160ms         40ms
Australia          180ms        280ms         30ms
```

With multi-region geo-routing: every user hits <60ms latency.

---

## 3. GGID Current State: Single Region

### Current Infrastructure

| Component | Status | Multi-Region Ready? |
|-----------|--------|-------------------|
| PostgreSQL | Single instance | ❌ No replication |
| Redis | Single instance | ❌ No cluster/replica |
| NATS | Single instance | ❌ No geo-replication |
| K8s (k3s) | Single cluster | ❌ Single region |
| Gateway | Single instance | ❌ No geo-routing |
| CMK/KMS | 7 providers | ⚠️ Per-region keys |
| Audit hash chain | HMAC per-tenant | ✅ Region-independent |

### Current Deployment (docker-compose / k3s)

```yaml
# deploy/docker-compose.yml (simplified)
services:
  postgres:     # Single instance, no replica
  redis:        # Single instance, no cluster
  nats:         # Single instance
  gateway:      # Single instance
  identity:     # Single instance
  auth:         # Single instance
  ...
```

---

## 4. Gap Analysis

| # | Gap | Impact |
|---|-----|--------|
| 1 | No PG replication | Can't failover, no data redundancy |
| 2 | No Redis cluster | Cache is single point of failure |
| 3 | No geo-routing | All users hit single region |
| 4 | No data residency | EU/China data in US region (non-compliant) |
| 5 | No conflict resolution | Active-active requires merge logic |
| 6 | No cross-region health checks | Can't detect regional failures |
| 7 | No region-aware client routing | SDK doesn't know nearest region |
| 8 | No async event replication | NATS events not cross-region |

---

## 5. Active-Active Architecture

```
                    ┌─────────────────────────────────────────┐
                    │         Global Load Balancer             │
                    │         (GeoDNS / Anycast)               │
                    │                                         │
                    │  Routes user to nearest region based    │
                    │  on GeoIP. Health-checks each region.    │
                    └────────┬──────────────┬─────────────────┘
                             │              │
                    ┌────────▼────┐  ┌──────▼───────┐  ┌────────────┐
                    │  US-East    │  │  EU-West     │  │ AP-SE      │
                    │  (primary)  │  │  (active)    │  │ (active)   │
                    │             │  │              │  │            │
                    │ ┌─────────┐ │  │ ┌─────────┐  │  │ ┌────────┐ │
                    │ │ Gateway │ │  │ │ Gateway │  │  │ │Gateway │ │
                    │ │ Identity│ │  │ │ Identity│  │  │ │Identity│ │
                    │ │ Auth    │ │  │ │ Auth    │  │  │ │Auth    │ │
                    │ │ OAuth   │ │  │ │ OAuth   │  │  │ │OAuth   │ │
                    │ │ Policy  │ │  │ │ Policy  │  │  │ │Policy  │ │
                    │ │ Audit   │ │  │ │ Audit   │  │  │ │Audit   │ │
                    │ └────┬────┘ │  │ └────┬────┘  │  │ └───┬────┘ │
                    │      │      │  │      │       │  │     │      │
                    │ ┌────▼────┐ │  │ ┌────▼────┐  │  │ ┌───▼────┐ │
                    │ │PostgreSQL│ │  │ │PostgreSQL│  │  │ │PostgreSQL│
                    │ │(primary)│  │  │ │(active) │  │  │ │(active)│  │
                    │ └────┬────┘ │  │ └────┬────┘  │  │ └───┬────┘ │
                    └──────┼──────┘  └──────┼───────┘  └─────┼──────┘
                           │                │                 │
                    ┌──────▼────────────────▼─────────────────▼──────┐
                    │         Logical Replication Mesh               │
                    │                                               │
                    │  Bidirectional logical replication:            │
                    │  US ↔ EU ↔ AP                                 │
                    │  Per-tenant: EU tenant data only in EU region  │
                    │  Per-table: audit_events replicated all→all    │
                    └───────────────────────────────────────────────┘
```

### Phased Approach

| Phase | Mode | Description | Duration |
|-------|------|-------------|----------|
| **Phase 1** | Active-Passive | PG streaming replication (US→EU read replica) | 2 sprints |
| **Phase 2** | Active-Passive + Failover | Automated failover (Patroni) | 1 sprint |
| **Phase 3** | Active-Active (read) | Both regions serve reads, writes to primary | 2 sprints |
| **Phase 4** | Active-Active (write) | Bidirectional logical replication + CRDT | 3 sprints |

---

## 6. PostgreSQL Multi-Region Replication

### Phase 1: Streaming Replication (Physical)

```ini
# Primary (US-East)
wal_level = replica
max_wal_senders = 10
archive_mode = on

# Standby (EU-West)
primary_conninfo = 'host=us-east-pg primary user=replicator password=...'
hot_standby = on                  # Allow read queries on standby
```

**Properties**: Exact binary copy, schema must match, async (RPO ~seconds), one-way (primary→standby).

### Phase 4: Logical Replication (Bidirectional)

```sql
-- On US-East: create subscription to EU-West
CREATE SUBSCRIPTION eu_west_sub
  CONNECTION 'host=eu-west-pg user=replicator'
  PUBLICATION eu_west_pub
  WITH (copy_data = false, create_slot = true);

-- On EU-West: create subscription to US-East
CREATE SUBSCRIPTION us_east_sub
  CONNECTION 'host=us-east-pg user=replicator'
  PUBLICATION us_east_pub
  WITH (copy_data = false, create_slot = true);

-- Publication: all tables except system tables
CREATE PUBLICATION us_east_pub FOR ALL TABLES;
```

### Per-Tenant Data Placement

```sql
-- EU tenant: data only in EU region
-- Using publication filters (PostgreSQL 16+)

-- EU-West publishes only EU tenants to US (for DR read-only):
CREATE PUBLICATION eu_tenant_dr FOR TABLE users
  WHERE (tenant_id IN (SELECT id FROM tenants WHERE region = 'eu'));
```

### Conflict Resolution

| Conflict Type | Resolution Strategy |
|---------------|-------------------|
| **Insert-Insert (same PK)** | Use region-prefixed UUIDs (no collision) |
| **Update-Update (same row)** | Last-Write-Wins (LWW) with timestamp |
| **Delete-Update** | Update wins (delete rejected if row was modified) |
| **Foreign key** | Use DEFERRABLE constraints |

```sql
-- Conflict resolution trigger
CREATE OR REPLACE FUNCTION resolve_conflict()
RETURNS TRIGGER AS $$
BEGIN
  -- Last-write-wins: compare updated_at timestamps
  IF NEW.updated_at < OLD.updated_at THEN
    RETURN OLD;  -- Keep existing (newer) version
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;
```

---

## 7. Redis Cross-Region Strategy

### Options

| Strategy | Latency | Complexity | Use Case |
|----------|---------|-----------|----------|
| **Independent** (each region own Redis) | 0ms cross-region | Low | Session cache (region-local) |
| **Replication** (primary → replica) | Async | Medium | Read-heavy cache |
| **Redis CRDT** (active-active) | Eventual | High | Counters, shared state |

**Recommendation**: **Independent Redis per region** — sessions are region-local (user authenticates in nearest region). Rate limits are per-region. Only shared state (tenant config) replicates via PG logical replication.

---

## 8. Geo-Routing & Traffic Management

### GeoDNS Configuration

```yaml
# Route53 / Cloudflare config
ggid.corp.com:
  - region: us-east-1
    records: [us-east-gateway-ip]
    geolocation: [NA, SA]
    health_check: https://us-east.ggid.corp.com/healthz
  - region: eu-west-1
    records: [eu-west-gateway-ip]
    geolocation: [EU, AF]
    health_check: https://eu-west.ggid.corp.com/healthz
  - region: ap-southeast-1
    records: [ap-se-gateway-ip]
    geolocation: [AS, OC]
    health_check: https://ap-se.ggid.corp.com/healthz
  failover:
    primary: us-east-1
    secondary: eu-west-1
```

### Client-Side Region Discovery

```bash
# SDK auto-discovers nearest region
GET https://ggid.corp.com/.well-known/regions

# Response:
{
  "regions": [
    { "id": "us-east", "endpoint": "https://us.ggid.corp.com", "latency_hint": "NA" },
    { "id": "eu-west", "endpoint": "https://eu.ggid.corp.com", "latency_hint": "EU" },
    { "id": "ap-se", "endpoint": "https://ap.ggid.corp.com", "latency_hint": "AS" }
  ],
  "default": "us-east"
}
```

---

## 9. Conflict Resolution

### CRDT (Conflict-Free Replicated Data Types)

| Data Type | CRDT | Use Case |
|-----------|------|----------|
| **Counter** | G-Counter (grow-only) | Rate limit counters |
| **Set** | OR-Set (observed-remove) | Active sessions, device list |
| **Map** | LWW-Map (last-write-wins) | User attributes |
| **Boolean** | Flag CRDT | Feature flags |

### Implementation

```go
// Region-aware write with timestamp
func (s *UserService) UpdateUser(ctx context.Context, req *UpdateUserRequest) error {
    req.UpdatedAt = time.Now().UTC()
    req.UpdatedBy = getRegionID(ctx)  // "us-east" / "eu-west"

    // Write locally → logical replication carries to other regions
    return s.repo.Update(ctx, req)
    // On conflict: LWW trigger in PG resolves using updated_at + updated_by
}
```

---

## 10. Data Residency

### Per-Tenant Region Configuration

```sql
ALTER TABLE tenants ADD COLUMN region VARCHAR(16) DEFAULT 'global';
-- Values: 'us', 'eu', 'cn', 'global' (replicated everywhere)

-- EU tenant: data only in EU region
UPDATE tenants SET region = 'eu' WHERE id = 'eu-tenant-uuid';

-- Publication filter ensures EU data doesn't replicate to US
```

### GDPR Article 44 Compliance

```
EU tenant:
  ├── User data: EU-West PG only (not replicated to US)
  ├── Audit events: EU-West PG + encrypted backup to EU S3
  ├── Sessions: EU-West Redis only
  └── Replication: NOT replicated cross-region (data sovereignty)

US tenant:
  ├── User data: US-East PG + replicated to EU (DR read-only)
  ├── Audit events: US-East PG
  └── Sessions: Nearest region Redis

China tenant (future):
  ├── Completely separate deployment (air-gapped from US/EU)
  ├── SM2/SM3/SM4 crypto (existing in key_provider.go:82)
  └── Alibaba Cloud KMS
```

---

## 11. Endpoint Precondition Check

### Existing Infrastructure (Reuse)

| Component | File | Status | Multi-Region Ready? |
|----------|------|--------|-------------------|
| PG connection pool | `pkg/tenant/rls_pool.go` | ✅ RLS on 27 tables | ✅ RLS enables per-tenant placement |
| Tenant context | `pkg/tenant/tenant.go:27` | ✅ | ✅ Add region field |
| Health checks | `*/cmd/main.go` | ✅ | ✅ Used by GeoDNS |
| K8s deployment | `deploy/k8s/` | ✅ | Per-region deployment |
| Docker Compose | `deploy/docker-compose.yml` | ✅ | Template per region |
| KMS providers | `pkg/crypto/key_provider.go:39` | ✅ 7 providers | ✅ Per-region keys |

### New Components Required

| Component | Purpose | Priority |
|-----------|---------|----------|
| PG streaming replication config | Primary → standby | P0 |
| GeoDNS configuration | Route users by geography | P1 |
| Region discovery API | `/.well-known/regions` | P1 |
| Per-tenant region column | Data residency | P1 |
| Logical replication (bidirectional) | Active-active writes | P2 |
| Conflict resolution triggers | LWW merge | P2 |
| Cross-region health monitor | Detect region failure | P1 |

---

## 12. API Design + Curl Commands

### Region Discovery

```bash
curl https://ggid.corp.com/.well-known/regions

{
  "regions": [
    { "id": "us-east", "endpoint": "https://us.ggid.corp.com", "status": "healthy", "latency_ms": 12 },
    { "id": "eu-west", "endpoint": "https://eu.ggid.corp.com", "status": "healthy", "latency_ms": 8 },
    { "id": "ap-se", "endpoint": "https://ap.ggid.corp.com", "status": "healthy", "latency_ms": 35 }
  ],
  "default": "us-east"
}
```

### Set Tenant Region

```bash
curl -X PUT https://ggid.corp.com/api/v1/tenants/{id}/region \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{"region": "eu", "reason": "GDPR compliance"}'

# Response:
{ "tenant_id": "...", "region": "eu", "replication": "eu-only", "updated_at": "..." }
```

### Cross-Region Status

```bash
curl https://ggid.corp.com/api/v1/admin/regions/status \
  -H "Authorization: Bearer $ADMIN_TOKEN"

{
  "regions": [
    { "id": "us-east", "role": "primary", "pg_lag_seconds": 0, "redis_status": "healthy" },
    { "id": "eu-west", "role": "active", "pg_lag_seconds": 2, "redis_status": "healthy" }
  ],
  "replication_lag_max": 2,
  "failover_ready": true
}
```

---

## 13. Database Schema Changes

```sql
-- Add region column to tenants
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS region VARCHAR(16) DEFAULT 'global';
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS preferred_region VARCHAR(16);

-- Add region tracking to all rows (for conflict resolution)
-- Already have updated_at on most tables; add region_origin
ALTER TABLE users ADD COLUMN IF NOT EXISTS region_origin VARCHAR(16) DEFAULT 'us-east';
ALTER TABLE audit_events ADD COLUMN IF NOT EXISTS region_origin VARCHAR(16) DEFAULT 'us-east';

-- Conflict resolution: last-write-wins trigger
CREATE OR REPLACE FUNCTION lww_resolve()
RETURNS TRIGGER AS $$
BEGIN
  IF TG_OP = 'UPDATE' THEN
    IF NEW.updated_at < OLD.updated_at THEN
      RETURN NULL;  -- Discard older update
    END IF;
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply to key tables
CREATE TRIGGER users_lww BEFORE UPDATE ON users
  FOR EACH ROW EXECUTE FUNCTION lww_resolve();

-- Logical replication publications (per-tenant region)
CREATE PUBLICATION eu_tenants FOR TABLE users, user_credentials
  WHERE (tenant_id IN (
    SELECT id FROM tenants WHERE region IN ('eu', 'global')
  ));
```

---

## 14. Performance Analysis

### Replication Lag

| Config | Typical Lag | Max Lag | RPO |
|--------|-----------|---------|-----|
| Sync streaming | 0ms | 0ms | 0 (zero data loss) |
| Async streaming | 1-5ms | 30s | < 30s |
| Logical (bidirectional) | 100-500ms | 5s | < 5s |
| Cross-continent (US↔EU) | 50-200ms | 10s | < 10s |

### Write Performance Impact

```
Single-region write:     2ms (local PG commit)
Sync replication write:  80ms (wait for remote ack) ← RTT to EU
Async replication write: 2ms (commit local, replicate async)
Logical replication:     2ms (commit local, logical async)
```

**Recommendation**: Async replication for <5ms write latency, accept RPO of <30s.

### Capacity Planning

| Metric | Single Region | 3-Region Active-Active |
|--------|--------------|----------------------|
| Max RPS | 5,000 (gateway) | 15,000 (3× gateway) |
| PG write RPS | 1,000 | 3,000 (split by region) |
| Storage | 1× | 3× (each region has full copy) |
| Network (cross-region) | 0 | 50-200ms RTT replication |

---

## 15. Implementation Backlog with DoD

### P0 — Active-Passive (2 sprints)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 1 | PG streaming replication (US→EU standby) | ✅ Standby receiving WAL ✅ Read queries on standby ✅ <30s lag ✅ ≥3 tests | 4d |
| 2 | Automated failover (Patroni or pg_auto_failover) | ✅ Primary failure → promote standby ✅ <60s failover ✅ ≥3 tests | 4d |
| 3 | Redis replica + sentinel | ✅ Replica syncs ✅ Sentinel failover ✅ ≥3 tests | 2d |
| 4 | GeoDNS configuration | ✅ Route by GeoIP ✅ Health check per region ✅ Failover routing | 2d |

### P1 — Active-Active Reads (1 sprint)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 5 | Region discovery API | ✅ `/.well-known/regions` ✅ Health status ✅ ≥3 tests | 2d |
| 6 | Both regions serve reads | ✅ Gateway routes to local PG ✅ Read replica for cross-region ✅ ≥3 tests | 3d |
| 7 | Cross-region health monitor | ✅ Monitor PG lag ✅ Alert on >30s lag ✅ ≥3 tests | 2d |

### P2 — Active-Active Writes (2 sprints)

| # | Task | DoD | Effort |
|---|------|-----|--------|
| 8 | Bidirectional logical replication | ✅ US↔EU bidirectional ✅ <5s lag ✅ ≥3 tests | 5d |
| 9 | Conflict resolution (LWW triggers) | ✅ Timestamp-based merge ✅ No data loss ✅ ≥3 tests | 3d |
| 10 | Per-tenant region column + publication filters | ✅ EU tenants not replicated to US ✅ GDPR compliant ✅ ≥3 tests | 3d |
| 11 | Region-aware SDK routing | ✅ SDK picks nearest region ✅ Failover to next region ✅ ≥3 tests | 2d |

### P3 — Advanced (Future)

| # | Task | DoD |
|---|------|-----|
| 12 | China region deployment | Air-gapped SM2/SM3/SM4 deployment |
| 13 | CRDT for rate limit counters | Cross-region consistent limits |
| 14 | Multi-region NATS | Geo-replicated event streams |
| 15 | Per-tenant write affinity | Write always goes to tenant's home region |

---

## 16. Competitive Differentiation

| Feature | GGID (target) | Okta | Microsoft Entra | Auth0 | Keycloak |
|---------|---------------|------|-----------------|-------|----------|
| **Multi-region** | Active-active (target) | Yes (11 regions) | Yes (60+ regions) | Yes (4 regions) | No (manual) |
| **Data residency** | Per-tenant region | Yes | Yes | Partial | No |
| **Failover** | <60s automated | ~30s | ~0s (anyCast) | ~60s | Manual |
| **Geo-routing** | GeoDNS + SDK | Anycast | Anycast | GeoDNS | No |
| **Conflict resolution** | LWW + CRDT | Proprietary | Proprietary | N/A | N/A |
| **PG logical replication** | Bidirectional | Proprietary | Proprietary | N/A | No |
| **Open source** | Yes | No | No | No | Yes |

**Key differentiator**: GGID would be the only open-source IAM with **per-tenant data residency** via PG logical replication publication filters — enabling GDPR/PIPL compliance without vendor lock-in.

---

## 17. Security Considerations

| Risk | Mitigation |
|------|-----------|
| **Replication credential theft** | Dedicated replication role with minimal privileges; TLS for replication connection |
| **Cross-region data leak** | Publication filters enforce per-tenant region; RLS prevents cross-tenant access |
| **Split-brain** | Quorum-based failover (Patroni with etcd); fencing (STONITH) |
| **Conflict data corruption** | LWW triggers are idempotent; audit trail records all writes |
| **Replication DDoS** | Rate limit WAL shipping; bandwidth throttling per region |
| **DNS hijack** | DNSSEC on geo-routing domain; health check before routing |

---

## References

- [PostgreSQL Streaming Replication](https://www.postgresql.org/docs/current/warm-standby.html) — Physical replication
- [PostgreSQL Logical Replication](https://www.postgresql.org/docs/current/logical-replication.html) — Bidirectional sync
- [Patroni](https://patroni.readthedocs.io/) — HA PostgreSQL with automatic failover
- [pg_auto_failover](https://github.com/citusdata/pg_auto_failover) — Simple PG failover
- [Redis Sentinel](https://redis.io/docs/management/sentinel/) — Redis HA + failover
- [CRDTs](https://crdt.tech/) — Conflict-free replicated data types
- [GDPR Article 44](https://gdpr-info.eu/art-44-gdpr/) — Data transfers outside EU
- [Route53 Geolocation](https://docs.aws.amazon.com/Route53/latest/DeveloperGuide/routing-policy-geo.html) — GeoDNS
- [Cloudflare Geo-Routing](https://developers.cloudflare.com/routing/) — Anycast routing
- [GGID Multi-Region (existing)](./multi-region.md) — Previous theoretical research (252 lines)
- [GGID PostgreSQL RLS](./postgresql-rls-implementation.md) — RLS enabling per-tenant placement
- [GGID Disaster Recovery](./disaster-recovery-backup.md) — Backup + replication foundation
- [GGID Tenant Package](../pkg/tenant/tenant.go) — Tenant context at line 27
- [GGID K8s Deploy](../deploy/k8s/) — Deployment manifests
- [GGID Docker Compose](../deploy/docker-compose.yml) — Single-region template
