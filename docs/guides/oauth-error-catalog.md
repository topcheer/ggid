# OAuth Error Catalog

All RFC error codes, HTTP status mapping, user-facing messages, retry guidance, developer documentation, and troubleshooting flowchart.

## Authorization Endpoint Errors

| Error Code | HTTP | User Message | Retry? | RFC |
|-----------|------|-------------|--------|-----|
| `invalid_request` | 302 | "The request is missing a required parameter." | Fix and retry | 6749 |
| `unauthorized_client` | 302 | "This app is not authorized for this flow." | Contact admin | 6749 |
| `access_denied` | 302 | "The user or admin denied the request." | User choice | 6749 |
| `unsupported_response_type` | 302 | "The requested response type is not supported." | Fix response_type | 6749 |
| `invalid_scope` | 302 | "The requested scope is invalid or exceeds granted." | Fix scopes | 6749 |
| `server_error` | 302/500 | "Authorization server encountered an error." | Retry after 5s | 6749 |
| `temporarily_unavailable` | 302/503 | "Service temporarily unavailable." | Retry after 30s | 6749 |
| `interaction_required` | 302 | "User interaction is required." | Redirect to login | OIDC |
| `login_required` | 302 | "User must re-authenticate." | Redirect to login | OIDC |
| `account_selection_required` | 302 | "User must select an account." | Show picker | OIDC |
| `consent_required` | 302 | "User consent is required." | Show consent | OIDC |
| `invalid_request_uri` | 302 | "The request_uri is invalid or expired." | Restart flow | JAR |
| `invalid_request_object` | 302 | "The signed request object is invalid." | Fix JWT | JAR |
| `request_not_supported` | 302 | "Request objects are not supported." | Use params directly | JAR |
| `request_uri_not_supported` | 302 | "request_uri is not supported." | Use params directly | JAR |

## Token Endpoint Errors

| Error Code | HTTP | Developer Message | Retry? | Action |
|-----------|------|------------------|--------|--------|
| `invalid_request` | 400 | "Missing or duplicate parameter." | Fix params | — |
| `invalid_client` | 401 | "Client authentication failed." | Check credentials | — |
| `invalid_grant` | 400 | "Authorization code expired, reused, or invalid." | Restart flow | — |
| `unauthorized_client` | 401 | "Client not authorized for this grant_type." | Fix config | — |
| `unsupported_grant_type` | 400 | "grant_type not supported." | Use supported type | — |
| `invalid_scope` | 400 | "Requested scope exceeds authorized scope." | Fix scopes | — |
| `expired_token` | 400 | "Device code expired." | Restart device flow | RFC 8628 |
| `access_denied` | 400 | "User denied the device authorization." | Stop polling | RFC 8628 |
| `authorization_pending` | 400 | "User hasn't completed authorization." | Poll after interval | RFC 8628 |
| `slow_down` | 400 | "Polling too fast." | Add 5s to interval | RFC 8628 |

## Introspection Errors

| Error Code | HTTP | Meaning |
|-----------|------|---------|
| `{"active": false}` | 200 | Token is invalid/expired/revoked |
| `invalid_client` | 401 | Resource server not authenticated |
| `invalid_request` | 400 | Missing token parameter |

## Registration Errors (RFC 7591)

| Error Code | HTTP | Meaning |
|-----------|------|---------|
| `invalid_client_metadata` | 400 | Missing/invalid field in registration |
| `invalid_redirect_uri` | 400 | Non-HTTPS or wildcard URI |
| `invalid_software_statement` | 400 | Bad signature or untrusted issuer |
| `access_denied` | 403 | Scope not allowed at registration level |

## User-Facing Messages

### Login Errors

| Scenario | Message (EN) |
|----------|-------------|
| Wrong password | "Incorrect email or password." |
| Account locked | "Your account has been locked. Please contact your administrator." |
| Account suspended | "This account is suspended. Contact support." |
| MFA failed | "Incorrect verification code. Please try again." |
| Rate limited | "Too many attempts. Please wait a moment." |
| Expired session | "Your session has expired. Please log in again." |

### Consent Errors

| Scenario | Message (EN) |
|----------|-------------|
| Access denied | "You denied access to this application." |
| State mismatch | "Security check failed. Please try again." |
| Invalid client | "This application is not recognized." |

## Retry Guidance

```yaml
retry_strategy:
  server_error:
    retry: true
    initial_delay: 5s
    max_delay: 60s
    backoff: exponential
    max_attempts: 3

  temporarily_unavailable:
    retry: true
    initial_delay: 30s
    max_attempts: 5

  authorization_pending:
    retry: true
    delay: interval_seconds  # From device flow response

  invalid_grant:
    retry: false
    action: "Restart authorization flow"

  invalid_client:
    retry: false
    action: "Check client credentials"

  access_denied:
    retry: false
    action: "User explicitly denied — do not retry"
```

## Troubleshooting Flowchart

```
Error received
    │
    ├── 400 Bad Request
    │   ├── invalid_request → Check params (missing/duplicate)
    │   ├── invalid_grant → Code expired? Already used? Restart flow
    │   ├── invalid_scope → Scope not authorized for client
    │   └── unsupported_grant_type → Check client config
    │
    ├── 401 Unauthorized
    │   ├── invalid_client → Check client_id/secret
    │   └── unauthorized_client → Add grant_type to client
    │
    ├── 403 Forbidden
    │   └── access_denied → User/admin denied, DCR scope not allowed
    │
    ├── 429 Too Many Requests
    │   └── Read Retry-After header, wait, retry
    │
    ├── 500 Server Error
    │   └── server_error → Retry with backoff (5s, 15s, 45s)
    │
    └── 503 Service Unavailable
        └── temporarily_unavailable → Retry after 30s
```

## Error Response Format

### Standard (RFC 6749)

```json
{
  "error": "invalid_grant",
  "error_description": "The authorization code has expired.",
  "error_uri": "https://docs.ggid.dev/errors/invalid_grant"
}
```

### OIDC (RFC 6749 + OIDC)

```json
{
  "error": "interaction_required",
  "error_description": "User must re-authenticate.",
  "state": "xyz",
  "session_state": "sess-abc"
}
```

## Monitoring

| Error Code | Expected Rate | Alert |
|-----------|--------------|-------|
| `invalid_client` | <0.5% | Spike → credential rotation or attack |
| `invalid_grant` | <1% | Spike → clock skew or code reuse bug |
| `server_error` | <0.1% | Any → investigate immediately |
| `access_denied` | <5% | Spike → possible phishing or broken consent |
| 429 rate | <2% | Spike → queue building up |

## See Also

- [OAuth State Management](oauth-state-management.md)
- [OAuth PKCE Deep Dive](oauth-pkce-deep-dive.md)
- [OAuth Client Lifecycle](oauth-client-lifecycle.md)
- [API Versioning Strategy](api-versioning-strategy.md)
