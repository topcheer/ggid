# Consent Management — User Guide

> Feature: F-50 Consent Management
> Location: **Consent** (`/consent`)

## What It Does

The Consent Management page provides a centralized interface for managing user consent records as required by GDPR (Articles 6, 7, 17), CCPA, and other privacy regulations. Administrators can view, grant, withdraw, and audit consent records, manage privacy policies, and process Data Subject Requests (DSR).

## How to Access

1. Log in to the GGID Admin Console.
2. Click **Consent** in the sidebar.

Alternatively, go to `/consent` directly.

## Tabs and Sections

### 1. Overview

High-level KPIs at a glance:

- **Total Consents**: All consent records across the tenant.
- **Active**: Currently valid consent grants.
- **Expired**: Consents past their expiration date.
- **Withdrawn**: Consents explicitly withdrawn by users or admins.
- **By Purpose**: Breakdown of consents grouped by purpose (marketing, analytics, third-party sharing, etc.) with color-coded distribution.

**Workflow — Review consent health:**
1. Open the Overview tab.
2. Check the Active vs Expired ratio.
3. If many expired consents, consider sending re-consent campaigns.

### 2. Consent Registry

Detailed table of all consent records with filtering:

- **Filter by Status**: All, Active, Expired, Withdrawn.
- **Filter by User**: Search by user ID.
- **Columns**: User, Purpose, Scopes, Client (OAuth app), Status, Granted At, Expires At.

**Workflow — Grant consent on behalf of a user:**
1. Click **Grant Consent**.
2. Enter the User ID.
3. Select a Purpose (e.g., marketing, analytics).
4. Enter scopes (e.g., `read:profile, write:preferences`).
5. Optionally enter the OAuth Client ID.
6. Click **Grant**.

**Workflow — Withdraw consent:**
1. Find the consent record in the table.
2. Click the **Withdraw** button (ban icon).
3. Enter a withdrawal reason (optional).
4. Confirm. The consent status changes to "Withdrawn".

### 3. Privacy Policies

View and manage privacy policy versions:

- **Current Version**: The active policy version.
- **Effective Date**: When the current policy took effect.
- **Version History**: Previous policy versions with dates.
- **Re-consent Required**: Whether a policy change triggers re-consent.

### 4. DSR Requests

Process Data Subject Requests per GDPR Article 15-22:

- **Request Types**: Access, Deletion (Right to Erasure), Portability, Rectification.
- **SLA Tracking**: Days remaining to fulfill the request (GDPR requires 30 days).
- **Status**: Pending, In Progress, Completed, Overdue.

**Workflow — Process a data deletion request:**
1. Go to the DSR tab.
2. Select type "Deletion".
3. Enter the user ID.
4. Click **Submit Request**.
5. The system creates the request and tracks the SLA.
6. When complete, mark it as fulfilled.

## API Endpoints

| Endpoint | Method | Purpose |
|----------|--------|--------|
| `/api/v1/identity/consent/registry` | GET | List consent records (supports `user_id`, `status` query params) |
| `/api/v1/identity/consent/registry` | POST | Grant new consent |
| `/api/v1/identity/consent/registry` | DELETE | Withdraw consent (by record params) |

### curl Examples

```bash
# List all active consents
TOKEN="your-jwt-token"
curl -k -H 'Accept-Encoding: identity' \
  "https://ggid.iot2.win/api/v1/identity/consent/registry?status=active" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001"

# Grant consent
NEW_TOKEN="your-jwt-token"
curl -k -H 'Accept-Encoding: identity' \
  -X POST "https://ggid.iot2.win/api/v1/identity/consent/registry" \
  -H "Authorization: Bearer $NEW_TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{"user_id":"user-123","purpose":"marketing","scopes":["read:profile"]}'

# Withdraw consent
curl -k -H 'Accept-Encoding: identity' \
  -X DELETE "https://ggid.iot2.win/api/v1/identity/consent/registry?user_id=user-123&purpose=marketing" \
  -H "Authorization: Bearer $NEW_TOKEN" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001"
```

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|--------|
| Consent list empty | No consents recorded or identity service unreachable | Check `ggid-identity` pod; verify users have gone through consent flow |
| Grant fails | Invalid user_id or missing required fields | Verify user exists; ensure purpose and scopes are provided |
| DSR request stuck | Backend processing delay | Check identity pod logs: `kubectl logs -n ggid deploy/ggid-identity` |
| Withdraw button missing | Consent already withdrawn or expired | Only active consents can be withdrawn |

## Best Practices

- **Regular audits**: Review the Consent Registry monthly for expired or stale records.
- **Prompt DSR fulfillment**: GDPR requires response within 30 days — monitor SLA closely.
- **Policy versioning**: Update privacy policy version when making material changes.
- **Minimal scopes**: Only grant scopes that are strictly necessary for the purpose.
- **Document withdrawals**: Always provide a reason when withdrawing consent for audit trails.
