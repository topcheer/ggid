# Onboarding & Multi-Tenant Login Design

> Status: **In Progress** | Owner: docs team | Last updated: 2026-07-15

## 1. Current State Analysis

### 1.1 Seed Data (`deploy/seed.sh`)

The seed script creates a single hardcoded default tenant and system roles/permissions:

```bash
TENANT="00000000-0000-0000-0000-000000000001"
# Inserts: tenants(name='Default', slug='default', plan='enterprise')
#          roles(admin, manager, user)
#          permissions(11 entries: iam:*)
#          role_permissions(admin → all)
```

**Gaps:**
- No admin user is created — the script only prints instructions to call `POST /api/v1/auth/register` manually.
- Tenant ID is hardcoded in both the seed script and the console's `api-config.ts`.
- No mechanism to detect "first run" vs. "already initialized."
- If the seed script runs twice, `ON CONFLICT DO NOTHING` silently skips — no indication of whether data was already present.

### 1.2 Login Page (`console/src/app/login/page.tsx`)

The login page imports `DEFAULT_TENANT_ID` from `@/lib/api-config`:

```typescript
const TENANT_ID = DEFAULT_TENANT_ID; // "00000000-0000-0000-0000-000000000001"
```

Every API call (login, MFA verify, social connectors) includes this as the `X-Tenant-ID` header. The user has **no way** to specify which tenant they belong to. The backend (`auth_service.go` `Login()`) requires `tenant.FromContext(ctx)` — if the header is missing or invalid, login fails.

**Gaps:**
- Single-tenant only — impossible to log in as a user from a different tenant.
- The tenant ID is a UUID, not user-friendly.
- No tenant discovery mechanism (slug → tenant_id resolution).

### 1.3 Existing Onboarding (`console/src/app/onboarding/page.tsx`)

A 3-step wizard exists but is client-side only:

| Step | Action | Server Interaction |
|------|--------|-------------------|
| 0 | Create Organization | `POST /api/v1/organizations` (fails silently → skip) |
| 1 | Add User | `POST /api/v1/auth/register` (fails silently → skip) |
| 2 | Generate API Key | `POST /api/v1/api-keys` (fails silently → skip) |

Completion is tracked via `localStorage.setItem("ggid_onboarding_completed", "true")`. Clearing browser data re-triggers onboarding. No backend state records whether initialization has occurred.

**Gaps:**
- No tenant creation step (assumes default tenant exists).
- No admin role assignment (register creates a user, but doesn't grant admin).
- No security configuration (password policy, MFA enforcement).
- Completion state is ephemeral (localStorage), not persistent (database).
- All API calls require authentication, but no token exists during onboarding.

### 1.4 Database Schema

**`tenants` table** (`deploy/migrations/01_all_up.sql:13`):

```sql
CREATE TABLE tenants (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(100) NOT NULL,
    slug        VARCHAR(50) NOT NULL UNIQUE,
    plan        tenant_plan NOT NULL DEFAULT 'free',
    status      tenant_status NOT NULL DEFAULT 'active',
    settings    JSONB NOT NULL DEFAULT '{}',
    max_users   INT NOT NULL DEFAULT 50,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

**`users` table** unique constraints:
```sql
CONSTRAINT users_tenant_username_uk UNIQUE (tenant_id, username),
CONSTRAINT users_tenant_email_uk    UNIQUE (tenant_id, email)
```

Usernames are scoped per-tenant — the same username can exist in different tenants.

**No `system_settings` or `system_config` table exists.**

---

## 2. Ideal Onboarding Flow

### 2.1 First-Run Detection

When the console loads, it calls a new unauthenticated endpoint:

```
GET /api/v1/system/status
```

Response:
```json
{
  "initialized": false,
  "tenant_count": 0,
  "user_count": 0,
  "version": "v0.4.0"
}
```

- `initialized: false` → redirect to `/onboarding`
- `initialized: true` → redirect to `/login`

**Implementation**: The auth service queries `SELECT COUNT(*) FROM users` and `SELECT COUNT(*) FROM tenants`. If both are 0 (or below a threshold), onboarding is triggered.

### 2.2 Onboarding Wizard (Server-Backed)

A new set of **unauthenticated** bootstrap endpoints (only active when `initialized == false`):

```
POST /api/v1/system/bootstrap/tenant      → Step 1
POST /api/v1/system/bootstrap/admin       → Step 2
POST /api/v1/system/bootstrap/security     → Step 3
POST /api/v1/system/bootstrap/complete     → Step 4
```

#### Step 1: Create Tenant

```
POST /api/v1/system/bootstrap/tenant
Body: { "name": "Acme Corp", "slug": "acme", "admin_email": "admin@acme.com" }
Response: { "tenant_id": "uuid", "setup_token": "one-time-token" }
```

- Creates the tenant record.
- Returns a `setup_token` used for subsequent bootstrap steps (replaces normal JWT auth).
- Validation: slug must be URL-safe, unique, 3-50 chars.

#### Step 2: Create Admin Account

```
POST /api/v1/system/bootstrap/admin
Headers: X-Setup-Token: <from step 1>
Body: { "username": "admin", "email": "admin@acme.com", "password": "SecurePass123!" }
Response: { "user_id": "uuid" }
```

- Creates the user, assigns `admin` system role.
- Seeds default roles + permissions for the new tenant (same as current `seed.sh`).

#### Step 3: Security Configuration

```
POST /api/v1/system/bootstrap/security
Headers: X-Setup-Token: <from step 1>
Body: {
  "password_policy": { "min_length": 12, "require_special": true },
  "enforce_mfa": false,
  "session_timeout": 3600
}
Response: { "status": "ok" }
```

- Stores settings in `tenants.settings` JSONB (existing column — no schema change needed).

#### Step 4: Complete

```
POST /api/v1/system/bootstrap/complete
Headers: X-Setup-Token: <from step 1>
Response: { "login_url": "/login" }
```

- Marks system as initialized.
- Invalidates the setup token.
- Returns a success page with a "Go to Login" button.

### 2.3 State Machine

```
                    ┌─────────────────────┐
                    │  System Boot        │
                    │  (no users/tenants) │
                    └──────────┬──────────┘
                               │
                    ┌──────────▼──────────┐
                    │  GET /system/status │
                    │  initialized=false  │
                    └──────────┬──────────┘
                               │
                    ┌──────────▼──────────┐
       ┌────────────│  Onboarding Wizard  │────────────┐
       │            └──────────┬──────────┘            │
       │                       │                       │
  ┌────▼───┐             ┌────▼────┐            ┌─────▼─────┐
  │ Step 1 │             │ Step 2  │            │  Step 3   │
  │ Tenant │────────────►│ Admin   │───────────►│ Security  │
  └────────┘             └─────────┘            └─────┬─────┘
                                                      │
                                               ┌──────▼──────┐
                                               │   Step 4    │
                                               │  Complete   │
                                               └──────┬──────┘
                                                      │
                                               ┌──────▼──────┐
                                               │ initialized │
                                               │   = true    │
                                               └──────┬──────┘
                                                      │
                                               ┌──────▼──────┐
                                               │  /login     │
                                               └─────────────┘
```

### 2.4 Bootstrap Token Security

The `setup_token` is critical because bootstrap endpoints bypass JWT auth:

- Generated as a random 32-byte hex string.
- Stored in Redis with a 30-minute TTL.
- Single-use per step (rotated after each call) — or reused with rate limiting.
- Invalidated permanently after `POST /bootstrap/complete`.
- Bootstrap endpoints return `403 Forbidden` once `initialized == true`.

---

## 3. Multi-Tenant Login Design

### Problem

Currently, the console hardcodes `X-Tenant-ID: 00000000-...-001` on every request. Users from any other tenant cannot log in.

### Option A: Tenant Input Field

Add a "Workspace" or "Organization slug" input on the login page:

```
┌───────────────────────────────┐
│  GGID Console                 │
│                               │
│  ┌─────────────────────────┐  │
│  │ Workspace (acme)        │  │
│  └─────────────────────────┘  │
│  ┌─────────────────────────┐  │
│  │ Username                │  │
│  └─────────────────────────┘  │
│  ┌─────────────────────────┐  │
│  │ Password                │  │
│  └─────────────────────────┘  │
│         [ Sign In ]           │
└───────────────────────────────┘
```

**Flow**: User types slug → frontend resolves slug to tenant_id → sends `X-Tenant-ID` header.

**New endpoint**:
```
GET /api/v1/system/tenant?slug=acme
Response: { "tenant_id": "uuid", "name": "Acme Corp", "branding": {...} }
```

**Pros**:
- Simple to implement — minimal backend change.
- Explicit — user knows which tenant they're logging into.
- Works for self-hosted and SaaS.

**Cons**:
- Extra field on login page.
- Users must remember their workspace slug.
- Requires a public (unauthenticated) tenant lookup endpoint.

### Option B: Auto-Discovery (Username → Tenant)

User enters just username + password. Backend searches all tenants for matching `(username, password)`.

```sql
SELECT tenant_id FROM users WHERE username = $1 AND password_hash = $2
```

**Pros**:
- Simplest UX — no tenant field needed.
- Familiar pattern (like personal apps).

**Cons**:
- **Security risk**: username enumeration across tenants. An attacker can determine if "admin" exists in any tenant.
- **Ambiguity**: if two tenants both have a user named "admin", login is ambiguous. Would need email-based lookup instead.
- **Performance**: must search across tenants (no RLS filter). Requires a global index on `users(username)` or `users(email)`, breaking tenant isolation patterns.
- **Not compatible with RLS**: the whole point of tenant-scoped queries is to always filter by `tenant_id`.

**Verdict: Not recommended.** Breaks multi-tenant isolation and introduces security vulnerabilities.

### Option C: Subdomain Resolution

Tenant is determined from the URL hostname:

```
acme.ggid.dev    → slug=acme → tenant_id lookup
admin.ggid.dev   → slug=admin → tenant_id lookup
localhost:3000   → default tenant (dev mode)
```

**Implementation**: Gateway middleware extracts subdomain, resolves to tenant_id, injects into context.

**Pros**:
- Best UX — zero friction, no extra input.
- Natural branding (each tenant gets their own URL).
- Industry standard (Slack, Notion, Linear all use this).
- No username cross-tenant leakage.

**Cons**:
- Requires DNS configuration (wildcard `*.ggid.dev`).
- Complex for self-hosted deployments (no wildcard DNS).
- Needs fallback for single-tenant / localhost deployments.

### Recommendation: **Hybrid (A + C)**

| Deployment | Mode | Tenant Resolution |
|-----------|------|-------------------|
| SaaS / Cloud | Subdomain (C) | `acme.ggid.dev` → auto-resolve |
| Self-hosted multi-tenant | Slug field (A) | User types workspace slug |
| Self-hosted single-tenant | Default | Auto-use the only tenant |

**Implementation priority**:
1. **Phase 1**: Add slug field (Option A) — works everywhere, no DNS needed.
2. **Phase 2**: Add subdomain resolution (Option C) — for SaaS deployments.

#### Login Page Changes (Phase 1)

```typescript
// Before:
const TENANT_ID = DEFAULT_TENANT_ID; // hardcoded

// After:
const [workspaceSlug, setWorkspaceSlug] = useState("");
const [tenantId, setTenantId] = useState<string | null>(null);

// Resolve slug → tenant_id on blur
const resolveTenant = async () => {
  const resp = await fetch(`${API_BASE}/api/v1/system/tenant?slug=${workspaceSlug}`);
  if (resp.ok) {
    const data = await resp.json();
    setTenantId(data.tenant_id);
  }
};

// Login uses resolved tenant_id
headers: { "X-Tenant-ID": tenantId }
```

**Single-tenant optimization**: If `GET /api/v1/system/status` returns `tenant_count: 1`, auto-resolve and hide the workspace field.

---

## 4. Data Model Changes

### 4.1 `tenants` Table — No Schema Change Needed

The existing `settings` JSONB column can store all onboarding configuration:

```json
{
  "password_policy": { "min_length": 12, "require_uppercase": true },
  "enforce_mfa": false,
  "session_timeout": 3600,
  "branding": { "logo_url": "", "primary_color": "#4F46E5" },
  "onboarding_completed_at": "2026-07-15T10:00:00Z"
}
```

No ALTER TABLE required. This is the **preferred approach** — avoids migrations.

### 4.2 New Table: `system_settings` (Recommended)

A simple key-value table for global (non-tenant-scoped) system state:

```sql
CREATE TABLE IF NOT EXISTS system_settings (
    key         VARCHAR(100) PRIMARY KEY,
    value       JSONB NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seed initial values
INSERT INTO system_settings (key, value) VALUES
    ('initialized', 'false'::jsonb),
    ('version', '"v0.4.0"'::jsonb),
    ('install_date', 'null'::jsonb);
```

**Why**: System initialization status is global, not per-tenant. Even with 100 tenants, the "is the system set up?" question is binary and system-wide.

**Key entries**:

| Key | Type | Purpose |
|-----|------|---------|
| `initialized` | boolean | `false` until first onboarding completes |
| `version` | string | Current schema/app version |
| `install_date` | timestamp | When the system was first set up |
| `maintenance_mode` | boolean | Admin toggle for maintenance window |

### 4.3 Alternative: Use `tenants` Row Count

Instead of a new table, check if any tenant + admin user exists:

```go
func (s *Service) IsInitialized(ctx context.Context) (bool, error) {
    count, err := s.db.QueryRow(ctx,
        "SELECT COUNT(*) FROM users WHERE status = 'active'"
    ).Scan(&count)
    return count > 0, err
}
```

**Pros**: Zero schema changes.
**Cons**: Can't store version, maintenance mode, or install date. Heuristic-based (what if all users are deleted?).

### 4.4 Recommendation

| Approach | Use When |
|----------|----------|
| `system_settings` table | **Recommended** — clean, extensible, future-proof |
| `tenants.settings` JSONB | For tenant-specific config (password policy, branding) |
| Row count heuristic | Quick MVP — no migration needed |

**Migration file** (`deploy/migrations/02_system_settings_up.sql`):

```sql
CREATE TABLE IF NOT EXISTS system_settings (
    key         VARCHAR(100) PRIMARY KEY,
    value       JSONB NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO system_settings (key, value) VALUES
    ('initialized', 'false'::jsonb)
ON CONFLICT (key) DO NOTHING;
```

---

## 5. API Surface Summary

### New Endpoints

| Method | Path | Auth | Purpose |
|--------|------|------|---------|
| `GET` | `/api/v1/system/status` | None | Check if system is initialized |
| `GET` | `/api/v1/system/tenant?slug=...` | None | Resolve tenant slug to ID (for login) |
| `POST` | `/api/v1/system/bootstrap/tenant` | None (bootstrap only) | Create initial tenant |
| `POST` | `/api/v1/system/bootstrap/admin` | Setup token | Create admin user |
| `POST` | `/api/v1/system/bootstrap/security` | Setup token | Configure security settings |
| `POST` | `/api/v1/system/bootstrap/complete` | Setup token | Finalize onboarding |

### Modified Endpoints

| Endpoint | Change |
|----------|--------|
| `POST /api/v1/auth/login` | Accept tenant slug OR tenant_id in `X-Tenant-ID` (resolve slug if not UUID) |
| Gateway middleware | Auto-resolve subdomain → tenant_id (Phase 2) |

---

## 6. Security Considerations

1. **Bootstrap endpoints must self-disable**: Once `initialized == true`, all bootstrap routes return `403`. This is enforced at the middleware level, not just per-handler.

2. **Setup token**:
   - 32-byte random hex, stored in Redis with 30-min TTL.
   - Single-use or rate-limited (max 5 calls).
   - Invalidated after completion or expiry.

3. **Tenant slug lookup**: The public `/system/tenant?slug=...` endpoint reveals tenant existence. Mitigate by:
   - Only returning `tenant_id` + `name` (no user data).
   - Rate limiting (10 lookups per IP per minute).
   - Optional: require slug to match a minimum length (3+ chars) to prevent brute-force.

4. **Password policy enforcement**: The bootstrap admin creation must validate password strength using the same `PasswordPolicy` config used in regular registration.

5. **Audit logging**: All bootstrap operations must emit audit events (`system.bootstrap.tenant_created`, `system.bootstrap.admin_created`, etc.).

---

## 7. Implementation Phases

### Phase 1: Minimum Viable Onboarding (1-2 sprints)

- [ ] Add `system_settings` table migration
- [ ] Implement `GET /api/v1/system/status` (unauthenticated)
- [ ] Implement bootstrap endpoints (tenant + admin + complete)
- [ ] Rewrite `/onboarding` page to use bootstrap API
- [ ] Remove localStorage-based completion tracking
- [ ] Auto-redirect: status check on app load

### Phase 2: Multi-Tenant Login (1 sprint)

- [ ] Add workspace slug field to login page
- [ ] Implement `GET /api/v1/system/tenant?slug=...`
- [ ] Resolve slug → tenant_id before login request
- [ ] Single-tenant optimization (hide slug field)

### Phase 3: Subdomain Resolution (1 sprint)

- [ ] Gateway middleware: extract subdomain from Host header
- [ ] DNS setup: wildcard `*.ggid.dev`
- [ ] Console: read tenant from URL instead of input field
- [ ] Self-hosted fallback: slug field when subdomain not available

### Phase 4: Polish (ongoing)

- [ ] Branding per tenant (logo, colors from `tenants.settings`)
- [ ] Email verification during onboarding
- [ ] Invite flow (admin invites additional users)
- [ ] Onboarding progress recovery (resume interrupted setup)

---

## 8. Implementation Status (2026-07-15)

### Gap Tracking

| Gap # | Feature | Status | Commit | Notes |
|-------|---------|--------|--------|-------|
| #13 | System initialization detection (`GET /api/v1/system/initialized`) | **DONE** | `6f23b400` | Unauthenticated endpoint, returns `{initialized: bool}` |
| #14 | Onboarding wizard (server-backed bootstrap flow) | **PARTIAL** | — | Login page shows warning when system not initialized; full 4-step bootstrap wizard (tenant creation, admin setup, security config) still **pending** |
| #15 | Tenant resolution (`GET /api/v1/tenants/resolve?slug=...`) | **DONE** | `6f23b400` | Unauthenticated slug → tenant_id lookup for multi-tenant login |
| #16 | Multi-tenant login (`tenant_slug` in `POST /api/v1/auth/login`) | **DONE** | `6f23b400` | Login accepts `tenant_slug` field as alternative to `X-Tenant-ID` header |
| #17 | All-in-one Docker IPv6 fix + run.sh launcher | **DONE** | `6f23b400` | Service URLs use `127.0.0.1` instead of `localhost`; `run.sh` one-command launcher |

### What Remains

**Gap #14 — Full Onboarding Wizard (pending):**
- The login page detects uninitialized systems and displays a warning, but the actual 4-step bootstrap flow (Steps 1-4 described in Section 2) is not yet implemented.
- Bootstrap endpoints (`POST /api/v1/system/bootstrap/*`) with setup_token authentication need to be built.
- The existing `/onboarding` page still uses localStorage for completion tracking — needs to be rewritten to use server-backed bootstrap APIs.
- The `system_settings` table migration is not yet created.

**Phase 2 — Multi-Tenant Login UI (partially done):**
- The backend supports `tenant_slug` in login requests (Gap #16).
- The `/tenants/resolve` API is available (Gap #15).
- The console login page has not yet been updated to include a workspace slug input field.

**Phase 3-4 — Subdomain resolution and polish:** Not started.

### Commits

- `3f49c3a5` — Initial design document (this file)
- `6f23b400` — Multi-tenant login + onboarding flow implementation (gaps #13, #15, #16, #17)

---

## 9. References

- Current seed script: `deploy/seed.sh`
- Current login page: `console/src/app/login/page.tsx`
- Current onboarding page: `console/src/app/onboarding/page.tsx`
- API config: `console/src/lib/api-config.ts`
- Auth service: `services/auth/internal/service/auth_service.go`
- Database schema: `deploy/migrations/01_all_up.sql`
- Tenant context: `pkg/tenant/`
