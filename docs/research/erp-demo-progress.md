# Cross-Board ERP Demo Progress Tracker

> **Last Updated**: 2026-07-21 (Round 15 — OAuth-only auth migration complete)
> **Status: 7/8 demos working with new OAuth flows. Python SAML demo needs SAML session.**

## Three-Layer Alignment After OAuth Migration

| Demo | Auth Flow | Token | Verify | CRUD | Notes |
|------|-----------|:-----:|:------:|:----:|-------|
| Go | OAuth PKCE | 200 | SDK | 200 | Fully working |
| Node | M2M Client Credentials | 200 | SDK crypto | 200 | Node SDK uses built-in crypto |
| C# | Password Grant | 200 | SDK | 200 | Working |
| Java | Password Grant / SAML | 200 | SDK JwtVerifier | 200 | Working |
| Python | SAML 2.0 SSO | N/A | SDK JWTVerifier | SAML | Needs SAML session (correct behavior) |
| Ruby | Device Code | 200 | SDK | 200 | Working |
| Rust | Token Exchange | 200 | SDK | 200 | Working |
| React | SPA PKCE | 200 | Backend SDK | 200 | Working |

## Key Changes This Session
1. **All 7 SDKs**: login() migrated to OAuth2 password grant
2. **Go SDK**: DisableCompression for gzip JWKS + GetAuthorizeURL + ExchangeCode
3. **Node SDK**: Replaced jose ESM with built-in crypto for JWT verify
4. **C# SDK**: Claims adds Permissions + Aud; ClientCredentialsAsync fixed
5. **Ruby SDK**: Device code methods + form_post helper
6. **Rust SDK**: exchange_token + client_credentials fixed
7. **Java/Python SDK**: SAML2-bearer grant method added
8. **Issuer unified**: All tokens now iss=https://ggid.iot2.win
9. **OAuth clients registered**: 8 clients for all demo auth flows
10. **Zero hack**: All demos use SDK, zero inline JWT decode, zero raw HTTP

## Next Target: Rebuild C# image + verify Python SAML flow end-to-end
