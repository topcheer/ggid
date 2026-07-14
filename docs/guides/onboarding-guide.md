# GGID Onboarding Guide

This guide walks new administrators through first-run setup of a GGID tenant — creating an admin account, configuring identity providers, adding users, and testing login.

## Prerequisites

- GGID Gateway running and accessible
- Default tenant ID: `00000000-0000-0000-0000-000000000001`
- Console accessible at `https://console.ggid.example.com`

## Step 1: Create Admin Account

Register your first administrator:

```bash
curl -X POST https://api.ggid.example.com/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{
    "username": "admin",
    "email": "admin@company.com",
    "password": "SecureAdminPass1!"
  }'
```

**Important**: The first registered user should be granted the `admin` role immediately.

## Step 2: Login as Admin

```bash
curl -X POST https://api.ggid.example.com/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{
    "username": "admin",
    "password": "SecureAdminPass1!"
  }'
```

Save the returned `access_token` for subsequent steps.

## Step 3: Create Admin Role

```bash
curl -X POST https://api.ggid.example.com/api/v1/roles \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "key": "admin",
    "name": "Administrator",
    "permissions": [
      "users:read", "users:write", "roles:read", "roles:write",
      "audit:read", "policies:write", "settings:write"
    ]
  }'
```

## Step 4: Assign Admin Role to Yourself

```bash
curl -X POST https://api.ggid.example.com/api/v1/users/$ADMIN_USER_ID/roles \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{"role_id": "$ADMIN_ROLE_ID"}'
```

## Step 5: Configure Authentication

### Password Policy

```bash
curl -X PUT https://api.ggid.example.com/api/v1/settings/password-policy \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "min_length": 12,
    "require_uppercase": true,
    "require_lowercase": true,
    "require_digit": true,
    "require_special": true
  }'
```

### Enable MFA (Recommended)

Enroll TOTP for the admin account:

1. Navigate to **Profile → Security → MFA** in Console
2. Scan QR code with authenticator app
3. Enter 6-digit code to confirm

### Configure SSO (Optional)

For enterprise SSO via SAML:

1. Navigate to **Settings → SSO → Add Provider**
2. Upload IdP metadata XML
3. Download SP metadata from GGID
4. Upload SP metadata to your IdP (Okta/Azure AD)
5. Test SAML login

For social login (Google, GitHub, etc.):

1. Navigate to **Settings → SSO → Social**
2. Enter client ID and secret for each provider
3. Test social login flow

## Step 6: Add Users

### Individual User

```bash
curl -X POST https://api.ggid.example.com/api/v1/users \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "username": "alice",
    "email": "alice@company.com",
    "password": "SecurePass1!",
    "name": "Alice Chen"
  }'
```

### Bulk Import

```bash
curl -X POST https://api.ggid.example.com/api/v1/users/import \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "users": [
      {"username": "bob", "email": "bob@company.com", "password": "Pass1!"},
      {"username": "carol", "email": "carol@company.com", "password": "Pass1!"}
    ]
  }'
```

## Step 7: Create Roles

### Standard Roles

```bash
# Developer role
curl -X POST https://api.ggid.example.com/api/v1/roles \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "key": "developer",
    "name": "Developer",
    "permissions": ["users:read", "audit:read"]
  }'

# Auditor role (read-only)
curl -X POST https://api.ggid.example.com/api/v1/roles \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "key": "auditor",
    "name": "Auditor",
    "permissions": ["audit:read", "users:read"]
  }'
```

## Step 8: Create Organizations

```bash
curl -X POST https://api.ggid.example.com/api/v1/organizations \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{"name": "Engineering", "description": "Engineering team"}'
```

Add members:
```bash
curl -X POST https://api.ggid.example.com/api/v1/organizations/$ORG_ID/members \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{"user_id": "$USER_ID", "role": "developer"}'
```

## Step 9: Configure Branding

```bash
curl -X PUT https://api.ggid.example.com/api/v1/branding \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "logo_url": "https://cdn.company.com/logo.svg",
    "primary_color": "#0066CC",
    "company_name": "Acme Corp"
  }'
```

## Step 10: Test Login

```bash
# Test password login
curl -X POST https://api.ggid.example.com/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{"username": "alice", "password": "SecurePass1!"}'

# Verify token works
curl https://api.ggid.example.com/api/v1/users/me \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"

# Test unauthorized access (should return 401)
curl https://api.ggid.example.com/api/v1/users \
  -H "X-Tenant-ID: $TENANT_ID"
```

## Step 11: Configure Monitoring

Set up alert rules for security events:

```bash
curl -X POST https://api.ggid.example.com/api/v1/audit/alerts/rules \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "name": "Failed login burst",
    "condition": "event_type=user.login AND result=failure COUNT > 10 IN 5m",
    "action": "email",
    "recipients": ["security@company.com"],
    "enabled": true
  }'
```

## Verification Checklist

- [ ] Admin account created with MFA
- [ ] Admin role created and assigned
- [ ] Password policy configured
- [ ] At least one SSO provider configured (if needed)
- [ ] Users imported/created
- [ ] Roles created (admin, developer, auditor)
- [ ] Organizations set up
- [ ] Branding customized
- [ ] Login test passed
- [ ] Unauthorized access returns 401
- [ ] Alert rules configured
- [ ] Console accessible and functional

## Next Steps

- Configure OAuth clients for your applications
- [Set up SCIM provisioning](scim-provisioning.md) from Okta/Azure AD
- [Configure webhooks](webhook-events-guide.md) for real-time notifications
- [Review the [security checklist](security-audit-checklist.md)
- [Set up [backup and recovery](backup-restore.md)

## See Also

- [Console Admin Guide](console-admin-guide.md)
- [Quick Start](quick-start.md)
- [Production Checklist](production-checklist.md)
- [Branding Guide](branding-guide.md)
