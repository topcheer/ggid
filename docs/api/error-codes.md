# API Error Codes Reference

> Complete reference of all GGID API error codes, grouped by service.

> **Note**: This is a mirror of [docs/api-error-codes.md](../api-error-codes.md) placed under `docs/api/` for discoverability.

---

## How GGID Errors Work

Every error response follows this format:

```json
{
  "error": {
    "code": "AUTH_INVALID_CREDENTIALS",
    "message": "Invalid username or password",
    "http_status": 401
  }
}
```

Error codes use the format: `{SERVICE}_{DESCRIPTION}`

---

## Auth Service

| Code | HTTP | Trigger | Solution |
|------|------|---------|----------|
| `AUTH_INVALID_CREDENTIALS` | 401 | Wrong username/password | Check credentials; use `username` field (not `email`) |
| `AUTH_USER_NOT_FOUND` | 404 | User doesn't exist in tenant | Verify tenant_id and username |
| `AUTH_USER_EXISTS` | 409 | Username already taken | Use different username |
| `AUTH_USER_INACTIVE` | 403 | Account suspended | Contact admin to reactivate |
| `AUTH_ACCOUNT_LOCKED` | 423 | Too many failed logins | Wait 60s or restart auth container (dev) |
| `AUTH_TOKEN_INVALID` | 401 | JWT signature/claims invalid | Obtain new token via login |
| `AUTH_TOKEN_EXPIRED` | 401 | Access token expired | Use refresh token |
| `AUTH_REFRESH_INVALID` | 401 | Refresh token invalid/used | Re-authenticate |
| `AUTH_MFA_REQUIRED` | 403 | MFA enrollment needed | Complete MFA setup |
| `AUTH_MFA_INVALID` | 401 | Wrong/expired MFA code | Provide valid 6-digit TOTP |
| `AUTH_MFA_ALREADY_ENROLLED` | 409 | MFA already configured | Disable existing first |
| `AUTH_PASSWORD_TOO_WEAK` | 400 | Password doesn't meet policy | 8+ chars, upper/lower/digit/special |
| `AUTH_LDAP_UNAVAILABLE` | 503 | LDAP server unreachable | Check LDAP_URL config |
| `AUTH_WEBAUTHN_REGISTRATION_FAILED` | 400 | Attestation verification failed | Check device compatibility |
| `AUTH_WEBAUTHN_AUTH_FAILED` | 401 | Assertion verification failed | Re-register device |
| `AUTH_RATE_LIMITED` | 429 | Too many login attempts | Wait 60s |

## OAuth Service

| Code | HTTP | Trigger | Solution |
|------|------|---------|----------|
| `OAUTH_INVALID_CLIENT` | 401 | Bad client ID/secret | Verify client credentials |
| `OAUTH_INVALID_GRANT` | 400 | Auth code invalid/expired | Restart authorization flow |
| `OAUTH_INVALID_REQUEST` | 400 | Missing parameter | Check grant_type, code, redirect_uri |
| `OAUTH_INVALID_SCOPE` | 400 | Scope not allowed | Reduce scope or update client |
| `OAUTH_UNAUTHORIZED_CLIENT` | 401 | Client not authorized for grant | Enable grant type |
| `OAUTH_UNSUPPORTED_GRANT_TYPE` | 400 | Grant type not supported | Use authorization_code/client_credentials |
| `OAUTH_ACCESS_DENIED` | 403 | User denied consent | N/A |
| `OAUTH_INVALID_REDIRECT_URI` | 400 | URI doesn't match | Register exact URI |
| `OAUTH_INVALID_STATE` | 400 | State mismatch (CSRF) | Restart flow with fresh state |
| `OAUTH_PKCE_REQUIRED` | 400 | PKCE missing | Include code_challenge + S256 |
| `OAUTH_INVALID_PKCE_VERIFIER` | 400 | Verifier doesn't match | Check code_verifier |
| `OAUTH_TOKEN_REVOKED` | 401 | Token revoked | Obtain new token |
| `OAUTH_INTROSPECTION_FAILED` | 401 | Introspection auth failed | Provide client credentials |
| `OAUTH_DPOP_PROOF_INVALID` | 400 | DPoP JWT invalid | Check DPoP header |

## Identity Service

| Code | HTTP | Trigger | Solution |
|------|------|---------|----------|
| `IDENTITY_USER_NOT_FOUND` | 404 | User ID not found | Verify user_id + tenant_id |
| `IDENTITY_USER_EXISTS` | 409 | Same username/email | Use different value |
| `IDENTITY_INVALID_USER_DATA` | 400 | Validation failed | Check required fields |
| `IDENTITY_TENANT_MISMATCH` | 403 | User in different tenant | Check JWT tenant_id |
| `IDENTITY_BULK_LIMIT` | 400 | Batch > 100 records | Split request |

## Policy Service

| Code | HTTP | Trigger | Solution |
|------|------|---------|----------|
| `POLICY_ROLE_NOT_FOUND` | 404 | Role ID missing | Verify role exists |
| `POLICY_ROLE_EXISTS` | 409 | Duplicate role key | Use different `key` |
| `POLICY_ROLE_KEY_REQUIRED` | 400 | Empty `key` field | Provide unique key |
| `POLICY_PERMISSION_DENIED` | 403 | Lacks permission | Assign role with permission |
| `POLICY_ABAC_RULE_INVALID` | 400 | Rule syntax error | Check expression |

## Org Service

| Code | HTTP | Trigger | Solution |
|------|------|---------|----------|
| `ORG_NOT_FOUND` | 404 | Org ID missing | Verify org_id |
| `ORG_EXISTS` | 409 | Same org name | Use different name |
| `ORG_MEMBER_EXISTS` | 409 | User already member | Remove first |
| `ORG_CIRCULAR_REFERENCE` | 400 | Parent creates cycle | Check hierarchy |

## Audit Service

| Code | HTTP | Trigger | Solution |
|------|------|---------|----------|
| `AUDIT_QUERY_TOO_BROAD` | 400 | No filters | Add tenant_id or date range |
| `AUDIT_EXPORT_LIMIT` | 400 | Export > 100k records | Narrow date range |
| `AUDIT_EVENT_NOT_FOUND` | 404 | Event ID missing | Verify event_id |
| `AUDIT_NATS_UNAVAILABLE` | 503 | NATS down | Check NATS connectivity |

## Gateway

| Code | HTTP | Trigger | Solution |
|------|------|---------|----------|
| `GATEWAY_ROUTE_NOT_FOUND` | 404 | No route matches | Check API path |
| `GATEWAY_BACKEND_UNAVAILABLE` | 503 | Circuit breaker open | Wait 30s or check backend |
| `GATEWAY_BODY_TOO_LARGE` | 413 | Body > 10MB | Split request |
| `GATEWAY_INVALID_JSON` | 400 | Malformed JSON | Fix JSON syntax |
| `GATEWAY_MISSING_TENANT` | 400 | No tenant context | Include X-Tenant-ID header |

---

## HTTP Status Summary

| Status | Meaning | Common Codes |
|--------|---------|-------------|
| 400 | Bad Request | Validation errors |
| 401 | Unauthorized | Token invalid/expired |
| 403 | Forbidden | Insufficient scope/permission |
| 404 | Not Found | Resource missing |
| 409 | Conflict | Duplicate resource |
| 413 | Too Large | Body > 10MB |
| 423 | Locked | Account locked |
| 429 | Rate Limited | Too many requests |
| 503 | Unavailable | Backend/circuit breaker |

---

*Last updated: 2025-07-11*