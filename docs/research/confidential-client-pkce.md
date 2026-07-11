# Confidential Clients & PKCE Enforcement

> OAuth 2.1 PKCE requirements, public vs confidential clients, GGID PKCE status.

---

## Client Types

| Type | Has Secret | Example | PKCE |
|------|-----------|---------|------|
| **Public** | No | SPA, mobile app | Required |
| **Confidential** | Yes | Server-side app | Recommended |

OAuth 2.1 mandates PKCE for **all** authorization code flows — even confidential clients.

---

## PKCE Flow

```
1. Client generates random code_verifier (43-128 chars)
2. Client computes code_challenge = BASE64URL(SHA256(code_verifier))
3. Client sends code_challenge in authorize request
4. Server stores code_challenge
5. Client sends code_verifier in token request
6. Server verifies SHA256(code_verifier) == stored code_challenge
```

This prevents authorization code interception attacks.

---

## GGID PKCE Implementation

### Status

| Feature | Status |
|---------|--------|
| PKCE in authorization code flow | Done |
| PKCE required for public clients | Done |
| PKCE enforced for all clients | Done (OAuth 2.1) |
| S256 challenge method | Done |
| Plain challenge method | Rejected (insecure) |

### Verification

```bash
# Authorization with PKCE
curl "http://localhost:8080/oauth/authorize?response_type=code&client_id=app1&code_challenge=...&code_challenge_method=S256&redirect_uri=..."
```

```bash
# Token exchange with verifier
curl -X POST http://localhost:8080/oauth/token \
  -d '{
    "grant_type": "authorization_code",
    "code": "auth_code_here",
    "code_verifier": "original_verifier_here",
    "client_id": "app1"
  }'
```

---

## Confidential Client Authentication

### Client Secret (Basic)

```bash
curl -u client_id:client_secret -X POST http://localhost:8080/oauth/token ...
```

### Client Secret (POST body)

```bash
curl -X POST http://localhost:8080/oauth/token \
  -d 'client_id=app&client_secret=secret&...'
```

### mTLS (RFC 8705)

```bash
curl --cert client.crt --key client.key -X POST https://ggid/oauth/token
```

---

## Compliance Checklist

- [ ] All clients use PKCE (public + confidential)
- [ ] S256 only (no `plain`)
- [ ] Confidential clients authenticated via secret or mTLS
- [ ] Public clients have no secret
- [ ] Redirect URIs exact-match (no wildcards)

---

*See: [OAuth Scopes Design](oauth-scopes-design.md) | [Token Binding & DPoP](token-binding-dpop.md) | [Security Audit](security-audit-checklist.md)*

*Last updated: 2025-07-11*
