# Delegation Management Guide

## Overview

GGID's delegation system allows a user (delegator) to grant scoped access to another user (delegatee) for a limited time. Delegations are tenant-scoped, revocable, and auditable — enabling secure temporary access without sharing credentials.

## Key Concepts

| Term | Description |
|------|-------------|
| **Delegator** | The user granting access to their permissions |
| **Delegatee** | The user receiving the delegated access |
| **Scopes** | Permission boundaries limiting what the delegatee can do |
| **Resource ID** | Optional specific resource the delegation applies to |
| **Expiry** | Time-limited validity window (mandatory) |

## API Endpoints

### List Delegations

```http
GET /api/v1/auth/delegations
X-Tenant-ID: <tenant-uuid>
X-User-ID: <user-uuid>
```

Returns all delegations where the current user is either delegator or delegatee.

**Response:**
```json
{
  "delegations": [
    {
      "id": "del_abc123",
      "tenant_id": "...",
      "delegator_id": "...",
      "delegatee_id": "...",
      "scopes": ["read:profile", "read:sessions"],
      "resource_id": "",
      "expires_at": "2026-07-20T12:00:00Z",
      "created_at": "2026-07-18T10:00:00Z"
    }
  ],
  "count": 1
}
```

### Create Delegation

```http
POST /api/v1/auth/delegations
X-Tenant-ID: <tenant-uuid>
X-User-ID: <user-uuid>
Content-Type: application/json

{
  "delegatee_id": "uuid-of-delegatee",
  "scopes": ["read:profile", "read:sessions"],
  "resource_id": "optional-resource-id",
  "expires_in_hours": 48
}
```

Alternative: use `expires_at` with ISO 8601 timestamp instead of `expires_in_hours`.

**Response:** `201 Created` with the full delegation object.

### Revoke Delegation

```http
DELETE /api/v1/auth/delegations/:id
X-Tenant-ID: <tenant-uuid>
X-User-ID: <user-uuid>
```

Immediately revokes the delegation. Sets `revoked_at` timestamp. Only the delegator can revoke.

**Response:** `200 OK`
```json
{
  "status": "revoked",
  "id": "del_abc123"
}
```

### Check Delegation Validity

```http
GET /api/v1/auth/delegations/check?delegator_id=<uuid>&delegatee_id=<uuid>&scope=read:profile
X-Tenant-ID: <tenant-uuid>
```

Verifies whether a valid (non-expired, non-revoked) delegation exists between two users for a specific scope.

**Response:**
```json
{
  "valid": true,
  "delegation": { ... }
}
```

## Data Model

### `user_delegations` Table

| Column | Type | Description |
|--------|------|-------------|
| `id` | TEXT (PK) | Unique delegation ID |
| `tenant_id` | UUID | Tenant scope |
| `delegator_id` | UUID | User granting access |
| `delegatee_id` | UUID | User receiving access |
| `scopes` | TEXT[] | Array of permitted scopes |
| `resource_id` | TEXT | Optional resource binding |
| `expires_at` | TIMESTAMPTZ | Mandatory expiry timestamp |
| `revoked_at` | TIMESTAMPTZ | NULL until revoked |
| `created_at` | TIMESTAMPTZ | Creation timestamp |

## Scope Design

Scopes follow the `<action>:<resource>` pattern:

| Scope | Description |
|-------|-------------|
| `read:profile` | View user profile |
| `read:sessions` | View active sessions |
| `write:profile` | Edit user profile |
| `admin:users` | Manage users (use with caution) |
| `read:audit` | View audit logs |

### Best Practices for Scopes

- **Principle of least privilege**: Grant only the minimum scopes needed
- **Resource-specific**: Use `resource_id` to limit delegation to a specific resource
- **Short TTL**: Prefer `expires_in_hours` with small values (1-48 hours)
- **Audit trail**: All delegation operations are recorded in the audit log

## Validation Rules

The `ValidateDelegation` function enforces:

1. **Scopes required** — at least one scope must be specified
2. **Expiry required** — `expires_at` must be set and in the future
3. **No self-delegation** — delegator and delegatee must differ
4. **Valid UUIDs** — both user IDs must be valid UUIDs
5. **Tenant match** — both users must belong to the same tenant

## Lifecycle

```
Create → Active → [Expiry Reached → Expired]
                 → [Revoked → Revoked]
```

1. **Create**: Delegator creates delegation with scopes + expiry
2. **Active**: Delegatee can exercise delegated permissions
3. **Expired**: Past `expires_at` — automatically invalid
4. **Revoked**: Explicitly cancelled by delegator via DELETE

## Security Considerations

- Delegations do NOT share credentials — the delegatee authenticates as themselves
- The delegation is checked at authorization time, not just at creation
- Revocation is immediate — no grace period
- All delegation operations (create, check, revoke) are audit-logged with actor, target, and scopes
- Use CAE (Continuous Access Evaluation) to re-validate delegations on active sessions

## Integration Points

- **Audit System**: Every delegation operation emits an audit event
- **Policy Engine**: Delegation scopes are evaluated during policy checks
- **OAuth Token Exchange (RFC 8693)**: Delegations can be exchanged for scoped tokens
- **Admin Console**: Manage delegations via the delegation management page
