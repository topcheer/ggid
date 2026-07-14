# ADR-0001: JWT RSA Shared Key Between Auth and Gateway

**Status:** Accepted
**Date:** 2025-Q1
**Deciders:** Architecture Team

---

## Context

GGID's Gateway service verifies JWTs issued by the Auth service. Both services need the same signing key to operate:

- **Auth service** signs JWTs with the key
- **Gateway** verifies JWT signatures with the same key
- **OAuth service** also signs some token types (client credentials, device flow)

### Requirements

1. All three services (gateway, auth, oauth) must share the same JWT secret
2. Secret must not be committed to source code
3. Secret must survive service restarts
4. Secret rotation should be possible without downtime (planned)
5. Works across Docker Compose, Kubernetes, and bare metal

---

## Decision

Use a **shared HMAC-SHA256 secret** distributed via environment variables.

### Current Approach

```bash
# Each service reads JWT_SECRET from environment
JWT_SECRET=your-secret-here
```

- Auth: uses `JWT_SECRET` to sign access tokens
- Gateway: uses `JWT_SECRET` to verify token signatures
- OAuth: uses `JWT_SECRET` for client credentials and token exchange

### Safety Mechanism

The auth service calls `log.Fatal` if `JWT_SECRET` is empty — no silent bypass:

```go
if jwtSecret == "" {
    log.Fatal("JWT_SECRET must be set")
}
```

---

## Alternatives Considered

### Option A: RSA Key Pair (Asymmetric)

Auth holds private key, gateway holds public key.

**Pros:** Gateway never has signing key, JWKS rotation
**Cons:** Higher CPU cost, more complex key management

**Status:** Planned for future (Phase 2)

### Option B: JWKS Endpoint

Gateway fetches public keys from `/.well-known/jwks.json`.

**Pros:** Industry standard, automatic key rotation
**Cons:** Adds network dependency, bootstrap problem (gateway needs auth to be up)

**Status:** Planned (partially implemented — `JWKS_REFRESH_INTERVAL` exists in config)

---

## Per-Deployment Strategy

### Docker Compose

```yaml
# docker-compose.yaml
services:
  gateway:
    environment:
      JWT_SECRET: "${JWT_SECRET}"
  auth:
    environment:
      JWT_SECRET: "${JWT_SECRET}"
  oauth:
    environment:
      JWT_SECRET: "${JWT_SECRET}"
```

Secret stored in `.env` file (gitignored):
```bash
# .env
JWT_SECRET=$(openssl rand -base64 32)
```

### Kubernetes

```yaml
# K8s Secret
apiVersion: v1
kind: Secret
metadata:
  name: ggid-jwt
  namespace: ggid
type: Opaque
data:
  JWT_SECRET: <base64-encoded-secret>
```

```yaml
# Deployment env reference
env:
  - name: JWT_SECRET
    valueFrom:
      secretKeyRef:
        name: ggid-jwt
        key: JWT_SECRET
```

Or via Helm:
```yaml
# values.yaml
gateway:
  env:
    JWT_SECRET: "your-secret"
auth:
  env:
    JWT_SECRET: "your-secret"
```

### Bare Metal

```bash
# /opt/ggid/config/ggid.env
JWT_SECRET=$(openssl rand -base64 32)
```

systemd reads this via `EnvironmentFile=/opt/ggid/config/ggid.env`.

---

## Consequences

### Positive

- Simple: one secret, one env var
- Fast: HMAC-SHA256 is 10x faster than RSA verification
- Works everywhere: env vars are universal
- Safe: `log.Fatal` prevents running without a secret

### Negative

- All services have signing capability (not just auth)
- Secret rotation requires restarting all services simultaneously
- No cryptographic separation between signer and verifier

### Future: JWKS Migration Path

1. Auth generates RSA key pair
2. Publishes public key at `/.well-known/jwks.json`
3. Gateway fetches and caches public keys
4. Gradual migration: support both HMAC and RSA during transition
5. Remove HMAC after all tokens have rotated

---

*Last updated: 2025-07-11*
