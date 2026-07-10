# LDAP / Active Directory Integration

## 1. Overview

LDAP (Lightweight Directory Access Protocol) is the backbone of enterprise identity
management. The dominant implementations are:

| Server | Vendor | Typical Use |
|--------|--------|-------------|
| **Active Directory (AD)** | Microsoft | Windows-centric enterprises |
| **OpenLDAP** | Open source | Linux/university environments |
| **FreeIPA / IdM** | Red Hat | Linux domain management |
| **389-DS** | Fedora/Red Hat | General-purpose directory |

GGID integrates LDAP as an authentication provider within its **provider chain**
(`pkg/authprovider`). When `LDAP_URL` is set, an `LDAPProvider` is appended after
the `LocalProvider`, enabling users stored in a corporate directory to authenticate
without a separate GGID-managed password.

**Integration patterns supported by GGID:**

- **Search-then-bind authentication** — bind with a service account, search for the
  user DN, then re-bind as the user to verify credentials.
- **Group-to-role mapping** — read `memberOf` attributes and map LDAP groups to
  GGID roles via `GroupRoleMapping`.
- **JIT auto-provisioning** — create a local shadow user on first successful
  LDAP login (`AutoProvision: true`).

---

## 2. Connection Configuration

### LDAP URL Formats

| Scheme | Port | TLS | Notes |
|--------|------|-----|-------|
| `ldap://host:389` | 389 | None | Plaintext; pair with START_TLS |
| `ldaps://host:636` | 636 | Implicit TLS | Deprecated (RFC 8460 recommends START_TLS) |
| `ldap://host:3268` | 3268 | START_TLS | AD Global Catalog (cross-domain search) |

**Recommended:** `ldap://host:389` + `START_TLS`. This upgrades the plaintext
connection to TLS after the initial handshake, allowing the same port for both
encrypted and unencrypted traffic while giving the client control over TLS
negotiation.

### TLS Configuration

GGID's `LDAPProvider` enforces minimum TLS 1.2 by default. A custom `*tls.Config`
can be passed via `LDAPConfig.TLSConfig` for enterprise CA certificates. GGID does
not currently expose a `LDAP_CA_CERT` env var — this is a roadmap gap.

### Bind DN and Password (Service Account)

The service account binds to search for user DNs before the user bind. It needs
read access to user attributes and group memberships. Use a dedicated read-only
account (not Domain Admin), store the password in `keys.env`, and rotate it
periodically per enterprise policy.

### GGID Environment Variables

Configured in `services/auth/cmd/main.go` (lines 75-99):

```bash
# Required
LDAP_URL=ldap://dc01.corp.local:389
LDAP_BIND_DN=cn=ggid-service,ou=services,dc=corp,dc=local
LDAP_BIND_PASSWORD=********          # from keys.env
LDAP_BASE_DN=dc=corp,dc=local

# Recommended
LDAP_START_TLS=true
LDAP_AUTO_PROVISION=true

# Directory-specific filter
LDAP_USER_FILTER=(sAMAccountName=%s)           # Active Directory
LDAP_USER_FILTER=(uid=%s)                      # OpenLDAP / FreeIPA
LDAP_USER_FILTER=(&(objectClass=user)(sAMAccountName=%s))  # AD strict
```

**Defaults when env vars are missing:**
- `BaseDN`: `dc=corp,dc=local`
- `UserFilter`: `(&(objectClass=inetOrgPerson)(uid=%s))`
- `StartTLS`: false (must be explicitly enabled)
- `AutoProvision`: false

---

## 3. Authentication Flow

### Search-Then-Bind (Implemented in GGID)

GGID implements the recommended search-then-bind pattern. The flow in
`LDAPProvider.Authenticate()` (ldap.go:91-154):

```
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│  1. Pool Bind │───▶│ 2. Search DN │───▶│ 3. User Bind │
│ (svc account) │    │ (UserFilter)  │    │ (user passwd) │
└──────────────┘    └──────────────┘    └──────────────┘
       │                                       │
       ▼                                       ▼
   return conn to pool              bind success → AuthResult
                                      bind fail   → Unauthenticated
```

1. **Pool bind**: acquire a pooled connection (or dial new), bind with service account.
2. **Search**: execute `UserFilter` against `BaseDN`, fetch `mail`, `displayName`,
   `memberOf`, `sAMAccountName`, `givenName`, `sn`.
3. **User bind**: dial a **fresh** connection and bind with the user DN + password.
   If the bind succeeds, the user is authenticated.
4. Return `AuthResult` with `ExternalID` (user DN) and attributes.

### Why Search-Then-Bind Instead of Direct Bind?

| Approach | Pro | Con |
|----------|-----|-----|
| **Direct bind** | One round-trip | Must know DN pattern (`cn=user,ou=...`); breaks on AD (cn) vs OpenLDAP (uid) |
| **Search-then-bind** | Works with any directory layout; returns attributes | Two round-trips; requires service account |

Direct bind (`cn={username},ou=users,dc=...`) assumes a fixed DN structure that varies
between directories. GGID correctly uses search-then-bind for maximum compatibility.

### Go Code Reference

Library: `github.com/go-ldap/ldap/v3` — the de facto Go LDAP client.

```go
// Simplified from ldap.go:91-154
conn, _ := p.getConn(ctx)                     // from pool
conn.Bind(p.cfg.BindDN, p.cfg.BindPassword)    // service bind
userDN, attrs, _ := p.searchUser(conn, username)
userConn, _ := p.dialFunc(ctx)                 // fresh connection
userConn.Bind(userDN, password)                // verify password
// success → AuthResult{ExternalID: userDN, Attributes: attrs}
```

---

## 4. Group Membership and Role Sync

### Reading Groups

| Directory | Method | Attribute |
|-----------|--------|-----------|
| **Active Directory** | `memberOf` back-link | Returns DN list directly on user object |
| **OpenLDAP** | Forward search | Query `groupOfNames` entries for `member={userDN}` |
| **FreeIPA** | `memberOf` plugin | Same as AD (requires `memberOf` overlay) |

GGID fetches `memberOf` as part of the user search and extracts group DN list:

```go
// ldap.go — searchUser() requests memberOf attribute
[]string{"mail", "displayName", "memberOf", "sAMAccountName", ...}

// ldap.go — mapGroupsToRoles() (line 359)
if memberOf, ok := attrs["memberOf"]; ok {
    groups = memberOf.([]string)  // or single string
}
```

**Limitation:** GGID reads only top-level `memberOf`. Nested groups are
**not** resolved recursively. AD resolves server-side; OpenLDAP does not.

### Group-to-Role Mapping

```go
type GroupRoleMapping struct {
    GroupDN string  // "cn=admins,ou=groups,dc=corp,dc=local"
    Role    string  // "admin"
}

// Config example
cfg.GroupRoleMappings = []GroupRoleMapping{
    {"cn=ggid-admins,ou=groups,dc=corp,dc=local", "admin"},
    {"cn=ggid-developers,ou=groups,dc=corp,dc=local", "developer"},
    {"cn=ggid-viewers,ou=groups,dc=corp,dc=local", "viewer"},
}
```

**Sync model:** On-login (real-time). GGID evaluates group membership at
authentication time and populates `AuthResult.Roles`. Periodic/scheduled sync
is not yet implemented.

---

## 5. Auto-Provisioning (JIT)

When `AutoProvision: true`, GGID creates a local **shadow user** on first
successful LDAP login. This eliminates the need to pre-create user accounts.

### Attribute Mapping

| LDAP Attribute | GGID Field | Notes |
|----------------|------------|-------|
| `cn` / `displayName` | `name` | Display name |
| `mail` | `email` | Primary email |
| `sAMAccountName` / `uid` | `username` | Login identifier |
| `title` | `job_title` | Optional |

### Shadow User Characteristics

- `NewUser: true` in `AuthResult` signals JIT creation
- No local password stored — credentials verified against LDAP on every login
- `MustLink: true` when auto-provision is off — requires manual linking
- Roles assigned from `GroupRoleMappings` result

### Deprovisioning (Gap)

Currently, when an LDAP user's account is disabled in the directory, GGID has no
mechanism to detect this proactively. The user simply cannot authenticate (bind
fails), but their local shadow account remains active. Scheduled reconciliation
to disable shadow users is a roadmap item.

---

## 6. Password Policy and Error Handling

### AD Password Policies

AD manages all password complexity, expiry, and history on the directory side.
GGID **must not** enforce its own password policy for LDAP users — defer all
validation to the directory.

### LDAP Error Handling (Gap)

GGID currently maps all LDAP bind failures to a generic `Unauthenticated` error:

```go
// ldap.go:126-130 — current error handling
if err := userConn.Bind(userDN, creds.Password); err != nil {
    return nil, errors.Unauthenticated("LDAP authentication failed")
}
```

**Desired improvement — granular error mapping:**

| LDAP Result Code | Meaning | User-Facing Message |
|-----------------|---------|-------------------|
| 49 (`invalidCredentials`) | Wrong password / disabled / expired | "Invalid username or password" |
| 49 + data 533 | Account disabled | "Account disabled — contact your administrator" |
| 49 + data 775 | Account locked out | "Account locked — contact admin" |
| 53 (`unwillingToPerform`) | Password expired | "Password expired — reset via AD" |
| 51 (`busy`) | Server busy | "Directory busy — retry" |

The `go-ldap` library returns result codes in `*ldap.Error`. GGID should parse
AD-specific sub-codes (533/701/773/775) embedded in error messages.

---

## 7. GGID Provider Chain Architecture

### Chain Mechanics

```
Auth Request → Chain.Authenticate()
                    │
                    ▼
            ┌──────────────┐     success
            │ LocalProvider │──────────────▶ return AuthResult
            │ (local DB)    │
            └──────┬───────┘
                   │ fail
                   ▼
            ┌──────────────┐     success
            │ LDAPProvider  │──────────────▶ return AuthResult (NewUser=true if JIT)
            │ (directory)   │
            └──────┬───────┘
                   │ fail
                   ▼
            ┌──────────────┐
            │ (future: OIDC │
            │  SAML, etc.)  │
            └──────────────┘
```

The `Chain` type (provider.go:64) iterates providers in order, returning the
first success or the last error. `ChainEnhanced` adds `OnlyTypes()` filtering
to restrict authentication to specific provider types per request.

### Current State Analysis

Based on `pkg/authprovider/ldap.go` source code:

| Feature | Status | Detail |
|---------|--------|--------|
| START_TLS | **Implemented** | `dial()` issues StartTLS when configured |
| LDAPS (implicit TLS) | **Implemented** | Detected via `ldaps://` URL prefix |
| TLS 1.2 minimum | **Implemented** | `tlsConfig()` enforces `VersionTLS12` |
| Custom CA / client certs | **Partial** | `TLSConfig` field exists but no env var to configure it |
| Connection pooling | **Implemented** | Buffered channel pool, default size 5 |
| Pool health check | **Implemented** | `IsClosing()` check on get; discard unhealthy |
| Search-then-bind | **Implemented** | Service bind → search → user bind |
| Group-to-role mapping | **Implemented** | `GroupRoleMappings` + `mapGroupsToRoles()` |
| Nested group resolution | **Not implemented** | Only top-level `memberOf` read |
| Auto-provisioning (JIT) | **Implemented** | `NewUser` flag set when `AutoProvision` |
| Granular error mapping | **Not implemented** | All bind failures → generic `Unauthenticated` |
| Scheduled sync | **Not implemented** | Group/role evaluated on-login only |
| Deprovisioning | **Not implemented** | Shadow users not reconciled |
| Multi-directory (per-tenant) | **Not implemented** | Single LDAP config via env vars |
| Filter injection protection | **Implemented** | `ldap.EscapeFilter(username)` used |

---

## 8. Active Directory Specifics

### Key AD Attributes

| Attribute | Description | Example |
|-----------|-------------|---------|
| `sAMAccountName` | Pre-Windows 2000 logon name | `jdoe` |
| `userPrincipalName` (UPN) | user@realm format | `jdoe@corp.local` |
| `userAccountControl` | Bitfield flags (see below) | `512` (normal account) |
| `pwdLastSet` | Last password change (AD filetime) | `133836480000000000` |
| `memberOf` | Group DN back-links | `["cn=admins,..."]` |
| `primaryGroupID` | Primary group RID (e.g., 513 = Domain Users) | `513` |

### userAccountControl Bit Flags

| Flag | Value | Meaning |
|------|-------|---------|
| `SCRIPT` | 1 | Logon script runs |
| `ACCOUNTDISABLE` | 2 | Account is disabled |
| `NORMAL_ACCOUNT` | 512 | Typical user account |
| `LOCKOUT` | 16 | Account locked out |
| `PASSWD_NOTREQD` | 32 | No password required |
| `DONT_EXPIRE_PASSWD` | 65536 | Password never expires |
| `PASSWD_EXPIRED` | 8388608 | Password has expired |
| `MDS_ENCRYPTED_TEXT_PWD_ALLOWED` | 128 | Send encrypted password |

Source: [Microsoft Learn — UserAccountControl flags](https://learn.microsoft.com/en-us/troubleshoot/windows-server/active-directory/useraccountcontrol-manipulate-account-properties)

### Global Catalog

AD Global Catalog (port 3268 plaintext / 3269 TLS) provides forest-wide search
across all domains. Useful for multi-domain enterprises, but only a subset of
attributes is replicated to the GC (typically `displayName`, `mail`,
`sAMAccountName`, `memberOf` — not custom attributes).

### Referrals

AD returns referrals (result code 10) when a search matches objects in another
domain or partition. The `go-ldap` library supports referral chasing, but GGID
does not currently configure it. For single-domain deployments, referrals are
rare; for multi-domain forests, GGID should expose a `LDAP_CHASE_REFERRALS`
config option.

### AD User Filter Examples

```bash
# Match by sAMAccountName (most common)
LDAP_USER_FILTER=(sAMAccountName=%s)

# Match by UPN (user@domain.com)
LDAP_USER_FILTER=(userPrincipalName=%s)

# Strict: must be an enabled user
LDAP_USER_FILTER=(&(objectClass=user)(objectCategory=person)(!(userAccountControl:1.2.840.113556.1.4.803:=2))(sAMAccountName=%s))
```

---

## 9. Comparison: AD vs OpenLDAP vs FreeIPA

| Feature | Active Directory | OpenLDAP | FreeIPA |
|---------|-----------------|----------|---------|
| Naming attribute | `sAMAccountName`, UPN | `uid`, `cn` | `uid`, `krbPrincipalName` |
| Object class | `user`, `person` | `inetOrgPerson` | `posixAccount`, `person` |
| Group model | `group` with `member` | `groupOfNames` | `posixGroup`, `groupOfNames` |
| Nested groups | Yes (native) | No (manual recursion) | Yes (via `memberOf` plugin) |
| Group membership attr | `memberOf` (back-link) | None (query forward) | `memberOf` (plugin) |
| Password policy | Fine-grained (FGPP) | `ppolicy` overlay | IPA password policies |
| Kerberos | Built-in (kdc) | Separate (MIT KDC) | Built-in (MIT KDC) |
| TLS | LDAPS (636) / START_TLS | START_TLS recommended | START_TLS recommended |
| Schema | Proprietary extensions | RFC 4512 standard | RFC + IPA extensions |
| Default filter | `(sAMAccountName=%s)` | `(uid=%s)` | `(uid=%s)` |
| Multi-domain | Forest + Global Catalog | Referrals | Cross-realm trusts |

**Key takeaway for GGID:** The `UserFilter` env var is the primary directory-specific
knob. GGID's search-then-bind approach works across all three directories with only
the filter and base DN changing.

---

## 10. Roadmap

| Phase | Feature | Effort | Priority |
|-------|---------|--------|----------|
| **1** | `LDAP_CA_CERT` env var for custom CA trust | 2h | High |
| **1** | Granular LDAP error mapping (result code 49 sub-codes) | 4h | High |
| **2** | Scheduled group/role sync (cron-based, not just on-login) | 1d | Medium |
| **2** | Deprovisioning: disable shadow users when LDAP auth consistently fails | 4h | Medium |
| **3** | Nested group resolution (recursive `memberOf` for OpenLDAP) | 4h | Medium |
| **3** | `LDAP_CHASE_REFERRALS` config for multi-domain forests | 4h | Low |
| **4** | Per-tenant LDAP configuration (multi-directory) | 3d | Future |
| **4** | LDIF-based attribute mapping config (not hardcoded) | 1d | Future |
| **5** | AD password change endpoint (LDAP modify on `unicodePwd`) | 1d | Future |
| **5** | Mutual TLS (client certificate) authentication | 4h | Future |
