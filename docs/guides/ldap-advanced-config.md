# LDAP Advanced Configuration Guide

Advanced LDAP configuration for GGID — complex directory trees, group filters, LDAPS/StartTLS, connection pooling, sync tuning, multi-domain trust.

## Environment Variables

```yaml
LDAP_URL: ldaps://ldap.example.com:636
LDAP_BIND_DN: cn=ggid-service,ou=services,dc=example,dc=com
LDAP_BIND_PASSWORD: ${LDAP_PASSWORD}
LDAP_BASE_DN: dc=example,dc=com
LDAP_USER_FILTER: (objectClass=person)
LDAP_USER_ID_ATTR: uid
LDAP_START_TLS: false  # Use ldaps:// instead
LDAP_AUTO_PROVISION: true  # Auto-create GGID user on first LDAP login
```
## LDAPS vs StartTLS

| Method | URL | Port | Notes |
|--------|-----|------|-------|
| LDAPS | `ldaps://` | 636 | TLS from connection start |
| StartTLS | `ldap://` + START_TLS | 389 | Upgrade to TLS after connect |

**Recommendation**: Use LDAPS (simpler, no plaintext window).

## Complex Directory Trees

```
dc=example,dc=com
├── ou=engineering
│   ├── ou=backend
│   │   ├── cn=alice (uid=alice)
│   └── ou=frontend
│       └── cn=bob (uid=bob)
├── ou=sales
│   └── cn=carol (uid=carol)
└── ou=services
    └── cn=ggid-service
```

### Multiple Base DNs

For complex trees, use multiple search bases:

```yaml
LDAP_BASE_DN: dc=example,dc=com
LDAP_SEARCH_SCOPES: sub  # sub | one | base
```

## Group Filter Optimization

```yaml
# Fast: Filter by specific group membership
LDAP_USER_FILTER: (&(objectClass=person)(memberOf=cn=ggid-users,ou=groups,dc=example,dc=com))

# Slower: All persons (filter in application)
LDAP_USER_FILTER: (objectClass=person)
```

## Connection Pooling

```yaml
LDAP_POOL_SIZE: 10        # Max connections
LDAP_POOL_MIN: 2          # Min idle connections
LDAP_TIMEOUT: 10s         # Search timeout
LDAP_CONN_TIMEOUT: 5s     # Connection timeout
```

## Multi-Domain Trust

For organizations with multiple LDAP domains:

```yaml
# Configure authprovider chain with multiple LDAP providers
LDAP_PROVIDERS:
  - name: corporate
    url: ldaps://ldap.corp.example.com:636
    base_dn: dc=corp,dc=example,dc=com
  - name: subsidiary
    url: ldaps://ldap.sub.example.com:636
    base_dn: dc=sub,dc=example,dc=com
```

GGID tries each provider in order until authentication succeeds.

## Troubleshooting

| Issue | Cause | Fix |
|-------|-------|-----|
| Connection timeout | Firewall / wrong port | Check port 636 reachable |
| Bind failed | Wrong DN or password | Verify bind DN + password |
| User not found | Wrong base DN or filter | Test with ldapsearch |
| TLS error | CA not trusted | Add CA cert to trust store |
| Slow searches | No index on filter attr | Add index on uid/memberOf |

```bash
# Test LDAP connection
ldapsearch -H ldaps://ldap.example.com:636 \
  -D "cn=ggid-service,ou=services,dc=example,dc=com" \
  -W -b "dc=example,dc=com" \
  "(uid=alice)"
```

## See Also

- LDAP Provider
- [Auth API](../api/auth.md)
- [Authentication Flows](authentication-flows.md)
