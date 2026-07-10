# SCIM 2.0 User Provisioning Guide

Automated user provisioning and deprovisioning via SCIM 2.0 (RFC 7643/7644).

---

## Overview

GGID implements SCIM 2.0 as a Service Provider (SP). External Identity Providers (IdPs) like Okta, Azure AD, and Google Workspace can automatically create, update, and deactivate users in GGID.

### Base URL

```
https://iam.example.com/scim/v2/
```

### Authentication

SCIM endpoints require a Bearer token (API key or JWT):

```
Authorization: Bearer <token>
```

---

## Endpoints

### Users

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/scim/v2/Users` | Create user |
| `GET` | `/scim/v2/Users` | List/filter users |
| `GET` | `/scim/v2/Users/{id}` | Get user by ID |
| `PUT` | `/scim/v2/Users/{id}` | Replace user |
| `PATCH` | `/scim/v2/Users/{id}` | Partial update |
| `DELETE` | `/scim/v2/Users/{id}` | Deactivate user |

### Groups

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/scim/v2/Groups` | Create group |
| `GET` | `/scim/v2/Groups` | List groups |
| `GET` | `/scim/v2/Groups/{id}` | Get group |
| `PUT` | `/scim/v2/Groups` | Replace group |
| `PATCH` | `/scim/v2/Groups/{id}` | Update membership |
| `DELETE` | `/scim/v2/Groups/{id}` | Delete group |

### Service Provider Config

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/scim/v2/ServiceProviderConfig` | Capabilities |
| `GET` | `/scim/v2/ResourceTypes` | Supported resource types |

---

## User Schema

### Create User

```bash
POST /scim/v2/Users
Authorization: Bearer <token>

{
  "schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"],
  "userName": "john.doe@example.com",
  "active": true,
  "name": {
    "givenName": "John",
    "familyName": "Doe"
  },
  "emails": [
    {
      "value": "john.doe@example.com",
      "type": "work",
      "primary": true
    }
  ],
  "title": "Software Engineer",
  "displayName": "John Doe"
}
```

### Response (201 Created)

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"],
  "userName": "john.doe@example.com",
  "active": true,
  "name": {
    "givenName": "John",
    "familyName": "Doe"
  },
  "emails": [{"value": "john.doe@example.com", "primary": true}],
  "meta": {
    "resourceType": "User",
    "created": "2024-07-10T12:00:00Z",
    "lastModified": "2024-07-10T12:00:00Z",
    "location": "https://iam.example.com/scim/v2/Users/550e8400-..."
  }
}
```

### Filtering

```bash
# Filter by email
GET /scim/v2/Users?filter=emails.value eq "john.doe@example.com"

# Filter by active status
GET /scim/v2/Users?filter=active eq true

# Pagination
GET /scim/v2/Users?startIndex=1&count=100

# Combined
GET /scim/v2/Users?filter=active eq true&startIndex=1&count=50
```

### Patch User (Update / Activate / Deactivate)

```bash
PATCH /scim/v2/Users/550e8400-...
{
  "schemas": ["urn:ietf:params:scim:api:messages:2.0:PatchOp"],
  "Operations": [
    {
      "op": "replace",
      "path": "active",
      "value": false
    }
  ]
}
```

This deactivates the user (SCIM soft delete).

---

## Integration with Okta

### 1. Create API Key in GGID

```bash
# Create an admin API key for SCIM
POST /api/v1/auth/api-keys
{
  "name": "Okta SCIM Provisioning",
  "scopes": ["scim:read", "scim:write"]
}
```

### 2. Configure Okta SCIM App

1. In Okta Admin → **Applications** → **Browse App Catalog**
2. Search for "SCIM 2.0" or create a custom SCIM app
3. Configure:
   - **Base URL:** `https://iam.example.com/scim/v2`
   - **Authentication:** Bearer Token
   - **Token:** (the API key from step 1)
4. Test connection → Okta validates by calling `GET /scim/v2/Users`

### 3. Configure Attribute Mapping

| Okta Attribute | SCIM Attribute | GGID Field |
|---------------|----------------|------------|
| Email | `userName` / `emails[0].value` | username / email |
| First Name | `name.givenName` | display_name (first part) |
| Last Name | `name.familyName` | display_name (last part) |
| Department | `title` | metadata.department |
| Status (Active/Suspended) | `active` | status (active/locked) |

### 4. Enable Provisioning

- **Assignment-based:** Users assigned to the Okta app are provisioned
- **Group-based:** Users in specific Okta groups are provisioned
- **Deprovisioning:** When removed from the app, `active` is set to `false`

---

## Integration with Azure AD

### 1. Enterprise Application

1. Azure Portal → **Enterprise Applications** → **New Application**
2. Create "Non-gallery" application
3. Configure **Provisioning** tab:
   - **Mode:** Automatic
   - **Admin Credentials:**
     - **Tenant URL:** `https://iam.example.com/scim/v2`
     - **Secret Token:** (API key)
   - **Test Connection** → verify

### 2. Attribute Mappings

| Azure AD Attribute | SCIM Attribute |
|-------------------|----------------|
| `userPrincipalName` | `userName` |
| `mail` | `emails[type eq "work"].value` |
| `givenName` | `name.givenName` |
| `surname` | `name.familyName` |
| `jobTitle` | `title` |
| `isActive` | `active` |

### 3. Provisioning Cycle

- **Initial sync:** All assigned users provisioned (bulk)
- **Delta sync:** Every 40 minutes, changes synced
- **Deprovisioning:** On unassignment, `active → false`

---

## Automatic Provisioning Flow

```
Okta/Azure AD                    GGID SCIM                   GGID Identity
     │                              │                            │
     │ POST /scim/v2/Users          │                            │
     ├─────────────────────────────►│                            │
     │                              │ Parse SCIM schema          │
     │                              │ Map to GGID user           │
     │                              ├───────────────────────────►│ Create user
     │                              │                            │ (status: active)
     │                              │                            │ Publish audit
     │  201 Created                 │◄───────────────────────────┤
     │◄─────────────────────────────┤                            │
     │                              │                            │
     │ (user removed from IdP)      │                            │
     │ PATCH /scim/v2/Users/{id}    │                            │
     │  {active: false}             │                            │
     ├─────────────────────────────►│                            │
     │                              ├───────────────────────────►│ Lock user
     │  200 OK                      │◄───────────────────────────┤
     │◄─────────────────────────────┤                            │
```

---

## Error Handling

SCIM errors use a standard format:

```json
{
  "schemas": ["urn:ietf:params:scim:api:messages:2.0:Error"],
  "status": "409",
  "detail": "User with this username already exists",
  "scimType": "uniqueness"
}
```

| Status | scimType | Cause |
|--------|----------|-------|
| 400 | `invalidSyntax` | Malformed JSON |
| 400 | `invalidFilter` | Bad filter expression |
| 401 | — | Invalid/expired token |
| 404 | — | Resource not found |
| 409 | `uniqueness` | Duplicate resource |
| 500 | — | Internal error |

---

## Best Practices

1. **Use pagination** — Always specify `count` to avoid large responses
2. **Idempotency** — Creating the same user twice returns 409, not a duplicate
3. **Use `externalId`** — Store the IdP's user ID for cross-referencing
4. **Monitor sync** — Check audit events for provisioning/deprovisioning activity
5. **Test deprovisioning** — Ensure users are locked (not deleted) on removal
6. **Rate limit awareness** — SCIM endpoints share the 100 req/min API limit
