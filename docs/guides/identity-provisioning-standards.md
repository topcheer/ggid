# Identity Provisioning Standards

## Overview

Identity provisioning is the process of creating, updating, and deprovisioning user identities and their access across systems. This guide covers SCIM 2.0, just-in-time vs scheduled provisioning, HR-driven provisioning, deprovisioning automation, reconciliation, error handling, and how GGID implements provisioning standards.

## SCIM 2.0 Deep Dive

### Overview

SCIM (System for Cross-domain Identity Management) 2.0 is the IETF standard (RFC 7643, 7644) for automated user provisioning across identity systems.

### Core Resources

| Resource | Endpoint | Operations |
|----------|----------|------------|
| User | /Users | CRUD + bulk + search |
| Group | /Groups | CRUD + bulk + search |
| ServiceProviderConfig | /ServiceProviderConfig | Read-only |
| ResourceType | /ResourceTypes | Read-only |
| Schema | /Schemas | Read-only |
| Bulk | /Bulk | Bulk operations |

### User Schema

```json
{
  "schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"],
  "id": "2819c223-7f76-453a-919d-413861904646",
  "externalId": "employee-12345",
  "userName": "john.doe@example.com",
  "name": {
    "familyName": "Doe",
    "givenName": "John",
    "formatted": "John Doe"
  },
  "emails": [
    {"value": "john.doe@example.com", "type": "work", "primary": true}
  ],
  "phoneNumbers": [
    {"value": "+1-555-123-4567", "type": "mobile"}
  ],
  "addresses": [
    {"type": "work", "locality": "San Francisco", "region": "CA"}
  ],
  "active": true,
  "displayName": "John Doe",
  "title": "Software Engineer",
  "department": "Engineering",
  "meta": {
    "resourceType": "User",
    "created": "2026-01-15T10:00:00Z",
    "lastModified": "2026-01-20T14:30:00Z",
    "location": "https://ggid.example.com/scim/v2/Users/2819c223-7f76-453a-919d-413861904646"
  }
}
```

### Enterprise User Extension

```json
{
  "schemas": [
    "urn:ietf:params:scim:schemas:core:2.0:User",
    "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User"
  ],
  "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User": {
    "employeeNumber": "12345",
    "costCenter": "ENG-001",
    "organization": "Engineering",
    "division": "Platform",
    "department": "Backend",
    "manager": {
      "value": "2819c223-7f76-453a-919d-413861904646",
      "$ref": "https://ggid.example.com/scim/v2/Users/2819c223-7f76-453a-919d-413861904646",
      "displayName": "Jane Smith"
    }
  }
}
```

### SCIM Operations

#### CRUD Operations

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | /Users | Create user |
| GET | /Users/{id} | Retrieve user |
| GET | /Users?filter=... | Search users |
| PUT | /Users/{id} | Replace user (full update) |
| PATCH | /Users/{id} | Modify user (partial update) |
| DELETE | /Users/{id} | Delete user |

#### PATCH Operation

```json
{
  "schemas": ["urn:ietf:params:scim:api:messages:2.0:PatchOp"],
  "Operations": [
    {
      "op": "replace",
      "path": "name.familyName",
      "value": "Smith"
    },
    {
      "op": "replace",
      "path": "emails[type eq \"work\"].value",
      "value": "john.smith@example.com"
    },
    {
      "op": "replace",
      "path": "active",
      "value": false
    },
    {
      "op": "add",
      "path": "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User:department",
      "value": "Security"
    }
  ]
}
```

#### Filtering

SCIM 2.0 filter syntax (RFC 7644 Section 3.4.2):

```
# Equals
GET /Users?filter=userName eq "john.doe@example.com"

# Contains
GET /Users?filter=emails.value co "example.com"

# Starts with
GET /Users?filter=name.familyName sw "D"

# Present
GET /Users?filter=emails pr and emails.type eq "work"

# Complex
GET /Users?filter=active eq true and (title co "Engineer" or title co "Developer")

# With attributes
GET /Users?filter=active eq true&attributes=userName,emails,name
```

#### Bulk Operations

```json
POST /Bulk
{
  "schemas": ["urn:ietf:params:scim:api:messages:2.0:BulkRequest"],
  "failOnErrors": 5,
  "Operations": [
    {
      "method": "POST",
      "path": "/Users",
      "bulkId": "bulk-user-1",
      "data": {"userName": "user1@example.com"}
    },
    {
      "method": "POST",
      "path": "/Users",
      "bulkId": "bulk-user-2",
      "data": {"userName": "user2@example.com"}
    },
    {
      "method": "PATCH",
      "path": "/Users/2819c223",
      "data": {"Operations": [{"op": "replace", "path": "active", "value": true}]}
    }
  ]
}
```

### Authentication

SCIM endpoints use bearer token authentication:

```
Authorization: Bearer <scim_token>
```

- Token is scoped to SCIM operations only
- Separate tokens per provisioning client (e.g., Okta, Azure AD, Workday)
- Token permissions: read-only, read-write, or admin

## JIT vs Scheduled Sync

### Just-in-Time (JIT) Provisioning

Provision users at authentication time, when they first access the application.

- **Trigger**: User's first successful authentication via federation (SAML/OIDC)
- **Process**: Extract user attributes from federation assertion, create/update user in local directory, assign default roles
- **Pros**: Zero administration, immediate access, no sync delays
- **Cons**: No pre-provisioning, attribute mapping at auth time, no deprovisioning trigger
- **Best for**: SaaS apps with federated authentication, low-complexity environments

**JIT flow**:
```
User -> IDP (SAML/OIDC) -> GGID receives assertion ->
  -> Check if user exists in local directory ->
    -> Yes: Update attributes if changed -> Authenticate
    -> No: Create user with assertion attributes -> Assign default roles -> Authenticate
```

### Scheduled Synchronization

Periodically sync user identities between source and target systems.

- **Trigger**: Scheduled job (e.g., every 15 minutes, hourly, daily)
- **Process**: Query source system for user changes, map attributes, create/update/disable users in target
- **Pros**: Full control, pre-provisioning, deprovisioning on source change, reconciliation
- **Cons**: Sync delay, more complex, requires source system access
- **Best for**: Enterprise environments, HR-driven provisioning, compliance requirements

**Scheduled sync flow**:
```
Scheduler -> Query HR system for changes since last sync ->
  -> For each change: Map attributes -> Create/Update/Disable user in GGID ->
  -> Record sync result -> Report errors -> Schedule next sync
```

### Hybrid Approach

Combine JIT and scheduled sync for the best of both worlds:

- **JIT**: For federated SaaS apps (immediate access)
- **Scheduled sync**: For HR-driven lifecycle management (accurate deprovisioning)
- **Reconciliation**: Scheduled job verifies both paths agree

## HR-Driven Provisioning

### Source Systems

| HR System | Integration Method | Common Use Case |
|-----------|-------------------|-----------------|
| Workday | SCIM 2.0 / REST API | Enterprise HR |
| SAP SuccessFactors | SCIM 2.0 / OData API | Enterprise HR |
| BambooHR | REST API | SMB HR |
| ADP | REST API | Payroll + HR |
| Personio | REST API | European SMB |
| HiBob | REST API | Modern HR |
| Okta Lifecycle | SCIM 2.0 | Identity-driven |
| Azure AD | SCIM 2.0 / Graph API | Microsoft ecosystem |

### Provisioning Triggers

| HR Event | Provisioning Action |
|----------|---------------------|
| New hire | Create user, assign role based on department/position |
| Department transfer | Update department, reassign roles, trigger access review |
| Promotion | Update title, add elevated roles per new position |
| Demotion | Remove elevated roles, trigger access review |
| Leave of absence | Disable user, preserve account for return |
| Termination | Disable user, revoke all access, schedule deletion |
| Rehire | Re-enable user, restore previous access (with review) |

### Attribute Mapping

| HR Attribute | GGID User Attribute | Notes |
|--------------|---------------------|-------|
| employeeId | externalId | Unique identifier from HR |
| email | emails[work].value | Primary email |
| firstName | name.givenName | |
| lastName | name.familyName | |
| fullName | displayName | |
| department | department / enterprise extension | |
| title | title | Job title |
| manager.email | manager (SCIM reference) | Manager relationship |
| status (active/inactive) | active | Employment status |
| location | addresses[work] | Work location |
| phone | phoneNumbers[mobile] | |

### Workday Integration Example

```
Workday -> SCIM 2.0 push -> GGID SCIM endpoint ->
  -> Create user (new hire) -> Assign default role for department ->
  -> Enable user -> Send welcome email with enrollment link

Workday -> SCIM 2.0 push -> GGID SCIM endpoint ->
  -> Disable user (termination) -> Revoke all sessions ->
  -> Revoke all tokens -> Remove from all groups ->
  -> Schedule account deletion after 90 days
```

## Deprovisioning Automation

### Deprovisioning Process

Deprovisioning must be immediate, complete, and verifiable.

```
Trigger (HR termination / admin action / access review) ->
  1. Disable user account (prevent new logins)
  2. Revoke all active sessions
  3. Revoke all refresh tokens
  4. Revoke all OAuth grants
  5. Remove from all groups/roles
  6. Disable all MFA devices
  7. Revoke all API keys
  8. Revoke all delegated permissions
  9. Update audit log with deprovisioning event
  10. Schedule account deletion (per retention policy)
```

### Deprovisioning Triggers

| Trigger | Latency | Scope |
|---------|--------|-------|
| HR termination | < 15 minutes (scheduled sync) or immediate (webhook) | Full deprovisioning |
| Admin manual action | Immediate | Full or partial |
| Access review revocation | Immediate (on certification) | Specific entitlements |
| Security incident | Immediate | Full + ban + alert |
| Inactivity timeout | Scheduled (e.g., 90 days inactive) | Disable + review |
| Contractor end date | Scheduled (on end date) | Full deprovisioning |

### Graduated Deprovisioning

Not all terminations require immediate full deprovisioning:

- **Immediate full**: Security incident, fraud, forced termination
- **Immediate disable + 24h deprovision**: Standard termination
- **Disable + 7-day grace**: Amicable departure with knowledge transfer
- **Disable + 30-day archive**: Contractor end date, expected return

### Data Retention After Deprovisioning

| Data Type | Retention | Action |
|-----------|-----------|--------|
| User profile | 90 days | Anonymize after retention |
| Audit logs | Per compliance (typically 7 years) | Preserve, anonymize user reference |
| Sessions/tokens | Immediate | Delete all |
| Group memberships | Immediate | Remove all |
| MFA devices | Immediate | Delete all |
| API keys | Immediate | Revoke all |
| User-generated content | Per data policy | Transfer ownership or archive |

## Reconciliation

### Purpose

Reconciliation verifies that provisioning state matches the source of truth, detecting and correcting drift.

### Reconciliation Process

```
1. Export all users from source (HR system)
2. Export all users from GGID
3. Compare by externalId/employeeId
4. Identify discrepancies:
   - Users in GGID but not in source: orphaned accounts
   - Users in source but not in GGID: provisioning failure
   - Attribute mismatches: sync drift
   - Status mismatches (active in GGID, inactive in source): deprovisioning gap
5. Generate reconciliation report
6. Auto-remediate where safe (e.g., update attributes)
7. Flag for manual review (e.g., orphaned accounts with active access)
```

### Reconciliation Schedule

| Frequency | Scope | Purpose |
|----------|-------|---------|
| Daily | Status check (active/inactive) | Quick deprovisioning gap detection |
| Weekly | Full attribute sync | Detect attribute drift |
| Monthly | Full reconciliation | Comprehensive audit |
| On-demand | Triggered by alert | Investigate specific discrepancies |

### Drift Detection

| Drift Type | Detection Method | Auto-Remediate |
|-----------|-----------------|----------------|
| Orphaned account (not in HR, active in GGID) | Full reconciliation | Disable + alert |
| Status mismatch (inactive in HR, active in GGID) | Daily status check | Disable |
| Attribute drift (name/email changed in HR) | Weekly full sync | Update |
| Role drift (role assigned outside provisioning) | Access review | Alert for review |
| Group drift (group membership changed manually) | Group reconciliation | Alert for review |

## Error Handling and Retry

### Error Categories

| Error | Cause | Action |
|-------|-------|--------|
| Source system unavailable | Network, maintenance | Retry with backoff, alert after 3 failures |
| SCIM API error (400) | Invalid data | Log, alert admin, skip record |
| SCIM API error (401/403) | Token expired/revoked | Refresh token, retry, alert if persistent |
| SCIM API error (404) | User not found | Skip update, log, reconcile later |
| SCIM API error (409) | Duplicate user | Merge or skip, log for review |
| SCIM API error (500) | Target system error | Retry with backoff, alert after 3 failures |
| Rate limited (429) | Too many requests | Honor Retry-After, backoff |
| Timeout | Slow response | Retry, increase timeout, alert if persistent |

### Retry Strategy

```
Attempt 1: Immediate
Attempt 2: 30 seconds
Attempt 3: 2 minutes
Attempt 4: 10 minutes
Attempt 5: 60 minutes
-> After 5 attempts: Log to dead letter queue, alert admin
```

### Dead Letter Queue

Failed provisioning operations that exhaust retries are sent to a dead letter queue for manual intervention:

- **Storage**: Persistent queue (database or message queue)
- **Content**: Original operation, error details, retry history
- **Processing**: Admin reviews, fixes data, requeues or discards
- **Alerting**: Alert when dead letter queue grows or critical operations fail
- **Cleanup**: Auto-remove resolved entries, retain for audit

### Monitoring and Alerting

| Metric | Alert Threshold |
|--------|----------------|
| Sync success rate | < 98% |
| Sync duration | > 2x normal |
| Failed operations | > 10 in a window |
| Dead letter queue size | > 50 entries |
| Orphaned accounts | > 0 |
| Deprovisioning delay | > 15 minutes |
| Source system availability | < 99.5% |

## GGID Provisioning Standards

### SCIM 2.0 Support

GGID implements SCIM 2.0 endpoints for inbound and outbound provisioning:

- **Endpoint**: `/scim/v2/Users`, `/scim/v2/Groups`
- **Authentication**: Bearer token, scoped to SCIM operations
- **Operations**: Full CRUD, PATCH, bulk, search with filtering
- **Schema**: Core User + Enterprise User extension
- **Custom attributes**: Support for custom schema extensions

### Provisioning Architecture

```
HR System (Workday/SAP/BambooHR)
  | SCIM 2.0 / REST API
  v
GGID SCIM Endpoint
  | Internal processing
  v
GGID Identity Service
  | Attribute mapping + role assignment
  v
GGID User Directory (PostgreSQL)
  | Provisioning events
  v
GGID Audit Service (NATS JetStream)
  | Notifications
  v
GGID Notification Service
```

### JIT Provisioning in GGID

When a federated user authenticates for the first time:

1. SAML/OIDC assertion received from IdP
2. Extract user attributes from assertion
3. Query GGID user directory by `userName` or `externalId`
4. If not found: Create user with assertion attributes, assign default role
5. If found: Update attributes if changed
6. Continue authentication flow

### Scheduled Sync in GGID

GGID supports scheduled sync via integration connectors:

- **Connector framework**: Pluggable connectors for different HR systems
- **Sync scheduler**: Cron-based scheduling with configurable intervals
- **Attribute mapping**: Configurable mapping table per connector
- **Delta sync**: Sync only changed records since last sync timestamp
- **Full sync**: Periodic full reconciliation (weekly/monthly)

### Deprovisioning in GGID

GGID's deprovisioning pipeline:

- **Session revocation**: Revoke all active sessions via session management API
- **Token revocation**: Revoke all OAuth tokens via token revocation endpoint
- **Group removal**: Remove from all groups via SCIM PATCH or internal API
- **MFA device removal**: Remove all registered MFA devices
- **API key revocation**: Revoke all API keys
- **Account disable**: Set `active=false` in user directory
- **Audit logging**: Record full deprovisioning event with all actions taken
- **Scheduled deletion**: Configurable retention period before account deletion

### Reconciliation in GGID

- **Scheduled reconciliation**: Daily status check, weekly attribute sync, monthly full
- **Reconciliation report**: Generated and stored in audit service
- **Alerting**: Orphaned accounts and deprovisioning gaps trigger alerts
- **Auto-remediation**: Status mismatches auto-remediated; attribute drift logged for review

## Best Practices

1. **Automate everything**: Manual provisioning is error-prone and slow
2. **Single source of truth**: HR system should be the authoritative source
3. **Deprovision immediately**: Delay in deprovisioning is the #1 security risk
4. **Reconcile regularly**: Drift detection prevents security gaps
5. **Handle errors gracefully**: Retry, dead letter queue, alerting
6. **Map attributes consistently**: Use a central attribute mapping table
7. **Test provisioning flows**: Verify create, update, disable, delete in staging
8. **Monitor provisioning health**: Track success rates, durations, errors
9. **Audit provisioning actions**: Every provisioning event should be auditable
10. **Plan for edge cases**: Rehires, leaves of absence, contractor conversions

## See Also

- [SCIM 2.0 Implementation](./scim-2-0-implementation.md)
- Access Lifecycle Management
- Access Review Guide
- Deprovisioning Automation
- Audit Logging Guide
- HR Integration Guide