# Tutorial: Multi-Tenant Setup

> Step-by-step guide to creating a new tenant, configuring SSO, inviting users, and setting up RBAC.

---

## Prerequisites

- GGID running via `docker compose up -d`
- Super-admin JWT (from default tenant login)
- `curl` and `jq` installed

```bash
# Login as super-admin on default tenant
export SUPER_ADMIN_TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{"username":"admin","password":"SecurePass123!"}' | jq -r .access_token)

echo "Token: ${SUPER_ADMIN_TOKEN:0:20}..."
```

---

## Step 1: Create a Tenant

```bash
curl -s -X POST http://localhost:8080/api/v1/tenants \
  -H "Authorization: Bearer $SUPER_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Acme Corporation",
    "plan": "enterprise",
    "max_users": 5000
  }' | jq .
```

Response:
```json
{
  "id": "55000000-0000-0000-0000-000000000002",
  "name": "Acme Corporation",
  "plan": "enterprise",
  "active": true
}
```

```bash
# Save tenant ID
export TENANT_ID="55000000-0000-0000-0000-000000000002"
```

---

## Step 2: Create Admin User for Tenant

```bash
# Register the first user in the new tenant
curl -s -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "username": "acme-admin",
    "email": "admin@acme.com",
    "password": "SecurePass123!",
    "first_name": "Jane",
    "last_name": "Doe"
  }' | jq .

# Save user ID
export ACME_USER_ID=$(curl -s http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer $SUPER_ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" | jq -r '.users[0].id')
```

---

## Step 3: Set Up RBAC Roles

### Create Custom Roles

```bash
# Create a manager role
curl -s -X POST http://localhost:8080/api/v1/roles \
  -H "Authorization: Bearer $SUPER_ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Manager",
    "key": "manager",
    "description": "Department manager",
    "permissions": ["read:users", "write:users", "read:orgs"]
  }' | jq .

# Create an auditor role
curl -s -X POST http://localhost:8080/api/v1/roles \
  -H "Authorization: Bearer $SUPER_ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Auditor",
    "key": "auditor",
    "description": "Read-only audit access",
    "permissions": ["read:audit", "read:users"]
  }' | jq .
```

### Assign Role to User

```bash
# Get role IDs
export MANAGER_ROLE_ID=$(curl -s http://localhost:8080/api/v1/roles \
  -H "Authorization: Bearer $SUPER_ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" | jq -r '.roles[] | select(.key=="manager") | .id')

# Assign manager role to user
curl -s -X POST "http://localhost:8080/api/v1/users/$ACME_USER_ID/roles" \
  -H "Authorization: Bearer $SUPER_ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Content-Type: application/json" \
  -d "{\"role_id\": \"$MANAGER_ROLE_ID\"}" | jq .
```

---

## Step 4: Create Organization Hierarchy

```bash
# Create root organization
curl -s -X POST http://localhost:8080/api/v1/orgs \
  -H "Authorization: Bearer $SUPER_ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Acme Corp",
    "description": "Root organization"
  }' | jq .

# Create sub-organization
export ROOT_ORG_ID=$(curl -s http://localhost:8080/api/v1/orgs \
  -H "Authorization: Bearer $SUPER_ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" | jq -r '.orgs[0].id')

curl -s -X POST http://localhost:8080/api/v1/orgs \
  -H "Authorization: Bearer $SUPER_ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"Engineering\",\"parent_org_id\":\"$ROOT_ORG_ID\"}" | jq .
```

---

## Step 5: Configure SSO (OIDC)

### Register External IdP

```bash
# Configure Azure AD as external IdP
curl -s -X POST http://localhost:8080/api/v1/oauth/clients \
  -H "Authorization: Bearer $SUPER_ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "azure-ad-acme",
    "client_secret": "your-azure-client-secret",
    "redirect_uris": ["https://login.microsoftonline.com/common/oauth2/nativeclient"],
    "grant_types": ["authorization_code"],
    "response_types": ["code"],
    "scope": "openid profile email"
  }' | jq .
```

### Configure SAML IdP

```bash
# Set SAML configuration
curl -s -X POST http://localhost:8080/api/v1/saml/config \
  -H "Authorization: Bearer $SUPER_ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "idp_metadata_url": "https://idp.acme.com/metadata",
    "sp_entity_id": "https://ggid.acme.com",
    "acs_url": "https://ggid.acme.com/api/v1/saml/acs"
  }' | jq .
```

---

## Step 6: Invite Users

### Direct Registration

```bash
# Register a new user
curl -s -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "username": "johnsmith",
    "email": "john@acme.com",
    "password": "Welcome2025!",
    "first_name": "John",
    "last_name": "Smith"
  }' | jq .
```

### Bulk Import via SCIM

```bash
# Bulk provision users via SCIM 2.0
curl -s -X POST http://localhost:8080/api/v1/scim/v2/Bulk \
  -H "Authorization: Bearer $SUPER_ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "Operations": [
      {
        "method": "POST",
        "path": "/Users",
        "data": {
          "userName": "alice@acme.com",
          "emails": [{"value":"alice@acme.com","type":"work"}],
          "name": {"givenName":"Alice","familyName":"Wonder"}
        }
      },
      {
        "method": "POST",
        "path": "/Users",
        "data": {
          "userName": "bob@acme.com",
          "emails": [{"value":"bob@acme.com","type":"work"}],
          "name": {"givenName":"Bob","familyName":"Builder"}
        }
      }
    ]
  }' | jq .
```

---

## Step 7: Verify Setup

```bash
# Login as the new tenant admin
curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{"username":"acme-admin","password":"SecurePass123!"}' | jq '{token: .access_token[0:20], scope: .user.scopes, roles: .user.roles}'

# Check audit trail for tenant
export ACME_TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{"username":"acme-admin","password":"SecurePass123!"}' | jq -r .access_token)

curl -s "http://localhost:8080/api/v1/audit/events?limit=5" \
  -H "Authorization: Bearer $ACME_TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" | jq '.events | length'
```

---

## Summary

You have successfully:
- [x] Created a new tenant
- [x] Created the first admin user
- [x] Defined RBAC roles with permissions
- [x] Created an organization hierarchy
- [x] Configured SSO (OIDC + SAML)
- [x] Invited users (direct + bulk SCIM)
- [x] Verified the audit trail

---

## Console View

In the admin console (`http://localhost:3000`):
1. **Dashboard** shows Acme Corp tenant metrics
2. **Users** lists all provisioned users
3. **Roles** shows Manager and Auditor roles
4. **Organizations** displays the hierarchy tree
5. **Audit** shows all events for this tenant

---

*Last updated: 2025-07-11*