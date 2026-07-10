# Design: Data Model

> **Status:** Implemented

Entity-Relationship diagram, index strategy, and RLS policies for the GGID database.

---

## ER Diagram

```
┌─────────────────┐       ┌─────────────────┐       ┌─────────────────┐
│    tenants      │       │     users       │       │  credentials    │
├─────────────────┤       ├─────────────────┤       ├─────────────────┤
│ id (PK)         │◄──┐   │ id (PK)         │   ┌──►│ id (PK)         │
│ name            │   │   │ tenant_id (FK)  │   │   │ tenant_id (FK)  │
│ created_at      │   │   │ username        │   │   │ user_id (FK)    │
└─────────────────┘   │   │ email           │   │   │ type            │
                      │   │ phone           │   │   │ secret (hashed) │
                      │   │ status          │   │   │ created_at      │
                      │   │ email_verified  │   │   └─────────────────┘
                      │   │ display_name    │   │
                      │   │ created_at      │   │   ┌─────────────────┐
                      │   └────────┬────────┘   │   │  user_roles     │
                      │            │            │   ├─────────────────┤
                      │            │            └──►│ id (PK)         │
                      ├─── FK references tenant_id     │ tenant_id (FK)  │
                      │            │                │ user_id (FK)    │
                      │            ▼                │ role_id (FK)    │
                      │   ┌─────────────────┐        └────────┬────────┘
                      │   │     roles       │                 │
                      │   ├─────────────────┤                 │
                      │   │ id (PK)         │◄────────────────┘
                      │   │ tenant_id (FK)  │
                      │   │ key             │     ┌─────────────────┐
                      │   │ name            │     │ role_permissions│
                      │   │ description     │     ├─────────────────┤
                      │   │ parent_role_id  │◄──┐ │ id (PK)         │
                      │   │ created_at      │   │ │ tenant_id (FK)  │
                      │   └─────────────────┘   │ │ role_id (FK)    │
                      │           ▲             │ │ resource        │
                      │           └── self-ref  │ │ action          │
                      │                         │ └─────────────────┘
                      │
                      │   ┌─────────────────┐
                      │   │ organizations   │
                      │   ├─────────────────┤
                      │   │ id (PK)         │
                      │   │ tenant_id (FK)  │
                      │   │ name            │
                      │   │ parent_id       │◄──┐
                      │   │ description     │   │ self-ref (tree)
                      │   │ created_at      │   │
                      │   └────────┬────────┘   │
                      │            │            │
                      │            ▼            │
                      │   ┌─────────────────┐   │
                      │   │  org_members    │   │
                      │   ├─────────────────┤   │
                      │   │ id (PK)         │   │
                      │   │ tenant_id (FK)  │   │
                      │   │ org_id (FK)     │───┘
                      │   │ user_id (FK)    │───► users
                      │   │ title           │
                      │   └─────────────────┘
                      │
                      │   ┌─────────────────┐     ┌─────────────────┐
                      │   │  departments    │     │     teams       │
                      │   ├─────────────────┤     ├─────────────────┤
                      │   │ id (PK)         │     │ id (PK)         │
                      │   │ tenant_id (FK)  │     │ tenant_id (FK)  │
                      │   │ org_id (FK)     │     │ org_id (FK)     │
                      │   │ name            │     │ name            │
                      │   │ parent_id       │     │ created_at      │
                      │   └─────────────────┘     └─────────────────┘
                      │
                      │   ┌─────────────────┐
                      │   │   policies      │
                      │   ├─────────────────┤
                      │   │ id (PK)         │
                      │   │ tenant_id (FK)  │
                      │   │ name            │
                      │   │ effect (allow/  │
                      │   │       deny)     │
                      │   │ actions (JSONB) │
                      │   │ resources(JSONB)│
                      │   │ conditions(JSON)│
                      │   │ priority        │
                      │   │ created_at      │
                      │   └─────────────────┘
                      │
                      │   ┌─────────────────┐     ┌─────────────────┐
                      │   │ audit_events    │     │ oauth_clients   │
                      │   ├─────────────────┤     ├─────────────────┤
                      │   │ id (PK)         │     │ id (PK)         │
                      │   │ tenant_id (FK)  │     │ tenant_id (FK)  │
                      │   │ actor_id        │     │ name            │
                      │   │ actor_name      │     │ client_id       │
                      │   │ action          │     │ client_secret   │
                      │   │ result          │     │ redirect_uris   │
                      │   │ resource_type   │     │ grant_types     │
                      │   │ resource_id     │     │ scopes          │
                      │   │ ip_address      │     │ created_at      │
                      │   │ user_agent      │     └─────────────────┘
                      │   │ metadata (JSONB)│
                      │   │ created_at      │
                      │   └─────────────────┘
                      │
                      └── All tables have tenant_id FK to tenants.id
```

---

## Table Definitions

### tenants

```sql
CREATE TABLE tenants (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);
```

### users

```sql
CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    username        TEXT NOT NULL,
    email           TEXT NOT NULL,
    phone           TEXT,
    status          TEXT DEFAULT 'active',  -- active|locked|inactive|suspended
    email_verified  BOOLEAN DEFAULT false,
    display_name    TEXT,
    locale          TEXT DEFAULT 'en-US',
    timezone        TEXT DEFAULT 'UTC',
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, username),
    UNIQUE(tenant_id, email)
);
```

### credentials

```sql
CREATE TABLE credentials (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type        TEXT NOT NULL DEFAULT 'password',  -- password|webauthn|oauth
    identifier  TEXT,          -- username for password, key_id for webauthn
    secret      TEXT NOT NULL,  -- Argon2id hash or public key
    metadata    JSONB,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, user_id, type)
);
```

### roles

```sql
CREATE TABLE roles (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    key             TEXT NOT NULL,
    name            TEXT NOT NULL,
    description     TEXT,
    parent_role_id  UUID REFERENCES roles(id),
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, key)
);
```

### role_permissions

```sql
CREATE TABLE role_permissions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    role_id     UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    resource    TEXT NOT NULL,
    action      TEXT NOT NULL,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);
```

### user_roles

```sql
CREATE TABLE user_roles (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id     UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, user_id, role_id)
);
```

### organizations

```sql
CREATE TABLE organizations (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    name        TEXT NOT NULL,
    parent_id   UUID REFERENCES organizations(id),
    description TEXT,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);
```

### org_members

```sql
CREATE TABLE org_members (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    org_id      UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title       TEXT,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, org_id, user_id)
);
```

### policies

```sql
CREATE TABLE policies (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    name        TEXT NOT NULL,
    description TEXT,
    effect      TEXT NOT NULL CHECK (effect IN ('allow', 'deny')),
    actions     JSONB NOT NULL DEFAULT '[]',
    resources   JSONB NOT NULL DEFAULT '[]',
    conditions  JSONB,
    priority    INT DEFAULT 0,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);
```

### audit_events

```sql
CREATE TABLE audit_events (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants(id),
    actor_id      UUID,
    actor_name    TEXT,
    action        TEXT NOT NULL,
    result        TEXT NOT NULL,  -- success|failure
    resource_type TEXT,
    resource_id   TEXT,
    ip_address    TEXT,
    user_agent    TEXT,
    metadata      JSONB,
    created_at    TIMESTAMPTZ DEFAULT NOW()
);
```

### oauth_clients

```sql
CREATE TABLE oauth_clients (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    name            TEXT NOT NULL,
    client_id       TEXT NOT NULL UNIQUE,
    client_secret   TEXT NOT NULL,
    redirect_uris   JSONB NOT NULL DEFAULT '[]',
    grant_types     JSONB NOT NULL DEFAULT '["authorization_code","refresh_token"]',
    scopes          JSONB NOT NULL DEFAULT '["openid","profile","email"]',
    created_at      TIMESTAMPTZ DEFAULT NOW()
);
```

---

## Index Strategy

### Rule: tenant_id First

All indexes on multi-tenant tables include `tenant_id` as the first column.

```sql
-- Users
CREATE INDEX idx_users_tenant_username ON users (tenant_id, username);
CREATE INDEX idx_users_tenant_email    ON users (tenant_id, email);
CREATE INDEX idx_users_tenant_status   ON users (tenant_id, status);

-- Credentials
CREATE INDEX idx_cred_tenant_user      ON credentials (tenant_id, user_id);

-- Roles
CREATE INDEX idx_roles_tenant_key      ON roles (tenant_id, key);
CREATE INDEX idx_roles_tenant_parent   ON roles (tenant_id, parent_role_id);

-- Role permissions
CREATE INDEX idx_role_perm_tenant_role ON role_permissions (tenant_id, role_id);
CREATE INDEX idx_role_perm_tenant_res  ON role_permissions (tenant_id, resource);

-- User roles
CREATE INDEX idx_user_roles_tenant_user ON user_roles (tenant_id, user_id);
CREATE INDEX idx_user_roles_tenant_role ON user_roles (tenant_id, role_id);

-- Organizations
CREATE INDEX idx_orgs_tenant_parent    ON organizations (tenant_id, parent_id);

-- Policies
CREATE INDEX idx_policies_tenant_name  ON policies (tenant_id, name);

-- Audit events (time-range queries)
CREATE INDEX idx_audit_tenant_time     ON audit_events (tenant_id, created_at DESC);
CREATE INDEX idx_audit_tenant_action   ON audit_events (tenant_id, action);
CREATE INDEX idx_audit_tenant_actor    ON audit_events (tenant_id, actor_id);

-- OAuth clients
CREATE UNIQUE INDEX idx_oauth_client_id ON oauth_clients (client_id);
```

### Partitioning (for audit_events at scale)

```sql
CREATE TABLE audit_events (...) PARTITION BY RANGE (created_at);

CREATE TABLE audit_events_2024_07 PARTITION OF audit_events
  FOR VALUES FROM ('2024-07-01') TO ('2024-08-01');
```

---

## RLS Policies

### Enable and Force RLS

```sql
-- For every multi-tenant table:
ALTER TABLE users          ENABLE ROW LEVEL SECURITY;
ALTER TABLE users          FORCE ROW LEVEL SECURITY;

ALTER TABLE credentials    ENABLE ROW LEVEL SECURITY;
ALTER TABLE credentials    FORCE ROW LEVEL SECURITY;

ALTER TABLE roles          ENABLE ROW LEVEL SECURITY;
ALTER TABLE roles          FORCE ROW LEVEL SECURITY;

ALTER TABLE role_permissions ENABLE ROW LEVEL SECURITY;
ALTER TABLE role_permissions FORCE ROW LEVEL SECURITY;

ALTER TABLE user_roles     ENABLE ROW LEVEL SECURITY;
ALTER TABLE user_roles     FORCE ROW LEVEL SECURITY;

ALTER TABLE organizations  ENABLE ROW LEVEL SECURITY;
ALTER TABLE organizations  FORCE ROW LEVEL SECURITY;

ALTER TABLE org_members    ENABLE ROW LEVEL SECURITY;
ALTER TABLE org_members    FORCE ROW LEVEL SECURITY;

ALTER TABLE policies       ENABLE ROW LEVEL SECURITY;
ALTER TABLE policies       FORCE ROW LEVEL SECURITY;

ALTER TABLE audit_events   ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_events   FORCE ROW LEVEL SECURITY;

ALTER TABLE oauth_clients  ENABLE ROW LEVEL SECURITY;
ALTER TABLE oauth_clients  FORCE ROW LEVEL SECURITY;
```

### Policy Definition

```sql
CREATE POLICY tenant_isolation ON users
    USING (tenant_id = current_setting('app.tenant_id')::uuid);
```

Applied to every table listed above.

### Application-Level Enforcement

```go
// At start of every transaction:
tx.Exec(ctx, fmt.Sprintf("SET LOCAL app.tenant_id = '%s'", tenantID))
// SET LOCAL does NOT support $1 parameters in pgx v5
// UUID strings are safe for fmt.Sprintf (injection-safe)
```

### Verification

```sql
-- Check all tables have RLS enabled and forced
SELECT relname, relrowsecurity, relforcerowsecurity
FROM pg_class
WHERE relnamespace = 'public'::regnamespace
  AND relkind = 'r'
ORDER BY relname;
-- All multi-tenant tables should show: t, t
```
