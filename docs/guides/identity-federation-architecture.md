# Identity Federation Architecture

Guide for SAML/OIDC federation, cross-domain trust, and metadata exchange in GGID.

## Overview

Federation enables users from one identity domain to access services in another domain without duplicate accounts. GGID supports both SAML and OIDC federation as both Identity Provider (IdP) and Service Provider (SP/Relying Party).

## Federation Topologies

### Hub-and-Spoke (Central)

```
          ┌──────────┐
          │   GGID   │ ← Central IdP
          │   IdP    │
          └────┬─────┘
     ┌─────────┼─────────┐
     ▼         ▼         ▼
  App A     App B     App C
  (SP)      (RP)      (SP)
```

GGID is the sole identity authority. All apps trust GGID.

### Cross-Domain Federation

```
  Corp A IdP ←──federation trust──→ Corp B IdP
     │                                  │
     ▼                                  ▼
  Corp A users                     Corp B users
  access Corp B apps               access Corp A apps
```

### OIDC Federation (RFC 8416 / OpenID Federation 1.0)

```
        Federation Authority (Trust Anchor)
              │
    ┌─────────┼─────────┐
    ▼         ▼         ▼
  Org A     Org B     GGID
  (OP)      (RP)      (OP+RP)
```

Trust is established via federation entity statements signed by a trust anchor.

## SAML Federation

### Metadata Exchange

```bash
# GGID publishes SP metadata
GET /saml/metadata.xml
# → EntityDescriptor with ACS URL, signing cert, NameID format

# GGID consumes IdP metadata
POST /api/v1/identity/federation/saml/import
{"metadata_url": "https://idp.partner.com/metadata.xml"}
# → Parses EntityID, SSO URL, certificates
```

### Trust Configuration

| Parameter | Value |
|-----------|-------|
| Entity ID | `https://auth.ggid.dev/saml` |
| ACS URL | `https://auth.ggid.dev/saml/acs` |
| SLO URL | `https://auth.ggid.dev/saml/slo` |
| NameID Format | `urn:oasis:names:tc:SAML:2.0:nameid-format:transient` |
| Signing | RSA-SHA256 |
| Encryption | AES-256-GCM (_assertion) |

### Attribute Release Policy

```yaml
attribute_release:
  - sp_entity_id: "https://app.partner.com"
    attributes:
      - "email"
      - "display_name"
      - "department"
    deny:
      - "ssn"
      - "salary"
  - sp_entity_id: "*"
    attributes:
      - "email"
      - "display_name"
```

Attributes are released per-SP — no blanket attribute dump.

## OIDC Federation

### Discovery

```bash
# GGID OIDC discovery
GET /.well-known/openid-configuration
# → {issuer, authorization_endpoint, token_endpoint, jwks_uri, ...}

# Partner RP uses discovery to configure automatically
```

### Client Registration (RFC 7591)

```bash
# Partner registers as RP
POST /api/v1/oauth/register
{
  "client_name": "Partner App",
  "redirect_uris": ["https://partner.com/callback"],
  "grant_types": ["authorization_code"],
  "response_types": ["code id_token"]
}
```

### UserInfo Claims

```bash
# Partner requests specific claims
GET /api/v1/oauth/userinfo
# → {"sub":"uuid","email":"user@corp.com","name":"Jane"}
```

Claims controlled by consent screen and scope grants.

## Cross-Domain Trust Establishment

### Step-by-Step

1. **Exchange metadata**: Each party publishes federation metadata
2. **Validate certificates**: Verify signing certs via trusted CA or fingerprint
3. **Configure attribute mapping**: Map source attributes to destination schema
4. **Test SSO**: Initiate login flow in both directions
5. **Enable SLO**: Configure single logout propagation
6. **Monitor**: Log all federated sign-ins for audit

### Trust Model Comparison

| Model | Trust Basis | Use Case |
|-------|------------|----------|
| Direct (pairwise) | Explicit per-partner config | B2B with known partners |
| Federation authority | Signed entity statements | Large-scale multi-org |
| PKI / CA based | Certificate chain | Government, regulated |
| Web of trust | Transitive trust | Academic (eduGAIN) |

## Attribute Mapping

Source IdP attributes → GGID canonical attributes:

```yaml
attribute_mappings:
  # SAML attribute → GGID field
  "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress": "email"
  "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name": "display_name"
  "urn:oid:2.5.4.11": "department"
  
  # OIDC claim → GGID field
  "preferred_username": "username"
  "given_name": "first_name"
  "family_name": "last_name"
```

## Privacy Considerations

| Principle | Implementation |
|-----------|---------------|
| Data minimization | Release only attributes SP needs |
| Pairwise identifiers | Different `sub` per SP (prevent correlation) |
| Consent | User sees what attributes are released |
| Pseudonymity | Allow federated login without real identity |
| Data retention | Federation logs expire after 90 days |
| Right to revoke | User can revoke SP access anytime |

### Pairwise Subject Claim

```go
func pairwiseSub(userID, sectorIdentifier string) string {
    h := hmac.New(sha256.New, pairwiseSecret)
    h.Write([]byte(userID + sectorIdentifier))
    return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}
// Each SP sees a different sub for the same user
```

## SLO (Single Logout)

```
User logs out of GGID
    │
    ▼
GGID sends SAML LogoutRequest to all active SPs
    │
    ▼
Each SP destroys local session
    │
    ▼
SP sends LogoutResponse to GGID
    │
    ▼
GGID confirms logout complete
```

OIDC uses back-channel logout tokens (RFC) or front-channel iframe logout.

## Monitoring

| Metric | Alert |
|--------|-------|
| Federation sign-in failures | >5% → cert expiry or config drift |
| Metadata refresh failures | Any → stale config risk |
| New SP attribute requests | Unexpected → possible data exfiltration |
| SLO propagation failures | >2% → sessions not terminated |
| Cross-domain impossible travel | Federation login from different geo than source |

## See Also

- [OAuth Client Lifecycle](oauth-client-lifecycle.md)
- [SAML Federation](../research/saml-federation.md)
- [Identity Lifecycle Automation](identity-lifecycle-automation.md)
- [Session Security](session-security.md)
