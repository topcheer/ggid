# Identity Federation Patterns

Hub-and-spoke, bilateral, multi-party federation, discovery patterns, trust lifecycle management, attribute mapping, and SLO propagation.

## Federation Patterns

### 1. Hub-and-Spoke (Central IdP)

```
              ┌──────────┐
              │   GGID    │ ← Central IdP
              │   IdP     │
              └────┬─────┘
         ┌────────┼────────┐
         ▼        ▼        ▼
       App A    App B    App C
       (SP)     (RP)     (SP)
```

| Aspect | Detail |
|--------|--------|
| Trust model | All apps trust GGID only |
| User store | Central (GGID) |
| Onboarding | New app registers with GGID |
| Best for | Single-org, greenfield |

### 2. Bilateral (Pairwise Trust)

```
  Org A IdP ←──trust──→ Org B IdP
       │                     │
       ▼                     ▼
   A users → B apps     B users → A apps
```

| Aspect | Detail |
|--------|--------|
| Trust model | Direct agreement between 2 parties |
| Metadata exchange | Manual or URL-based |
| Attribute mapping | Per-partner config |
| Best for | B2B partnerships |

### 3. Multi-Party Federation

```
        Federation Authority (Trust Anchor)
                    │
        ┌───────────┼───────────┐
        ▼           ▼           ▼
      Org A       Org B       GGID
      (IdP+SP)   (IdP+SP)   (IdP+SP)
```

| Aspect | Detail |
|--------|--------|
| Trust model | Transitive via trust anchor |
| Scale | 100s of organizations |
| Discovery | Federation metadata (RFC 8416) |
| Best for | Academic (eduGAIN), government, consortia |

### 4. Brokered Federation

```
  User → Broker (GGID) → Routes to appropriate IdP
                         ├── Corporate SAML
                         ├── Social Google
                         ├── Partner OIDC
                         └── Local password
```

| Aspect | Detail |
|--------|--------|
| Trust model | Broker is trusted intermediary |
| User choice | User picks IdP at login |
| Best for | Multi-source identity aggregation |

## Discovery Patterns

### Email-Based Discovery

```bash
GET /api/v1/auth/discover?email=user@corp.com
# → {
#   "providers": [
#     {"type": "saml", "name": "Corp SSO", "redirect": "https://idp.corp.com/sso"},
#     {"type": "local", "name": "Password"}
#   ]
# }
```

### Domain-Based Routing

```yaml
discovery:
  rules:
    - domain: "@corp.com"
      idp: "saml-corporate"
    - domain: "@partner.com"
      idp: "oidc-partner"
    - domain: "@gmail.com"
      idp: "social-google"
    - default
      idp: "local"
```

### Client-Specific Discovery

```bash
# Only show IdPs this OAuth client supports
GET /api/v1/auth/discover?client_id=app-123
# → Only authorized IdPs for this client
```

### OIDC Issuer Discovery

```bash
# Standard .well-known discovery
GET https://partner.com/.well-known/openid-configuration
# → Auto-configure endpoints, JWKS, scopes
```

## Trust Lifecycle Management

### Trust Establishment

```
1. Exchange metadata (SAML) or discovery doc (OIDC)
2. Verify certificates (fingerprint or CA chain)
3. Configure attribute mapping
4. Test SSO in staging
5. Enable in production
6. Document in trust registry
```

### Trust Monitoring

```bash
# Health check all federation partners
GET /api/v1/identity/federation/health
# → [
#   {"partner": "Corp A", "status": "healthy", "last_sso": "2m ago"},
#   {"partner": "Corp B", "status": "degraded", "issue": "cert_expiring_5d"},
#   {"partner": "Google", "status": "healthy", "last_sso": "30s ago"}
# ]
```

### Trust Renewal

```yaml
renewal:
  certificates:
    check_interval: "daily"
    alert_before_expiry: "30d"
    auto_renew: false  # Manual for federation trust

  metadata:
    refresh_interval: "hourly"
    alert_on_change: true
```

### Trust Revocation

```bash
# Revoke trust with a partner
POST /api/v1/identity/federation/{id}/revoke
{
  "reason": "Security incident",
  "effective": "immediate",
  "notify_users": true
}
# → All active SSO sessions from this IdP terminated
```

## Attribute Mapping

### Per-Partner Mapping

```yaml
attribute_mappings:
  "saml-corp-a":
    # SAML attribute URN → GGID field
    "http://...emailaddress": "email"
    "http://...name": "display_name"
    "urn:oid:2.5.4.11": "department"

  "oidc-partner-b":
    # OIDC claim → GGID field
    "email": "email"
    "name": "display_name"
    "groups": "groups"

  "social-google":
    "email": "email"
    "name": "display_name"
    "picture": "avatar_url"
```

### Transformation Rules

```yaml
transformations:
  - field: "department"
    rules:
      - if: "value == 'R&D'"
        set: "Engineering"    # Normalize naming
      - if: "value == ''"
        set: "Unassigned"     # Default
```

## SLO Propagation

### SAML SLO

```
User logs out of GGID
  → GGID sends LogoutRequest to all active SPs
  → Each SP destroys local session
  → SP sends LogoutResponse
  → GGID completes logout
```

### OIDC Backchannel Logout

```
User logs out of GGID
  → GGID sends logout_token to each RP's backchannel_logout_uri
  → Each RP terminates session by sid
  → No browser redirect needed
```

### Cross-Protocol SLO

```
User federated via SAML to App A, via OIDC to App B
  → GGID sends SAML LogoutRequest to App A
  → GGID sends OIDC backchannel logout to App B
  → Both sessions terminated
```

## Monitoring

| Metric | Alert |
|--------|-------|
| SSO success rate per IdP | <95% → cert or config issue |
| SLO propagation failures | >2% → sessions not terminated |
| Metadata refresh failures | Any → stale config risk |
| New federation partners | Log for review |
| Cert expiry | <30 days → renew |

## See Also

- [Identity Federation Architecture](identity-federation-architecture.md)
- [Identity Provider Configuration](identity-provider-configuration.md)
- [OIDC Backchannel Logout](oidc-backchannel-logout.md)
- [Consent Management Design](consent-management-design.md)
