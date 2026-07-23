# GGID SCIM 2.0 Sync Status API

> Provides visibility into inbound/outbound SCIM provisioning status for IdP integration verification.

## GET /api/v1/scim/sync-status

Returns the current sync state for all configured SCIM sources.

### Response

```json
{
  "sources": [
    {
      "source_type": "azure-ad",
      "source_id": "aad-tenant-123",
      "last_sync": "2026-07-23T10:00:00Z",
      "last_sync_status": "success",
      "users_synced": 1250,
      "groups_synced": 45,
      "errors": [],
      "next_sync": "2026-07-23T11:00:00Z"
    },
    {
      "source_type": "okta",
      "source_id": "okta-org-456",
      "last_sync": "2026-07-23T09:30:00Z",
      "last_sync_status": "partial",
      "users_synced": 980,
      "groups_synced": 30,
      "errors": [
        {"user": "john@example.com", "error": "duplicate email"}
      ],
      "next_sync": "2026-07-23T10:30:00Z"
    }
  ],
  "pending_deprovisioning": 5,
  "pending_assignments": 12
}
```

## GET /api/v1/scim/sync-status/{sourceId}/history

Returns sync history for a specific source.

### Query Parameters
- `days` (default: 7): Number of days of history
- `page` (default: 1): Page number
- `page_size` (default: 20): Results per page

## POST /api/v1/scim/sync-status/{sourceId}/trigger

Manually trigger a sync for a specific source. Returns 202 with a job ID.

## Console Integration

The SCIM Settings page (`/settings/scim-provisioning`) should display:
1. Per-source status card (last sync, user count, error count)
2. Sync history table with expandable error details
3. "Trigger Sync Now" button per source
4. Deprovisioning queue count badge

## Reverse Sync Confirmation

When GGID makes changes (role assignment, deprovisioning), the system should:
1. Record the change in `scim_outbound_queue` table
2. Attempt to push to the IdP via SCIM PATCH/PUT
3. Record confirmation when IdP acknowledges
4. Alert if outbound sync fails after 3 retries

```sql
-- Migration: outbound SCIM sync queue
CREATE TABLE IF NOT EXISTS scim_outbound_queue (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id   TEXT NOT NULL,
    user_id     UUID NOT NULL,
    operation   TEXT NOT NULL,  -- 'update', 'deprovision', 'assign_role'
    payload     JSONB NOT NULL,
    status      TEXT NOT NULL DEFAULT 'pending',  -- pending, sent, confirmed, failed
    attempts    INT NOT NULL DEFAULT 0,
    last_error  TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_scim_outbound_status ON scim_outbound_queue (status, created_at);
CREATE INDEX idx_scim_outbound_source ON scim_outbound_queue (source_id, status);
```
