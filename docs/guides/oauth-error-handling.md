# OAuth Error Handling Guide

## Standard Error Codes

GGID follows RFC 6749 error response format:

```json
{
  "error": "invalid_grant",
  "error_description": "The authorization code has expired",
  "error_uri": "https://docs.ggid.dev/errors/oauth#invalid_grant"
}
```

## Error Reference

| Error | HTTP | Cause | Client Action |
|-------|------|-------|--------------|
| `invalid_request` | 400 | Missing/invalid parameter | Fix request, retry |
| `invalid_client` | 401 | Bad client credentials | Re-authenticate |
| `invalid_grant` | 400 | Expired/used code or token | Re-authorize user |
| `unauthorized_client` | 400 | Client not authorized for grant | Contact admin |
| `unsupported_grant_type` | 400 | Grant not supported | Use valid grant |
| `invalid_scope` | 400 | Unknown/excessive scope | Request valid scope |
| `access_denied` | 403 | User/resource owner denied | Inform user |
| `server_error` | 500 | Internal failure | Retry with backoff |
| `temporarily_unavailable` | 503 | Service overload | Retry later |

## OIDC Additional Errors

| Error | Cause |
|-------|-------|
| `interaction_required` | Need user interaction (consent) |
| `login_required` | User not authenticated |
| `account_selection_required` | Multiple accounts, user must select |
| `consent_required` | User consent needed |
| `invalid_request_uri` | Back-channel invalid request_uri |

## PKCE Error Handling

```
If code_verifier mismatch → invalid_grant
If code_verifier wrong format → invalid_request
If S256 hash mismatch → invalid_grant
```

## Retry Strategy

```go
func doTokenExchange(code string) error {
    for attempt := 0; attempt < 3; attempt++ {
        resp, err := exchange(code)
        if err == nil { return nil }

        switch resp.Error {
        case "server_error", "temporarily_unavailable":
            time.Sleep(backoff(attempt)) // 1s, 2s, 4s
            continue
        default:
            return err // Non-retryable
        }
    }
    return errors.New("max retries exceeded")
}
```

## Logging

- Log `error` and `error_description` for all OAuth failures
- Do NOT log tokens, codes, or secrets
- Include `client_id`, `redirect_uri`, `grant_type` for debugging

## See Also

- [OAuth API](../api/oauth.md)
- OAuth Scope Design
