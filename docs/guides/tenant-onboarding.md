# Tenant Onboarding Guide

> Step-by-step: create a tenant, configure IdP, import users, set roles, apply branding.

---

## Step 1: Create Tenant

```bash
curl -X POST http://localhost:8080/api/v1/tenants \
  -H "Authorization: Bearer $ADMIN_JWT" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Acme Corp",
    "domain": "acme.com",
    "plan": "enterprise"
  }'
```

Save the returned tenant ID.

## Step 2: Configure Identity Provider

### Option A: SAML
```bash
curl -X POST http://localhost:8080/api/v1/tenants/$TENANT_ID/saml/config \
  -H "Authorization: Bearer $ADMIN_JWT" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "idp_metadata_url": "https://acme.okta.com/.../metadata",
    "entity_id": "https://acme.com/saml/sp",
    "acs_url": "https://ggid.example.com/saml/acs"
  }'
```

### Option B: OIDC
```bash
curl -X POST http://localhost:8080/api/v1/tenants/$TENANT_ID/oidc/config \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "issuer": "https://login.microsoftonline.com/.../v2.0",
    "client_id": "azure-id",
    "client_secret": "azure-secret",
    "redirect_uri": "https://ggid.example.com/oidc/callback"
  }'
```

## Step 3: Import Users

```bash
# Via SCIM bulk
curl -X POST http://localhost:8080/scim/v2/Bulk \
  -H "Authorization: Bearer $ADMIN_JWT" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "Operations": [
      {"method": "POST", "path": "/Users", "data": {"userName": "alice", "emails": [{"value":"alice@acme.com"}]}},
      {"method": "POST", "path": "/Users", "data": {"userName": "bob", "emails": [{"value":"bob@acme.com"}]}}
    ]
  }'
```

## Step 4: Create Roles & Assign

```bash
TENANT=$TENANT_ID

# Create roles
curl -X POST http://localhost:8080/api/v1/roles \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"name":"Admin","key":"admin"}'

curl -X POST http://localhost:8080/api/v1/roles \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"name":"Member","key":"member"}'

# Assign to users
curl -X POST http://localhost:8080/api/v1/users/$ALICE_ID/roles \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"role_id":"'$ADMIN_ROLE_ID'"}'
```

## Step 5: Apply Branding

```bash
curl -X PUT http://localhost:8080/api/v1/tenants/$TENANT_ID/branding \
  -H "X-Tenant-ID: $TENANT" \
  -d '{
    "logo_url": "https://acme.com/logo.png",
    "primary_color": "#FF5733",
    "login_bg_color": "#1a1a2e"
  }'
```

## Step 6: Verify

```bash
# Login as tenant user
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"username":"alice","password":"..."}'
```

---

*See: [Multi-Tenant Guide](multi-tenant-guide.md) | [Per-Tenant IdP](per-tenant-idp.md) | [RBAC Guide](role-based-access.md)*

*Last updated: 2025-07-11*
