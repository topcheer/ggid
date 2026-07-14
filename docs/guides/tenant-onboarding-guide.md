# Tenant Onboarding Guide

This guide covers the complete tenant onboarding workflow in GGID, from creation to go-live and post-onboarding health checks.

## Tenant Creation Flow

```
1. Admin creates tenant → assigns tenant admin
2. Initial configuration → SSO, MFA, password policy, branding
3. Default roles + permissions setup
4. SCIM/OAuth client provisioning
5. Tenant isolation verification
6. Data migration (if applicable)
7. Go-live checklist
8. Post-onboarding health check
```

### Step 1: Create Tenant

```bash
# Create tenant via API
curl -X POST https://auth.ggid.example.com/api/v1/admin/tenants \
  -H "Authorization: Bearer <platform-admin-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Acme Corporation",
    "domain": "acme.com",
    "plan": "enterprise",
    "admin_email": "admin@acme.com",
    "admin_name": "Jane Smith"
  }'
```

Response:
```json
{
  "tenant_id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Acme Corporation",
  "domain": "acme.com",
  "status": "active",
  "admin_user_id": "660e8400-e29b-41d4-a716-446655440001"
}
```

### Step 2: Assign Tenant Admin

The first user in a new tenant is automatically assigned the `tenant-admin` role:

```bash
# Verify admin user created
curl https://auth.ggid.example.com/api/v1/users/660e8400-... \
  -H "X-Tenant-ID: 550e8400-..." \
  -H "Authorization: Bearer <platform-admin-token>"
```

## Initial Configuration Checklist

### SSO Configuration

```yaml
tenant:
  sso:
    enabled: true
    providers:
      - type: "saml"
        entity_id: "https://acme.com/saml"
        idp_metadata_url: "https://idp.acme.com/metadata"
        name_id_format: "emailAddress"
      - type: "oidc"
        issuer: "https://idp.acme.com"
        client_id: "acme-ggid"
        client_secret: "<secret>"
        scopes: ["openid", "profile", "email"]
```

**Checklist**:
- [ ] IdP metadata imported and validated
- [ ] SP metadata exported to IdP
- [ ] Test SSO login with IdP test account
- [ ] Configure attribute mappings
- [ ] Set up SLO (Single Logout) if required

### MFA Configuration

```yaml
tenant:
  mfa:
    required: true
    methods:
      - totp
      - webauthn
    enforce_for_admins: true
    grace_period: 7d        # Users have 7 days to enroll
    backup_codes: true
    recovery_key: true
```

**Checklist**:
- [ ] MFA policy defined (required vs optional)
- [ ] TOTP configured (Google Authenticator, Authy)
- [ ] WebAuthn configured (passkeys, security keys)
- [ ] Backup codes generated for initial admin
- [ ] Recovery flow tested

### Password Policy

```yaml
tenant:
  password_policy:
    min_length: 12
    require_uppercase: true
    require_lowercase: true
    require_digit: true
    require_special: true
    history_count: 12       # Can't reuse last 12 passwords
    max_age: 90d            # Must change every 90 days
    breach_check: true      # Check against HIBP
    pepper: true            # Server-side pepper
```

**Checklist**:
- [ ] Password complexity rules set
- [ ] Password history enabled
- [ ] Breach detection (HIBP) configured
- [ ] Password pepper enabled
- [ ] Test password reset flow

### Branding

```yaml
tenant:
  branding:
    logo_url: "https://acme.com/logo.png"
    primary_color: "#0066CC"
    secondary_color: "#FFFFFF"
    login_page_title: "Acme Corporation - Sign In"
    favicon_url: "https://acme.com/favicon.ico"
    custom_css: "/etc/ggid/tenants/acme/custom.css"
    email_templates:
      welcome: "acme-welcome.html"
      password_reset: "acme-reset.html"
      mfa_enrollment: "acme-mfa.html"
```

**Checklist**:
- [ ] Logo uploaded
- [ ] Color scheme configured
- [ ] Custom login page tested
- [ ] Email templates customized
- [ ] Favicon set

## Default Roles and Permissions

### Standard Role Set

```bash
# Create default roles for new tenant
curl -X POST https://auth.ggid.example.com/api/v1/roles \
  -H "X-Tenant-ID: 550e8400-..." \
  -H "Authorization: Bearer <tenant-admin-token>" \
  -d '{
    "key": "tenant-admin",
    "name": "Tenant Administrator",
    "permissions": ["*"]
  }'
```

| Role | Key | Permissions | Assigned To |
|---|---|---|---|
| Tenant Admin | `tenant-admin` | All tenant operations | Initial admin |
| Security Admin | `security-admin` | Security config, audit, MFA reset | Security team |
| User Admin | `user-admin` | User CRUD, role assignment | Helpdesk |
| App Admin | `app-admin` | OAuth client management | Development leads |
| Read-Only | `viewer` | Read all resources, no writes | Auditors |
| User | `user` | Self-service profile, password | All users |

### Permission Matrix

| Permission | tenant-admin | security-admin | user-admin | app-admin | viewer | user |
|---|---|---|---|---|---|---|
| users:read | ✓ | ✓ | ✓ | - | ✓ | (self) |
| users:write | ✓ | - | ✓ | - | - | (self) |
| roles:read | ✓ | ✓ | ✓ | - | ✓ | - |
| roles:write | ✓ | - | ✓ | - | - | - |
| security:config | ✓ | ✓ | - | - | - | - |
| audit:read | ✓ | ✓ | - | - | ✓ | - |
| oauth:manage | ✓ | - | - | ✓ | - | - |
| mfa:reset | ✓ | ✓ | - | - | - | - |

## SCIM/OAuth Client Provisioning

### SCIM Endpoint

```bash
# Configure SCIM for tenant
curl -X POST https://auth.ggid.example.com/api/v1/admin/scim/config \
  -H "X-Tenant-ID: 550e8400-..." \
  -H "Authorization: Bearer <tenant-admin-token>" \
  -d '{
    "endpoint": "https://auth.ggid.example.com/scim/v2/550e8400-...",
    "auth_token": "<generate-secure-token>",
    "provisioning_mode": "bidirectional"
  }'
```

**Checklist**:
- [ ] SCIM endpoint generated
- [ ] Auth token configured in IdP
- [ ] Test user provisioning (create/update/delete)
- [ ] Test group sync
- [ ] Verify deprovisioning works

### OAuth Client Registration

```bash
# Register OAuth client for tenant app
curl -X POST https://auth.ggid.example.com/api/v1/oauth/register \
  -H "Authorization: Bearer <tenant-admin-token>" \
  -d '{
    "client_name": "Acme Web App",
    "redirect_uris": ["https://app.acme.com/callback"],
    "grant_types": ["authorization_code", "refresh_token"],
    "response_types": ["code"],
    "scope": "openid profile email groups",
    "token_endpoint_auth_method": "client_secret_basic"
  }'
```

Response:
```json
{
  "client_id": "acme-web-app-001",
  "client_secret": "<generated-secret>",
  "client_id_issued_at": 1700000000
}
```

**Checklist**:
- [ ] Client registered with correct redirect URIs
- [ ] Required scopes configured
- [ ] Client secret stored securely
- [ ] Test OAuth flow end-to-end
- [ ] Configure PKCE for public clients

## Tenant Isolation Verification

### Row-Level Security (RLS)

GGID uses PostgreSQL Row-Level Security to enforce tenant isolation:

```sql
-- Verify RLS is enabled
SELECT relname, relrowsecurity
FROM pg_class
WHERE relname IN ('users', 'roles', 'audit_events');

-- Test isolation: query as tenant A, should not see tenant B data
SET app.tenant_id = '550e8400-...';
SELECT count(*) FROM users;  -- Should only return tenant A users
```

### Verification Steps

```bash
# 1. Create test user in tenant A
curl -X POST .../api/v1/users -H "X-Tenant-ID: tenantA" -d '{"username":"testA"}'

# 2. Try to access tenant A user from tenant B
curl .../api/v1/users/testA -H "X-Tenant-ID: tenantB"
# Expected: 404 Not Found

# 3. Verify JWT contains tenant_id
# Decode JWT payload → check tenant_id claim matches

# 4. Verify audit logs are tenant-scoped
curl .../api/v1/audit/events -H "X-Tenant-ID: tenantA"
# Should only return tenant A events
```

**Checklist**:
- [ ] RLS policies verified on all tables
- [ ] Cross-tenant access returns 404
- [ ] JWT tenant_id claim validated on every request
- [ ] Audit logs are tenant-scoped
- [ ] No shared resources between tenants

## Data Migration

### From Existing System

```yaml
migration:
  source: "legacy-iam"  # or "keycloak", "auth0", "okta"
  batch_size: 1000
  parallelism: 4
  dry_run: true         # Test first
  fields:
    users:
      - source: "username"
        target: "username"
      - source: "email"
        target: "email"
      - source: "password_hash"
        target: "password_hash"
        transform: "rehash-argon2id"
```

### Migration Steps

1. **Export** users, roles, groups from source system
2. **Transform** data to GGID schema
3. **Dry run** — validate without writing
4. **Import** in batches with progress tracking
5. **Verify** — compare counts, spot-check records
6. **Password migration** — mark for rehash on next login
7. **Cutover** — update DNS, redirect old endpoints

**Checklist**:
- [ ] Source data exported and validated
- [ ] Field mapping defined and tested
- [ ] Dry run completed successfully
- [ ] User count matches source
- [ ] Password hashes migrated (will be rehashed on first login)
- [ ] Role assignments preserved
- [ ] Rollback plan ready

## Go-Live Checklist

### Pre-Launch

- [ ] Tenant created and configured
- [ ] SSO tested with production IdP
- [ ] MFA enrollment tested
- [ ] Password policy validated
- [ ] Branding applied and tested
- [ ] Default roles and permissions assigned
- [ ] SCIM provisioning tested (if applicable)
- [ ] OAuth clients registered and tested
- [ ] Tenant isolation verified
- [ ] Data migration completed (if applicable)
- [ ] Audit logging verified
- [ ] Rate limits configured
- [ ] DNS configured for custom domain
- [ ] TLS certificates valid
- [ ] Backup configured

### Launch

- [ ] Invite initial users
- [ ] Monitor login success rate
- [ ] Monitor error rates
- [ ] Verify audit events flowing
- [ ] Check Redis connectivity
- [ ] Check database connectivity
- [ ] Verify health endpoints responding

### Post-Launch

- [ ] Daily health check for first week
- [ ] Review audit logs for anomalies
- [ ] Collect user feedback
- [ ] Monitor token issuance rates
- [ ] Verify SCIM sync health
- [ ] Check rate limit metrics

## Post-Onboarding Health Check

```bash
# Run health check script
curl https://auth.ggid.example.com/api/v1/admin/tenants/550e8400-.../health \
  -H "Authorization: Bearer <platform-admin-token>"
```

Response:
```json
{
  "tenant_id": "550e8400-...",
  "status": "healthy",
  "checks": {
    "database": "ok",
    "redis": "ok",
    "sso": "ok",
    "scim": "ok",
    "oauth_clients": 3,
    "users": 150,
    "mfa_enrollment_rate": 0.87,
    "audit_events_24h": 3421,
    "rate_limit_denied_24h": 12
  }
}
```

### Health Check Metrics

| Metric | Healthy | Warning | Critical |
|---|---|---|---|
| Login success rate | >95% | 85-95% | <85% |
| MFA enrollment rate | >80% | 50-80% | <50% |
| API error rate | <1% | 1-5% | >5% |
| Audit events/day | >0 | 0 (misconfigured) | N/A |
| Rate limit denials | <100/day | 100-1000/day | >1000/day |

## Best Practices

1. **Test everything before go-live** — SSO, MFA, SCIM, OAuth flows
2. **Start small** — onboard a pilot group before all users
3. **Monitor closely** — check metrics daily for the first week
4. **Have a rollback plan** — know how to revert if issues arise
5. **Document configuration** — keep record of all tenant settings
6. **Regular health checks** — automated monitoring post-launch
7. **User training** — provide guides for MFA enrollment and SSO login
8. **Phase the migration** — don't migrate all users simultaneously
