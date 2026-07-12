# OIDC Federation Guide

This guide covers OpenID Connect Federation — trust anchors, entity statements, trust chains, metadata discovery, and multi-party federation.

## Overview

OIDC Federation (spec: `openid-federation-1_0`) enables scalable trust between organizations without pre-registered bilateral integrations. Instead of manually exchanging metadata with each IdP/SP, federation members trust a common Trust Anchor.

## Core Concepts

### Trust Anchor

A Trust Anchor is a root entity that federation members trust. It signs Entity Statements for its members.

```
Trust Anchor (e.g., eduGAIN, government agency)
  ├── Trusts: University A (IdP)
  ├── Trusts: University B (IdP)
  └── Trusts: Publisher C (SP)
```

### Entity Statement

A JWT signed by the Trust Anchor describing an entity:

```json
{
  "iss": "https://trust-anchor.example.org",
  "sub": "https://idp.university-a.edu",
  "iat": 1706104200,
  "exp": 1706190600,
  "jwks": {"keys": [...]},
  "metadata": {
    "openid_provider": {
      "issuer": "https://idp.university-a.edu",
      "authorization_endpoint": "https://idp.university-a.edu/authorize",
      "jwks_uri": "https://idp.university-a.edu/jwks.json"
    }
  },
  "constraints": {
    "scopes_supported": ["openid", "email", "profile"]
  }
}
```

### Trust Chain

To trust an SP, the IdP builds a trust chain:

```
Entity (SP) → Subordinate Statement (signed by SP)
  → Intermediate Statement (signed by federation operator)
    → Trust Anchor Statement (signed by Trust Anchor)
```

Verification: Validate each signature up the chain to the Trust Anchor.

## Metadata Discovery

### Federation Endpoint

```
GET https://trust-anchor.example.org/entities/{entity_id}
```

Returns the Entity Statement for the given entity.

### Trust Mark

Trust Marks are badges issued by authorities:

```
GET https://trust-anchor.example.org/trust-mark/{entity_id}
```

| Trust Mark | Meaning |
|-----------|---------|
| `https://refeds.org/sirtfi` | Incident response certified |
| `https://eduGAIN.org/member` | eduGAIN member |

## Multi-Party Federation

```
Federation A (Trust Anchor A)
  ↕ Bilateral trust
Federation B (Trust Anchor B)
```

Cross-federation trust via Trust Anchor delegation. Entity in Federation A can access SP in Federation B.

## GGID Federation Support

### As Federation Member

```yaml
oidc_federation:
  trust_anchor: "https://trust-anchor.example.org"
  trust_anchor_jwks: "https://trust-anchor.example.org/jwks.json"
  entity_id: "https://ggid.example.com"
  organization_name: "Acme Corp"
  contacts: ["security@acme.com"]
```

### Automatic Metadata Discovery

GGID automatically discovers and validates federation members:

1. Fetch Entity Statement from Trust Anchor
2. Validate Trust Anchor signature
3. Extract OIDC metadata
4. Cache metadata (TTL: 1 hour)
5. Use discovered metadata for OIDC flow

## Federation vs Bilateral

| Aspect | Bilateral (current) | Federation |
|--------|-------------------|------------|
| Setup | Manual per partner | Automatic via Trust Anchor |
| Scaling | O(n²) | O(n) |
| Trust | Direct | Transitive via anchor |
| Revocation | Per partner | Via anchor |
| Use case | Enterprise SSO | Research/education/gov |

## See Also

- [OAuth API](../api/oauth.md)
- [Authentication Flows](authentication-flows.md)
- [SAML Federation Guide](saml-federation-guide.md)
