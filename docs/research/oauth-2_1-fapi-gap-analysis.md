# OAuth 2.1 / FAPI 2.0 Compliance Gap — Research Update

*Research date: 2026-07-15*

## Summary

OAuth 2.1 (draft, but de facto standard) consolidates RFC 6749 + RFC 7636 (PKCE) + RFC 9700 (Security BCP) and removes dangerous legacy options. FAPI 2.0 Security Profile is final and widely adopted by financial/enterprise IAM systems.

## Mandatory changes for GGID

| Requirement | OAuth 2.0 | OAuth 2.1 / FAPI 2.0 | GGID Status |
|-------------|-----------|----------------------|-------------|
| PKCE | Optional for confidential | Mandatory for all authorization_code | PARTIAL |
| Implicit grant | Allowed | Removed | NEEDS_REVIEW |
| ROPC / password grant | Allowed | Removed | NEEDS_REVIEW |
| Redirect URI matching | Implementation-defined | Exact string comparison | NEEDS_REVIEW |
| Refresh token rotation | Optional | Mandatory for public clients | IMPLEMENTED |
| Bearer token in query | Allowed | Forbidden | NEEDS_REVIEW |
| state parameter | Recommended | Absorbed by PKCE (still for app state) | IMPLEMENTED |

## Recommended actions

1. Add `oauth_2_1_strict` config flag (default false for backward compatibility).
2. Reject authorization requests without `code_challenge` when strict mode is enabled.
3. Reject `response_type=token` (implicit) and `grant_type=password` in strict mode.
4. Enforce exact redirect URI matching in strict mode.
5. Add FAPI 2.0 client profile flag with PAR/JAR/JARM/DPoP requirements.

## Next step

Add to backlog as P2 competitive/compliance gap after HSM/KMS phase 1 is complete.