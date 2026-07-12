# LDAP Integration Guide

This guide covers integrating GGID with LDAP directories (OpenLDAP, Active Directory, and other LDAPv3-compliant servers).

## Overview

GGID's LDAP provider authenticates users against an external LDAP directory, enabling single-source-of-truth identity management while leveraging GGID's token issuance, RBAC, and audit capabilities.

## LDAP Schema Mapping

### inetOrgPerson → GGID User

| LDAP Attribute | GGID Field | Notes |
|---|---|---|
| `uid` | `username` | Primary login identifier |
| `mail` | `email` | Unique email address |
| `cn` | `display_name` | Full name |
| `givenName` | `first_name` | Optional |
| `sn` | `last_name` | Optional |
| `telephoneNumber` | `phone` | Optional |
| `title` | `job_title` | Optional |
| `departmentNumber` | `department` | Optional |
| `employeeNumber` | `employee_id` | Optional |

### Active Directory Mapping

| AD Attribute | GGID Field |
|---|---|
| `sAMAccountName` | `username` |
| `userPrincipalName` | `email` |
| `displayName` | `display_name` |
| `givenName` | `first_name` |
| `sn` | `last_name` |

## Bind Strategies

### Simple Bind

```
ldap://dc01.example.com:389
Bind DN: cn=svc-ggid,ou=service-accounts,dc=example,dc=com
Password: <stored in env var LDAP_BIND_PASSWORD>
```

Use simple bind for service accounts with dedicated access rights. Never use anonymous bind for authentication.

### SASL Bind

SASL (Simple Authentication and Security Layer) supports:
- **GSSAPI** — Kerberos tickets (preferred for AD)
- **EXTERNAL** — TLS client certificates
- **DIGEST-MD5** — Challenge-response

```go
l, err := ldap.DialURL("ldaps://dc01.example.com:636")
// SASL GSSAPI bind
err = l.GSSAPIBind("", "svc-ggid@EXAMPLE.COM")
```

## Search Filter Design

### User Search Filter

```
# OpenLDAP (inetOrgPerson)
(&(objectClass=inetOrgPerson)(uid={username}))

# Active Directory
(&(objectClass=user)(sAMAccountName={username}))

# Multi-attribute search
(&(objectClass=inetOrgPerson)(|(uid={username})(mail={username})))
```

### Group Search Filter

```
# OpenLDAP (groupOfNames)
(&(objectClass=groupOfNames)(member={userDN}))

# POSIX groups
(&(objectClass=posixGroup)(memberUid={uid}))

# Active Directory
(&(objectClass=group)(member={userDN}))
```

## Group Synchronization

### memberUid vs Member DN

| Style | Format | Directory |
|---|---|---|
| `memberUid` | `memberUid: jdoe` | OpenLDAP POSIX groups |
| `member` (DN) | `member: cn=jdoe,ou=users,dc=example,dc=com` | OpenLDAP groupOfNames, AD |

GGID handles both formats automatically:

```yaml
ldap:
  group_member_attribute: memberUid  # or "member" for DN-based
  group_object_class: posixGroup     # or "groupOfNames"
```

### Group → Role Mapping

```yaml
ldap_group_role_mapping:
  "cn=admins,ou=groups,dc=example,dc=com": "platform-admin"
  "cn=developers,ou=groups,dc=example,dc=com": "developer"
  "cn=readonly,ou=groups,dc=example,dc=com": "viewer"
```

## LDIF Provisioning

### Create User

```ldif
dn: uid=jdoe,ou=users,dc=example,dc=com
objectClass: inetOrgPerson
objectClass: organizationalPerson
objectClass: person
objectClass: top
uid: jdoe
cn: John Doe
sn: Doe
givenName: John
mail: jdoe@example.com
userPassword: {SSHA}hashedpassword
```

### Add to Group

```ldif
dn: cn=developers,ou=groups,dc=example,dc=com
changetype: modify
add: memberUid
memberUid: jdoe
```

## Configuration

### OpenLDAP

```bash
LDAP_URL=ldap://openldap:389
LDAP_BIND_DN=cn=admin,dc=example,dc=com
LDAP_BIND_PASSWORD=secret
LDAP_BASE_DN=dc=example,dc=com
LDAP_USER_FILTER=(uid={username})
LDAP_GROUP_FILTER=(memberUid={uid})
LDAP_START_TLS=true
LDAP_AUTO_PROVISION=true
```

### Active Directory

```bash
LDAP_URL=ldaps://dc01.example.com:636
LDAP_BIND_DN=svc-ggid@example.com
LDAP_BIND_PASSWORD=secret
LDAP_BASE_DN=dc=example,dc=com
LDAP_USER_FILTER=(sAMAccountName={username})
LDAP_GROUP_FILTER=(member={dn})
LDAP_AUTO_PROVISION=true
```

## Auto-Provisioning on First Login

When `LDAP_AUTO_PROVISION=true`, GGID creates a local user record on the first successful LDAP bind. The provisioned user inherits:

- Read-only attributes from LDAP (email, display name, phone)
- Default tenant assignment
- Group-based role mapping

```go
if ldapAutoProvision {
    user, err := provisionUser(ldapUser)
    if err != nil {
        return nil, fmt.Errorf("auto-provision failed: %w", err)
    }
    // Assign roles based on LDAP group membership
    assignRolesFromGroups(user, ldapGroups)
}
```

## START_TLS vs LDAPS

| Feature | START_TLS | LDAPS |
|---|---|---|
| Port | 389 | 636 |
| TLS | Upgraded after connect | From connection start |
| Fallback | Can detect and reject | No plaintext phase |
| Recommendation | Preferred (RFC 4513) | Use for AD |

**Always use TLS.** Never transmit credentials over plaintext LDAP (port 389 without START_TLS).

## Connection Pooling

GGID maintains a connection pool to the LDAP server:

```yaml
ldap:
  pool_size: 10
  pool_timeout: 30s
  idle_timeout: 5m
  max_lifetime: 30m
```

- Each goroutine borrows a connection from the pool
- Connections are health-checked before reuse
- Failed connections are automatically replaced

## Troubleshooting

### Common Issues

| Symptom | Cause | Fix |
|---|---|---|
| `Invalid credentials` | Wrong bind DN or password | Verify with `ldapwhoami` |
| `No such object` | Base DN mismatch | Check `LDAP_BASE_DN` |
| `Connection refused` | Wrong host/port or firewall | Verify `LDAP_URL` |
| `TLS handshake failed` | CA certificate mismatch | Add CA to trust store |
| `Filter error` | Malformed filter syntax | Test with `ldapsearch` |

### Debug Commands

```bash
# Test simple bind
ldapwhoami -x -H ldap://openldap:389 -D "cn=admin,dc=example,dc=com" -w secret

# Search for user
ldapsearch -x -H ldap://openldap:389 -b "dc=example,dc=com" "(uid=jdoe)"

# Test START_TLS
ldapsearch -ZZ -x -H ldap://openldap:389 -b "dc=example,dc=com" "(uid=jdoe)"

# Check group membership
ldapsearch -x -H ldap://openldap:389 -b "cn=developers,ou=groups,dc=example,dc=com" "(objectClass=*)"
```

### Logs

GGID logs LDAP operations at debug level:

```
DEBUG ldap: binding as cn=admin,dc=example,dc=com
DEBUG ldap: searching uid=jdoe in dc=example,dc=com
DEBUG ldap: found user cn=John Doe, DN=uid=jdoe,ou=users,dc=example,dc=com
DEBUG ldap: user has 3 groups: [developers admins readonly]
```