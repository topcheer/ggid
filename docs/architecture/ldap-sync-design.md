# LDAP/Active Directory Sync Architecture

> Status: DESIGN — Round 68 (2026-07-16)
> Auth login: IMPLEMENTED (`pkg/authprovider/ldap.go`)
> Directory sync: NOT IMPLEMENTED (this document defines the target architecture)

## 1. Current State

### What Works: LDAP Authentication

LDAP login is fully implemented in `pkg/authprovider/ldap.go`:

- **Connection pool** (`PoolSize`, default 5) with health tracking
- **Service bind + user search + re-bind** authentication flow (3-step)
- **StartTLS** and **LDAPS** support with custom CA pool
- **Auto-provisioning** (JIT user creation on first login)
- **Group-to-role mapping** (`GroupRoleMappings`)
- Tested against an embedded LDAP test server (`ldap_server_test.go`)

### What Doesn't Work: Directory Sync

The following features return mock/stub data or are missing entirely:

| Feature | Current State | Required |
|---------|--------------|----------|
| Config GET (`GET /ldap/sync-config`) | Returns hardcoded fake data | Read from DB (`idp_configs` table) |
| Config PUT (`PUT /ldap/sync-config`) | Accepts JSON but doesn't persist | Write to DB |
| Test connection (`POST /ldap/sync-config/test`) | 404 — no route | New endpoint |
| User sync (`POST /ldap/sync`) | 404 — no endpoint | New endpoint |
| Group sync | Not implemented | New endpoint + handler |
| Sync status (`GET /ldap/sync-status`) | Hardcoded mock in `sync_status_handler.go` | Real sync state from DB |
| Scheduled sync | No scheduler | Cron/ticker in identity service |

## 2. Target Architecture

### 2.1 Components

```
┌──────────────────────────────────────────────────────────┐
│                    Identity Service                        │
│                                                            │
│  ┌──────────────┐    ┌──────────────┐    ┌─────────────┐ │
│  │ Config Store │    │  Sync Engine │    │  Scheduler  │ │
│  │ (idp_configs) │───▶│              │◀───│  (ticker)   │ │
│  └──────────────┘    └──────┬───────┘    └─────────────┘ │
│                             │                             │
│  ┌──────────────────────────┼──────────────────────────┐ │
│  │         LDAP Sync Service                            │ │
│  │                                                       │ │
│  │  1. Connect (pool)    ──▶  pkg/authprovider/ldap.go  │ │
│  │  2. Search users      ──▶  LDAP SearchRequest        │ │
│  │  3. Map attributes    ──▶  LDAP → domain.User        │ │
│  │  4. Upsert users      ──▶  identity repository       │ │
│  │  5. Sync groups       ──▶  LDAP groups → roles       │ │
│  │  6. Publish events    ──▶  NATS audit publisher      │ │
│  └──────────────────────────────────────────────────────┘ │
│                                                            │
│  HTTP Routes:                                              │
│    GET  /api/v1/identity/ldap/sync-config                  │
│    PUT  /api/v1/identity/ldap/sync-config                  │
│    POST /api/v1/identity/ldap/sync-config/test             │
│    POST /api/v1/identity/ldap/sync                         │
│    GET  /api/v1/identity/ldap/sync-status                  │
└──────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────┐
│  LDAP / AD      │
│  (OpenLDAP/AD)  │
│  Port 389/636   │
└─────────────────┘
```

### 2.2 Configuration Storage

LDAP sync config should be stored in the existing `idp_configs` table (already used by SCIM/SAML config):

```
Table: idp_configs
  id           UUID PK
  tenant_id    UUID
  provider     VARCHAR  -- 'ldap', 'scim', 'saml'
  config_json  JSONB    -- {server_url, bind_dn, base_dn, ...}
  enabled      BOOLEAN
  created_at   TIMESTAMPTZ
  updated_at   TIMESTAMPTZ
```

The `config_json` field stores the `LDAPSyncConfig` struct as JSON, reusing the existing `idpconfig.Store` interface.

### 2.3 Sync Flow (User Import)

```
1. Trigger (manual POST /ldap/sync OR scheduler tick)
   │
2. Load LDAP config from idp_configs
   │
3. Acquire connection from pool (pkg/authprovider/ldap.go)
   │
4. Search users with configured UserFilter
   - Paginate with SimplePaging (size=100)
   - Request attributes: cn, mail, uid, sAMAccountName, memberOf
   │
5. For each LDAP user entry:
   a. Map attributes → domain.User (using AttributeMapping config)
   b. Check if user exists by external_id (LDAP DN)
   c. If new → CreateUser (status=active, source=ldap)
   d. If existing → UpdateUser (sync display_name, email, groups)
   e. Skip deleted LDAP entries (set status=inactive)
   │
6. Publish audit event (user.sync.ldap, count, errors)
   │
7. Update sync_status record (last_sync, status, counts)
```

### 2.4 Group Sync

```
1. Search LDAP groups with GroupFilter + GroupBaseDN
   │
2. For each group:
   a. Map to application role (using GroupRoleMappings config)
   b. List group members (member attribute)
   c. For each member:
      - AssignRole(userID, roleID, ScopeOrganization, ...)
      - Uses policy service RoleService.AssignRole()
   │
3. Remove role assignments for users no longer in the LDAP group
```

### 2.5 Attribute Mapping

Default attribute mapping (configurable via `AttributeMapping` in config):

| LDAP Attribute | domain.User Field | Default |
|---------------|-------------------|---------|
| `uid` or `sAMAccountName` | `username` | required |
| `mail` | `email` | required |
| `cn` or `displayName` | `display_name` | optional |
| `memberOf` | `groups` (for role mapping) | optional |
| `DN` (distinguishedName) | `external_id` | auto |

### 2.6 Scheduler

A background goroutine in the identity service:

```go
// In identity cmd/main.go or service init
ticker := time.NewTicker(time.Duration(config.SyncIntervalMins) * time.Minute)
go func() {
    for range ticker.C {
        if config.Enabled {
            syncService.RunSync(ctx)
        }
    }
}()
```

- Default interval: 15 minutes (configurable)
- Respects `enabled` flag in config
- Manual sync via `POST /ldap/sync` bypasses the interval check

### 2.7 Sync Status Tracking

```sql
Table: idp_sync_runs
  id            UUID PK
  tenant_id     UUID
  provider      VARCHAR  -- 'ldap'
  started_at    TIMESTAMPTZ
  completed_at  TIMESTAMPTZ
  status        VARCHAR  -- 'success', 'failed', 'in_progress'
  users_synced  INT
  users_created INT
  users_updated INT
  errors_count  INT
  error_details JSONB
```

## 3. API Endpoints

### GET /api/v1/identity/ldap/sync-config
Returns current LDAP sync configuration from DB.

### PUT /api/v1/identity/ldap/sync-config
Updates LDAP sync configuration. Stores in `idp_configs`.

### POST /api/v1/identity/ldap/sync-config/test
Tests LDAP connection without performing a sync:
- Connects to LDAP server
- Binds with service account
- Runs a count-only search
- Returns: latency, users found, groups found

### POST /api/v1/identity/ldap/sync
Triggers a manual sync. Returns immediately with a sync run ID.
- Optional `?dry_run=true` to preview changes without writing
- Optional `?full=true` to bypass incremental sync

### GET /api/v1/identity/ldap/sync-status
Returns last sync result + next scheduled sync time.
Reads from `idp_sync_runs` table.

## 4. Files to Implement

| File | Purpose |
|------|---------|
| `services/identity/internal/service/ldap_sync_service.go` | Core sync engine: search, map, upsert |
| `services/identity/internal/server/ldap_sync_config_handler.go` | Replace mock handler with real DB-backed handler |
| `services/identity/internal/server/ldap_sync_handler.go` | POST /ldap/sync + test connection handler |
| `services/identity/internal/server/sync_status_handler.go` | Replace hardcoded mock with DB query |
| Migration: `add_idp_sync_runs.sql` | Create sync run tracking table |

## 5. Reuse Existing Code

- **Connection pool**: `pkg/authprovider/ldap.go` — `LDAPProvider` already has pool, dial, TLS, bind
- **Config store**: `services/identity/internal/idpconfig/idpconfig.go` — `Store` interface for idp_configs
- **Role assignment**: `services/policy/internal/service/role_service.go` — `AssignRole()` method
- **Audit publisher**: `pkg/audit` — `Publisher.PublishAsync()` for sync events
- **External identity**: `services/identity/internal/domain/external_identity.go` — already has LDAP linking

## 6. Security Considerations

- **Bind credentials**: Store encrypted in DB (use `pkg/crypto` AES-256-GCM)
- **Connection**: Always use StartTLS or LDAPS in production
- **Rate limiting**: Prevent concurrent sync runs per tenant (mutex or DB lock)
- **Audit**: Every sync run must publish an audit event with counts
- **Soft delete**: Users removed from LDAP should be deactivated, not hard-deleted
