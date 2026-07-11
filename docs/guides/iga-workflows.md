# IGA Access Request Workflows

> Identity Governance and Administration: create, approve, and deny access requests with automated provisioning.

---

## Overview

GGID's IGA module implements access request workflows for governing privilege escalation:

```
User → Submit Request → Admin Reviews → Approve/Deny → Auto-Provision Role
```

---

## API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `POST` | `/api/v1/access-requests` | Bearer JWT | Create access request |
| `GET` | `/api/v1/access-requests` | Bearer JWT | List requests |
| `POST` | `/api/v1/access-requests/{id}/approve` | `admin` role | Approve request |
| `POST` | `/api/v1/access-requests/{id}/deny` | `admin` role | Deny request |

---

## 1. Create Access Request

```bash
curl -X POST http://localhost:8080/api/v1/access-requests \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "usr_abc123",
    "role_id": "550e8400-e29b-41d4-a716-446655440000",
    "reason": "Need admin access for production deployment",
    "duration": "24h"
  }'
```

**Response (201):**
```json
{
  "id": "req_001",
  "user_id": "usr_abc123",
  "role_id": "550e8400-...",
  "status": "pending",
  "reason": "Need admin access for production deployment",
  "duration": "24h",
  "created_at": "2025-07-11T12:00:00Z"
}
```

---

## 2. List Access Requests

```bash
# All pending requests
curl -s "http://localhost:8080/api/v1/access-requests?status=pending" \
  -H "Authorization: Bearer $ADMIN_JWT" \
  -H "X-Tenant-ID: $TENANT" | jq .
```

```bash
# Requests for a specific user
curl -s "http://localhost:8080/api/v1/access-requests?user_id=usr_abc123" \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT" | jq .
```

**Response (200):**
```json
{
  "requests": [
    {
      "id": "req_001",
      "user_id": "usr_abc123",
      "role_id": "550e8400-...",
      "status": "pending",
      "reason": "Need admin access for production deployment",
      "duration": "24h",
      "created_at": "2025-07-11T12:00:00Z"
    }
  ],
  "total": 1
}
```

---

## 3. Approve a Request

```bash
curl -X POST http://localhost:8080/api/v1/access-requests/req_001/approve \
  -H "Authorization: Bearer $ADMIN_JWT" \
  -H "X-Tenant-ID: $TENANT" \
  -H "Content-Type: application/json" \
  -d '{"comment":"Approved for deployment window"}'
```

**Response (200):**
```json
{
  "id": "req_001",
  "status": "approved",
  "approved_by": "usr_admin",
  "approved_at": "2025-07-11T12:05:00Z",
  "comment": "Approved for deployment window",
  "provisioned": true
}
```

When approved, the role is **automatically assigned** to the user via `/api/v1/users/{id}/roles`.

---

## 4. Deny a Request

```bash
curl -X POST http://localhost:8080/api/v1/access-requests/req_001/deny \
  -H "Authorization: Bearer $ADMIN_JWT" \
  -H "X-Tenant-ID: $TENANT" \
  -H "Content-Type: application/json" \
  -d '{"comment":"Use service account instead"}'
```

**Response (200):**
```json
{
  "id": "req_001",
  "status": "denied",
  "denied_by": "usr_admin",
  "denied_at": "2025-07-11T12:05:00Z",
  "comment": "Use service account instead"
}
```

---

## Request Lifecycle

```
pending → approved (role auto-provisioned)
       → denied    (no change)
       → expired   (not approved within 72h)
       → cancelled (user withdraws)
```

| Status | Description |
|--------|-------------|
| `pending` | Awaiting admin review |
| `approved` | Role automatically assigned to user |
| `denied` | Request rejected |
| `expired` | Not reviewed within 72 hours |
| `cancelled` | User withdrew the request |

---

## Audit

All access request actions are audited:

| Action | Event Type |
|--------|-----------|
| Create request | `iga.request_created` |
| Approve | `iga.request_approved` |
| Deny | `iga.request_denied` |
| Expire | `iga.request_expired` |

Query via: `GET /api/v1/audit/events?action=iga.*`

---

## Console UI

The Admin Console has a dedicated Access Requests page at `/access-requests` showing:
- Pending requests with one-click approve/deny
- Request history with filtering
- User context (current roles, last login)

---

*See: [RBAC Guide](role-based-access.md) | [Admin API](../api/admin-api.md) | [REST API Reference](../api/rest-api.md)*

*Last updated: 2025-07-11*
