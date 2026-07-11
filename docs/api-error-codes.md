# GGID API Error Codes Reference

Complete reference for all GGID error codes with HTTP status, description, and
retry guidance.

---

## Error Response Format

All errors follow a standard JSON structure:

```json
{
  "error": "invalid_request",
  "code": "AUTH_INVALID_CREDENTIALS",
  "description": "The username or password is incorrect.",
  "request_id": "req-abc-123",
  "retry_after": null
}
```

---

## AUTH_* Errors

| Code | HTTP | Description | Retry? |
|------|------|-------------|--------|
| `AUTH_INVALID_CREDENTIALS` | 401 | Wrong username or password | Yes (after delay) |
| `AUTH_ACCOUNT_LOCKED` | 423 | Account locked after too many failures | After lockout expires |
| `AUTH_ACCOUNT_SUSPENDED` | 403 | Account is suspended by admin | No |
| `AUTH_ACCOUNT_NOT_FOUND` | 404 | User does not exist | No |
| `AUTH_EMAIL_NOT_VERIFIED` | 403 | Email verification required | No |
| `AUTH_TOKEN_EXPIRED` | 401 | Access token has expired | Use refresh token |
| `AUTH_TOKEN_INVALID` | 401 | Token signature invalid or malformed | No |
| `AUTH_TOKEN_REVOKED` | 401 | Token has been revoked | Re-authenticate |
| `AUTH_REFRESH_TOKEN_INVALID` | 401 | Refresh token is invalid or reused | Re-authenticate |
| `AUTH_MFA_REQUIRED` | 403 | MFA verification needed | With MFA code |
| `AUTH_MFA_INVALID_CODE` | 401 | Wrong MFA code | Yes (limited) |
| `AUTH_PASSWORD_TOO_WEAK` | 400 | Password doesn't meet policy | No |
| `AUTH_PASSWORD_REUSE` | 400 | Password matches recent password | No |
| `AUTH_SESSION_EXPIRED` | 401 | Session has expired | Re-authenticate |

---

## OAUTH_* Errors

| Code | HTTP | Description | Retry? |
|------|------|-------------|--------|
| `OAUTH_INVALID_CLIENT` | 401 | Client ID or secret invalid | No |
| `OAUTH_INVALID_GRANT` | 400 | Authorization code is expired or invalid | No |
| `OAUTH_INVALID_REQUEST` | 400 | Missing required parameter | Fix and retry |
| `OAUTH_INVALID_SCOPE` | 400 | Requested scope not allowed | No |
| `OAUTH_UNAUTHORIZED_CLIENT` | 403 | Client not authorized for grant type | No |
| `OAUTH_UNSUPPORTED_GRANT_TYPE` | 400 | Grant type not supported | No |
| `OAUTH_REDIRECT_URI_MISMATCH` | 400 | Redirect URI doesn't match registration | No |
| `OAUTH_ACCESS_DENIED` | 403 | User denied consent | No |
| `OAUTH_SERVER_ERROR` | 500 | Internal OAuth server error | Yes (backoff) |

---

## POLICY_* Errors

| Code | HTTP | Description | Retry? |
|------|------|-------------|--------|
| `POLICY_ACCESS_DENIED` | 403 | RBAC/ABAC check failed | No |
| `POLICY_ROLE_EXISTS` | 409 | Role with same key already exists | No |
| `POLICY_ROLE_NOT_FOUND` | 404 | Role does not exist | No |
| `POLICY_INVALID_PERMISSION` | 400 | Permission string malformed | No |
| `POLICY_QUOTA_EXCEEDED` | 429 | Maximum roles for tier reached | No |

---

## IDENTITY_* Errors

| Code | HTTP | Description | Retry? |
|------|------|-------------|--------|
| `IDENTITY_USER_EXISTS` | 409 | Username or email already registered | No |
| `IDENTITY_USER_NOT_FOUND` | 404 | User does not exist | No |
| `IDENTITY_INVALID_EMAIL` | 400 | Email format invalid | No |
| `IDENTITY_INVALID_USERNAME` | 400 | Username format invalid | No |
| `IDENTITY_DUPLICATE_EMAIL` | 409 | Email already in use | No |
| `IDENTITY_ORG_NOT_FOUND` | 404 | Organization does not exist | No |

---

## SCIM_* Errors

| Code | HTTP | Description | Retry? |
|------|------|-------------|--------|
| `SCIM_INVALID_FILTER` | 400 | SCIM filter syntax error | Fix and retry |
| `SCIM_RESOURCE_NOT_FOUND` | 404 | SCIM resource (User/Group) not found | No |
| `SCIM_TOO_MANY` | 400 | Too many resources returned | Add pagination |
| `SCIM_INVALID_SCHEMA` | 400 | Request schema doesn't match SCIM spec | No |
| `SCIM_MUTABILITY` | 409 | Attempted to modify read-only attribute | No |

---

## TENANT_* Errors

| Code | HTTP | Description | Retry? |
|------|------|-------------|--------|
| `TENANT_NOT_FOUND` | 404 | Tenant ID does not exist | No |
| `TENANT_SUSPENDED` | 403 | Tenant is suspended | No |
| `TENANT_QUOTA_EXCEEDED` | 429 | Tenant has exceeded tier quota | Upgrade tier |
| `TENANT_INVALID_HEADER` | 400 | Missing or invalid X-Tenant-ID header | No |

---

## RATE_LIMIT_* Errors

| Code | HTTP | Description | Retry? |
|------|------|-------------|--------|
| `RATE_LIMIT_EXCEEDED` | 429 | Rate limit hit | After Retry-After |
| `RATE_LIMIT_QUOTA_EXCEEDED` | 429 | Monthly/daily quota exceeded | Next period |

### 429 Response Headers

```
X-RateLimit-Limit: 60
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1721034600
Retry-After: 42
```

---

## WEBHOOK_* Errors

| Code | HTTP | Description | Retry? |
|------|------|-------------|--------|
| `WEBHOOK_DELIVERY_FAILED` | N/A | Webhook delivery to subscriber failed | Auto-retry (3x) |
| `WEBHOOK_SIGNATURE_INVALID` | 401 | Webhook signature verification failed | No |
| `WEBHOOK_EVENT_UNKNOWN` | 400 | Event type not recognized | No |

---

## Validation Errors (400)

```json
{
  "error": "validation_failed",
  "code": "VALIDATION_ERROR",
  "details": [
    { "field": "email", "message": "must be a valid email address" },
    { "field": "password", "message": "must be at least 12 characters" }
  ]
}
```

---

## Retry Guidance

| HTTP Status | Retry? | Backoff Strategy |
|-------------|--------|-----------------|
| 400 | No | Fix request |
| 401 | No | Re-authenticate |
| 403 | No | Fix permissions |
| 404 | No | Check resource ID |
| 409 | No | Handle conflict |
| 429 | Yes | Exponential: 1s, 2s, 4s, 8s (max 60s) |
| 500 | Yes | Exponential: 1s, 2s, 4s (max 3 retries) |
| 502, 503 | Yes | Fixed 5s delay (up to 10 retries) |

---

## References

- [API Reference](./api-reference.md) — Endpoint documentation
- [Rate Limiting](./api-rate-limiting.md) — Rate limit details
- [SDK Guide](./sdk-guide.md) — SDK error handling

---

## Per-Service Error Codes

### Auth Service Errors

| Code | HTTP | Message | Cause | Resolution |
|------|------|---------|-------|------------|
| `auth.invalid_credentials` | 401 | Invalid username or password | Wrong password | Check credentials, retry |
| `auth.account_locked` | 423 | Account locked after {N} failed attempts | 5+ failed logins | Wait 30 min or admin unlock |
| `auth.account_disabled` | 403 | Account is deactivated | Admin deactivated | Contact administrator |
| `auth.token_expired` | 401 | Token has expired | JWT `exp` passed | Refresh token |
| `auth.token_invalid` | 401 | Invalid token signature | Tampered JWT | Re-authenticate |
| `auth.refresh_token_reuse` | 401 | Refresh token reuse detected | Stolen token | Re-authenticate (all tokens revoked) |
| `auth.mfa_required` | 403 | MFA challenge required | MFA enabled | Complete MFA flow |
| `auth.mfa_invalid` | 401 | Invalid MFA code | Wrong TOTP code | Retry with new code |
| `auth.password_too_weak` | 400 | Password does not meet policy | Short, missing chars | Use stronger password |

```bash
# Trigger: wrong password
curl -sX POST "$GW/api/v1/auth/login" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"username":"user","password":"wrong"}' | jq .
# {"error":"auth.invalid_credentials","message":"Invalid username or password"}

# Trigger: expired token
curl -s "$GW/api/v1/users" \
  -H "Authorization: Bearer expired.jwt.token" \
  -H "X-Tenant-ID: $TENANT" | jq .
# {"error":"auth.token_expired","message":"Token has expired"}
```

### Identity Service Errors

| Code | HTTP | Message | Cause |
|------|------|---------|-------|
| `identity.user_not_found` | 404 | User not found | Wrong ID or tenant |
| `identity.duplicate_username` | 409 | Username already exists | Registration conflict |
| `identity.duplicate_email` | 409 | Email already registered | Registration conflict |
| `identity.invalid_user_data` | 400 | Invalid user data | Missing required field |
| `identity.scim_invalid_filter` | 400 | Invalid SCIM filter syntax | Malformed filter |

```bash
# Trigger: duplicate registration
curl -sX POST "$GW/api/v1/auth/register" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"username":"existing","password":"Test123!"}' | jq .
# {"error":"identity.duplicate_username","message":"Username already exists"}
```

### Policy Service Errors

| Code | HTTP | Message | Cause |
|------|------|---------|-------|
| `policy.role_not_found` | 404 | Role not found | Wrong role ID |
| `policy.duplicate_role_key` | 409 | Role key already exists | UNIQUE(tenant_id, key) conflict |
| `policy.access_denied` | 403 | Insufficient permissions | User lacks required role |
| `policy.evaluation_error` | 500 | Policy evaluation failed | Malformed policy rule |

```bash
# Trigger: create role with duplicate key
curl -sX POST "$GW/api/v1/roles" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT" \
  -d '{"name":"Admin","key":"admin"}' | jq .
# {"error":"policy.duplicate_role_key","message":"Role key 'admin' already exists"}
```

### OAuth Service Errors

| Code | HTTP | Message | Cause |
|------|------|---------|-------|
| `oauth.invalid_grant` | 400 | Invalid authorization grant | Expired/used code |
| `oauth.invalid_client` | 401 | Client authentication failed | Wrong secret |
| `oauth.invalid_redirect_uri` | 400 | Redirect URI mismatch | Not registered |
| `oauth.pkce_required` | 400 | code_challenge is required | Missing PKCE |
| `oauth.invalid_scope` | 400 | Requested scope not allowed | Client lacks scope |
| `oauth.unsupported_grant_type` | 400 | Grant type not supported | Implicit/password disabled |
| `oauth.authorization_pending` | 400 | Authorization pending (device flow) | User hasn't approved |

```bash
# Trigger: missing PKCE
curl -s "$GW/oauth/authorize?response_type=code&client_id=app&redirect_uri=https://app/cb" | jq .
# {"error":"oauth.pkce_required","message":"code_challenge is required"}

# Trigger: wrong client secret
curl -sX POST "$GW/oauth/token" \
  -d "grant_type=client_credentials&client_id=x&client_secret=wrong" | jq .
# {"error":"oauth.invalid_client","message":"Client authentication failed"}
```

### Gateway Errors

| Code | HTTP | Message | Cause |
|------|------|---------|-------|
| `gateway.missing_tenant` | 412 | Missing X-Tenant-ID header | Header omitted |
| `gateway.tenant_not_found` | 404 | Tenant not found | Invalid tenant ID |
| `gateway.cross_tenant_denied` | 403 | Cross-tenant access denied | JWT tenant ≠ header |
| `gateway.rate_limited` | 429 | Rate limit exceeded | Too many requests |
| `gateway.backend_unavailable` | 502 | Backend service unavailable | Service down |
| `gateway.circuit_open` | 503 | Circuit breaker open | Backend failing repeatedly |

```bash
# Trigger: missing tenant header
curl -s "$GW/api/v1/users" -H "Authorization: Bearer $TOKEN" | jq .
# {"error":"gateway.missing_tenant","message":"X-Tenant-ID header required"}

# Trigger: rate limited
for i in $(seq 1 100); do curl -s "$GW/api/v1/auth/login" ...; done
# {"error":"gateway.rate_limited","message":"Rate limit exceeded. Retry after 30s"}
```

### Audit Service Errors

| Code | HTTP | Message | Cause |
|------|------|---------|-------|
| `audit.query_too_broad` | 400 | Query range too broad | Date range > 90 days without filter |
| `audit.export_limit` | 400 | Export exceeds max records | > 100k records requested |

### SCIM Errors (RFC 7644)

| Code | HTTP | scimType | Cause |
|------|------|----------|-------|
| `invalidFilter` | 400 | invalidFilter | Malformed filter expression |
| `uniqueness` | 409 | uniqueness | Attribute uniqueness violated |
| `invalidPath` | 400 | invalidPath | PATCH path not found |
| `noTarget` | 400 | noTarget | PATCH target missing |
| `tooMany` | 400 | tooMany | Results exceed maxResults |

---

## Complete Error Reference by Service

### Auth Service Errors

| Error Code | HTTP Status | Description | How to Fix |
|-----------|-------------|-------------|------------|
| `AUTH_INVALID_CREDENTIALS` | 401 | Username or password incorrect | Check credentials; verify `username` field is used (not `email`) |
| `AUTH_USER_NOT_FOUND` | 404 | User does not exist in tenant | Verify tenant_id and username |
| `AUTH_USER_EXISTS` | 409 | Username already taken | Use a different username |
| `AUTH_USER_INACTIVE` | 403 | Account suspended or deactivated | Contact admin to reactivate |
| `AUTH_ACCOUNT_LOCKED` | 423 | Account locked after too many failures | Wait 60s or restart auth container in dev |
| `AUTH_TOKEN_INVALID` | 401 | JWT signature or claims invalid | Obtain new token via login |
| `AUTH_TOKEN_EXPIRED` | 401 | Access token has expired | Use refresh token to get new access token |
| `AUTH_REFRESH_INVALID` | 401 | Refresh token invalid or already used | Re-authenticate |
| `AUTH_MFA_REQUIRED` | 403 | MFA enrollment or verification required | Complete MFA setup or provide MFA code |
| `AUTH_MFA_INVALID` | 401 | MFA code is wrong or expired | Provide valid 6-digit TOTP code |
| `AUTH_MFA_ALREADY_ENROLLED` | 409 | MFA already configured for this method | Disable existing MFA first |
| `AUTH_PASSWORD_TOO_WEAK` | 400 | Password doesn't meet policy | Use 8+ chars with upper, lower, digit, special |
| `AUTH_LDAP_UNAVAILABLE` | 503 | LDAP server unreachable | Check LDAP_URL and LDAP_BIND_DN config |
| `AUTH_LDAP_BIND_FAILED` | 401 | LDAP bind failed for user | Check LDAP credentials or filter |
| `AUTH_WEBAUTHN_REGISTRATION_FAILED` | 400 | WebAuthn attestation verification failed | Check device compatibility |
| `AUTH_WEBAUTHN_AUTH_FAILED` | 401 | WebAuthn assertion verification failed | Re-register device |
| `AUTH_RATE_LIMITED` | 429 | Too many login attempts | Wait 60s or contact admin |

### OAuth Service Errors

| Error Code | HTTP Status | Description | How to Fix |
|-----------|-------------|-------------|------------|
| `OAUTH_INVALID_CLIENT` | 401 | Client ID or secret invalid | Verify client credentials |
| `OAUTH_INVALID_GRANT` | 400 | Authorization code invalid or expired | Restart authorization flow |
| `OAUTH_INVALID_REQUEST` | 400 | Missing required parameter | Check `grant_type`, `code`, `redirect_uri` |
| `OAUTH_INVALID_SCOPE` | 400 | Requested scope not allowed for client | Reduce scope or update client config |
| `OAUTH_UNAUTHORIZED_CLIENT` | 401 | Client not authorized for this grant type | Enable grant type for client |
| `OAUTH_UNSUPPORTED_GRANT_TYPE` | 400 | Grant type not supported | Use `authorization_code`, `client_credentials`, or `device_code` |
| `OAUTH_ACCESS_DENIED` | 403 | Resource owner denied authorization | User declined consent |
| `OAUTH_INVALID_REDIRECT_URI` | 400 | Redirect URI doesn't match registered | Register exact URI in client config |
| `OAUTH_INVALID_STATE` | 400 | State parameter mismatch (CSRF) | Restart authorization flow with fresh state |
| `OAUTH_PKCE_REQUIRED` | 400 | PKCE code challenge required | Include `code_challenge` and `code_challenge_method=S256` |
| `OAUTH_INVALID_PKCE_VERIFIER` | 400 | Code verifier doesn't match challenge | Verify `code_verifier` matches original `code_challenge` |
| `OAUTH_TOKEN_NOT_FOUND` | 404 | Token not found during introspection | Token may be expired or revoked |
| `OAUTH_TOKEN_REVOKED` | 401 | Token has been revoked | Obtain new token |
| `OAUTH_INTROSPECTION_FAILED` | 401 | Introspection endpoint authentication failed | Provide valid client credentials |
| `OAUTH_DPOP_PROOF_INVALID` | 400 | DPoP proof JWT validation failed | Check DPoP header format and signature |
| `OAUTH_DEVICE_CODE_EXPIRED` | 400 | Device authorization code expired | Request new device code |

### Identity Service Errors

| Error Code | HTTP Status | Description | How to Fix |
|-----------|-------------|-------------|------------|
| `IDENTITY_USER_NOT_FOUND` | 404 | User ID not found in tenant | Verify user_id and tenant_id |
| `IDENTITY_USER_EXISTS` | 409 | User with same username/email exists | Use different username/email |
| `IDENTITY_INVALID_USER_DATA` | 400 | Validation failed on user fields | Check required fields, email format |
| `IDENTITY_TENANT_MISMATCH` | 403 | User does not belong to requesting tenant | Check JWT tenant_id matches user's tenant |
| `IDENTITY_BULK_LIMIT` | 400 | Bulk operation exceeds max batch size | Limit to 100 records per request |

### Policy Service Errors

| Error Code | HTTP Status | Description | How to Fix |
|-----------|-------------|-------------|------------|
| `POLICY_ROLE_NOT_FOUND` | 404 | Role ID not found | Verify role exists in tenant |
| `POLICY_ROLE_EXISTS` | 409 | Role with same key already exists | Use different role `key` |
| `POLICY_ROLE_KEY_REQUIRED` | 400 | Role `key` field is empty | Provide unique `key` (UNIQUE constraint) |
| `POLICY_PERMISSION_DENIED` | 403 | User lacks required permission | Assign role with needed permissions |
| `POLICY_ABAC_RULE_INVALID` | 400 | ABAC rule condition syntax error | Check expression syntax |
| `POLICY_ABAC_EVAL_ERROR` | 500 | Error evaluating ABAC condition | Check attribute names in rule |

### Org Service Errors

| Error Code | HTTP Status | Description | How to Fix |
|-----------|-------------|-------------|------------|
| `ORG_NOT_FOUND` | 404 | Organization ID not found | Verify org_id |
| `ORG_EXISTS` | 409 | Organization with same name exists | Use different name |
| `ORG_MEMBER_EXISTS` | 409 | User already in organization | Remove first, then re-add |
| `ORG_CIRCULAR_REFERENCE` | 400 | Parent org would create cycle | Check org hierarchy tree |
| `ORG_MAX_DEPTH_EXCEEDED` | 400 | Organization tree too deep | Restructure hierarchy |

### Audit Service Errors

| Error Code | HTTP Status | Description | How to Fix |
|-----------|-------------|-------------|------------|
| `AUDIT_QUERY_TOO_BROAD` | 400 | Query missing required filters | Add tenant_id or date range filter |
| `AUDIT_EXPORT_LIMIT` | 400 | Export exceeds 100k records | Narrow date range |
| `AUDIT_EVENT_NOT_FOUND` | 404 | Specific event ID not found | Verify event_id |
| `AUDIT_NATS_UNAVAILABLE` | 503 | Cannot publish to NATS | Check NATS connectivity |

### Gateway Errors

| Error Code | HTTP Status | Description | How to Fix |
|-----------|-------------|-------------|------------|
| `GATEWAY_ROUTE_NOT_FOUND` | 404 | No route matches the path | Check API path spelling |
| `GATEWAY_BACKEND_UNAVAILABLE` | 503 | Circuit breaker open or backend down | Wait or check service health |
| `GATEWAY_BODY_TOO_LARGE` | 413 | Request body exceeds 10MB | Split request or reduce size |
| `GATEWAY_INVALID_JSON` | 400 | Request body is not valid JSON | Fix JSON syntax |
| `GATEWAY_MISSING_TENANT` | 400 | No tenant context found | Include `X-Tenant-ID` header or valid JWT |

---

## HTTP Status Code Summary

| HTTP Status | Meaning | GGID Usage |
|-------------|---------|------------|
| 200 | OK | Successful GET, PUT, PATCH |
| 201 | Created | Successful POST (create) |
| 204 | No Content | Successful DELETE |
| 400 | Bad Request | Validation errors, malformed input |
| 401 | Unauthorized | Missing or invalid JWT |
| 403 | Forbidden | Insufficient scope or permission |
| 404 | Not Found | Resource doesn't exist |
| 409 | Conflict | Duplicate resource |
| 413 | Payload Too Large | Body > 10MB |
| 423 | Locked | Account locked (brute force) |
| 429 | Too Many Requests | Rate limit exceeded |
| 500 | Internal Server Error | Unhandled exception |
| 503 | Service Unavailable | Backend down, circuit open |

---

*Last updated: 2025-07-11*
