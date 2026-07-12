# Identity Federation

SAML federation, OIDC federation, cross-domain SSO, metadata exchange, discovery, trust framework interoperability, eduGAIN case study, and multi-IdP routing.

## Federation Models

| Model | Trust | Scale | Example |
|-------|-------|-------|---------|
| Bilateral | Direct SP↔IdP | 2 parties | B2B partnership |
| Hub-and-spoke | Central IdP | Single org | Corporate SSO |
| Multi-party | Trust anchor | 100s of orgs | eduGAIN, InCommon |
| Brokered | Intermediary routes | Multi-source | GGID broker |

## SAML Federation

### Trust Establishment

```
1. SP registers metadata with federation operator
2. IdP registers metadata with federation operator
3. Federation publishes aggregate metadata
4. SP/IdP consume aggregate → trust all federation members
```

### Metadata Exchange

```xml
<!-- SP metadata -->
<EntityDescriptor entityID="https://sp.ggid.dev/saml">
  <SPSSODescriptor>
    <KeyDescriptor use="signing">...</KeyDescriptor>
    <AssertionConsumerService Location="https://sp.ggid.dev/saml/acs"/>
  </SPSSODescriptor>
</EntityDescriptor>
```

### SAML SSO Flow

```
User → SP → Redirect to IdP with AuthnRequest
     IdP → Authenticate user
     IdP → POST SAML assertion to SP ACS
     SP → Verify signature, extract attributes
     SP → Create session
```

## OIDC Federation

### Entity Statements

```json
{
  "iss": "https://federation.ggid.dev",
  "sub": "https://idp.partner.com",
  "iat": 1700000000,
  "exp": 1700086400,
  "jwks": {"keys": [...]},
  "metadata": {
    "openid_provider": {
      "issuer": "https://idp.partner.com",
      "authorization_endpoint": "...",
      "token_endpoint": "..."
    }
  },
  "authority_hints": ["https://federation.ggid.dev"]
}
```

### Trust Chain Resolution

```
1. Client discovers IdP entity configuration
2. Fetches entity statement from federation operator
3. Verifies trust chain: leaf → intermediate → trust anchor
4. If valid → trust IdP for authentication
```

## Cross-Domain SSO

### SAML Cross-Domain

```
User in Org A → Access app in Org B
  → Org B SP redirects to Org A IdP (federated trust)
  → Org A IdP authenticates + issues assertion
  → Org B SP verifies via federation metadata
  → User logged in at Org B
```

### OIDC Cross-Domain

```
User → App (RP) → Redirect to home IdP
  → IdP authenticates → ID token
  → RP verifies via federation trust chain
  → User logged in
```

## Discovery Patterns

### WebFinger

```bash
GET https://ggid.dev/.well-known/webfinger?resource=acct:jane@corp.com
# → {
#   "subject": "acct:jane@corp.com",
#   "links": [{"rel": "http://openid.net/specs/connect/1.0/issuer",
#              "href": "https://idp.corp.com"}]
# }
```

### Email-Based

```bash
GET /api/v1/auth/discover?email=jane@corp.com
# → {"providers": [{"type":"saml","name":"Corp SSO","redirect":"..."}]}
```

### DNS-Based

```bash
# DNS TXT record
_corp.com.openid._finger TXT "https://idp.corp.com/.well-known/openid-configuration"
```

## Trust Framework Interoperability

| Framework | Region | Protocol |
|-----------|--------|----------|
| eduGAIN | Global (academic) | SAML |
| InCommon | US (academic) | SAML |
| eIDAS | EU (government) | SAML/OIDC |
| PIV/CAC | US (federal) | SAML/PKI |
| eHerkenning | Netherlands | SAML |

### eIDAS Integration

```yaml
eidas:
  enabled: true
  node_url: "https://eidas-node.gov.eu"
  trust_anchor: "EU eIDAS root CA"
  minimum_loa: "substantial"  # low/substantial/high
```

## eduGAIN Case Study

```
University A (IdP) ←─eduGAIN─→ University B (SP)
    │                              │
    └── Registers metadata         └── Consumes aggregate
        with eduGAIN                  + filters by entity category

Student from University A:
  → Accesses University B's library
  → Redirected to University A IdP
  → Authenticates with institutional credentials
  → SAML assertion sent to University B
  → Library access granted
  → No separate account at University B
```

### Entity Categories (Trust Filtering)

| Category | Meaning |
|----------|---------|
| `research-and-scholarship` | R&S profile compliant |
| `code-of-conduct` | GDPR CoC compliant |
| `sirtfi` | Security incident response capable |

## Multi-IdP Routing

```yaml
routing:
  rules:
    - domain: "@corp.com"
      idp: "saml-corporate"
    - domain: "@partner.com"
      idp: "oidc-partner"
    - domain: "@university.edu"
      idp: "edugain-federated"
    - default
      idp: "local"
```

### Login Flow with Routing

```
User enters email
  → Router evaluates domain/regex
  → @corp.com → SAML IdP redirect
  → @partner.com → OIDC IdP redirect
  → @university.edu → eduGAIN discovery
  → Other → Local password
```

## Monitoring

| Metric | Alert |
|--------|-------|
| Federation metadata refresh failure | Any → stale config |
| SSO success rate per IdP | <95% → cert or config |
| SLO propagation failures | >2% → sessions not terminated |
| Trust chain verification failures | Spike → federation issue |
| New federation members | Log for review |

## See Also

- [Identity Federation Patterns](identity-federation-patterns.md)
- [Identity Provider Configuration](identity-provider-configuration.md)
- [SAML SP Implementation](saml-sp-implementation.md)
- [SAML Metadata Management](saml-metadata-management.md)
- [OIDC Backchannel Logout](oidc-backchannel-logout.md)