# OAuth Client Lifecycle

Complete guide for managing OAuth client applications from registration to retirement.

## Stages

```
Registration → Onboarding → Active → Maintenance → Deprecation → Retirement
```

## 1. Registration

### Dynamic Registration (RFC 7591)

```bash
POST /api/v1/oauth/register
{
  "client_name": "My Mobile App",
  "redirect_uris": ["https://app.example.com/callback"],
  "grant_types": ["authorization_code", "refresh_token"],
  "response_types": ["code"],
  "token_endpoint_auth_method": "client_secret_basic",
  "scope": "openid profile email users:read"
}
# → 201 with client_id, client_secret
```

### Manual Registration (Console)

Admin Console → OAuth Clients → New Client → Fill wizard:

| Field | Required | Example |
|-------|----------|---------|
| Client name | Yes | "My Mobile App" |
| Redirect URIs | Yes | Exact HTTPS URLs |
| Grant types | Yes | authorization_code + refresh_token |
| Token auth method | Yes | client_secret_basic |
| Scopes | Yes | openid profile users:read |
| PKCE required | Recommended | true for SPA/mobile |

### Security Review

- Redirect URI must be exact match (no wildcards)
- HTTP redirect URIs rejected except for localhost
- Implicit flow discouraged — require PKCE
- Sector identifier URI for pairwise subject claims

## 2. Onboarding Wizard

The onboarding wizard guides developers through:

1. **Choose app type**: Web / SPA / Mobile / Backend service
2. **Configure auth method**: PKCE / client_secret / mTLS / private_key_jwt
3. **Set scopes**: Select from approved scope catalog
4. **Test integration**: Sandbox environment with sample credentials
5. **Review security**: Automated checks for redirect URI, PKCE, consent screen

## 3. Credential Rotation

```bash
# Generate new secret, keep old valid for grace period
POST /api/v1/oauth/clients/{client_id}/rotate-secret
# → {"new_secret": "...", "old_secret_valid_until": "2025-02-01T00:00:00Z"}

# After grace period, old secret invalidated automatically
```

Rotation triggers:
- Annual policy
- Suspected compromise
- Personnel turnover
- Key custodian change

## 4. Scope Changes

```bash
# Request additional scopes
PATCH /api/v1/oauth/clients/{client_id}
{
  "scope": "openid profile email users:read users:write"
}
```

Approval workflow:
1. Developer requests scope expansion
2. Admin reviews (risk assessment)
3. If `users:write` or admin scopes → require CISO approval
4. Changes logged in audit trail
5. User re-consent required on next authorization

## 5. Deployment Promotion (Dev → Prod)

```
Dev (sandbox.ggid.dev)
  → Staging (staging.ggid.dev)
    → Production (auth.ggid.dev)
```

| Environment | Client ID | Secret | Scopes |
|-------------|-----------|--------|--------|
| Dev | Auto-generated | Auto | All (testing) |
| Staging | Separate ID | Separate | Subset |
| Production | Separate ID | Separate | Approved only |

**Never share secrets across environments.**

## 6. Deprecation

When a client is deprecated but not yet retired:

- New authorizations blocked
- Existing refresh tokens honored until expiry
- Admin notified weekly
- Users see deprecation notice on consent screen
- 90-day countdown to retirement

```bash
PATCH /api/v1/oauth/clients/{client_id}
{"status": "deprecated", "deprecation_notice": "Use v2 client after Mar 2025"}
```

## 7. Retirement

```bash
DELETE /api/v1/oauth/clients/{client_id}
```

Retirement checklist:
1. Revoke all active access tokens
2. Revoke all refresh tokens
3. Revoke all grants
4. Remove from consent registry
5. Archive client metadata (7 years)
6. Notify connected users
7. Update discovery metadata

## Lifecycle Monitoring

| Metric | Alert Threshold |
|--------|----------------|
| Token exchange failures | >5% of requests |
| Scope expansion requests | >3/month for single client |
| Unused client (no auth in 90 days) | Flag for deprecation |
| Refresh token reuse | Immediate (theft indicator) |

## See Also

- [OAuth API](../api/oauth.md)
- [OAuth Error Handling](oauth-error-handling.md)
- OAuth Scope Design
- [JWT Security Best Practices](jwt-security-best-practices.md)
