# Digital Identity Lifecycle

Joiner (provisioning), mover (role change), leaver (deprovisioning), automation hooks, and SCIM integration points.

## Joiner → Mover → Leaver

```
Joiner          Mover           Leaver
  │               │               │
  ▼               ▼               ▼
Provision    Role Change    Deprovision
Email        Attribute      Session Kill
Roles        Update         Data Retain
Groups       Access Review  Archive
```

## Joiner (Onboarding)

### Provisioning Flows

| Source | Mechanism | Trigger |
|--------|-----------|---------|
| HR system | SCIM create | New hire in HRIS |
| SSO JIT | Auto-provision on login | First SSO login |
| Admin manual | Console create | Admin adds user |
| API programmatic | POST /users | Script/integration |
| Self-service | Registration page | User signs up |

### Onboarding Steps

```yaml
joiner:
  - step: create_identity
    action: POST /scim/v2/Users
    assign_default_roles: [user]
    assign_default_groups: [all_users]

  - step: send_activation
    action: email activation link
    ttl: 7_days

  - step: first_login
    action: force_password_set
    action: mfa_enrollment_required
    action: accept_terms

  - step: provision_apps
    action: webhook to connected apps
    action: SCIM sync to downstream
```

## Mover (Role/Attribute Change)

### Triggers

| Trigger | Action |
|---------|--------|
| Department change | Update attributes, reassign groups |
| Promotion | Add roles, expand access |
| Demotion | Remove roles, reduce access |
| Transfer | Revoke old dept access, add new dept |
| Project change | Add project-specific roles (JIT) |

### Automation

```bash
# HR system sends update via SCIM PATCH
PATCH /scim/v2/Users/uuid
{"Operations": [
  {"op": "replace", "path": "department", "value": "Product"},
  {"op": "replace", "path": "title", "value": "Senior PM"}
]}
```

```go
func onAttributeChange(user *User, changes map[string]interface{}) {
    // Re-evaluate dynamic roles
    roleService.ReassignDynamicRoles(user.ID)

    // Update group membership
    groupService.SyncGroupsByAttribute(user.ID, changes)

    // Trigger access review
    accessReviewService.ScheduleForUser(user.ID)

    // Notify connected apps
    webhookService.Notify("user.updated", user)

    // Audit
    audit.Log("user.moved", user, changes)
}
```

## Leaver (Offboarding)

### Deprovisioning Checklist

```yaml
leaver:
  immediate:
    - revoke_all_sessions
    - revoke_all_tokens
    - disable_mfa_factors
    - set_status: suspended

  within_24h:
    - remove_from_all_groups
    - revoke_oauth_grants
    - revoke_api_keys
    - disable_in_connected_apps (SCIM PATCH active=false)
    - send_webhook: user.deprovisioned

  retain:
    - audit_logs: 7_years
    - anonymize_after: 90_days
    - archive_to_cold_storage
```

### Session Invalidations

```bash
POST /api/v1/admin/users/{user_id}/deprovision
{
  "reason": "departure",
  "effective": "immediate",
  "revoke_sessions": true,
  "revoke_tokens": true,
  "notify_apps": true
}
```

## SCIM Integration Points

| Lifecycle Event | SCIM Operation |
|----------------|---------------|
| Join | POST /scim/v2/Users |
| Update attributes | PATCH /scim/v2/Users/{id} |
| Suspend | PATCH (active: false) |
| Reinstate | PATCH (active: true) |
| Leave | DELETE /scim/v2/Users/{id} |
| Group add | PATCH /scim/v2/Groups/{id} (add member) |
| Group remove | PATCH (remove member) |

## Monitoring

| Metric | Alert |
|--------|-------|
| Provisioning latency | >30s → backlog |
| Deprovisioning failures | Any → security risk |
| Dormant accounts (>90d) | Flag for cleanup |
| Mover access review completion | <90% → overdue |

## See Also

- [Identity Lifecycle Automation](identity-lifecycle-automation.md)
- [User Provisioning Pipeline](user-provisioning-pipeline.md)
- [SCIM 2.0 Implementation](scim-2-0-implementation.md)
- [Access Request Lifecycle](access-request-lifecycle.md)
