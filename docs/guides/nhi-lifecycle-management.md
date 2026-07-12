# Non-Human Identity (NHI) Lifecycle Management

NHI types, provisioning automation, credential rotation, orphan detection, decommissioning, inventory best practices, access review, and risk scoring.

## NHI Types

| Type | Identifier | Auth Method | Typical Lifetime |
|------|-----------|-------------|-----------------|
| Service account | `svc-uuid` | Client credentials | Permanent (managed) |
| API key | `ak-uuid` | Bearer token | 90 days (rotating) |
| AI agent | `agent-uuid` | DPoP-bound token | 90 days (auto-decommission) |
| IoT device | `device-uuid` | mTLS cert | Device lifetime |
| OAuth client | `client-id` | Client secret / PKCE | App lifetime |
| CI/CD pipeline | `pipeline-uuid` | OIDC token | Pipeline lifetime |

## Provisioning Automation

```bash
# Auto-provision NHI from Infrastructure-as-Code
POST /api/v1/admin/nhi/provision
{
  "type": "service_account",
  "name": "billing-processor",
  "owner": "team-finance",
  "scopes": ["users:read", "invoices:read"],
  "rotation_days": 90,
  "max_session_hours": 1
}
# → {nhi_id, client_id, client_secret}
```

### IaC Integration

```yaml
# Terraform module
resource "ggid_nhi" "billing" {
  type         = "service_account"
  name         = "billing-processor"
  scopes       = ["users:read", "invoices:read"]
  rotation_days = 90
}
# → Provisioned automatically with terraform apply
```

## Credential Rotation Policy

| NHI Type | Rotation Period | Method | Grace Period |
|----------|----------------|--------|-------------|
| Service account secret | 90 days | Generate new, revoke old after 7d | 7 days |
| API key | 90 days | Rolling (new + old valid) | 7 days |
| AI agent token | 15 min (access) | Automatic via refresh | — |
| mTLS cert | 90 days | cert-manager auto | 15 days |
| OAuth client secret | 180 days | Admin initiates | 30 days |

## Orphan Detection

```bash
# Find NHIs with no activity in 90 days
GET /api/v1/admin/nhi/orphans
# → {
#   "orphans": [
#     {"nhi_id": "svc-old", "type": "service_account", "last_used": "2024-10-01", "days_idle": 105},
#     {"nhi_id": "ak-unused", "type": "api_key", "last_used": "never", "days_idle": 200}
#   ],
#   "total": 2,
#   "recommendation": "decommission after owner verification"
# }
```

### Orphan Criteria

| Signal | Threshold |
|--------|----------|
| No API calls | 90 days |
| No token refresh | 30 days |
| Owner team dissolved | Immediate |
| No IaC reference | Flag for review |

## Decommissioning Workflow

```bash
POST /api/v1/admin/nhi/{nhi_id}/decommission
{
  "reason": "Service retired",
  "revoke_credentials": true,
  "revoke_sessions": true,
  "notify_owner": true,
  "archive_audit": true
}
```

### Steps

1. Revoke all active tokens + sessions
2. Revoke credentials (secret/key/cert)
3. Remove from access policies
4. Notify owner team
5. Archive audit trail (7 years)
6. Mark NHI as decommissioned (soft delete)
7. Purge after 90 days

## NHI Inventory Best Practices

```bash
# Full inventory export
GET /api/v1/admin/nhi/inventory?format=csv
# → Columns: nhi_id, type, name, owner, scopes, last_used, created, expires, status
```

### Inventory Requirements

| Requirement | Implementation |
|-------------|---------------|
| Every NHI has an owner | `owner` field required at provisioning |
| Every NHI has a purpose | `description` field required |
| Scopes documented | `scopes` array with justification |
| Last activity tracked | Updated on every API call |
| Expiry date set | Auto-expire or rotation schedule |

## Access Review for NHIs

| Review Type | Frequency | Reviewer | Action |
|-------------|-----------|---------|--------|
| Scope review | Quarterly | NHI owner | Remove unused scopes |
| Activity review | Monthly | Security team | Flag inactive NHIs |
| Owner verification | Bi-annual | Team lead | Confirm NHI still needed |
| Privilege check | Monthly | Automated | Alert on scope > registered |

## Risk Scoring Model

| Signal | Weight | Score Impact |
|--------|--------|-------------|
| Idle > 90 days | 20 | High risk (likely orphan) |
| Has admin scope | 25 | High risk (blast radius) |
| No owner | 30 | Critical (unmanageable) |
| Scope never used | 15 | Medium (overgranted) |
| Credential expired | 10 | Low (already safe) |
| Multiple active tokens | 15 | Medium (token sprawl) |

```bash
GET /api/v1/admin/nhi/risk-scores
# → [
#   {"nhi_id": "svc-old", "risk_score": 65, "level": "high", "factors": ["idle_90d", "no_owner"]},
#   {"nhi_id": "svc-billing", "risk_score": 25, "level": "medium", "factors": ["admin_scope"]}
# ]
```

## The 78% Gap Statistic

Industry research shows **78% of organizations** have NHIs they can't inventory. GGID addresses this by:

1. **Mandatory registration** — No NHI can access APIs without registration
2. **Automatic discovery** — Shadow NHI scanning (unregistered tokens/keys)
3. **IaC provisioning** — All NHIs created via Terraform, tracked in state
4. **Continuous inventory** — Real-time NHI dashboard with last-seen tracking
5. **Orphan alerts** — Automatic detection of unused NHIs

## Monitoring

| Metric | Alert |
|--------|-------|
| Orphaned NHIs | >5% of total → cleanup campaign |
| Expired credentials in use | Any → rotation failure |
| Shadow NHIs detected | Any → register or revoke |
| High-risk NHIs (score >60) | Any → review |
| NHI without owner | Any → assign or decommission |

## See Also

- [Agentic AI Governance](agentic-ai-governance.md)
- [Agent Identity Delegation](agent-identity-delegation.md)
- [Secret Sprawl Prevention](secret-sprawl-prevention.md)
- [Secrets Rotation Automation](secrets-rotation-automation.md)
- [Access Reviews](access-reviews.md)