# OAuth 2.1 / FAPI 2.0 Compliance Gap Analysis

*Research date: 2026-07-14*
## Key requirements for GGID

1. **Mandatory PKCE for all authorization_code clients** — including confidential clients.
2. **Reject implicit grant** and **Resource Owner Password Credentials (ROPC)**.
3. **Exact redirect URI matching** (no path-based or wildcard looseness).
4. **Refresh token rotation** with automatic reuse detection.
5. **PAR (Pushed Authorization Requests)** should be default for high-assurance clients.
6. **JAR/JARM** for request/response integrity.
7. **Sender-constrained tokens** via DPoP or mTLS for FAPI 2.0.

## GGID current status (arch assessment)

| Requirement | Status | Notes |
|-------------|--------|-------|
| PKCE | PARTIAL | PKCE exists; not enforced for all clients |
| Implicit/ROPC | NEEDS_REVIEW | Implicit may still be present in grant_types |
| Redirect URI matching | NEEDS_REVIEW | Likely exact, but needs audit |
| Refresh token rotation | IMPLEMENTED | Exists |
| PAR | IMPLEMENTED | Exists |
| JAR/JARM | IMPLEMENTED | JAR exists; JARM may not |
| DPoP | IMPLEMENTED | Exists |

## Recommended backlog items

1. Add `oauth_2_1_strict` config flag to OAuth service.
2. Reject authorization requests without `code_challenge` when flag is enabled.
3. Remove `implicit` and `password` from allowed default grant_types in strict mode.
4. Add FAPI 2.0 compliance test suite.
5. Document migration guide for existing GGID deployments.

