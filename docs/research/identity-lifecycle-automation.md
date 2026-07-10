# Identity Lifecycle Automation

> Research document for automating the Joiner-Mover-Leaver (JML) lifecycle in GGID.
> Covers JIT provisioning, deprovisioning cascade, role mining, access review, and IGA integration.

## 1. Overview

The identity lifecycle follows three phases:

- **Joiner** — new person enters; identity created, roles and entitlements assigned.
- **Mover** — role/department change; old permissions revoked, new ones granted.
  Most error-prone phase — leading source of **access creep**.
- **Leaver** — departure; all access revoked promptly and completely.

**Automation goals:** reduce manual admin, ensure compliance (SOC 2, SOX, ISO 27001, GDPR),
speed provisioning from days to minutes, eliminate orphaned accounts.

**Key phases of identity governance:**

| Phase | Description |
|-------|-------------|
| Provisioning | Create user, assign base role, set MFA |
| Role assignment | Grant/modify roles based on job function |
| Access review | Periodic certification of who has access to what |
| Deprovisioning | Disable account, revoke all tokens/sessions/MFA/grants |

**IGA (Identity Governance and Administration) market:** SailPoint IdentityIQ/Identity
Security Cloud, Okta IGA (formerly identity governance), Saviynt EIC, One Identity,
Oracle Identity Governance. These tools handle JML automation, access certification,
role mining, and compliance reporting for enterprise deployments.

---

## 2. JIT (Just-in-Time) Provisioning

JIT provisioning creates user accounts at login time rather than in advance. When a user
authenticates via SAML/OIDC for the first time, GGID inspects the IdP-provided attributes
and creates a local account automatically.

### SCIM-Based JIT (SAML/OIDC)

When a user logs in via SAML or OIDC for the first time, GGID checks whether a local user
exists (by email or NameID). If not, it auto-creates an account using attributes from the
assertion or ID token:

- Map IdP attributes (`email`, `given_name`, `family_name`, `groups`) to the local user model
- Assign a default role based on IdP group membership (e.g. `admins` group → admin role)
- Per-tenant JIT config: enabled/disabled flag, default role, attribute mapping table

GGID already has the data model for this — `IdPConfig` in `idp_federation.go` includes
`AutoProvision bool` and `AttrMap map[string]string`. The missing piece is the JIT
provisioner logic that runs during the SAML/OIDC callback handler:

```go
// JITProvisioner auto-creates users from IdP attributes during first login.
type JITProvisioner struct {
    identityClient IdentityClient
    idpConfig      IdPConfig
    auditPublisher *audit.Publisher
}

func (j *JITProvisioner) ProvisionOrCreate(ctx context.Context, attrs map[string]string) (*domain.User, error) {
    email := attrs[j.idpConfig.AttrMap["email"]]
    user, _ := j.identityClient.FindByEmail(ctx, email)
    if user != nil {
        return user, nil // existing user
    }
    if !j.idpConfig.AutoProvision {
        return nil, ErrUserNotFound
    }
    newUser := &domain.User{
        ID: uuid.New(), Email: email,
        DisplayName: attrs[j.idpConfig.AttrMap["name"]],
        Status: domain.UserStatusActive,
    }
    if err := j.identityClient.Create(ctx, newUser); err != nil {
        return nil, fmt.Errorf("jit create: %w", err)
    }
    role := j.resolveDefaultRole(attrs["groups"])
    _ = j.identityClient.AssignRole(ctx, newUser.ID, role)
    j.auditPublisher.PublishAsync(audit.NewEvent("user.jit_provisioned",
        "success", newUser.TenantID, newUser.ID))
    return newUser, nil
}
```

### LDAP-Sourced Provisioning

For organizations using LDAP/Active Directory as the system of record, GGID can run a
scheduled sync job that queries LDAP for user changes and applies them:

- **Create:** new LDAP users appear in GGID automatically
- **Update:** attribute changes (name, email, department) sync to the GGID user record
- **Disable:** users disabled in LDAP are disabled in GGID (triggers deprovisioning cascade)
- **Group sync:** LDAP group membership maps to GGID roles

GGID already has an LDAP auth provider (`local_provider.go` + LDAP wiring in `cmd/main.go`).
The missing piece is a scheduled sync job:

```go
// LDAPSyncJob runs on a cron schedule to sync LDAP → GGID.
type LDAPSyncJob struct {
    ldap        *authprovider.LDAPProvider
    identitySvc IdentityService
    interval    time.Duration
}

func (j *LDAPSyncJob) Run(ctx context.Context) error {
    users, err := j.ldap.Search(ctx, j.ldap.BaseDN, j.ldap.UserFilter)
    if err != nil {
        return fmt.Errorf("ldap search: %w", err)
    }
    for _, lu := range users {
        existing, _ := j.identitySvc.FindByExternalID(ctx, lu.DN)
        switch {
        case existing == nil:
            j.createUser(ctx, lu)
        case !lu.Active && existing.Status != domain.UserStatusDisabled:
            j.identitySvc.Disable(ctx, existing.ID) // triggers cascade
        default:
            j.updateUser(ctx, existing, lu)
        }
    }
    return nil
}
```
Config: sync interval (default 15 min), LDAP filter, attribute mapping, dry-run mode.


### Inbound SCIM Provisioning

External IdPs (Okta, Entra ID, Azure AD) can push user lifecycle events to GGID via
SCIM 2.0. GGID already implements the SCIM 2.0 endpoints:

```
POST   /scim/v2/Users         → createUser
GET    /scim/v2/Users/{id}    → getUser
PUT    /scim/v2/Users/{id}    → updateUser (full replace)
PATCH  /scim/v2/Users/{id}    → updateUser (partial: activate/deactivate, group changes)
DELETE /scim/v2/Users/{id}    → deleteUser (sets status=disabled)
```

When an IdP sends a PATCH setting `active=false`, GGID should trigger the deprovisioning
cascade (Section 3). When group membership changes arrive via PATCH, GGID updates role
assignments accordingly.

---

## 3. Deprovisioning Cascade

When a user is disabled or deleted, a multi-step cascade ensures **complete access
revocation**. This is the most security-critical automation in the lifecycle — a missed
step means an orphaned session or token that an attacker can exploit.

### Cascade Steps

1. **Disable user account** — set `status=disabled` in identity service
2. **Revoke all active sessions** — call `SessionService.RevokeAllForUser()` (Redis)
3. **Revoke all tokens** — call `TokenService.RevokeAllForUser()` (refresh tokens + JWT jti blacklist)
4. **Remove from all groups/roles** — delete role assignments in policy service
5. **Revoke OAuth grants** — delete user consent records for all relying parties
6. **Disable MFA credentials** — mark TOTP/WebAuthn credentials as inactive (prevent step-up)
7. **Emit audit events** — one event per cascade step via NATS (`user.deprovisioned`, `session.revoked`, etc.)
8. **Emit CAEP/RISC events** — notify external relying parties via SSE/CAEP feed
9. **Schedule data retention** — GDPR: delete or anonymize PII after N days (configurable per tenant)

### DeprovisioningService

GGID already has partial deprovisioning in `AuthService.LogoutAll()` which revokes
sessions + refresh tokens. The proposed `DeprovisioningService` extends this to a full
orchestration:

```go
// DeprovisioningService orchestrates the full cascade.
// Steps 1–6 are transactional; steps 7–9 are best-effort via NATS.
type DeprovisioningService struct {
    identitySvc    IdentityService
    sessionSvc     *SessionService
    tokenSvc       *TokenService
    policySvc      PolicyService
    auditPublisher *audit.Publisher
}

func (d *DeprovisioningService) DeprovisionUser(
    ctx context.Context, tenantID, userID uuid.UUID, reason string,
) error {
    if err := d.identitySvc.Disable(ctx, tenantID, userID); err != nil {
        return fmt.Errorf("disable user: %w", err)
    }
    _ = d.sessionSvc.RevokeAllForUser(ctx, tenantID, userID, uuid.Nil)
    _ = d.tokenSvc.RevokeAllForUser(ctx, tenantID, userID)
    _ = d.policySvc.RemoveAllAssignments(ctx, tenantID, userID)
    _ = d.policySvc.RevokeAllGrants(ctx, tenantID, userID)
    _ = d.identitySvc.DisableAllMFA(ctx, tenantID, userID)
    d.auditPublisher.PublishAsync(audit.Event{
        Action: "user.deprovisioned", Result: "success",
        TenantID: tenantID, ActorID: userID,
        Metadata: map[string]any{"reason": reason},
    })
    d.publishCAEP(ctx, tenantID, userID, "sessions-revoked")
    d.scheduleRetention(ctx, tenantID, userID, 90*24*time.Hour)
    return nil
}
```

### Sequence Diagram

```
Admin/API          Identity         Auth(Token/Sess)    Policy         Audit(NATS)
    │                  │                  │                │                │
    │── DisableUser ──▶│                  │                │                │
    │                  │── status=disabled│                │                │
    │                  │                  │                │                │
    │── Deprovision ────────────────────▶│                │                │
    │                  │  RevokeAllForUser (sessions)      │                │
    │                  │  RevokeAllForUser (tokens+jti)    │                │
    │                  │                  │                │                │
    │── RemoveAssignments ───────────────────────────────▶│                │
    │                  │                  │  delete roles  │                │
    │                  │                  │  delete grants │                │
    │                  │                  │                │                │
    │── DisableAllMFA ─▶│                  │                │                │
    │                  │                  │                │                │
    │── Audit + CAEP ─────────────────────────────────────────────────────▶│
    │                  │                  │                │  PublishAsync  │
    │                  │                  │                │  CAEP subject  │
    │◀── Done ─────────│                  │                │                │
```

Failed cascade steps are retried via a NATS JetStream durable consumer. Each step is
idempotent (re-running a revoke on an already-revoked session is a no-op).

---

## 4. Role Mining

Role mining analyzes existing access patterns to discover optimal role definitions. This
helps organizations transition from ad-hoc individual grants to a clean RBAC model.

**Algorithms:**

| Algorithm | Description | Use case |
|-----------|-------------|----------|
| Intersection | Find common permission sets shared by many users | Core roles (e.g. "all engineers have repo:read") |
| Union | Aggregate all permissions of a user group into one role | Functional roles (e.g. "frontend team") |
| Clustering | Group users by similar permission vectors (k-means, hierarchical) | Discover hidden role patterns |

**Input:** user-permission matrix — who has access to what (from policy service grants).
**Output:** suggested roles with member lists and permission sets.

```go
type RoleMiner struct { policySvc PolicyService }

type SuggestedRole struct {
    Name        string      `json:"name"`
    Permissions []string    `json:"permissions"`
    MemberCount int         `json:"member_count"`
    Members     []uuid.UUID `json:"members"`
}

func (rm *RoleMiner) MineRoles(ctx context.Context, tenantID uuid.UUID) ([]SuggestedRole, error) {
    grants, err := rm.policySvc.ListAllGrants(ctx, tenantID)
    if err != nil {
        return nil, err
    }
    matrix := buildUserPermMatrix(grants) // map[userID]map[perm]bool
    permSets := findCommonSets(matrix, minMembers)
    return rankRoles(permSets), nil
}
```

**GGID approach:** query the policy service for current grants via `ListAllGrants`,
build the user-permission matrix in memory, and run intersection/clustering to suggest
roles. Results are reviewed by an admin before promotion to actual role definitions.

---

## 5. Access Review Automation

Periodic access certification (also called "access recertification" or "user access
review") is a compliance requirement for SOC 2, SOX, and ISO 27001. Managers or
application owners must periodically review and attest that each team member's access
is still appropriate.

**Triggers:** quarterly schedule, role change event, high-risk alert, new hire probation end.

**Workflow:**

1. **Generate** access report per user/role from policy service
2. **Assign** to reviewer (direct manager or resource owner)
3. **Notify** reviewer via email + NATS event
4. **Review** — reviewer approves or revokes each access item
5. **Auto-revoke** denied items and expired reviews (cron job)
6. **Audit trail** — every decision is logged for compliance evidence

```go
type AccessReview struct {
    ID         uuid.UUID    `json:"id"`
    TenantID   uuid.UUID    `json:"tenant_id"`
    ReviewerID uuid.UUID    `json:"reviewer_id"`
    RevieweeID uuid.UUID    `json:"reviewee_id"`
    Items      []AccessItem `json:"items"`
    Deadline   time.Time    `json:"deadline"`
    Status     string       `json:"status"` // pending|approved|revoked|expired
}

type AccessItem struct {
    ResourceID uuid.UUID `json:"resource_id"`
    Permission string    `json:"permission"`
    Decision   string    `json:"decision"` // ""|"approve"|"revoke"
}
```

**API endpoints:**

```
GET  /api/v1/access-reviews               — list reviews (filter by reviewer, status)
GET  /api/v1/access-reviews/{id}          — get review detail with items
POST /api/v1/access-reviews/{id}/decision — submit decision (approve/revoke per item)
POST /api/v1/access-reviews               — admin: create review campaign
```

**Auto-revoke cron:** a scheduled job processes all reviews past their deadline with
undecided items — those items are auto-revoked (fail-closed for compliance). This ensures
that even if a reviewer ignores the notification, access doesn't persist unreviewed.

---

## 6. IGA Integration

### SailPoint Integration

GGID acts as a **managed system** in SailPoint IdentityIQ/Identity Security Cloud:

- **Connector:** SCIM 2.0 — SailPoint calls GGID's `/scim/v2/` endpoints
- **Aggregation:** SailPoint runs `GET /scim/v2/Users` to pull the full user inventory
- **Provisioning:** SailPoint sends `POST`/`PATCH`/`DELETE` to manage lifecycle
- **Certification:** SailPoint aggregates GGID access data and runs access review campaigns
  in its own UI, with revocation results pushed back via SCIM PATCH

**Configuration:** register GGID's SCIM base URL + bearer token in SailPoint's application
definition. Map SailPoint identity attributes to SCIM user schema fields.

### Okta IGA Integration

Okta's lifecycle management workflows integrate similarly:

- **Lifecycle workflow:** Okta → SCIM → GGID (on joiner/mover/leaver events)
- **Access requests:** Okta workflow → GGID API → approval → grant
- **Group push:** Okta groups → SCIM PATCH group membership → GGID roles

### GGID Native IGA (Future)

Long-term, GGID can reduce dependency on external IGA tools by offering built-in:

- Access review workflows (Section 5)
- Role mining (Section 4)
- Compliance reporting (export to CSV/PDF for auditors)
- Policy violation detection (detect toxic combinations — e.g. user with both
  "payment:approve" and "vendor:create")

---

## 7. GGID Current Capabilities

| Capability | Status | Gap |
|-----------|--------|-----|
| JIT via SAML | Partial | `IdPConfig.AutoProvision` + `AttrMap` defined but JIT logic not wired into SAML callback handler |
| JIT via OIDC | Not implemented | No OIDC claim → user auto-creation |
| LDAP sync | Manual | `LDAPProvider` exists for auth, but no scheduled sync job |
| SCIM provisioning (inbound) | Skeleton | CRUD endpoints exist; PATCH group membership → role mapping not done |
| SCIM provisioning (outbound) | Not implemented | GGID cannot push to downstream SCIM apps |
| Deprovisioning cascade | Partial | `LogoutAll()` revokes sessions+tokens; no MFA/grant/role/group revocation, no CAEP |
| Role mining | Not implemented | — |
| Access review | Not implemented | — |
| IGA connector (SCIM for SailPoint/Okta) | Skeleton | SCIM endpoints exist but not certified against SailPoint/Okta connectors |
| Audit event emission | Implemented | `pkg/audit.Publisher` via NATS JetStream |
| Session revocation | Implemented | `SessionService.RevokeAllForUser` |
| Token revocation | Implemented | `TokenService.RevokeAllForUser` + jti blacklist |

---

## 8. Roadmap

| Phase | Capability | Priority | Effort | Rationale |
|-------|-----------|----------|--------|-----------|
| 1 | Deprovisioning cascade | P0 | ~2 weeks | Security-critical — orphaned sessions/tokens are exploitable |
| 2 | JIT provisioning (SAML) | P1 | ~1 week | `IdPConfig` model exists; needs callback handler wiring |
| 2 | JIT provisioning (OIDC) | P1 | ~1 week | Parse ID token claims → create user |
| 3 | SCIM inbound completion | P1 | ~2 weeks | PATCH group→role mapping, SailPoint/Okta certification |
| 4 | Access review automation | P2 | ~3 weeks | New service + API + cron + notification |
| 5 | Role mining | P3 | ~2 weeks | Analytical — query grants, suggest roles |
| 5 | LDAP scheduled sync | P3 | ~1 week | Cron job wrapping existing LDAPProvider |
| 6 | SCIM outbound (downstream apps) | P3 | ~3 weeks | Push lifecycle events to Slack, Google Workspace, etc. |

**Phase 1 (deprovisioning cascade)** is the highest-impact, lowest-risk improvement —
it leverages existing `SessionService`, `TokenService`, and `audit.Publisher`; the main
new code is the orchestration + MFA/grant revocation. **Phases 2–3** unlock enterprise
SSO (Okta/Entra auto-provisioning). **Phases 4–5** move GGID toward a native IGA platform.
