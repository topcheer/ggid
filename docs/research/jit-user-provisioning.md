# Just-in-Time (JIT) User Provisioning: Multi-Source Auto-Provisioning Engine for GGID

> **Focus**: A comprehensive JIT provisioning engine that creates, updates, and de-provisions user accounts on-the-fly from multiple identity sources (SAML IdP, OIDC social login, LDAP/AD, SCIM inbound) — without pre-provisioning, with strict attribute mapping, role assignment, and audit trails.
>
> **Author**: ggcxf (researcher) | **Date**: 2026-07-17 | **Status**: Research Complete
>
> **Related**: `data-migration-bulk-import.md` covers lazy migration (legacy DB → GGID). This document covers **JIT provisioning from external IdPs** — SAML, OIDC, LDAP, SCIM.

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [What is JIT Provisioning?](#2-what-is-jit-provisioning)
3. [JIT vs SCIM vs Batch Provisioning](#3-jit-vs-scim-vs-batch-provisioning)
4. [Industry Landscape](#4-industry-landscape)
5. [GGID Current State Analysis](#5-ggid-current-state-analysis)
6. [Gap Analysis](#6-gap-analysis)
7. [Proposed Architecture: Universal JIT Engine](#7-proposed-architecture-universal-jit-engine)
8. [Attribute Mapping DSL](#8-attribute-mapping-dsl)
9. [JIT Flows by Protocol](#9-jit-flows-by-protocol)
10. [Database Schema](#10-database-schema)
11. [API Design](#11-api-design)
12. [Security Considerations](#12-security-considerations)
13. [Performance Considerations](#13-performance-considerations)
14. [Console UI Design](#14-console-ui-design)
15. [Competitive Differentiation](#15-competitive-differentiation)
16. [Implementation Backlog](#16-implementation-backlog)

---

## 1. Executive Summary

When users authenticate via SSO (SAML/OIDC) or LDAP federation, they expect instant access without waiting for an admin to pre-create their account. **Just-in-Time (JIT) provisioning** creates the user account automatically on first login, maps attributes from the external IdP to GGID's schema, assigns roles based on group/attribute mappings, and logs the provisioning event.

GGID currently supports LDAP JIT provisioning (`identity_service.go:286` — `ProvisionFromLDAP()`), LDAP auto-provision toggle (`main.go:112`), and SAML attribute extraction (`oauth/server.go:883`). However, the implementation is **LDAP-only and incomplete**:

1. **No SAML JIT** — SAML assertions contain user attributes but no user is created
2. **No OIDC JIT** — Social login (Google/GitHub) doesn't auto-provision users
3. **No SCIM inbound JIT** — SCIM endpoints exist but don't auto-create from external IdP pushes
4. **No attribute mapping engine** — Attribute names are hardcoded per protocol
5. **No role/group mapping** — JIT-provisioned users don't get roles from external group memberships
6. **No JIT update** — Existing JIT users' attributes aren't updated on subsequent logins
7. **No JIT de-provisioning** — No mechanism to disable users when they're removed from external IdP

**Recommendation**: Build a **Universal JIT Engine** that handles provisioning from all identity sources through a single configurable pipeline: extract → map → provision/update → assign roles → audit.

**Estimated effort**: 3 sprints for MVP (SAML + OIDC JIT + attribute mapping + role assignment) + 2 sprints for SCIM inbound + update/deprovision + Console UI.

---

## 2. What is JIT Provisioning?

### Definition

JIT provisioning creates or updates a user account **at the moment of authentication**, using attributes from the identity provider's assertion/token. The user does not need to be pre-created.

### How It Works

```
1. User attempts to access application via SSO
2. Redirected to external IdP (SAML/OIDC/LDAP)
3. User authenticates at IdP
4. IdP returns assertion with user attributes:
   - email, name, department, groups, roles
5. GGID receives assertion:
   a. Check: does user with this external_id exist in GGID?
      - YES → Update attributes (if JIT update enabled)
      - NO  → Create new user from attributes (if JIT provision enabled)
6. Map external groups → GGID roles
7. Assign roles to user
8. Log provisioning event to audit
9. Issue GGID tokens → user gets access
```

### Key Properties

| Property | Description |
|----------|-------------|
| **Zero-wait onboarding** | Users get access on first login — no admin pre-creation |
| **Attribute-driven** | User profile built from IdP attributes (email, name, department) |
| **Role mapping** | External groups/roles mapped to GGID roles |
| **Update-on-login** | Subsequent logins update attributes if they changed |
| **Idempotent** | Multiple logins by same user don't create duplicates |
| **Auditable** | Every provisioning event logged with source + attributes |
| **Configurable** | Admins control which attributes map to which GGID fields |

---

## 3. JIT vs SCIM vs Batch Provisioning

| Feature | JIT | SCIM | Batch/HR Sync |
|---------|-----|------|---------------|
| **When** | First login | Pre-login (push) | Scheduled (nightly) |
| **Direction** | Pull (from assertion) | Push (from IdP) | Push (from HR/AD) |
| **Latency** | Instant | Near real-time | Hours |
| **Account creation** | Automatic on login | IdP pushes user | Sync job creates |
| **Attribute updates** | On each login | On push | On sync cycle |
| **De-provisioning** | ❌ (needs SCIM) | ✅ (DELETE) | ✅ (missing → disable) |
| **Group sync** | Partial (from claims) | ✅ (Groups endpoint) | ✅ |
| **Best for** | SSO onboarding | Enterprise lifecycle | HR-driven joiner/mover/leaver |
| **Complexity** | Low | Medium | High |
| **Recommended** | **JIT + SCIM hybrid** | Yes | For large enterprises |

### The Recommended Hybrid Model

```
SCIM (day-one provisioning):
  - HR system creates user → pushes to GGID via SCIM
  - User has account before first login
  - Group memberships sync continuously
  - Deprovisioning: HR removes user → SCIM DELETE → GGID disables

JIT (fallback/supplementary):
  - User logs in via SAML SSO before SCIM push arrives
  - JIT creates account from SAML assertion
  - Subsequent SCIM pushes UPDATE the JIT-created account
  - Prevents "account not ready" situations
```

---

## 4. Industry Landscape

### Comparison Matrix

| Feature | Okta | Auth0 | Keycloak | WorkOS | **GGID (target)** |
|---------|------|-------|----------|--------|-------------------|
| **SAML JIT** | Yes | Yes | Yes | Yes | **Target** |
| **OIDC JIT** | Yes | Yes | Yes | Yes | **Target** |
| **LDAP JIT** | Yes | Via custom | Yes | No | **Existing** |
| **SCIM inbound** | Yes | Yes | Yes | Yes | **Existing (partial)** |
| **Attribute mapping UI** | Visual mapper | Custom rules | Mapper per IdP | Fixed mapping | **Configurable DSL** |
| **Role mapping** | Group→Role rules | Via Actions | Mapper per IdP | Fixed | **From groups** |
| **JIT update** | Yes | Yes | Yes | Yes | **Target** |
| **JIT de-provision** | Via SCIM | Via SCIM | Via SCIM | Via SCIM | **Via SCIM + CAEP** |
| **Open source** | No | No | Yes | No | **Yes (Apache 2.0)** |

---

## 5. GGID Current State Analysis

### Existing JIT Infrastructure

| Component | File | Status |
|-----------|------|--------|
| LDAP JIT | `services/identity/internal/service/identity_service.go:286` | **Implemented** — `ProvisionFromLDAP()` |
| LDAP auto-provision toggle | `services/auth/cmd/main.go:112` | **Implemented** — `LDAP_AUTO_PROVISION=true` |
| SAML attribute extraction | `services/oauth/internal/server/server.go:883` | **Implemented** — `saml.ExtractAttributes()` |
| SAML token issuance | `services/oauth/internal/service/oauth_service.go:752` | **Implemented** — `IssueSAMLToken()` |
| IdP config (per-tenant) | `services/identity/internal/idpconfig/idpconfig.go:58` | **Implemented** — `Create()` for SAML/OIDC/LDAP |
| Auth provider chain | `pkg/authprovider/` | **Implemented** — Local + LDAP chain |
| SCIM 2.0 handler | `services/identity/internal/scim/handler.go` | **Implemented** — user/group CRUD |
| External identity linking | `services/identity/internal/service/identity_service.go:293` | **Implemented** — `FindExternalIdentity()` |
| Joiner flow handler | `services/identity/internal/server/joiner_flow_handler.go` | **Implemented** — auto-provision apps |

### What's Missing

| # | Gap | Impact |
|---|-----|--------|
| 1 | **No SAML JIT** | SAML assertion received but no user created — existing logic only issues token |
| 2 | **No OIDC JIT** | Social login doesn't create user — returns error "user not found" |
| 3 | **No attribute mapping engine** | `ProvisionFromLDAP()` hardcodes `sAMAccountName`, `mail`, `displayName` |
| 4 | **No role/group mapping** | JIT users created with no roles — must be manually assigned |
| 5 | **No JIT update** | Second login doesn't update user attributes if they changed in IdP |
| 6 | **No JIT de-provision** | User removed from IdP but stays active in GGID forever |
| 7 | **No per-IdP JIT config** | JIT is all-or-nothing; can't enable for one IdP and not another |
| 8 | **No JIT audit trail** | No dedicated event for "user provisioned from IdP X" |

---

## 6. Gap Analysis

### Scenarios That Fail Today

| # | Scenario | Current | Expected |
|---|----------|---------|----------|
| 1 | "User logs in via SAML SSO for first time" | Error: user not found | Auto-create from SAML attributes |
| 2 | "Google social login for new user" | Error: user not found | Auto-create from OIDC claims |
| 3 | "IdP sends group 'engineering' — map to GGID role" | No mapping | Configurable: engineering → developer role |
| 4 | "User's department changes in AD" | GGID keeps old value | JIT update on next login |
| 5 | "User removed from LDAP group" | GGID keeps user active | Deprovision or disable based on policy |

---

## 7. Proposed Architecture: Universal JIT Engine

```
                    ┌──────────────────────────────────────────────┐
                    │         Auth / OAuth Service                  │
                    │                                              │
                    │   User authenticates via:                    │
                    │   ├── SAML IdP → SAML assertion              │
                    │   ├── OIDC IdP → ID Token / UserInfo        │
                    │   ├── LDAP     → LDAP attributes             │
                    │   └── SCIM    → SCIM user push               │
                    │           │                                  │
                    │           ▼                                  │
                    │   ┌──────────────────────────────────────┐   │
                    │   │   Universal JIT Engine               │   │
                    │   │                                      │   │
                    │   │  1. Extract attributes (per protocol)│   │
                    │   │  2. Resolve external ID              │   │
                    │   │  3. Check: user exists?              │   │
                    │   │     ├── YES → Update (if enabled)    │   │
                    │   │     └── NO  → Create (if enabled)    │   │
                    │   │  4. Map attributes (DSL)             │   │
                    │   │  5. Map groups → roles               │   │
                    │   │  6. Assign roles                     │   │
                    │   │  7. Audit event                      │   │
                    │   │  8. Link external identity            │   │
                    │   └──────────────────────────────────────┘   │
                    │           │                                  │
                    │           ▼                                  │
                    │   ┌──────────────────────────────────────┐   │
                    │   │   Identity Service                   │   │
                    │   │   (user CRUD + external linking)     │   │
                    │   └──────────────────────────────────────┘   │
                    └──────────────────────────────────────────────┘
```

### JIT Engine Core

```go
// services/identity/internal/service/jit_engine.go

// JITEngine handles just-in-time provisioning from all identity sources.
type JITEngine struct {
    identityService *IdentityService
    policyService   *PolicyService
    auditPublisher  audit.Publisher
    configs         JITConfigStore
}

// JITRequest is the input to the JIT engine.
type JITRequest struct {
    TenantID     uuid.UUID
    Source       string            // "saml", "oidc", "ldap", "scim"
    ExternalID   string            // Unique ID from IdP (NameID, sub, DN)
    Attributes   map[string][]string  // Raw attributes from IdP
    IdPConfigID  uuid.UUID         // Which IdP configuration triggered this
}

// JITResult is the outcome of JIT provisioning.
type JITResult struct {
    User       *domain.User
    Created    bool
    Updated    bool
    RolesAdded []string
    RolesRemoved []string
}

// Process handles a JIT provisioning request.
func (e *JITEngine) Process(ctx context.Context, req *JITRequest) (*JITResult, error) {
    // 1. Get JIT config for this IdP
    config, err := e.getConfig(ctx, req.TenantID, req.Source, req.IdPConfigID)
    if err != nil || !config.Enabled {
        return nil, ErrJITDisabled
    }
    
    // 2. Resolve external identity
    existing, _ := e.identityService.repo.FindExternalIdentity(ctx, req.TenantID, req.Source, req.ExternalID)
    
    result := &JITResult{}
    
    if existing != nil {
        // User exists — update if configured
        if config.UpdateOnLogin {
            user, err := e.updateUser(ctx, existing.UserID, req, config)
            result.User = user
            result.Updated = true
            e.audit(ctx, "jit.update", req, result)
        } else {
            result.User, _ = e.identityService.repo.GetUserByID(ctx, req.TenantID, existing.UserID)
        }
    } else {
        // User doesn't exist — create if configured
        if !config.ProvisionOnLogin {
            return nil, ErrUserNotFound
        }
        user, err := e.createUser(ctx, req, config)
        result.User = user
        result.Created = true
        
        // Link external identity
        e.identityService.repo.LinkExternalIdentity(ctx, req.TenantID, user.ID, req.Source, req.ExternalID)
        
        e.audit(ctx, "jit.create", req, result)
    }
    
    // 3. Map groups → roles
    roles := e.mapGroupsToRoles(req.Attributes, config.RoleMappings)
    if len(roles) > 0 {
        added, removed := e.syncRoles(ctx, result.User.ID, roles)
        result.RolesAdded = added
        result.RolesRemoved = removed
    }
    
    return result, nil
}
```

---

## 8. Attribute Mapping DSL

A declarative DSL maps external IdP attributes to GGID user fields:

```yaml
# JIT configuration for a SAML IdP
source: saml
idp_name: "Corporate Okta"
enabled: true
provision_on_login: true
update_on_login: true

# Attribute mapping
attribute_mapping:
  email:         "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress"
  display_name:  "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name"
  first_name:    "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/givenname"
  last_name:     "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/surname"
  department:    "department"
  title:         "title"
  phone:         "telephonenumber"

# Group → Role mapping
role_mappings:
  - external_group: "CN=engineering,OU=groups"
    ggid_role: "developer"
  - external_group: "CN=admins,OU=groups"
    ggid_role: "admin"
  - external_group: "CN=contractors,OU=groups"
    ggid_role: "viewer"

# Security constraints
security:
  allowed_domains: ["corp.com", "subsidiary.com"]
  require_verified_email: true
  reject_if_no_groups: false
  default_role: "viewer"  # if no group matches
```

### OIDC Variant

```yaml
source: oidc
idp_name: "Google Workspace"
enabled: true
provision_on_login: true
update_on_login: true

attribute_mapping:
  email:        "email"
  display_name: "name"
  first_name:   "given_name"
  last_name:    "family_name"
  avatar:       "picture"
  locale:       "locale"

role_mappings:
  - external_claim: "hd"         # hosted domain
    claim_value: "corp.com"
    ggid_role: "employee"
  - external_claim: "groups"
    claim_value: "engineering"
    ggid_role: "developer"

security:
  allowed_domains: ["corp.com"]
  require_verified_email: true
```

### LDAP Variant

```yaml
source: ldap
idp_name: "Corporate Active Directory"
enabled: true
provision_on_login: true
update_on_login: true

attribute_mapping:
  username:      "sAMAccountName"
  email:         "mail"
  display_name:  "displayName"
  department:    "department"
  title:         "title"
  phone:         "telephoneNumber"
  manager:       "manager"

role_mappings:
  - ldap_group: "CN=Domain Admins,CN=Users"
    ggid_role: "admin"
  - ldap_group: "CN=Engineers,OU=Groups"
    ggid_role: "developer"
```

---

## 9. JIT Flows by Protocol

### SAML JIT Flow

```
User → SP (GGID) → Redirect to SAML IdP → User authenticates
→ IdP sends SAML Response with attributes
→ GGID /saml/acs handler:
  1. Parse assertion (saml.ParseAssertion)
  2. Extract NameID (external_id)
  3. Extract attributes (saml.ExtractAttributes)
  4. Call JIT Engine:
     - Check if external identity exists
     - Create or update user
     - Map groups → roles
  5. Issue SAML token / JWT
  6. Return to application
```

### OIDC JIT Flow

```
User → Application → Redirect to OIDC Provider → User authenticates
→ Provider returns authorization code
→ GGID exchanges code for tokens
→ GGID calls UserInfo endpoint (or decodes ID Token)
→ Extract claims: sub, email, name, groups
→ Call JIT Engine:
  - Check if external identity (sub) exists
  - Create or update user from claims
  - Map groups → roles
→ Issue GGID tokens
```

### LDAP JIT Flow (Existing)

```
User → GGID login → Auth provider chain:
  1. Try Local provider (password hash) — miss
  2. Try LDAP provider — hit (bind succeeds)
  3. LDAP provider returns AuthResult with attributes
  4. Call JIT Engine (ProvisionFromLDAP):
     - Check external identity
     - Create user from LDAP attributes
     - Map groups → roles
→ Issue tokens
```

### SCIM Inbound JIT Flow

```
External IdP → SCIM POST /Users → GGID SCIM handler:
  1. Parse SCIM user JSON
  2. Check if user already exists (by email or externalId)
  3. If not: create user from SCIM attributes
  4. Map SCIM groups → GGID roles
  5. Return 201 Created
→ User is pre-provisioned before first login
```

---

## 10. Database Schema

```sql
-- JIT configurations (per IdP per tenant)
CREATE TABLE jit_configs (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    source              VARCHAR(32) NOT NULL,         -- 'saml', 'oidc', 'ldap', 'scim'
    idp_config_id       UUID,                          -- Reference to idpconfig
    name                VARCHAR(128) NOT NULL,
    enabled             BOOLEAN DEFAULT true,
    provision_on_login  BOOLEAN DEFAULT true,
    update_on_login     BOOLEAN DEFAULT true,
    deprovision_on_removal BOOLEAN DEFAULT false,      -- Disable user when removed from IdP
    config_yaml         TEXT NOT NULL,                 -- Full JIT config (attribute mapping, role mapping, security)
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, source, idp_config_id)
);

-- JIT provisioning log (audit trail)
CREATE TABLE jit_events (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    user_id             UUID,                           -- GGID user ID (null if provisioning failed)
    source              VARCHAR(32) NOT NULL,
    external_id         VARCHAR(256) NOT NULL,
    action              VARCHAR(32) NOT NULL,           -- 'create', 'update', 'role_assign', 'deprovision'
    attributes_snapshot JSONB,                          -- Attributes received from IdP
    roles_assigned      JSONB DEFAULT '[]',
    roles_removed       JSONB DEFAULT '[]',
    success             BOOLEAN NOT NULL,
    error_message       TEXT,
    idp_config_id       UUID,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- External identity links (existing, enhanced)
-- This may already exist as a table — extending with metadata
CREATE TABLE external_identities (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL,
    user_id             UUID NOT NULL,
    provider_type       VARCHAR(32) NOT NULL,           -- 'saml', 'oidc', 'ldap', 'scim'
    provider_name       VARCHAR(128),                   -- "Corporate Okta", "Google"
    external_id         VARCHAR(256) NOT NULL,          -- NameID, sub, DN, externalId
    external_email      VARCHAR(256),
    external_groups     JSONB DEFAULT '[]',             -- Groups from last assertion
    last_synced_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, provider_type, external_id)
);

-- JIT role mappings (extracted from config for queryability)
CREATE TABLE jit_role_mappings (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    jit_config_id       UUID NOT NULL REFERENCES jit_configs(id) ON DELETE CASCADE,
    tenant_id           UUID NOT NULL,
    external_group      VARCHAR(256) NOT NULL,          -- "CN=engineering,..." or "engineering"
    ggid_role_key       VARCHAR(128) NOT NULL,
    priority            INT DEFAULT 0,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_jit_configs_tenant ON jit_configs (tenant_id, source, enabled);
CREATE INDEX idx_jit_events_tenant ON jit_events (tenant_id, created_at DESC);
CREATE INDEX idx_jit_events_user ON jit_events (tenant_id, user_id, created_at DESC);
CREATE INDEX idx_external_identities_lookup ON external_identities (tenant_id, provider_type, external_id);
CREATE INDEX idx_external_identities_user ON external_identities (tenant_id, user_id);
CREATE INDEX idx_jit_role_mappings_config ON jit_role_mappings (jit_config_id);
```

---

## 11. API Design

### JIT Configuration Management

```
# Create JIT configuration
POST /api/v1/identity/jit/configs
Content-Type: application/json

{
    "source": "saml",
    "idp_config_id": "uuid",
    "name": "Corporate Okta JIT",
    "enabled": true,
    "provision_on_login": true,
    "update_on_login": true,
    "deprovision_on_removal": false,
    "config_yaml": "<YAML content>"
}

# List JIT configurations
GET /api/v1/identity/jit/configs?tenant_id={tenant}

# Test JIT with sample attributes
POST /api/v1/identity/jit/configs/{id}/test
{
    "external_id": "alice@corp.com",
    "attributes": {
        "email": ["alice@corp.com"],
        "groups": ["engineering", "on-call"]
    }
}

Response:
{
    "action": "would_create",
    "mapped_fields": {
        "email": "alice@corp.com",
        "display_name": "Alice Chen"
    },
    "mapped_roles": ["developer"],
    "user_would_be_created": true
}
```

### JIT Event History

```
# Get JIT provisioning events
GET /api/v1/identity/jit/events?tenant_id={tenant}&limit=50

Response:
{
    "events": [
        {
            "id": "uuid",
            "user_id": "uuid",
            "user_email": "alice@corp.com",
            "source": "saml",
            "external_id": "alice@corp.com",
            "action": "create",
            "roles_assigned": ["developer"],
            "success": true,
            "created_at": "2026-07-17T10:15:00Z"
        }
    ],
    "total": 1247
}
```

---

## 12. Security Considerations

| Risk | Mitigation |
|------|-----------|
| **Unauthorized account creation** | JIT only processes assertions from trusted IdPs (configured per-tenant) |
| **Attribute spoofing** | SAML assertions must be signed; OIDC tokens must have valid signature from trusted issuer |
| **Email domain injection** | `allowed_domains` config rejects assertions from unapproved domains |
| **Role escalation** | Role mappings are admin-configured; users can't self-assign roles |
| **Deprovisioning gap** | JIT deprovision check + CAEP session-revoked events + SCIM DELETE |
| **Duplicate accounts** | External identity link (tenant + provider + external_id) is unique |
| **PII in logs** | Attributes snapshot stored in encrypted column; PII fields obfuscated in logs |

---

## 13. Performance Considerations

| Operation | Latency | Notes |
|-----------|---------|-------|
| External identity lookup (indexed) | <2ms | UUID + provider_type + external_id |
| User creation | 3-5ms | INSERT + external identity link |
| User update | 2-3ms | UPDATE on changed fields only |
| Role assignment | 2-5ms | Per role: check + assign/revoke |
| JIT total overhead (first login) | 10-20ms | Create path |
| JIT total overhead (subsequent) | 5-10ms | Update path (check + optional update) |

---

## 14. Console UI Design

### JIT Configuration Dashboard

```
┌──────────────────────────────────────────────────────────────────┐
│  JIT Provisioning                                                │
│                                                                  │
│  ┌────────────────┐  ┌────────────────┐  ┌────────────────┐     │
│  │  Total JIT      │  │  This Month    │  │  Deprovisioned │     │
│  │  Users: 1,247   │  │  Created: 89   │  │  This Month: 5 │     │
│  └────────────────┘  └────────────────┘  └────────────────┘     │
│                                                                  │
│  Configured Sources                                              │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │ ● Corporate Okta (SAML)     Active   850 users provisioned │  │
│  │   Provision + Update + Role Mapping                        │  │
│  │   [Configure] [Test] [View Events]                         │  │
│  ├────────────────────────────────────────────────────────────┤  │
│  │ ● Google Workspace (OIDC)   Active   320 users provisioned │  │
│  │   Provision + Update                                       │  │
│  │   [Configure] [Test] [View Events]                         │  │
│  ├────────────────────────────────────────────────────────────┤  │
│  │ ● Active Directory (LDAP)   Active   77 users provisioned  │  │
│  │   Provision + Update + Deprovision                         │  │
│  │   [Configure] [Test] [View Events]                         │  │
│  └────────────────────────────────────────────────────────────┘  │
│                                                                  │
│  + Add JIT Source                                                │
│    [SAML] [OIDC] [LDAP] [SCIM]                                  │
│                                                                  │
│  Recent Events                                                   │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │ 10:15  CREATE   alice@corp.com    SAML → developer role   │  │
│  │ 10:12  UPDATE   bob@corp.com      LDAP → dept updated      │  │
│  │ 09:45  CREATE   carol@corp.com    OIDC → employee role     │  │
│  │ 09:30  ROLES    dave@corp.com     SAML → admin added       │  │
│  └────────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────┘
```

---

## 15. Competitive Differentiation

| Feature | GGID (target) | Okta | Auth0 | Keycloak | WorkOS |
|---------|---------------|------|-------|----------|--------|
| **Multi-source JIT** | **SAML+OIDC+LDAP+SCIM** | Yes | Yes | SAML+LDAP | SAML+OIDC |
| **Attribute mapping DSL** | **YAML declarative** | Visual UI | JS Actions | Per-IdP config | Fixed |
| **Role mapping** | **From groups/claims** | Group rules | Actions | Mapper | Fixed |
| **JIT update** | **Yes** | Yes | Yes | Yes | Yes |
| **JIT deprovision** | **Via SCIM + CAEP** | Via SCIM | Via SCIM | Via SCIM | Via SCIM |
| **Dry-run/test** | **Yes** | No | No | No | No |
| **Open source** | **Yes (Apache 2.0)** | No | No | Yes | No |

**Key differentiator**: GGID would be the only open-source IAM with declarative attribute mapping DSL, multi-source JIT (4 protocols), and dry-run test mode.

---

## 16. Implementation Backlog

### P0 — Core JIT Engine (3 sprints)

| # | Task | Description | Effort |
|---|------|-------------|--------|
| 1 | JIT config data model | PostgreSQL tables for configs, events, external identities, role mappings | 2 days |
| 2 | Universal JIT engine | Protocol-agnostic Process() pipeline: extract → resolve → create/update → map → audit | 5 days |
| 3 | Attribute mapping DSL parser | Parse YAML mapping definitions | 3 days |
| 4 | SAML JIT integration | Wire into /saml/acs handler — create/update from SAML assertion | 3 days |
| 5 | OIDC JIT integration | Wire into OIDC callback — create/update from ID Token/UserInfo | 3 days |
| 6 | Role/group mapping engine | Map external groups to GGID roles on provisioning | 3 days |
| 7 | JIT config API | CRUD endpoints for configurations + test mode | 3 days |
| 8 | Unit tests | 90%+ coverage for engine, mapper, role sync | 3 days |

### P1 — Enhanced Features (2 sprints)

| # | Task | Description | Effort |
|---|------|-------------|--------|
| 9 | LDAP JIT enhancement | Refactor existing ProvisionFromLDAP() to use universal engine | 2 days |
| 10 | SCIM inbound JIT | Enhance SCIM handler to create from external IdP pushes | 3 days |
| 11 | JIT update on login | Update user attributes + roles on each login (if changed) | 3 days |
| 12 | JIT deprovisioning | Disable users removed from external IdP (via SCIM DELETE + periodic check) | 3 days |
| 13 | Dry-run test mode | Simulate JIT with sample attributes, no DB writes | 2 days |
| 14 | Integration tests | End-to-end JIT for all 4 protocols | 3 days |

### P2 — Console UI (2 sprints)

| # | Task | Description | Effort |
|---|------|-------------|--------|
| 15 | JIT dashboard | Stats cards + source list + recent events | 3 days |
| 16 | Config editor | YAML editor with validation + role mapping table | 3 days |
| 17 | Test/simulation UI | Input form for sample attributes → preview mapped result | 2 days |
| 18 | Event log viewer | Filterable JIT event history with drill-down | 2 days |

### P3 — Advanced Features (Future)

| # | Task | Description |
|---|------|-------------|
| 19 | CAEP-driven deprovisioning | Real-time deprovision when CAEP session-revoked received |
| 20 | Multi-IdP conflict resolution | User exists in multiple IdPs — merge or link strategy |
| 21 | HR-driven provisioning | Workday/BambooHR integration for joiner/mover/leaver |
| 22 | Provisioning webhooks | Notify external systems when user is provisioned/deprovisioned |
| 23 | Attribute transformation | Custom CEL expressions for complex attribute mappings |
| 24 | Provisioning policies | Per-application provisioning rules (app A allows JIT, app B requires SCIM) |

---

## References

- [Descope: JIT Provisioning Guide](https://docs.descope.com/sso/jit-provisioning) — SSO JIT best practices
- [WorkOS: SCIM vs JIT](https://workos.com/guide/scim-vs-jit) — Comparison guide
- [Clerk: SCIM vs JIT Provisioning](https://clerk.com/articles/scim-vs-jit-provisioning-when-to-use-each) — When to use each approach
- [RFC 7643: SCIM Core Schema](https://datatracker.ietf.org/doc/html/rfc7643) — User/Group schema
- [RFC 7644: SCIM Protocol](https://datatracker.ietf.org/doc/html/rfc7644) — Provisioning protocol
- [SAML 2.0 Assertion Attributes](https://docs.oasis-open.org/security/saml/v2.0/saml-core-2.0-os.pdf) — Attribute statements
- [OIDC Standard Claims](https://openid.net/specs/openid-connect-core-1_0.html#StandardClaims) — User info claims
- [GGID ProvisionFromLDAP](../services/identity/internal/service/identity_service.go) — Existing LDAP JIT at line 286
- [GGID SAML Attribute Extraction](../services/oauth/internal/server/server.go) — SAML parsing at line 883
- [GGID SCIM Handler](../services/identity/internal/scim/handler.go) — SCIM 2.0 CRUD at line 37
