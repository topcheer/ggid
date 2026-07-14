# Access Request Lifecycle

Request → justification → approval chain → provisioning → review → expiry → revoke, with SLA management, auto-approval rules, and audit trail.

## Lifecycle Stages

```
1. Request → User requests access to a resource/role
2. Justification → User provides business reason
3. Approval → Manager/owner approves or denies
4. Provisioning → Access granted (roles assigned, scopes added)
5. Active → User uses the access
6. Review → Periodic review by owner
7. Expiry → Time-boxed access expires
8. Revocation → Access removed (expiry, denial, or revocation)
```

## Submit Request

```bash
POST /api/v1/policy/access-requests
{
  "requester_id": "user-uuid",
  "resource_type": "role",
  "resource_id": "role-prod-deploy",
  "justification": "Need to deploy hotfix for INC-2025-0142",
  "duration_hours": 4,
  "urgency": "high"
}
# → 201 {request_id: "ar-uuid", status: "pending"}
```

## Approval Chain

### Single Approver

```bash
# Manager approves
POST /api/v1/policy/access-requests/ar-uuid/approve
{
  "approver_id": "manager-uuid",
  "decision": "approved",
  "comment": "Approved for hotfix deployment"
}
# → {status: "approved", provisioned: true}
```

### Multi-Step Chain

```yaml
approval_chain:
  - step: 1
    approver: "direct_manager"
    sla_hours: 2
    required: true

  - step: 2
    approver: "resource_owner"
    sla_hours: 4
    required: true

  - step: 3
    approver: "security_team"
    sla_hours: 8
    required: "if resource.classification == 'restricted'"
```

### Parallel Approval

```yaml
parallel_approval:
  approvers: ["manager", "security_officer"]
  require_all: true  # Both must approve
  sla_hours: 4
```

## Auto-Approval Rules

```yaml
auto_approval:
  - name: "dev-access-for-engineers"
    condition: |
      requester.department == "Engineering" &&
      resource.type == "role" &&
      resource.environment == "development"
    action: "auto_approve"
    duration_hours: 8
    notify: ["manager"]

  - name: "read-access-same-dept"
    condition: |
      requester.department == resource.department &&
      resource.permission == "read"
    action: "auto_approve"
    duration_hours: 24
```

### Auto-Approval Limits

| Constraint | Value |
|-----------|-------|
| Max duration (auto-approved) | 24 hours |
| Max scope (auto-approved) | Read-only |
| Never auto-approve | Admin, restricted, production-write |
| Rate limit | 10/day per user |

## Provisioning

```go
func provisionAccess(request *AccessRequest) error {
    // 1. Assign role/scopes
    roleSvc.AssignRole(request.RequesterID, request.ResourceID)

    // 2. Set expiry
    expiry := time.Now().Add(time.Duration(request.DurationHours) * time.Hour)
    store.SetAccessExpiry(request.RequesterID, request.ResourceID, expiry)

    // 3. Notify user
    notify.Send(request.RequesterID, "access_granted", map[string]interface{}{
        "resource": request.ResourceID,
        "expires":  expiry,
    })

    // 4. Audit
    audit.Log("access.provisioned", map[string]interface{}{
        "user_id":    request.RequesterID,
        "resource":   request.ResourceID,
        "approved_by": request.ApproverID,
        "expires_at": expiry,
        "request_id": request.ID,
    })

    return nil
}
```

## Review

### Scheduled Reviews

```bash
# Quarterly access review
POST /api/v1/admin/access-reviews
{
  "scope": "role:prod-deploy",
  "reviewer": "resource-owner-uuid",
  "period": "Q1-2025"
}
# → Generates review with all active grants for this role
```

### Review Actions

```bash
# Reviewer decides on each grant
POST /api/v1/admin/access-reviews/{review_id}/items/{item_id}
{
  "decision": "revoke",  // or "retain"
  "reason": "No longer needed for current project"
}
```

## Expiry

```go
// Cron job runs every 5 minutes
func checkExpiringAccess() {
    expiring := store.GetExpiringBefore(time.Now())

    for _, grant := range expiring {
        // Revoke access
        roleSvc.RevokeRole(grant.UserID, grant.ResourceID)

        // Notify user
        notify.Send(grant.UserID, "access_expired", map[string]interface{}{
            "resource": grant.ResourceID,
            "expired_at": grant.Expiry,
        })

        // Audit
        audit.Log("access.expired", grant)
    }

    // Notify before expiry (24h warning)
    upcoming := store.GetExpiringBefore(time.Now().Add(24 * time.Hour))
    for _, grant := range upcoming {
        if !grant.NotifiedExpiry {
            notify.Send(grant.UserID, "access_expiring_soon", grant)
            grant.NotifiedExpiry = true
            store.Update(grant)
        }
    }
}
```

## Revocation

```bash
# Manual revocation (any time)
DELETE /api/v1/policy/access-requests/ar-uuid
# → Access revoked immediately, tokens invalidated

# User-initiated (give up voluntarily)
POST /api/v1/policy/access-requests/ar-uuid/relinquish
```

## SLA Management

| Request Urgency | Target SLA | Escalation |
|----------------|-----------|------------|
| Critical (P0) | 1 hour | Escalate after 30 min |
| High (P1) | 4 hours | Escalate after 2 hours |
| Normal (P2) | 24 hours | Escalate after 12 hours |
| Low (P3) | 72 hours | No escalation |

### Escalation

```go
func checkSLABreaches() {
    pending := store.GetPendingRequests()
    for _, req := range pending {
        elapsed := time.Since(req.CreatedAt)
        sla := getSLA(req.Urgency)

        if elapsed > sla/2 && !req.Escalated {
            // First escalation: notify backup approver
            notify.Send(req.BackupApprover, "approval_escalation", req)
            req.Escalated = true
        }

        if elapsed > sla {
            // SLA breach: auto-deny or auto-approve based on policy
            if autoApproveOnBreach {
                approve(req)
            } else {
                deny(req, "SLA breach - request expired")
            }
        }
    }
}
```

## Audit Trail

```json
{
  "request_id": "ar-uuid",
  "events": [
    {"event": "access.requested", "user": "...", "timestamp": "..."},
    {"event": "access.justified", "reason": "...", "timestamp": "..."},
    {"event": "access.approved", "approver": "...", "step": 1, "timestamp": "..."},
    {"event": "access.approved", "approver": "...", "step": 2, "timestamp": "..."},
    {"event": "access.provisioned", "role": "...", "expires": "...", "timestamp": "..."},
    {"event": "access.expired", "timestamp": "..."}
  ]
}
```

## Monitoring

| Metric | Target | Alert |
|--------|--------|-------|
| Request approval time (P50) | <2h | — |
| Request approval time (P99) | <SLA | Breach → escalate |
| Auto-approval rate | 30-50% | >70% → rules too loose |
| Revocation rate after review | Track | Trend up → access creep |
| Expired but still active | 0 | Any → cron failure |

## See Also

- [RBAC Design Patterns](rbac-design-patterns.md)
- [Policy Engine Internals](policy-engine-internals.md)
- [Delegated Administration](delegated-administration.md)
- [User Provisioning Pipeline](user-provisioning-pipeline.md)
- [Access Reviews](access-reviews.md)
