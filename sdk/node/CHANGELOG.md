# Changelog

## [1.0.0] - 2025-07-24

### Added
- OAuth2 password grant login with client_id + tenant support
- JWT verification via JWKS + RS256 (OIDC discovery auto-config)
- Client credentials grant (M2M, RFC 6749 §4.4)
- Token refresh (offline_access scope)
- Token introspection (RFC 7662)
- Authorization code + PKCE exchange
- SAML2 bearer token exchange (RFC 7522)
- Agent token exchange (RFC 8693)
- Device authorization flow (RFC 8628)
- User/role/org management API
- RBAC permission checking
- ABAC policy evaluation

### Security
- RS256 signature verification with JWKS key rotation
- Tenant isolation enforcement (app-level + gateway)
- No hardcoded secrets or inline JWT decoding
