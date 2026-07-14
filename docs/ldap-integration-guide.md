# GGID LDAP/Active Directory Integration Guide

Complete guide for integrating LDAP and Active Directory with GGID for
enterprise authentication, group synchronization, and auto-provisioning.

---

## Table of Contents

- [Overview](#overview)
- [LDAP Provider Configuration](#ldap-provider-configuration)
- [Bind DN and Authentication](#bind-dn-and-authentication)
- [User Search Filters](#user-search-filters)
- [START_TLS vs LDAPS](#start_tls-vs-ldaps)
- [Group Synchronization](#group-synchronization)
- [Auto-Provisioning](#auto-provisioning)
- [Active Directory Specifics](#active-directory-specifics)
- [OpenLDAP Specifics](#openldap-specifics)
- [Troubleshooting](#troubleshooting)

---

## Overview

GGID's LDAP provider allows users to authenticate against an existing LDAP or
Active Directory server. When a user logs in, GGID:

1. Binds to the LDAP server using a service account
2. Searches for the user by their username/email
3. Attempts to bind as that user with the provided password
4. Optionally syncs group memberships and auto-provisions the user

```
User → GGID Gateway → Auth Service → LDAP Server
                                       ├── Search for user DN
                                       ├── Bind as user (verify password)
                                       ├── Fetch groups
                                       └── Return to GGID
```

---

## LDAP Provider Configuration

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `LDAP_URL` | Yes | — | LDAP server URL (e.g., `ldap://dc01.corp.local:389`) |
| `LDAP_BIND_DN` | Yes | — | Service account DN for searching |
| `LDAP_BIND_PASSWORD` | Yes | — | Service account password |
| `LDAP_BASE_DN` | Yes | — | Base DN for user searches |
| `LDAP_USER_FILTER` | No | `(uid=%s)` | Search filter with `%s` placeholder |
| `LDAP_GROUP_DN` | No | — | Base DN for group searches |
| `LDAP_GROUP_FILTER` | No | `(member=%s)` | Group search filter |
| `LDAP_START_TLS` | No | `true` | Upgrade plaintext connection to TLS |
| `LDAP_AUTO_PROVISION` | No | `false` | Auto-create GGID user on first login |
| `LDAP_INSECURE_SKIP_VERIFY` | No | `false` | Skip TLS cert verification (dev only) |

### Docker Compose Configuration

```yaml
services:
  auth:
    environment:
      LDAP_URL: "ldap://dc01.corp.local:389"
      LDAP_BIND_DN: "CN=ggid-svc,OU=Service Accounts,DC=corp,DC=local"
      LDAP_BIND_PASSWORD: "${LDAP_BIND_PASSWORD}"
      LDAP_BASE_DN: "DC=corp,DC=local"
      LDAP_USER_FILTER: "(sAMAccountName=%s)"
      LDAP_GROUP_DN: "OU=Groups,DC=corp,DC=local"
      LDAP_GROUP_FILTER: "(member=%s)"
      LDAP_START_TLS: "true"
      LDAP_AUTO_PROVISION: "true"
```

### Kubernetes Secret

```bash
# Create LDAP secret
kubectl create secret generic ggid-ldap \
  --from-literal=LDAP_BIND_PASSWORD='YourSecurePassword' \
  -n ggid

# Reference in deployment
env:
  - name: LDAP_BIND_PASSWORD
    valueFrom:
      secretKeyRef:
        name: ggid-ldap
        key: LDAP_BIND_PASSWORD
```

---

## Bind DN and Authentication

### Service Account Requirements

The bind DN is a service account that GGID uses to search the LDAP directory.
It needs:

- **Search permission** on the user and group base DNs
- **Read permission** on user attributes (uid, mail, displayName, memberOf)
- **No write permission** required (GGID is read-only against LDAP)

### Active Directory Bind DN

```
LDAP_BIND_DN=CN=ggid-svc,OU=Service Accounts,DC=corp,DC=local
```

### OpenLDAP Bind DN

```
LDAP_BIND_DN=cn=ggid-svc,ou=services,dc=example,dc=com
```

### Verifying Bind Credentials

```bash
# Test bind credentials with ldapsearch
ldapsearch -x -H ldap://dc01.corp.local:389 \
  -D "CN=ggid-svc,OU=Service Accounts,DC=corp,DC=local" \
  -w "$LDAP_BIND_PASSWORD" \
  -b "DC=corp,DC=local" \
  "(sAMAccountName=testuser)" dn mail displayName

# Expected output:
# dn: CN=Test User,OU=Users,DC=corp,DC=local
# mail: testuser@corp.local
# displayName: Test User
```

---

## User Search Filters

The user filter determines how GGID maps a login username to an LDAP entry.
The `%s` placeholder is replaced with the user's input.

### Active Directory (Default)

```
LDAP_USER_FILTER=(sAMAccountName=%s)
```

Maps `john.doe` → searches for `sAMAccountName=john.doe`.

### OpenLDAP / POSIX

```
LDAP_USER_FILTER=(uid=%s)
```

### Email-Based Login

```
LDAP_USER_FILTER=(mail=%s)
```

Maps `john.doe@corp.com` → searches for `mail=john.doe@corp.com`.

### Multi-Attribute Search

Some directories use different attributes for different users:

```
LDAP_USER_FILTER=(|(uid=%s)(mail=%s)(sAMAccountName=%s))
```

This searches for the username in uid, mail, or sAMAccountName.

---

## START_TLS vs LDAPS

GGID supports two TLS modes for LDAP:

### START_TLS (Recommended)

Upgrades a plaintext LDAP connection to TLS after the initial handshake:

```
LDAP_URL=ldap://dc01.corp.local:389
LDAP_START_TLS=true
```

- Uses port **389**
- TLS negotiated after connection
- Allows fallback to plaintext (disabled by default in GGID)
- Most flexible — supported by all modern LDAP servers

### LDAPS (LDAP over TLS)

Establishes a TLS connection from the start:

```
LDAP_URL=ldaps://dc01.corp.local:636
LDAP_START_TLS=false
```

- Uses port **636**
- TLS from the beginning
- Certificate must be valid for the hostname
- Simpler but less flexible

### Choosing

| Aspect | START_TLS | LDAPS |
|--------|-----------|-------|
| Port | 389 | 636 |
| TLS | Negotiated after connect | From the start |
| Fallback | Can downgrade (rare) | No fallback |
| Firewall | One port (389) | Separate port (636) |
| Recommendation | Preferred | Use if server requires |

### Certificate Authority

For self-signed or internal CA certificates:

```bash
# Add CA cert to the container
COPY corp-ca.crt /usr/local/share/ca-certificates/
RUN update-ca-certificates

# Or set env var
LDAP_CACERT_FILE=/etc/ssl/certs/corp-ca.pem

# Or skip verification (DEV ONLY)
LDAP_INSECURE_SKIP_VERIFY=true
```

---

## Group Synchronization

GGID can map LDAP groups to GGID roles for authorization.

### How It Works

```
1. User authenticates via LDAP
2. GGID queries LDAP for user's groups (memberOf attribute)
3. LDAP groups are mapped to GGID roles
4. JWT includes mapped roles
```

### Group Mapping Configuration

```bash
# Base DN for group searches
LDAP_GROUP_DN=OU=Groups,DC=corp,DC=local

# Filter to find groups for a user
LDAP_GROUP_FILTER=(member=%s)
```

### Group-to-Role Mapping

Configure group-to-role mapping via the GGID API:

```bash
# Map LDAP group "GGID-Admins" to GGID role "admin"
curl -X POST $API/api/v1/settings/ldap/group-mapping \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "X-Tenant-ID: $TENANT_ID" \
    -d '{
        "ldap_group": "CN=GGID-Admins,OU=Groups,DC=corp,DC=local",
        "ggid_role": "admin"
    }'

# Map LDAP group "GGID-Editors" to GGID role "editor"
curl -X POST $API/api/v1/settings/ldap/group-mapping \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -d '{
        "ldap_group": "CN=GGID-Editors,OU=Groups,DC=corp,DC=local",
        "ggid_role": "editor"
    }'
```

### Nested Groups

Active Directory supports nested groups. GGID resolves nested membership:

```
User: john.doe
  └── memberOf: CN=Devs,OU=Groups,DC=corp,DC=local
       └── memberOf: CN=Engineering,OU=Groups,DC=corp,DC=local
            └── memberOf: CN=All-Staff,OU=Groups,DC=corp,DC=local
```

GGID resolves all three groups and applies the most permissive role.

---

## Auto-Provisioning

When `LDAP_AUTO_PROVISION=true`, GGID automatically creates a local user record
on first successful LDAP login.

### Flow

```
1. User submits username/password
2. GGID authenticates against LDAP → success
3. GGID checks if user exists locally
   ├── Exists → update groups from LDAP → issue JWT
   └── Missing → create user → assign groups → issue JWT
4. User is now provisioned in GGID
```

### Provisioned Attributes

| LDAP Attribute | GGID Field |
|----------------|------------|
| `sAMAccountName` or `uid` | `username` |
| `mail` | `email` |
| `displayName` | `name` |
| `memberOf` | `roles` (via group mapping) |

### JIT (Just-In-Time) Provisioning

Auto-provisioning is a form of JIT provisioning. The user is created on first
login, not pre-loaded:

```bash
# Enable auto-provisioning
LDAP_AUTO_PROVISION=true

# When a user logs in for the first time:
# {"event":"user.auto_provisioned","source":"ldap","username":"john.doe"}
```

### Deprovisioning

When a user is disabled in LDAP/AD, GGID detects it on next login attempt:

```
1. User tries to log in
2. GGID attempts LDAP bind → fails (account disabled)
3. GGID marks local user as `suspended`
4. User can no longer authenticate
```

For immediate deprovisioning, use a scheduled sync job:

```bash
# Sync LDAP user status (run every hour)
curl -X POST $API/api/v1/settings/ldap/sync \
    -H "Authorization: Bearer $ADMIN_TOKEN"
```

---

## Active Directory Specifics

### Common Attributes

| AD Attribute | Description | Use in GGID |
|--------------|-------------|-------------|
| `sAMAccountName` | Login name (pre-W2K) | Username |
| `userPrincipalName` | Login name (modern) | Email/UPN |
| `mail` | Email address | Email |
| `displayName` | Full name | Name |
| `memberOf` | Group memberships | Role mapping |
| `userAccountControl` | Account status | Active/disabled |

### Checking if User is Active

```bash
# userAccountControl bit flags:
#   2   = ACCOUNTDISABLE
#   16  = LOCKOUT
#   512 = NORMAL_ACCOUNT
#   66048 = NORMAL_ACCOUNT + DONT_EXPIRE_PASSWORD

# Filter for active users only:
LDAP_USER_FILTER=(&(sAMAccountName=%s)(!(userAccountControl:1.2.840.113556.1.4.803:=2)))
```

### Active Directory Referrals

AD may return referrals for searches across domains. Disable referral chasing
for predictable behavior:

```bash
LDAP_DISABLE_REFERRALS=true
```

---

## OpenLDAP Specifics

### Common Attributes

| OpenLDAP Attribute | Description |
|--------------------|-------------|
| `uid` | User ID (login name) |
| `cn` | Common name |
| `sn` | Surname |
| `mail` | Email address |
| `memberOf` | Group memberships (if overlay enabled) |

### memberOf Overlay

OpenLDAP doesn't include `memberOf` by default. Enable it:

```ldif
# Enable memberOf overlay
dn: cn=module{0},cn=config
changetype: modify
add: olcModuleLoad
olcModuleLoad: memberof

dn: olcOverlay={0}memberof,olcDatabase={1}mdb,cn=config
objectClass: olcConfig
objectClass: olcMemberOf
objectClass: olcOverlayConfig
olcOverlay: memberof
```

### Seed Users for Testing

```bash
# Create test users in OpenLDAP
ldapadd -x -D "cn=admin,dc=example,dc=com" -w admin -H ldap://localhost:389 << 'EOF'
dn: uid=john.doe,ou=users,dc=example,dc=com
objectClass: inetOrgPerson
uid: john.doe
cn: John Doe
sn: Doe
mail: john@example.com
userPassword: {SSHA}...

dn: cn=admins,ou=groups,dc=example,dc=com
objectClass: groupOfNames
cn: admins
member: uid=john.doe,ou=users,dc=example,dc=com
EOF
```

---

## Troubleshooting

### Common Issues

| Issue | Cause | Fix |
|-------|-------|-----|
| `LDAP connection refused` | Wrong host/port | Verify `LDAP_URL`, check firewall |
| `Invalid credentials` | Wrong bind DN/password | Test with `ldapsearch` |
| `User not found` | Wrong base DN or filter | Verify with `ldapsearch` manually |
| `TLS handshake failed` | Invalid CA cert | Set `LDAP_CACERT_FILE` or use `INSECURE_SKIP_VERIFY` (dev) |
| `Group sync empty` | `memberOf` not enabled | Enable memberOf overlay (OpenLDAP) |
| `Auto-provision fails` | Missing required attributes | Ensure LDAP returns `mail` and `displayName` |

### Debug Mode

```bash
# Enable LDAP debug logging
LOG_LEVEL=debug
LDAP_DEBUG=true

# This logs:
# - LDAP bind attempts
# - Search queries and results
# - Group lookups
# - Auto-provisioning events
```

### Diagnostic Commands

```bash
# 1. Test LDAP connectivity
ldapsearch -x -H ldap://dc01.corp.local:389 \
  -D "$LDAP_BIND_DN" -w "$LDAP_BIND_PASSWORD" \
  -b "$LDAP_BASE_DN" "(sAMAccountName=testuser)"

# 2. Test START_TLS
ldapsearch -ZZ -x -H ldap://dc01.corp.local:389 \
  -D "$LDAP_BIND_DN" -w "$LDAP_BIND_PASSWORD" \
  -b "$LDAP_BASE_DN" "(uid=testuser)"

# 3. Check group membership
ldapsearch -x -H ldap://dc01.corp.local:389 \
  -D "$LDAP_BIND_DN" -w "$LDAP_BIND_PASSWORD" \
  -b "$LDAP_GROUP_DN" "(member=CN=Test User,OU=Users,DC=corp,DC=local)" cn

# 4. Verify GGID auth flow
curl -X POST $API/api/v1/auth/login \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{"username":"testuser","password":"TestPass123!"}'
```

---

## References

- [Configuration Reference](./configuration.md) — All env vars
- Auth Providers — Provider API
- [Troubleshooting](./troubleshooting.md) — Common issues
