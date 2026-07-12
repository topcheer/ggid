# SAML SP-Initiated vs IdP-Initiated SSO

Comparison of SP-initiated and IdP-initiated SAML SSO flows, with security considerations and best practices.

> **Related**: [SAML Federation Guide](saml-federation-guide.md)

## Flow Comparison

### SP-Initiated (Recommended)

```
User → SP (GGID) → Redirect to IdP → User authenticates → SAML Response → GGID → JWT
```

**Pros**: Most secure, standard flow, RelayState protects against CSRF.

### IdP-Initiated

```
User → IdP (Okta) → Clicks GGID app → Unsolicited SAML Response → GGID → JWT
```

**Pros**: Better UX for users starting at IdP portal.

**Cons**: Vulnerable to login CSRF (attacker injects their SAML response into victim's session).

## Security Considerations

| Aspect | SP-Initiated | IdP-Initiated |
|--------|-------------|---------------|
| CSRF risk | Low (RelayState) | Higher (no RelayState) |
| Replay risk | Low (InResponseTo) | Higher |
| User tracking | Full flow visible | Response appears unsolicited |
| GGID mitigation | Standard | Validate NotOnOrAfter + freshness |

## Relay State

SP-initiated SSO includes a RelayState parameter that carries the originally requested URL:

```
GET /saml/login?redirect_to=/dashboard
  → AuthnRequest includes RelayState=/dashboard
  → IdP returns RelayState in response
  → GGID redirects user to /dashboard
```

## IdP Discovery

For multi-IdP setups, GGID can discover the correct IdP:

1. **Email domain**: `@company.com` → Okta
2. **DNS**: `company.ggid.example.com` → configured IdP
3. **Cookie**: Remember user's IdP choice
4. **Manual**: Let user select from IdP list

## Error Handling

| Error | Cause | User Action |
|-------|-------|-------------|
| Invalid signature | Wrong IdP cert | Contact admin |
| Conditions not met | Clock skew | Check NTP |
| Audience mismatch | Entity ID wrong | Verify SP entity ID |
| Response expired | NotOnOrAfter passed | Retry login |

## Best Practices

- [ ] Prefer SP-initiated when possible
- [ ] Validate InResponseTo for IdP-initiated
- [ ] Enforce NotBefore/NotOnOrAfter conditions
- [ ] Clock skew tolerance <= 60 seconds
- [ ] Audience restriction validated
- [ ] RelayState sanitized (no open redirects)

## See Also

- [SAML Federation Guide](saml-federation-guide.md)
- [Authentication Flows](authentication-flows.md)
- [Per-Tenant IdP](per-tenant-idp.md)
