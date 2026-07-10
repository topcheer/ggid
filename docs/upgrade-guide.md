# GGID Upgrade Guide

How to safely upgrade GGID across versions, including database migrations,
API deprecations, and breaking change procedures.

---

## Table of Contents

- [Versioning Policy](#versioning-policy)
- [Database Migration Strategy](#database-migration-strategy)
- [API Deprecation Process](#api-deprecation-process)
- [Breaking Change Communication](#breaking-change-communication)
- [Upgrade Checklist: v0.x to v1.0](#upgrade-checklist-v0x-to-v10)
- [Step-by-Step Upgrade Procedure](#step-by-step-upgrade-procedure)
- [Rollback Procedure](#rollback-procedure)

---

## Versioning Policy

GGID follows [Semantic Versioning (SemVer)](https://semver.org/) for all releases.

### Version Format

```
vMAJOR.MINOR.PATCH
```

| Version Component | When It Changes | Examples |
|-------------------|----------------|----------|
| **MAJOR** | Breaking changes (API, config, DB schema) | v0.x → v1.0, v1.x → v2.0 |
| **MINOR** | New features (backward compatible) | v1.2.x → v1.3.0 |
| **PATCH** | Bug fixes (backward compatible) | v1.3.0 → v1.3.1 |

### Pre-1.0 (Current Phase)

While GGID is in v0.x:
- MINOR version increments may include breaking changes
- PATCH versions are always safe to apply
- Breaking changes are clearly documented in release notes
- At least one minor version of deprecation warning is given

### Release Cadence

| Release Type | Frequency | Contents |
|-------------|-----------|----------|
| PATCH | As needed | Bug fixes, security patches |
| MINOR | Every 4-6 weeks | New features, non-breaking changes |
| MAJOR | As needed (rare) | Breaking changes, major architecture shifts |

---

## Database Migration Strategy

GGID uses **forward-only** migrations. There is no automatic rollback — down
migrations exist only for emergency use and are not tested in production.

### Migration Principles

1. **Additive changes first** — New columns are added as nullable, with defaults
2. **Deploy code before enforcing** — New code writes to new columns but doesn't require them
3. **Backfill asynchronously** — Large data migrations happen in background jobs
4. **Deprecate in a later release** — Old columns removed only in the next major version

### Migration Files

```
deploy/migrations/
  001_initial_schema.sql
  002_add_webauthn_backup_flags.sql
  003_add_scim_groups.sql
  004_add_oidc_federation.sql
  ...
```

Each migration is numbered sequentially. The migrations table tracks applied state:

```sql
SELECT * FROM schema_migrations ORDER BY version;
```

### Running Migrations

```bash
# Check current migration version
bash deploy/migrate.sh status

# Apply pending migrations (forward-only)
bash deploy/migrate.sh up

# In Docker Compose, migrations run automatically via init container
docker compose up -d
```

### Safe Migration Patterns

#### Add Column (Safe)

```sql
-- v1.2.0 migration: add nullable column
ALTER TABLE users ADD COLUMN mfa_secret TEXT;

-- v1.2.1 migration: backfill data in background
-- (Go migration runner, not SQL)
UPDATE users SET mfa_secret = '...' WHERE mfa_secret IS NULL;

-- v1.3.0 migration: make NOT NULL (after all rows are backfilled)
ALTER TABLE users ALTER COLUMN mfa_secret SET NOT NULL;
```

#### Rename Column (Multi-Release)

```sql
-- Release 1: add new column
ALTER TABLE users ADD COLUMN email_address TEXT;
UPDATE users SET email_address = email;

-- Release 2: deploy code that uses new column (dual-write)
-- Code writes to both email and email_address

-- Release 3: stop reading old column
-- Code reads only email_address

-- Release 4: drop old column
ALTER TABLE users DROP COLUMN email;
```

#### Change Column Type (Multi-Release)

```sql
-- Release 1: add new column with new type
ALTER TABLE roles ADD COLUMN permissions_new JSONB;

-- Release 2: backfill from old column
UPDATE roles SET permissions_new = to_jsonb(string_to_array(permissions, ','));

-- Release 3: deploy code using new column

-- Release 4: drop old column, rename new
ALTER TABLE roles DROP COLUMN permissions;
ALTER TABLE roles RENAME COLUMN permissions_new TO permissions;
```

---

## API Deprecation Process

When an API endpoint or field needs to be removed or changed, GGID follows
a structured deprecation process.

### Deprecation Timeline

```
v1.2.0 — Endpoint marked deprecated in OpenAPI spec
         Deprecation header added to responses
         Migration guide published
    │
    │  (minimum 2 minor releases = ~8-12 weeks)
    │
v1.4.0 — Endpoint still available (last chance)
         Warning logs for continued use
    │
v2.0.0 — Endpoint removed
```

### Deprecation Signals

```http
HTTP/1.1 200 OK
Deprecation: true
Sunset: Sat, 1 Mar 2025 00:00:00 GMT
Link: <https://docs.ggid.dev/migration/v2>; rel="deprecation"
```

| Header | Description |
|--------|-------------|
| `Deprecation: true` | This endpoint is deprecated |
| `Sunset: <date>` | Date when the endpoint will be removed |
| `Link: rel="deprecation"` | URL to migration documentation |

### Deprecation in OpenAPI

```yaml
paths:
  /api/v1/users/import:
    post:
      deprecated: true
      summary: "[DEPRECATED] Use POST /api/v1/users/batch instead"
      description: |
        This endpoint is deprecated as of v1.2.0 and will be
        removed in v2.0.0. Use the batch creation endpoint instead.
```

### Field-Level Deprecation

Deprecated fields in JSON responses include a `_deprecated` marker:

```json
{
  "user_id": "550e8400-...",
  "username": "john.doe",
  "email": "john@example.com",
  "email_address": "john@example.com",
  "_deprecations": {
    "email": "Use 'email_address' instead. Removed in v2.0.0."
  }
}
```

---

## Breaking Change Communication

### Communication Channels

| Channel | Timing | Audience |
|---------|--------|----------|
| Release notes | At release | All users |
| Deprecation headers | At runtime | API consumers |
| Blog post | 2 weeks before major release | Community |
| Email notification | 4 weeks before major release | Enterprise customers |
| Slack/Discord announcement | 1 week before major release | Community |
| Console banner | At upgrade | Admins |

### Release Note Template

```markdown
## Breaking Changes in vX.Y.0

### Removed
- `POST /api/v1/users/import` — Use `POST /api/v1/users/batch` instead
- `email` field in User response — Use `email_address` instead
- `DATABASE_URL` env var for Policy/Org/Audit — Use `DB_HOST`/`DB_PORT`/etc.

### Changed
- JWT signing algorithm default changed from HS256 to RS256
- Minimum password length increased from 8 to 12
- Rate limit headers renamed: `X-RateLimit-Remaining` → `X-RateLimit-Limit`

### Added
- `POST /api/v1/users/batch` — Batch user creation (replaces import)
- `email_address` field in User response
- RS256 JWT signing (HS256 still supported for backward compatibility)

### Migration Steps
1. Update your integration to use new endpoints
2. Test against staging environment
3. Update SDK to latest version
4. Schedule upgrade window
```

---

## Upgrade Checklist: v0.x to v1.0

The transition from pre-release (v0.x) to stable (v1.0) includes several
breaking changes. Use this checklist to prepare.

### Pre-Upgrade (2-4 weeks before)

- [ ] Read the v1.0 release notes thoroughly
- [ ] Review the migration guide for each deprecated API
- [ ] Update SDK dependencies to latest pre-release (`go get github.com/ggid/sdk-go@v1.0.0-rc1`)
- [ ] Audit your integration for deprecated endpoints (check `Deprecation` headers)
- [ ] Update configuration for renamed environment variables
- [ ] Schedule maintenance window for upgrade
- [ ] Backup PostgreSQL database (`pg_dump`)
- [ ] Notify users of scheduled downtime (if any)

### During Upgrade

- [ ] Stop all GGID services (`docker compose down` or `kubectl scale --replicas=0`)
- [ ] Backup database (point-in-time snapshot)
- [ ] Pull v1.0 images (`docker pull ghcr.io/ggid/gateway:v1.0.0`)
- [ ] Run migrations (`bash deploy/migrate.sh up`)
- [ ] Verify migrations applied (`SELECT * FROM schema_migrations ORDER BY version`)
- [ ] Start services (`docker compose up -d`)
- [ ] Run health checks (`curl localhost:8080/healthz`)
- [ ] Run E2E test suite (`bash deploy/e2e-docker-test.sh`)

### Post-Upgrade (verify)

- [ ] Verify all services healthy
- [ ] Test login flow (password, MFA, WebAuthn)
- [ ] Verify JWT validation works (old tokens may need refresh)
- [ ] Check audit events flowing (`curl $API/api/v1/audit/events`)
- [ ] Monitor error rates for 1 hour
- [ ] Remove deprecated env vars from configuration
- [ ] Update internal documentation with new API endpoints

### Key Changes in v1.0

| Change | v0.x | v1.0 |
|--------|------|------|
| JWT algorithm | HS256 (symmetric) | RS256 (asymmetric, default) |
| Password min length | 8 | 12 |
| Rate limit header | `X-RateLimit-Remaining` | `RateLimit-Remaining` (RFC 9211) |
| User import | `POST /users/import` | `POST /users/batch` |
| Policy/Org/Audit DB config | `DATABASE_URL` | `DB_HOST`, `DB_PORT`, etc. |
| SCIM endpoints | `/scim/v2/Users` | `/scim/v2/Users` (unchanged) |
| WebAuthn | Experimental | Stable (FIDO2 certified) |

---

## Step-by-Step Upgrade Procedure

### Docker Compose

```bash
# 1. Stop services (graceful)
docker compose down

# 2. Backup database
docker exec ggid-postgres pg_dump -U ggid ggid > backup-pre-v1.0.sql

# 3. Update image tags in docker-compose.yaml
# Change all image tags from :latest to :v1.0.0
sed -i 's|ghcr.io/ggid/.*:latest|ghcr.io/ggid/\1:v1.0.0|' deploy/docker-compose.yaml

# Or manually pull specific version
docker pull ghcr.io/ggid/gateway:v1.0.0
docker pull ghcr.io/ggid/auth:v1.0.0
# ... etc for all services

# 4. Start services (migrations run automatically)
docker compose up -d

# 5. Wait for healthchecks
sleep 30
docker compose ps

# 6. Verify
bash deploy/e2e-docker-test.sh
```

### Kubernetes (Helm)

```bash
# 1. Update Helm repo
helm repo update

# 2. Dry-run the upgrade (see what changes)
helm upgrade ggid ggid/ggid \
  -f ggid-values.yaml \
  -n ggid \
  --dry-run

# 3. Backup database (snapshot or pg_dump)
kubectl exec -n ggid ggid-postgres-0 -- pg_dump -U ggid ggid > backup.sql

# 4. Perform rolling update
helm upgrade ggid ggid/ggid \
  -f ggid-values.yaml \
  -n ggid \
  --timeout 10m \
  --wait

# 5. Check rollout status
kubectl rollout status deployment/ggid-gateway -n ggid
kubectl rollout status deployment/ggid-auth -n ggid

# 6. Run health checks
kubectl exec -n ggid deploy/ggid-gateway -- curl localhost:8080/healthz
```

### Build from Source

```bash
# 1. Checkout the release tag
git fetch --tags
git checkout v1.0.0

# 2. Build all services
go build -o bin/ ./services/*/cmd

# 3. Run migrations
go run ./cmd/migrate up

# 4. Restart services
systemctl restart ggid-gateway ggid-auth ggid-identity
```

---

## Rollback Procedure

If the upgrade fails, follow these steps to roll back.

### Step 1: Stop Services

```bash
docker compose down
# or
kubectl scale deployment -l app.kubernetes.io/part-of=ggid --replicas=0 -n ggid
```

### Step 2: Restore Database

```bash
# From the pre-upgrade backup
docker exec -i ggid-postgres psql -U ggid ggid < backup-pre-v1.0.sql

# Or restore from snapshot (AWS RDS)
aws rds restore-db-instance-from-db-snapshot \
  --db-instance-identifier ggid-rollback \
  --db-snapshot-identifier ggid-pre-v1.0-snapshot
```

### Step 3: Deploy Previous Version

```bash
# Docker Compose
sed -i 's|:v1.0.0|:v0.9.0|' deploy/docker-compose.yaml
docker compose up -d

# Kubernetes
helm rollback ggid 0 -n ggid  # rollback to previous revision
```

### Step 4: Verify

```bash
bash deploy/e2e-docker-test.sh
```

> **Warning:** Database rollbacks are not guaranteed. Forward-only migrations
> may have introduced schema changes that old code doesn't understand. Always
> test the rollback procedure in staging before relying on it in production.

---

## References

- [Deployment Guide](./deployment-guide.md) — Initial deployment
- [Configuration Reference](./configuration.md) — Environment variables
- [Contributing Guide](./contributing.md) — Release process
- [Migration Guide](./migration-guide.md) — Migrating from other IAM platforms
