# LDAP Directory Integration Guide

Connection pooling, search optimization, group membership resolution, nested groups, LDAPS/StartTLS, multi-directory federation, and sync tuning.

## Connection Management

### Connection Pooling

```go
type LDAPPool struct {
    conns    chan *ldap.Conn
    url      string
    bindDN   string
    bindPass string
    maxSize  int
}

func NewLDAPPool(url, bindDN, bindPass string, size int) *LDAPPool {
    pool := &LDAPPool{
        conns:    make(chan *ldap.Conn, size),
        maxSize:  size,
    }
    for i := 0; i < size; i++ {
        conn, err := ldap.DialURL(url)
        if err != nil { continue }
        conn.Bind(bindDN, bindPass)
        pool.conns <- conn
    }
    return pool
}

func (p *LDAPPool) Get() *ldap.Conn { return <-p.conns }
func (p *LDAPPool) Put(conn *ldap.Conn) { p.conns <- conn }
```

| Pool Size | Concurrent Users | Recommended |
|-----------|-----------------|-------------|
| 5 | <100 | Small org |
| 10 | 100-1000 | Default |
| 20 | 1000-10000 | Enterprise |
| 50 | >10000 | Large directory |

### Connection Health

```go
func (p *LDAPPool) GetHealthy() (*ldap.Conn, error) {
    for {
        conn := p.Get()
        if err := conn.Conn().SetReadDeadline(time.Now().Add(100 * time.Millisecond)); err != nil {
            conn.Close()
            continue // Get a new connection
        }
        return conn, nil
    }
}
```

## LDAPS / StartTLS

### LDAPS (Port 636)

```go
conn, err := ldap.DialURL("ldaps://ldap.corp.com:636",
    ldap.DialWithTLSConfig(&tls.Config{
        ServerName:         "ldap.corp.com",
        MinVersion:         tls.VersionTLS12,
        InsecureSkipVerify: false, // Must verify
    }),
)
```

### StartTLS (Port 389 → TLS Upgrade)

```go
conn, err := ldap.DialURL("ldap://ldap.corp.com:389")
if err != nil { return err }
// Upgrade to TLS
err = conn.StartTLS(&tls.Config{
    ServerName: "ldap.corp.com",
})
```

| Method | Port | When to Use |
|--------|------|-------------|
| LDAPS | 636 | Dedicated TLS port (simplest) |
| StartTLS | 389 | Same port, opportunistic upgrade |

Always verify the server certificate. Never set `InsecureSkipVerify: true` in production.

## Search Optimization

### Indexing on LDAP Server

```ldif
# Ensure these attributes are indexed on the directory server
attributeType cn  pres,eq,sub
attributeType uid  pres,eq
attributeType mail  pres,eq
attributeType memberOf  pres,eq
```

### Efficient Search Patterns

```go
// BAD: Subtree search with broad filter
search := ldap.NewSearchRequest(
    "dc=corp,dc=com",          // Entire tree
    ldap.ScopeWholeSubtree,    // Expensive
    ldap.NeverDerefAliases,
    0, 0,                      // No limits
    "(objectClass=*)",         // All objects
    []string{},
    nil,
)

// GOOD: Scoped search with specific filter
search := ldap.NewSearchRequest(
    "ou=users,dc=corp,dc=com", // Specific OU
    ldap.ScopeSingleLevel,     // Only immediate children
    ldap.NeverDerefAliases,
    100,                       // Size limit
    10,                        // Time limit (seconds)
    "(uid=jane)",              // Specific filter
    []string{"mail", "cn", "memberOf"},
    nil,
)
```

### Search Best Practices

| Practice | Rationale |
|----------|-----------|
| Scope to specific OU | Avoid full subtree scans |
| Use indexed attributes | `uid`, `mail`, `cn` are indexed |
| Set time limit | Prevent runaway queries |
| Set size limit | Cap result set |
| Request only needed attributes | Reduce payload |

## Group Membership Resolution

### Direct Membership

```go
func getUserGroups(conn *ldap.Conn, userDN string) ([]string, error) {
    search := ldap.NewSearchRequest(
        userDN,
        ldap.ScopeBaseObject,
        ldap.NeverDerefAliases, 0, 0,
        "(objectClass=*)",
        []string{"memberOf"},
        nil,
    )
    result, err := conn.Search(search)
    if err != nil { return nil, err }
    
    groups := []string{}
    for _, entry := range result.Entries {
        for _, attr := range entry.Attributes {
            groups = append(groups, attr.Values...)
        }
    }
    return groups, nil
}
```

### Nested Groups

AD supports nested groups — a user in "Engineers" which is in "Tech" which is in "Staff":

```go
// LDAP_MATCHING_RULE_IN_CHAIN (1.2.840.113556.1.4.1941)
// Resolves entire nesting chain in one query
filter := fmt.Sprintf(
    "(member:1.2.840.113556.1.4.1941:=%s)",
    ldap.EscapeFilter(userDN),
)
search := ldap.NewSearchRequest(
    "ou=groups,dc=corp,dc=com",
    ldap.ScopeWholeSubtree,
    ldap.NeverDerefAliases, 0, 0,
    filter,
    []string{"cn"},
    nil,
)
```

| Directory | Nested Support | Filter Rule |
|-----------|--------------|-------------|
| Active Directory | ✅ Native | `:1.2.840.113556.1.4.1941:` |
| OpenLDAP | ✅ Via `memberOf` overlay | Manual recursion |
| FreeIPA | ✅ Via group membership | API call |

### Recursive Resolution (Non-AD)

```go
func resolveNestedGroups(conn *ldap.Conn, userDN string, depth int) ([]string, error) {
    if depth > 10 { return nil, ErrMaxDepth }
    
    direct := getDirectGroups(conn, userDN)
    all := make(map[string]bool)
    
    for _, g := range direct {
        all[g] = true
        // Recursively resolve parent groups
        parents := getDirectGroups(conn, g)
        for _, p := range parents {
            all[p] = true
        }
    }
    return keys(all), nil
}
```

## Auto-Provisioning

```go
func authenticateLDAP(conn *ldap.Conn, username, password, baseDN, filter string) (*User, error) {
    // 1. Bind with service account
    if err := conn.Bind(serviceBindDN, serviceBindPass); err != nil {
        return nil, ErrServiceBind
    }
    
    // 2. Search for user
    userFilter := strings.Replace(filter, "{username}", ldap.EscapeFilter(username), 1)
    search := ldap.NewSearchRequest(baseDN, ldap.ScopeWholeSubtree, ...)
    result, err := conn.Search(search)
    if err != nil || len(result.Entries) == 0 {
        return nil, ErrUserNotFound
    }
    
    userDN := result.Entries[0].DN
    
    // 3. Bind as user (verify password)
    if err := conn.Bind(userDN, password); err != nil {
        return nil, ErrInvalidCredentials
    }
    
    // 4. Auto-provision if configured
    user := mapLDAPEntry(result.Entries[0])
    if autoProvision {
        provisionUser(user) // Create GGID user from LDAP attributes
    }
    
    return user, nil
}
```

## Multi-Directory Federation

```yaml
directories:
  - name: "Corporate AD"
    url: "ldaps://ad.corp.com:636"
    base_dn: "dc=corp,dc=com"
    priority: 1               # Primary
    
  - name: "Partner LDAP"
    url: "ldaps://ldap.partner.com:636"
    base_dn: "ou=partners,dc=partner,dc=com"
    priority: 2               # Fallback
    
  - name: "Dev LDAP"
    url: "ldap://dev-ldap.internal:389"
    base_dn: "ou=dev,dc=internal"
    priority: 3               # Dev/test
```

### Routing Logic

```go
func authenticate(username, password string) (*User, error) {
    for _, dir := range directories {
        user, err := dir.Authenticate(username, password)
        if err == nil { return user, nil }
        if err == ErrInvalidCredentials { return nil, err } // Real denial
        // Connection error → try next directory
    }
    return nil, ErrAllDirectoriesDown
}
```

## Sync Tuning

| Parameter | Default | Tuning Advice |
|-----------|---------|---------------|
| Sync interval | 6 hours | Hourly for volatile orgs, daily for stable |
| Batch size | 500 | Increase for large directories |
| Page size | 1000 | LDAP Simple Paged Results |
| Timeout | 30s | Increase for slow directories |
| Connection pool | 10 | Match to sync parallelism |

```bash
# Sync with paging
GET /api/v1/identity/federation/ldap/{id}/sync?pageSize=1000
# → {"synced": 5000, "created": 12, "updated": 47, "deactivated": 3, "duration_ms": 3200}
```

## Monitoring

| Metric | Alert |
|--------|-------|
| LDAP bind failures | >1% → check service account |
| Search latency | >500ms → add indexes or narrow scope |
| Sync failures | Any → connectivity issue |
| Connection pool exhaustion | Pool at 100% → increase pool size |
| Certificate expiry | <30 days → renew |

## See Also

- [Identity Provider Configuration](identity-provider-configuration.md)
- [SCIM 2.0 Implementation](scim-2-0-implementation.md)
- [Identity Lifecycle Automation](identity-lifecycle-automation.md)
- [Identity Federation Architecture](identity-federation-architecture.md)
