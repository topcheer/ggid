# LDAP Directory Synchronization

Guide for configuring real-time and scheduled directory synchronization
between LDAP/Active Directory and GGID. Covers provider configuration, AD vs
OpenLDAP differences, auto-provisioning, group-to-role mapping, delta sync,
and START_TLS encryption.

> For basic LDAP authentication (login only without sync), see
> [LDAP Integration Guide](ldap-integration-guide.md).

---

## Table of Contents

- [Overview](#overview)
- [Sync Architecture](#sync-architecture)
- [Provider Configuration](#provider-configuration)
- [Active Directory vs OpenLDAP](#active-directory-vs-openldap)
- [Auto-Provisioning](#auto-provisioning)
- [Group-to-Role Mapping](#group-to-role-mapping)
- [Delta Sync](#delta-sync)
- [Full Sync](#full-sync)
- [START_TLS Configuration](#start_tls-configuration)
- [Monitoring](#monitoring)
- [Troubleshooting](#troubleshooting)

---

## Overview

Directory synchronization keeps GGID user accounts, group memberships, and role
assignments in sync with an external LDAP/AD directory. Unlike simple LDAP
authentication (which only validates passwords at login), directory sync:

- Pre-provisions users before first login
- Maps LDAP groups to GGID roles automatically
- Detects disabled/deleted accounts in real time
- Maintains consistent identity data across systems

### Sync Modes

| Mode | Trigger | Latency | Use Case |
|------|---------|---------|----------|
| **Real-time** | LDAP persistent search | < 1 second | High-security, immediate deprovisioning |
| **Scheduled delta** | Cron interval (default 5 min) | 1-5 minutes | Most deployments |
| **Scheduled full** | Cron interval (default 1 hour) | 5-30 minutes | Reconciliation, drift correction |
| **Manual** | Admin API call | On demand | Troubleshooting, initial import |

---

## Sync Architecture

```
LDAP/AD Directory                    GGID
  │                                    │
  │  1. Scheduled or real-time query   │
  │◄───────────────────────────────────┤
  │  2. Changed entries (LDAP results) │
  │───────────────────────────────────►│
  │                                    │ 3. Map attributes
  │                                    │ 4. Create/update/deactivate user
  │                                    │ 5. Sync group memberships
  │                                    │ 6. Apply role mappings
  │                                    │ 7. Emit audit events
  │                                    │
```

### Sync Pipeline

```
Fetch → Parse → Transform → Diff → Apply → Audit
  │        │         │         │       │       │
  │        │         │         │       │       └── Emit event to NATS
  │        │         │         │       └── Create/Update/Deactivate
  │        │         │         └── Compare with stored state
  │        │         └── Map LDAP attrs → GGID schema
  │         └── Parse LDAP entry
  └── Query LDAP (search scope, filter)
```

---

## Provider Configuration

### Configuration via Environment Variables

```bash
# .env
LDAP_URL=ldap://dc01.corp.example.com:389
LDAP_BIND_DN=CN=ggid-sync,OU=Service Accounts,DC=corp,DC=example,DC=com
LDAP_BIND_PASSWORD=secure-password
LDAP_BASE_DN=DC=corp,DC=example,DC=com
LDAP_USER_FILTER=(objectClass=user)
LDAP_GROUP_FILTER=(objectClass=group)
LDAP_USER_ID_ATTR=sAMAccountName
LDAP_EMAIL_ATTR=mail
LDAP_DISPLAY_NAME_ATTR=displayName
LDAP_START_TLS=true
LDAP_AUTO_PROVISION=true
LDAP_SYNC_ENABLED=true
LDAP_SYNC_INTERVAL=5m
LDAP_FULL_SYNC_INTERVAL=1h
LDAP_GROUP_ROLE_MAPPING=CN=GGID-Admins:*:admin;CN=GGID-Developers:*:developer;CN=GGID-Viewers:*:viewer
```

### Configuration via YAML

```yaml
ldap:
  sync:
    enabled: true
    mode: "delta"              # delta | realtime | full
    interval: "5m"
    full_sync_interval: "1h"
    batch_size: 500
    timeout: "30s"

  connection:
    url: "ldap://dc01.corp.example.com:389"
    bind_dn: "CN=ggid-sync,OU=Service Accounts,DC=corp,DC=example,DC=com"
    bind_password: "${LDAP_BIND_PASSWORD}"
    start_tls: true
    insecure_skip_verify: false
    pool_size: 10

  user_sync:
    base_dn: "DC=corp,DC=example,DC=com"
    filter: "(objectClass=user)"
    scope: "subtree"
    id_attr: "sAMAccountName"
    email_attr: "mail"
    display_name_attr: "displayName"
    first_name_attr: "givenName"
    last_name_attr: "sn"
    status_attr: "userAccountControl"
    active_value: "512"        # AD: NORMAL_ACCOUNT
    inactive_value: "514"      # AD: ACCOUNTDISABLE
    last_modified_attr: "whenChanged"

  group_sync:
    base_dn: "DC=corp,DC=example,DC=com"
    filter: "(objectClass=group)"
    scope: "subtree"
    member_attr: "member"
    nested_groups: true
    max_depth: 5
```

---

## Active Directory vs OpenLDAP

### Attribute Mapping Differences

| GGID Field | Active Directory | OpenLDAP |
|------------|-----------------|----------|
| User ID | `sAMAccountName` | `uid` |
| Email | `mail` | `mail` |
| Display Name | `displayName` | `cn` |
| First Name | `givenName` | `givenName` |
| Last Name | `sn` | `sn` |
| Phone | `telephoneNumber` | `telephoneNumber` |
| Department | `department` | `ou` |
| Manager | `manager` | `manager` |
| Status | `userAccountControl` (bitmask) | `nsAccountLock` (boolean) |
| Last Modified | `whenChanged` | `modifyTimestamp` |
| Distinguished Name | `distinguishedName` | `dn` |
| Member Of | `memberOf` | `memberOf` |

### Active Directory Specifics

```yaml
ldap:
  user_sync:
    # AD uses sAMAccountName for login IDs
    id_attr: "sAMAccountName"
    # AD userAccountControl bitmask:
    #   512   = NORMAL_ACCOUNT (active)
    #   514   = ACCOUNTDISABLE (disabled)
    #   66048 = NORMAL_ACCOUNT + PASSWD_DONT_EXPIRE
    #   66050 = ACCOUNTDISABLE + PASSWD_DONT_EXPIRE
    status_attr: "userAccountControl"
    active_mask: "2"           # bit 1 = ACCOUNTDISABLE; if unset, account is active
    # AD uses whenChanged for delta detection
    last_modified_attr: "whenChanged"
    # AD user filter
    filter: "(&(objectClass=user)(objectCategory=person))"
  group_sync:
    # AD groups use member attribute
    member_attr: "member"
    filter: "(objectClass=group)"
```

### OpenLDAP Specifics

```yaml
ldap:
  user_sync:
    id_attr: "uid"
    filter: "(objectClass=inetOrgPerson)"
    status_attr: "nsAccountLock"
    active_value: "FALSE"
    inactive_value: "TRUE"
    last_modified_attr: "modifyTimestamp"
  group_sync:
    member_attr: "member"
    filter: "(objectClass=groupOfNames)"
```

### Nested Groups

AD supports nested groups (a group can be a member of another group). GGID
resolves nested memberships:

```yaml
ldap:
  group_sync:
    nested_groups: true
    max_depth: 5               # Prevent infinite loops
```

```
Group: GGID-Admins
  └── Member: CN=GGID-Platform,OU=Groups,DC=corp
       └── Member: CN=jane.doe,OU=Users,DC=corp  → Resolved as admin
```

OpenLDAP `groupOfNames` does not support nesting by default. Use
`groupOfURLs` or `memberOf` overlay for dynamic groups.

---

## Auto-Provisioning

When auto-provisioning is enabled, GGID automatically creates user accounts
for LDAP users who don't exist in GGID yet.

### Provisioning Flow

```
1. User attempts login (username + password)
2. GGID binds to LDAP, authenticates user
3. User not found in GGID database
4. Auto-provision:
   a. Read user attributes from LDAP
   b. Create GGID user record
   c. Assign default tenant
   d. Sync group memberships
   e. Apply role mappings
5. Issue JWT, user logged in
```

### Configuration

```yaml
ldap:
  auto_provision: true
  provisioning:
    default_tenant: "00000000-0000-0000-0000-000000000001"
    default_status: "active"
    default_roles: ["viewer"]
    link_existing_by_email: true
    update_on_login: true      # Re-sync attrs on each login
```

### Attribute Sync on Login

```yaml
ldap:
  provisioning:
    update_on_login: true
    sync_attributes:
      - email
      - display_name
      - first_name
      - last_name
      - phone
      - department
    sync_groups: true
```

| Setting | Behavior |
|---------|----------|
| `update_on_login: true` | Re-fetch LDAP attrs every login |
| `update_on_login: false` | Only sync at provisioning time; schedule handles updates |
| `sync_groups: true` | Re-evaluate group memberships on each login |

### JIT (Just-In-Time) Provisioning

Auto-provisioning is JIT: the account is created at the moment of first login.
No pre-creation or batch import needed.

---

## Group-to-Role Mapping

GGID maps LDAP groups to GGID roles, enabling automatic role assignment based
on directory membership.

### Configuration Format

```
LDAP_GROUP_DN:TENANT_ID:ROLE_NAME
```

### Example

```bash
LDAP_GROUP_ROLE_MAPPING="\
CN=GGID-Admins,OU=Groups,DC=corp,DC=example,DC=com:00000000-0000-0000-0000-000000000001:admin;\
CN=GGID-Developers,OU=Groups,DC=corp,DC=example,DC=com:00000000-0000-0000-0000-000000000001:developer;\
CN=GGID-Viewers,OU=Groups,DC=corp,DC=example,DC=com:00000000-0000-0000-0000-000000000001:viewer;\
CN=GGID-Security,OU=Groups,DC=corp,DC=example,DC=com:00000000-0000-0000-0000-000000000001:security_admin"
```

### YAML Configuration

```yaml
ldap:
  group_role_mapping:
    - ldap_group: "CN=GGID-Admins,OU=Groups,DC=corp,DC=example,DC=com"
      tenant_id: "00000000-0000-0000-0000-000000000001"
      role: "admin"

    - ldap_group: "CN=GGID-Developers,OU=Groups,DC=corp,DC=example,DC=com"
      tenant_id: "00000000-0000-0000-0000-000000000001"
      role: "developer"

    - ldap_group: "CN=Finance-Users,OU=Groups,DC=corp,DC=example,DC=com"
      tenant_id: "00000000-0000-0000-0000-000000000002"
      role: "viewer"
```

### Mapping Resolution

During sync, GGID:

1. Fetches user's `memberOf` groups from LDAP
2. Resolves nested groups (if enabled)
3. Matches group DNs against configured mappings
4. **Grants** roles for matching groups
5. **Revokes** roles previously assigned via sync but no longer matching

### Dynamic Role Sync Example

```
Before sync:
  User: jane.doe
  LDAP groups: GGID-Admins, Finance-Team
  GGID roles: admin (from sync), viewer (manual)

After LDAP change (removed from GGID-Admins):
  User: jane.doe
  LDAP groups: Finance-Team
  GGID roles: viewer (manual only — admin revoked by sync)
```

### Mapping Priority

When a user belongs to multiple mapped groups, the **highest-privilege** role
wins:

| Priority | Role |
|----------|------|
| 1 (highest) | `super_admin` |
| 2 | `admin` |
| 3 | `security_admin` |
| 4 | `developer` |
| 5 (lowest) | `viewer` |

---

## Delta Sync

Delta sync fetches only entries that changed since the last sync, using the
directory's modification timestamp.

### How It Works

```
1. Read last_sync_timestamp from GGID database
2. Query LDAP: (whenChanged >= last_sync_timestamp)
3. Process changed entries
4. Update last_sync_timestamp to query execution time
```

### LDAP Query

```ldap
# Active Directory
(&(objectClass=user)(whenChanged>=20240115000000.0Z))

# OpenLDAP
(&(objectClass=inetOrgPerson)(modifyTimestamp>=20240115000000Z))
```

### Configuration

```yaml
ldap:
  sync:
    mode: "delta"
    interval: "5m"
    last_modified_attr: "whenChanged"
    # AD timestamp format: YYYYMMDDHHMMSS.0Z
    # OpenLDAP timestamp format: YYYYMMDDHHMMSSZ
    timestamp_format: "ad"
```

### Delta Sync Events

| LDAP Change | GGID Action |
|-------------|-------------|
| New user created | Auto-provision user |
| User attributes modified | Update user record |
| User disabled (UAC=514) | Deactivate user, revoke sessions |
| User deleted from LDAP | Deactivate user (configurable: delete vs deactivate) |
| Group membership added | Grant mapped role |
| Group membership removed | Revoke mapped role |
| Group created/deleted | Create/remove group in GGID |

### Handling Tombstones

AD creates tombstone objects when entries are deleted. GGID filters these:

```ldap
(&(!(isDeleted=TRUE))(objectClass=user)(whenChanged>=...))
```

For hard deletions (no tombstone), GGID detects missing users during full sync.

---

## Full Sync

Full sync reconciles the entire directory, detecting drift that delta sync may
miss (e.g., clock skew, missed entries, corrupted sync state).

### When to Run Full Sync

- After initial setup
- After LDAP server migration
- When delta sync count seems off
- Weekly as a safety net (configurable)

### Configuration

```yaml
ldap:
  sync:
    full_sync_interval: "24h"
    full_sync_batch_size: 1000
    # On missing user in LDAP:
    missing_user_action: "deactivate"  # deactivate | delete | ignore
```

### Full Sync Logic

```
1. Fetch all LDAP users in scope
2. Fetch all GGID users from LDAP source
3. Diff the two sets:
   a. In LDAP but not GGID → provision
   b. In both but attributes differ → update
   c. In GGID but not LDAP → deactivate/delete
4. Fetch all LDAP groups
5. Sync group memberships for each user
6. Apply role mappings
7. Update sync metadata
```

### Manual Full Sync

```bash
curl -X POST https://iam.example.com/api/v1/admin/ldap/sync \
  -H "Authorization: Bearer <admin-token>" \
  -d '{ "mode": "full" }'
```

---

## START_TLS Configuration

START_TLS upgrades a plain LDAP connection (port 389) to encrypted TLS,
providing the same security as LDAPS (port 636) while allowing the same port
for both encrypted and unencrypted connections.

### Why START_TLS over LDAPS?

| Feature | START_TLS (389) | LDAPS (636) |
|---------|-----------------|-------------|
| Encryption | Negotiated after connect | Always encrypted |
| Port | 389 (standard) | 636 (deprecated) |
| Certificate validation | Configurable | Configurable |
| Connection pooling | Better (plain first, then upgrade) | Direct TLS pool |
| AD recommendation | Preferred | Legacy |

### Configuration

```yaml
ldap:
  connection:
    url: "ldap://dc01.corp.example.com:389"   # Plain LDAP URL
    start_tls: true
    tls_config:
      min_version: "1.2"
      max_version: "1.3"
      server_name: "dc01.corp.example.com"    # For cert validation
      ca_cert: "/etc/ggid/ldap-ca.pem"        # Internal CA certificate
      insecure_skip_verify: false             # Never true in production
```

### Certificate Validation

```go
// GGID validates the LDAP server certificate during START_TLS:
tlsConfig := &tls.Config{
    ServerName:         "dc01.corp.example.com",
    MinVersion:         tls.VersionTLS12,
    RootCAs:            caCertPool,    // Internal CA
    InsecureSkipVerify: false,         // MUST be false in production
}
```

### Common Certificate Issues

| Issue | Cause | Fix |
|-------|-------|-----|
| "certificate signed by unknown authority" | Internal CA not trusted | Add CA cert to `ca_cert` |
| "x509: cannot validate certificate for X" | SAN mismatch | Set `server_name` to match cert SAN |
| "tls: handshake failure" | Version mismatch | Check `min_version` / `max_version` |

---

## Monitoring

### Sync Metrics

GGID exposes sync metrics via Prometheus:

```
# Users synced in last cycle
g gid_ldap_sync_users_total{direction="delta"} 142
ggid_ldap_sync_users_total{direction="full"} 1523

# Sync duration
g gid_ldap_sync_duration_seconds{direction="delta"} 2.34
g gid_ldap_sync_duration_seconds{direction="full"} 45.6

# Errors
g gid_ldap_sync_errors_total{type="connection"} 0
g gid_ldap_sync_errors_total{type="parse"} 0
g gid_ldap_sync_errors_total{type="provision"} 3

# Last successful sync
g gid_ldap_sync_last_success_timestamp_seconds 1705312200
```

### Alerting Rules

```yaml
groups:
  - name: ldap-sync
    rules:
      - alert: LDAPSyncStale
        expr: time() - ggid_ldap_sync_last_success_timestamp_seconds > 900
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "LDAP sync has not completed in 15+ minutes"

      - alert: LDAPSyncErrors
        expr: rate(ggid_ldap_sync_errors_total[5m]) > 0.1
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "LDAP sync error rate is high"

      - alert: LDAPConnectionFailed
        expr: ggid_ldap_connection_status == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Cannot connect to LDAP server"
```

### Sync Status API

```bash
curl https://iam.example.com/api/v1/admin/ldap/sync/status \
  -H "Authorization: Bearer <admin-token>"
```

```json
{
  "status": "healthy",
  "mode": "delta",
  "last_delta_sync": "2024-01-15T10:05:00Z",
  "last_full_sync": "2024-01-15T06:00:00Z",
  "next_delta_sync": "2024-01-15T10:10:00Z",
  "next_full_sync": "2024-01-16T06:00:00Z",
  "stats": {
    "total_users_synced": 1523,
    "total_groups_synced": 47,
    "users_provisioned": 12,
    "users_deactivated": 3,
    "roles_assigned": 34,
    "roles_revoked": 8
  },
  "errors": []
}
```

---

## Troubleshooting

### "LDAP connection failed"

| Cause | Fix |
|-------|-----|
| Wrong URL | Verify `LDAP_URL` format: `ldap://` or `ldaps://` |
| Firewall blocking | Ensure port 389/636 is open |
| DNS resolution | Verify LDAP server FQDN resolves |

### "Invalid credentials" on bind

| Cause | Fix |
|-------|-----|
| Wrong bind DN | Check service account DN format |
| Password expired | Reset service account password in AD |
| Account locked | Unlock service account |

### Delta sync missing changes

| Cause | Fix |
|-------|-----|
| Clock skew between GGID and LDAP | Sync NTP on both servers |
| `whenChanged` not indexed | Add index in AD: `whenChanged` attribute |
| Timestamp format mismatch | Check `timestamp_format` (ad vs openldap) |
| Large batch exceeded | Reduce `batch_size` |

### Group mapping not applied

| Cause | Fix |
|-------|-----|
| Group DN case mismatch | LDAP DNs are case-insensitive; verify exact format |
| Nested group beyond max_depth | Increase `max_depth` or flatten groups |
| User not in expected group | Check `memberOf` in LDAP directly |

### "Size limit exceeded"

```bash
# Increase server-side limit or page results
LDAP_SYNC_PAGE_SIZE=1000
```

AD default limit is 1000 entries. Configure GGID to use paged results:

```yaml
ldap:
  sync:
    paging: true
    page_size: 500
```
