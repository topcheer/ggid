# GGID LDAP/Active Directory Integration

Complete guide for integrating LDAP or Active Directory with GGID for
enterprise authentication.

---

## Table of Contents

- [Overview](#overview)
- [LDAP Provider Configuration](#ldap-provider-configuration)
- [Bind DN and Service Account](#bind-dn-and-service-account)
- [User Search Filters](#user-search-filters)
- [Group Sync and Role Mapping](#group-sync-and-role-mapping)
- [START_TLS vs LDAPS](#start_tls-vs-ldaps)
- [Auto-Provisioning](#auto-provisioning)
- [Troubleshooting](#troubleshooting)

---

## Overview

GGID integrates with LDAP/AD as an authentication provider. When LDAP is
configured, the auth chain becomes: **Local → LDAP**. Users authenticate with
their directory credentials without needing a separate GGID password.

---

## LDAP Provider Configuration

### Environment Variables

| Variable | Required | Example | Description |
|----------|----------|---------|-------------|
| `LDAP_URL` | Yes | `ldap://dc01.corp.local:389` | LDAP server URL |
| `LDAP_BIND_DN` | Yes | `svc-ggid@corp.local` | Service account DN |
| `LDAP_BIND_PASSWORD` | Yes | `********` | Service account password |
| `LDAP_BASE_DN` | Yes | `DC=corp,DC=local` | Search base |
| `LDAP_USER_FILTER` | No | `(sAMAccountName=%s)` | User search filter |
| `LDAP_START_TLS` | No | `true` | Use START_TLS |
| `LDAP_AUTO_PROVISION` | No | `true` | Auto-create GGID users |

### Docker Compose

```yaml
auth:
  environment:
    LDAP_URL: ldap://openldap:389
    LDAP_BIND_DN: cn=admin,dc=corp,dc=local
    LDAP_BIND_PASSWORD: ${LDAP_PASSWORD}
    LDAP_BASE_DN: dc=corp,dc=local
    LDAP_USER_FILTER: (uid=%s)
    LDAP_START_TLS: "true"
    LDAP_AUTO_PROVISION: "true"
```

### Kubernetes Secret

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: ggid-ldap
type: Opaque
stringData:
  LDAP_URL: ldap://dc01.corp.local:389
  LDAP_BIND_DN: svc-ggid@corp.local
  LDAP_BIND_PASSWORD: "your-password"
  LDAP_BASE_DN: DC=corp,DC=local
  LDAP_USER_FILTER: "(sAMAccountName=%s)"
  LDAP_START_TLS: "true"
```

---

## Bind DN and Service Account

GGID needs a service account to search the directory for users before
authenticating them.

### Active Directory

```
Bind DN: svc-ggid@corp.local
        or CN=svc-ggid,OU=Service Accounts,DC=corp,DC=local
Password: ********
```

**Required permissions:** Read access to user objects (cn, mail,
sAMAccountName, memberOf). No write permissions needed.

### OpenLDAP

```
Bind DN: cn=ggid-service,ou=services,dc=corp,dc=local
Password: ********
```

### Verify Bind

```bash
# Test bind with ldapsearch
ldapsearch -x -H ldap://dc01.corp.local:389 \
  -D "svc-ggid@corp.local" -W \
  -b "DC=corp,DC=local" \
  "(sAMAccountName=jdoe)" mail memberOf
```

---

## User Search Filters

The `%s` placeholder is replaced with the username at login time.

### Active Directory (Default)

```
LDAP_USER_FILTER=(sAMAccountName=%s)
```

### OpenLDAP

```
LDAP_USER_FILTER=(uid=%s)
```

### Email-Based Login

```
LDAP_USER_FILTER=(mail=%s)
```

### Multi-Attribute (uid or email)

```
LDAP_USER_FILTER=(|(uid=%s)(mail=%s))
```

---

## Group Sync and Role Mapping

Map LDAP groups to GGID RBAC roles:

### Configuration

```bash
curl -X PUT $API/api/v1/settings/ldap/group-mapping \
  -H "Authorization: Bearer $JWT" \
  -d '{
    "mappings": {
      "CN=Admins,OU=Groups,DC=corp,DC=local": "admin",
      "CN=Developers,OU=Groups,DC=corp,DC=local": "editor",
      "CN=Viewers,OU=Groups,DC=corp,DC=local": "viewer"
    },
    "sync_on_login": true,
    "remove_unmapped_roles": false
  }'
```

### How Group Sync Works

1. User authenticates with LDAP credentials
2. GGID reads `memberOf` attribute from LDAP
3. Each LDAP group DN is matched against the mapping table
4. Matching GGID roles are assigned/revoked
5. User's JWT includes updated roles

---

## START_TLS vs LDAPS

| Feature | START_TLS | LDAPS |
|---------|-----------|-------|
| Port | 389 | 636 |
| Encryption | Upgrades plain connection | Native TLS from start |
| Cert validation | After upgrade | From connection start |
| Recommended | Yes (modern) | Legacy |

### Certificate Configuration

```bash
# For self-signed CA, add the CA cert to the container
LDAP_URL=ldap://dc01.corp.local:389
LDAP_START_TLS=true

# Mount CA certificate
docker run -v /path/to/ca-cert.pem:/etc/ssl/certs/company-ca.crt auth-service
```

### Verify TLS

```bash
# Test START_TLS
ldapsearch -ZZ -H ldap://dc01.corp.local:389 \
  -D "svc-ggid@corp.local" -W \
  -b "DC=corp,DC=local" "(sAMAccountName=test)"

# Test LDAPS
ldapsearch -H ldaps://dc01.corp.local:636 \
  -D "svc-ggid@corp.local" -W \
  -b "DC=corp,DC=local" "(sAMAccountName=test)"
```

---

## Auto-Provisioning

When `LDAP_AUTO_PROVISION=true`, GGID automatically creates a local user
record on first LDAP login. Subsequent logins update the local record.

### Provisioned Attributes

| GGID Field | LDAP Attribute | Notes |
|------------|---------------|-------|
| `username` | `sAMAccountName` or `uid` | From login |
| `email` | `mail` | Required |
| `name` | `displayName` or `cn` | Full name |
| `status` | `active` | Always active on provision |

### Disable Auto-Provision

When `false`, users must be pre-created in GGID before they can log in via
LDAP. This is useful for stricter access control.

---

## Troubleshooting

| Issue | Cause | Fix |
|-------|-------|-----|
| `invalid credentials` | Wrong bind DN/password | Verify with `ldapsearch` |
| `user not found` | Wrong base DN or filter | Test filter in `ldapsearch` |
| `connection refused` | LDAP server unreachable | Check `LDAP_URL`, firewall |
| `TLS handshake failed` | CA not trusted | Mount CA cert into container |
| `no email returned` | User missing `mail` attr | Add email in AD/LDAP |
| `group sync not working` | `memberOf` not populated | Enable memberOf overlay (OpenLDAP) |

### Diagnostic Commands

```bash
# Test connectivity
nc -zv dc01.corp.local 389

# Test bind + search
ldapsearch -x -H ldap://dc01.corp.local:389 \
  -D "svc-ggid@corp.local" -W \
  -b "DC=corp,DC=local" \
  "(sAMAccountName=jdoe)"

# Check auth container logs
docker logs ggid-auth 2>&1 | grep -i ldap

# Verify group membership
ldapsearch -x -H ldap://dc01.corp.local:389 \
  -D "svc-ggid@corp.local" -W \
  -b "CN=John Doe,OU=Users,DC=corp,DC=local" memberOf
```

---

## References

- [Configuration Reference](./configuration-reference.md) — All env vars
- [Auth Provider Guide](./plugin-development.md) — Custom auth providers
- [LDAP Directory Sync](./ldap-directory-sync.md) — Real-time/delta/full sync
- [Multi-Tenancy Guide](./multi-tenancy-guide.md) — Tenant isolation
