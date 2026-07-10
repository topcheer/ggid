# ADR-003: JWT Over Server-Side Sessions

- **Status:** Accepted
- **Date:** 2024-01-25

## Context

GGID needs an authentication mechanism for API consumers. The two primary
options are:

1. **Server-side sessions** — store session state in Redis, issue session ID
in a cookie. Every request requires a Redis lookup.
2. **JWT (JSON Web Tokens)** — issue a signed token containing claims. The
token is stateless and verified by the public key alone.

### Forces

- Services are stateless and horizontally scalable
- The Gateway must verify tokens without calling the auth service on every request
- Multiple services need to extract user identity (tenant, roles, scopes)
- Mobile apps and SPAs are primary clients (cookie-based sessions have CSRF risks)
- Token revocation is needed (logout, password change)
- Refresh token rotation is a security best practice

## Decision

We chose **JWT (RS256) with refresh tokens and JWKS endpoint**.

### Design

- **Signing**: RSA 2048-bit keys (RS256). Private key in auth service,
public key distributed via `/.well-known/jwks.json`.
- **Access token**: 1-hour TTL. Contains `sub`, `tenant_id`, `roles`, `scopes`, `exp`.
- **Refresh token**: 30-day TTL. Rotated on each use.
- **Verification**: Gateway fetches the public key at startup and verifies
JWTs locally — no per-request call to auth service.
- **Revocation**: Access tokens are short-lived (1h). For immediate revocation,
the auth service maintains a token blocklist in Redis (checked at refresh time).
- **JWKS**: Public key rotated by replacing the RSA key pair and updating JWKS.
Clients cache keys with a configurable TTL.

## Consequences

### Positive

- Stateless verification: Gateway verifies JWTs without calling auth service
- Horizontal scaling: no shared session store needed for token verification
- Standard OIDC/OAuth2 token format — works with existing JWT libraries
- Claims are self-contained (user ID, tenant, roles) — no extra DB lookup
- Works natively with mobile/SPA clients (no cookie/CSRF complexity)

### Negative

- Immediate token revocation is approximate (access tokens valid until expiry)
- Key rotation requires coordination between auth service and gateway
- JWT payload size grows with more claims (roles, scopes)
- No built-in server-side session metadata (IP, device info) — needs separate store

### Neutral

- JWKS caching TTL defaults to 15 minutes (configurable via SDK)
- RSA 2048 is the default; larger keys (4096) can be configured for higher security
