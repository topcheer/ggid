# Access Reviews & Certification Campaigns Guide

This guide covers running access certification campaigns, detecting orphaned accounts, and managing identity governance in GGID.

## Overview

Access reviews (also called access recertification or certification campaigns) are a core IGA (Identity Governance & Administration) process. They ensure that users retain only the access they need, reducing risk from privilege creep, orphaned accounts, and toxic combinations.

## Access Certification Campaigns

### What Is a Certification Campaign?

A certification campaign is a periodic review where managers or application owners verify that each user's access is still appropriate:

```
Campaign Created → Reviewers Notified → Review Access Items → Certify/Revoke → Auto-Provision Changes → Report
```

### Campaign Types

| Type | Reviewer | Scope | Frequency |
|------|----------|-------|-----------|
| **Manager review** | Direct manager | Reports' access | Quarterly |
| **Application owner** | App admin | Users with app access | Semi-annually |
| **Role review** | Role owner | Users assigned to role | Annually |
| **Privileged access** | Security team | Admin/elevated roles | Monthly |
| **New joiner audit** | HR/IT | Access granted in last 90 days | Monthly |
| **Leaver audit** | System | Accounts active after termination | Continuous |

### Creating a Campaign

GGID's access request system (`services/identity/internal/domain/access_request.go`) provides the foundation. A certification campaign builds on this:

```bash
# Create a campaign via API
curl -X POST https://api.ggid.example.com/api/v1/access-requests/campaign \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "name": "Q1 2025 Manager Review",
    "type": "manager_review",
    "scope": {
      "roles": ["developer", "admin", "auditor"],
      "orgs": ["engineering", "sales"]
    },
    "reviewers": {
      "mode": "manager"
    },
    "deadline": "2025-03-31T23:59:59Z",
    "actions_on_no_response": "revoke"
  }'
```

### Campaign Lifecycle

| Status | Description |
|--------|-------------|
| `pending` | Campaign created, reviewers not yet notified |
| `active` | Reviewers notified, awaiting decisions |
| `reviewing` | At least one reviewer has started |
| `completed` | All items reviewed or deadline passed |
| `remediated` | Revocation/provisioning changes applied |

### Reviewing Access Items

Each reviewer sees a list of access items for their scope:

```
┌─────────────────────────────────────────────────────┐
│  Access Review: Q1 2025                              │
├─────────────────────────────────────────────────────┤
│  Alice Chen (alice@example.com)                      │
│  ┌───────────────────────────────────────────────┐  │
│  │ Role: developer                               │  │
│  │ Assigned: 2024-06-15                          │  │
│  │ Last used: 2025-01-10                         │  │
│  │ [✓ Certify]  [✗ Revoke]  [⏳ Delegate]       │  │
│  └───────────────────────────────────────────────┘  │
│  ┌───────────────────────────────────────────────┐  │
│  │ Role: admin (ELEVATED)                        │  │
│  │ Assigned: 2024-01-01                          │  │
│  │ Last used: 2024-08-22 (INACTIVE 5 months)     │  │
│  │ [✓ Certify]  [✗ Revoke]  [⏳ Delegate]       │  │
│  └───────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────┘
```

## Orphaned Account Detection

### What Is an Orphaned Account?

An orphaned account is a user account that remains active but has no legitimate owner:

| Type | Detection Method |
|------|-----------------|
| **Unassigned user** | User exists but not in any organization or group |
| **Inactive user** | No login for N days (e.g., 90) |
| **Leaked account** | User left the org but account still active |
| **Dormant role** | Role assigned but never used (no permission checks) |
| **Stale session** | Session active but user terminated |

### Running Orphan Detection

```bash
# Find users with no login in 90 days
curl -X GET "https://api.ggid.example.com/api/v1/users?last_login_before=2024-10-24T00:00:00Z&status=active" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"

# Find users not in any organization
curl -X GET "https://api.ggid.example.com/api/v1/users?org_id=none" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"

# Find users with elevated roles and no recent activity
curl -X GET "https://api.ggid.example.com/api/v1/users?role=admin&last_login_before=2024-12-24T00:00:00Z" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"
```

### Automated Orphan Detection Script

```bash
#!/bin/bash
# orphan-detect.sh — Run weekly via cron

THRESHOLD_DAYS=90
CUTOFF=$(date -d "-${THRESHOLD_DAYS} days" -u +%Y-%m-%dT%H:%M:%SZ)

# 1. Inactive users
INACTIVE=$(curl -s "https://api.ggid.example.com/api/v1/users?last_login_before=${CUTOFF}&status=active&pageSize=1000" \
  -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT")

# 2. Orphaned (no org assignment)
ORPHANS=$(curl -s "https://api.ggid.example.com/api/v1/users?org_id=none" \
  -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT")

# 3. Generate report
echo "Orphan Detection Report — $(date)" > /tmp/orphan-report.txt
echo "Inactive (> ${THRESHOLD_DAYS}d): $(echo $INACTIVE | jq '.total')" >> /tmp/orphan-report.txt
echo "No org assignment: $(echo $ORPHANS | jq '.total')" >> /tmp/orphan-report.txt

# 4. Alert security team
mail -s "Orphan Account Report" security@company.com < /tmp/orphan-report.txt
```

## Privilege Creep Detection

### Toxic Combinations

| Combination | Risk |
|-------------|------|
| `users:write` + `audit:read` | Can create users and cover tracks |
| `roles:write` + `policies:write` | Can grant themselves any permission |
| `admin` + `apikeys:write` | Can create persistent backdoor keys |
| `settings:write` + `certificates:write` | Can replace signing keys |

### Segregation of Duties (SoD)

Define SoD policies that flag conflicting role assignments:

```json
{
  "sod_policies": [
    {
      "name": "No self-grant admin",
      "rule": "users:write + roles:write",
      "action": "flag_for_review"
    },
    {
      "name": "No audit tampering",
      "rule": "admin + audit:delete",
      "action": "deny"
    }
  ]
}
```

## Access Review Best Practices

### Frequency

| Access Type | Review Frequency | Rationale |
|-------------|-----------------|-----------|
| Privileged (admin, root) | Monthly | High risk |
| Standard roles | Quarterly | Medium risk |
| Application access | Semi-annually | Lower risk |
| Service accounts | Annually | Rarely changes |
| Contractor/vendor | Monthly | High turnover |

### Reviewer Assignment

| Method | Description | Pros | Cons |
|--------|-------------|------|------|
| Direct manager | User's manager reviews | Knows user's needs | May rubber-stamp |
| Application owner | App admin reviews | Knows app requirements | Doesn't know user |
| Peer review | Fellow team member | Understands day-to-day | Awkward dynamic |
| Self-attestation | User reviews own access | Efficient | No independent check |
| Automated | Rule-based decision | Scalable | Misses edge cases |

## GGID Implementation

### Current Capabilities

| Capability | Status | Location |
|------------|--------|----------|
| Access request workflow | Done | `services/identity/internal/domain/access_request.go` |
| Role assignment tracking | Done | Policy service |
| User listing & filtering | Done | Identity service |
| Audit trail for role changes | Done | Audit service |
| Organization membership | Done | Org service |
| Session activity tracking | Done | Auth service (Redis sessions) |

### Gaps for Full Access Reviews

| Gap | Priority | Effort |
|-----|----------|--------|
| Certification campaign engine | P1 | Large |
| Reviewer notification system | P1 | Medium |
| Automated revocation on campaign completion | P1 | Medium |
| Orphan detection dashboard | P1 | Medium |
| SoD policy enforcement | P2 | Medium |
| Privilege creep analytics | P2 | Large |
| Access review reporting (compliance) | P2 | Medium |

## Compliance Alignment

| Standard | Requirement | GGID Support |
|----------|-------------|--------------|
| **SOC 2 CC6** | Review logical access | Campaign system (roadmap) |
| **ISO 27001 A.9** | Access rights review | Campaign system (roadmap) |
| **NIST 800-53 AC-2** | Account management | User lifecycle (done) |
| **GDPR Art. 32** | Access controls | RBAC + ABAC (done) |
| **HIPAA 164.308** | Access management | RBAC + audit (done) |
| **SOX** | Access reviews for financial systems | Campaign system (roadmap) |

## See Also

- [Access Requests (IGA)](iga-workflows.md)
- [SCIM Provisioning](scim-provisioning.md)
- Audit Guide
- [Security Center](security-audit-checklist.md)
