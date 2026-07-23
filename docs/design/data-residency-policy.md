# Data Residency Policy

> Enables per-tenant geographic data residency enforcement.

## Design

GGID's `geofencing_handler.go` already blocks authentication from restricted regions. This extends the concept to **data storage residency** — ensuring tenant data stays within a designated geographic region.

## Tenant Configuration

```sql
-- Migration: tenant data residency configuration
CREATE TABLE IF NOT EXISTS tenant_data_residency (
    tenant_id    UUID PRIMARY KEY,
    region       TEXT NOT NULL,          -- 'eu-west-1', 'us-east-1', 'ap-east-1'
    enforced     BOOLEAN NOT NULL DEFAULT true,
    allowed_origins TEXT[] NOT NULL DEFAULT '{}', -- IP/CIDR restrictions
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

## API

### GET /api/v1/admin/data-residency
List all tenant residency policies.

### PUT /api/v1/admin/data-residency/{tenantId}
Set a tenant's data residency policy.

```json
{
  "region": "eu-west-1",
  "enforced": true,
  "allowed_origins": ["10.0.0.0/8"]
}
```

### GET /api/v1/admin/data-residency/{tenantId}
Get a specific tenant's residency policy.

## Enforcement Points

1. **Authentication**: `geofencing_handler.go` already checks IP → region. Extended to verify tenant residency policy.
2. **Data writes**: Gateway middleware tags requests with `X-Tenant-Region`. Audit service stores events with region tag.
3. **Backups**: `backup-verify.sh` can check that tenant data is only in the designated region's DB.
4. **Console**: Settings page shows tenant residency status with compliance indicators.

## Multi-Region Deployment Integration

With `values-ha.yaml`, each region runs its own PG instance. The `tenant_data_residency` table maps tenants to regions, and the gateway routes writes to the correct regional PG.

## Current Status

- **Code exists**: `geofencing_handler.go` (auth-level geo-blocking)
- **Pilot scope**: API + DB migration + admin UI read-only
- **Full implementation**: Requires multi-region PG routing (future v3)
