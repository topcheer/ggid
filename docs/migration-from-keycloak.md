# Migrating from Keycloak to GGID

Step-by-step guide for migrating from Keycloak to the GGID IAM Platform.

---

## Overview

| Aspect | Keycloak | GGID |
|--------|----------|------|
| Language | Java | Go (10x lower memory) |
| Architecture | Monolith | Microservices |
| Multi-tenancy | Realms (separate schemas) | RLS (shared tables) |
| Authorization | RBAC | RBAC + ABAC hybrid |
| Audit | Database writes | NATS JetStream pipeline |
| Image size | ~600MB | ~20-35MB per service |
| Startup | 10-30s | < 2s per service |
| Admin UI | Angular (legacy) | Next.js 15 (modern) |

---

## Phase 1: Assessment

### Export Keycloak Realm

Keycloak provides a built-in realm export:

```bash
# Export realm (offline — stop Keycloak first)
/opt/keycloak/bin/kc.sh export \
  --dir /tmp/keycloak-export \
  --realm your-realm \
  --users realm_file

# Or via Docker
docker exec keycloak /opt/keycloak/bin/kc.sh export \
  --dir /tmp/export \
  --realm your-realm \
  --users realm_file
```

This produces a JSON file containing:
- Realm configuration
- Clients (applications)
- Roles
- Users (with password hashes)
- Groups
- Identity providers (SSO)

### Map Keycloak Concepts to GGID

| Keycloak | GGID | Notes |
|----------|------|-------|
| Realm | Tenant | 1:1 mapping |
| Client | OAuth Client | Redirect URIs, grant types |
| Realm Role | Role | Direct mapping |
| Client Role | Role (scoped) | Prefix with client name |
| Group | Organization | Org tree with memberships |
| User | User | Direct mapping |
| User Attributes | User Metadata | JSON key-values |
| Identity Provider | Social / OIDC Config | SAML or OIDC federation |
| Required Actions | Auth Hooks | e.g., "VERIFY_EMAIL" → email verification |
| Realm Events | Audit Events | Different action names |
| Keycloak Session | JWT Session | Stateless (no server-side session) |

---

## Phase 2: Export Users

### Parse Keycloak Export

```python
import json

with open("/tmp/keycloak-export/your-realm-realm.json") as f:
    realm = json.load(f)

users = realm.get("users", [])
print(f"Found {len(users)} users")
```

### Transform to GGID Format

```python
import uuid

TENANT_ID = "00000000-0000-0000-0000-000000000001"

def transform_user(kc_user):
    # Keycloak attributes
    attrs = kc_user.get("attributes", {})

    username = kc_user.get("username", "")
    email = kc_user.get("email", "")
    if not email:
        email = f"{username}@migrated.local"

    # Password hash — Keycloak uses PBKDF2
    # GGID uses Argon2id, so we can't directly import hashes
    credentials = kc_user.get("credentials", [])

    return {
        "username": username,
        "email": email,
        "email_verified": kc_user.get("emailVerified", False),
        "enabled": kc_user.get("enabled", True),
        "display_name": attrs.get("displayName", [username])[0] if isinstance(attrs.get("displayName"), list) else username,
        "first_name": kc_user.get("firstName", ""),
        "last_name": kc_user.get("lastName", ""),
        "metadata": {
            k: v[0] if isinstance(v, list) else v
            for k, v in attrs.items()
        },
        "tenant_id": TENANT_ID,
        "keycloak_id": kc_user["id"],
        "has_password": len(credentials) > 0,
    }

ggid_users = [transform_user(u) for u in users]

with open("ggid_users_import.json", "w") as f:
    json.dump(ggid_users, f, indent=2)

print(f"Transformed {len(ggid_users)} users")
```

### Password Migration

Keycloak uses **PBKDF2** with HMAC-SHA256. GGID uses **Argon2id**.
The hashes are not compatible. Options:

1. **Forced Reset (recommended):**
   - Import users without passwords
   - Send "Welcome to GGID — Set Your Password" email
   - Users authenticate via email link on first login

2. **Lazy Migration Bridge:**
   - Deploy a bridge that proxies login to Keycloak
   - On successful Keycloak login, create GGID user with the same password (Argon2id)
   - After 30 days, all active users have migrated; decommission Keycloak

```python
# Lazy migration bridge
def login(username, password):
    # Try GGID first
    result = ggid_login(username, password)
    if result.ok:
        return result.tokens

    # Fall back to Keycloak
    kc_result = keycloak_login(username, password)
    if kc_result.ok:
        # Create user in GGID
        ggid_register(username, password, email=kc_result.email)
        return ggid_login(username, password).tokens

    return Unauthorized
```

---

## Phase 3: Migrate Roles

### Parse Keycloak Roles

```python
def transform_role(kc_role):
    return {
        "key": kc_role["name"].lower().replace(" ", "_"),
        "name": kc_role["name"],
        "description": kc_role.get("description", ""),
    }

realm_roles = realm.get("roles", [])
ggid_roles = [transform_role(r) for r in realm_roles if not r.get("clientRole", False)]

# Client roles (prefix with client name)
client_roles = []
for client in realm.get("clients", []):
    client_id = client.get("clientId", "")
    for role in client.get("roles", []):
        transformed = transform_role(role)
        transformed["key"] = f"{client_id}_{transformed['key']}"
        client_roles.append(transformed)

all_roles = ggid_roles + client_roles
```

### Import Roles to GGID

```bash
TOKEN=$(curl -s -X POST "$GW/api/v1/auth/login" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{"username":"admin","password":"Admin@123456"}' | jq -r '.access_token')

cat ggid_roles.json | jq -c '.[]' | while read -r role; do
  curl -s -X POST "$GW/api/v1/roles" \
    -H "Authorization: Bearer $TOKEN" \
    -H "X-Tenant-ID: $TENANT_ID" \
    -H "Content-Type: application/json" \
    -d "$role"
done
```

---

## Phase 4: Migrate Groups → Organizations

Keycloak groups map to GGID organizations:

```python
def transform_group_to_org(kc_group, parent_org_id=None):
    return {
        "name": kc_group["name"],
        "parent_id": parent_org_id,
        "description": kc_group.get("attributes", {}).get("description", [""])[0],
    }

def transform_memberships(kc_group, org_id):
    memberships = []
    for member in kc_group.get("userMemberships", []):
        memberships.append({
            "org_id": org_id,
            "user_username": member,  # resolve to user_id later
        })
    return memberships
```

### Role Mapping for Groups

Keycloak allows roles on groups. Map these to GGID org-level role assignments:

| Keycloak | GGID |
|----------|------|
| Group + Role | Org Member + Role |
| Subgroup | Child Organization |
| Top-level Group | Root Organization |

---

## Phase 5: Migrate Identity Providers (SSO)

| Keycloak IdP | GGID Config |
|--------------|-------------|
| SAML IdP | OIDC/SAML federation config |
| OIDC IdP (Google) | `AUTH_GOOGLE_CLIENT_ID` + secret |
| OIDC IdP (Microsoft) | `AUTH_MICROSOFT_CLIENT_ID` + secret |
| LDAP/AD Federation | `LDAP_URL` + bind credentials |

### SAML Federation

```python
def transform_saml_idp(kc_idp):
    return {
        "entity_id": kc_idp["config"]["entityId"],
        "sso_url": kc_idp["config"]["singleSignOnServiceUrl"],
        "slo_url": kc_idp["config"].get("singleLogoutServiceUrl", ""),
        "x509_cert": kc_idp["config"]["signingCertificate"],
        "name_id_format": kc_idp["config"].get("nameIDPolicyFormat", "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress"),
    }
```

### LDAP Configuration

```bash
# Keycloak LDAP federation → GGID LDAP env vars
LDAP_URL=ldap://your-ad-server:389
LDAP_BIND_DN=cn=svc-ggid,dc=corp,dc=local
LDAP_BIND_PASSWORD=your-bind-password
LDAP_BASE_DN=dc=corp,dc=local
LDAP_USER_FILTER=(sAMAccountName=%s)
LDAP_START_TLS=true
LDAP_AUTO_PROVISION=true
```

---

## Phase 6: Migrate Clients (Applications)

```python
def transform_client(kc_client):
    redirect_uris = []
    for redirect in kc_client.get("redirectUris", []):
        redirect_uris.append(redirect)

    return {
        "name": kc_client.get("clientId", ""),
        "client_id": kc_client.get("clientId", ""),
        "client_secret": kc_client.get("secret", ""),
        "redirect_uris": redirect_uris,
        "grant_types": kc_client.get("standardFlowEnabled", True) and \
                       ["authorization_code", "refresh_token"] or \
                       ["client_credentials"],
        "scopes": ["openid", "profile", "email"],
    }
```

---

## Phase 7: Migrate Required Actions → Auth Hooks

| Keycloak Required Action | GGID Hook |
|--------------------------|-----------|
| `VERIFY_EMAIL` | Built-in email verification |
| `CONFIGURE_TOTP` | Built-in MFA setup |
| `UPDATE_PASSWORD` | Built-in password policy |
| `UPDATE_PROFILE` | `pre-registration` hook |
| Custom required action | `post-login` hook |

---

## Phase 8: Cutover

### Parallel Run Strategy

```
Week 1: Keycloak primary, GGID imported (read-only verification)
Week 2: GGID primary (with lazy migration to Keycloak for stragglers)
Week 3: Keycloak decommissioned
```

### Post-Migration Checklist

- [ ] All users can log in to GGID
- [ ] Roles and permissions verified
- [ ] Organizations match Keycloak group hierarchy
- [ ] SAML/OIDC federation providers working
- [ ] LDAP connectivity verified
- [ ] OAuth clients updated in all applications
- [ ] Audit events flowing correctly
- [ ] Keycloak realm exported as backup
- [ ] Keycloak service stopped
- [ ] DNS updated to GGID Gateway

---

## API Mapping

| Keycloak API | GGID API |
|--------------|----------|
| `POST /realms/{r}/protocol/openid-connect/token` | `POST /api/v1/auth/login` |
| `GET /realms/{r}/users` | `GET /api/v1/users` |
| `POST /realms/{r}/users` | `POST /api/v1/users` |
| `GET /realms/{r}/users/{id}` | `GET /api/v1/users/{id}` |
| `PUT /realms/{r}/users/{id}` | `PUT /api/v1/users/{id}` |
| `DELETE /realms/{r}/users/{id}` | `DELETE /api/v1/users/{id}` |
| `GET /realms/{r}/roles` | `GET /api/v1/roles` |
| `POST /realms/{r}/roles` | `POST /api/v1/roles` |
| `POST /realms/{r}/users/{id}/role-mappings/realm` | `POST /api/v1/users/{id}/roles` |
| `GET /realms/{r}/groups` | `GET /api/v1/orgs` |
| `GET /realms/{r}/events` | `GET /api/v1/audit/events` |

---

## Realm Export to GGID Import

### Step 1: Export Keycloak Realm

```bash
# Export realm as JSON (run on Keycloak server)
/opt/keycloak/bin/kc.sh export \
  --realm my-realm \
  --dir /tmp/realm-export \
  --users realm_file

# This creates:
# /tmp/realm-export/my-realm-realm.json
# /tmp/realm-export/my-realm-users-0.json
```

### Step 2: Understand the Export Structure

```json
{
  "realm": "my-realm",
  "enabled": true,
  "users": [
    {
      "id": "kc-uuid-1",
      "username": "john.doe",
      "email": "john@example.com",
      "enabled": true,
      "emailVerified": true,
      "firstName": "John",
      "lastName": "Doe",
      "credentials": [
        { "type": "password", "hashedSaltedValue": "...", "salt": "..." }
      ],
      "realmRoles": ["user", "editor"],
      "clientRoles": { "my-app": ["admin"] },
      "groups": ["/Engineering/Backend"]
    }
  ],
  "roles": {
    "realm": [
      { "name": "user", "description": "Standard user" },
      { "name": "admin", "description": "Administrator" }
    ]
  },
  "groups": [
    {
      "name": "Engineering",
      "subGroups": [
        { "name": "Backend" },
        { "name": "Frontend" }
      ]
    }
  ],
  "clients": [
    {
      "clientId": "my-app",
      "enabled": true,
      "protocol": "openid-connect",
      "redirectUris": ["https://app.example.com/callback"],
      "secret": "******",
      "standardFlowEnabled": true,
      "directAccessGrantsEnabled": true,
      "bearerOnly": false
    }
  ]
}
```

### Step 3: Conversion Script

```python
#!/usr/bin/env python3
"""
Convert Keycloak realm export to GGID import format.
Usage: python3 convert_keycloak.py <realm.json> > ggid-import.json
"""

import json
import sys
import uuid

def convert_keycloak_to_ggid(kc_data):
    """Convert Keycloak realm export to GGID import format."""

    # --- Users ---
    ggid_users = []
    for kc_user in kc_data.get('users', []):
        ggid_user = {
            'external_id': kc_user['id'],
            'username': kc_user['username'],
            'email': kc_user.get('email', ''),
            'name': f"{kc_user.get('firstName', '')} {kc_user.get('lastName', '')}".strip(),
            'status': 'active' if kc_user.get('enabled', True) else 'suspended',
            'email_verified': kc_user.get('emailVerified', False),
            # Keycloak stores hashed passwords — can't migrate plaintext
            # Users will need to reset passwords after import
            'require_password_reset': True,
            'roles': kc_user.get('realmRoles', []),
            'groups': kc_user.get('groups', []),
        }
        ggid_users.append(ggid_user)

    # --- Roles ---
    ggid_roles = []
    for kc_role in kc_data.get('roles', {}).get('realm', []):
        ggid_roles.append({
            'key': kc_role['name'],
            'name': kc_role.get('description', kc_role['name']),
            'description': kc_role.get('description', ''),
        })

    # --- Groups → Organizations ---
    ggid_orgs = []
    for kc_group in kc_data.get('groups', []):
        ggid_orgs.append({
            'name': kc_group['name'],
            'display_name': kc_group['name'],
        })
        # Sub-groups become child orgs
        for sub in kc_group.get('subGroups', []):
            ggid_orgs.append({
                'name': f"{kc_group['name']}/{sub['name']}",
                'display_name': sub['name'],
                'parent': kc_group['name'],
            })

    # --- Clients → OAuth Clients ---
    ggid_clients = []
    for kc_client in kc_data.get('clients', []):
        if not kc_client.get('enabled', True):
            continue
        # Skip built-in Keycloak clients
        if kc_client['clientId'] in ['account', 'admin-cli', 'broker', 'realm-management', 'security-admin-console']:
            continue

        ggid_clients.append({
            'name': kc_client['clientId'],
            'client_id': kc_client['clientId'],
            'client_secret': kc_client.get('secret', ''),
            'redirect_uris': kc_client.get('redirectUris', []),
            'grant_types': convert_grant_types(kc_client),
            'scopes': ['openid', 'profile', 'email'],
        })

    return {
        'format': 'ggid-import-v1',
        'tenant': kc_data['realm'],
        'users': ggid_users,
        'roles': ggid_roles,
        'organizations': ggid_orgs,
        'oauth_clients': ggid_clients,
    }

def convert_grant_types(kc_client):
    """Map Keycloak flow flags to OAuth grant types."""
    grants = []
    if kc_client.get('standardFlowEnabled'):
        grants.append('authorization_code')
    if kc_client.get('directAccessGrantsEnabled'):
        grants.append('password')
    if kc_client.get('serviceAccountsEnabled'):
        grants.append('client_credentials')
    if kc_client.get('implicitFlowEnabled'):
        grants.append('implicit')  # Note: GGID may reject this (OAuth 2.1)
    return grants

if __name__ == '__main__':
    with open(sys.argv[1]) as f:
        kc_data = json.load(f)

    ggid_data = convert_keycloak_to_ggid(kc_data)
    json.dump(ggid_data, sys.stdout, indent=2)
```

### Step 4: Import to GGID

```bash
# Convert
python3 convert_keycloak.py my-realm-realm.json > ggid-import.json

# Import users
for user in $(jq -c '.users[]' ggid-import.json); do
    curl -X POST $API/api/v1/users \
        -H "Authorization: Bearer $ADMIN_TOKEN" \
        -H "X-Tenant-ID: $TENANT_ID" \
        -H "Content-Type: application/json" \
        -d "$user"
done

# Import roles
for role in $(jq -c '.roles[]' ggid-import.json); do
    curl -X POST $API/api/v1/roles \
        -H "Authorization: Bearer $ADMIN_TOKEN" \
        -H "X-Tenant-ID: $TENANT_ID" \
        -d "$role"
done

# Import OAuth clients
for client in $(jq -c '.oauth_clients[]' ggid-import.json); do
    curl -X POST $API/api/v1/oauth/clients \
        -H "Authorization: Bearer $ADMIN_TOKEN" \
        -H "X-Tenant-ID: $TENANT_ID" \
        -d "$client"
done
```

### Step 5: SAML Client Migration

Keycloak SAML clients map to GGID SAML service providers:

```bash
# Register SAML SP in GGID for each Keycloak SAML client
curl -X POST $API/api/v1/saml/sp \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "X-Tenant-ID: $TENANT_ID" \
    -d '{
        "name": "My SAML App",
        "entity_id": "https://app.example.com",
        "assertion_consumer_service_url": "https://app.example.com/saml/acs",
        "name_id_format": "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress"
    }'

# Point the app to GGID IdP metadata:
# https://iam.example.com/saml/metadata
```

---

## User Migration Considerations

### Password Migration

Keycloak stores passwords using PBKDF2 with realm-specific salts. GGID uses
bcrypt (cost 12) or Argon2id. **Passwords cannot be directly migrated.**

**Options:**

1. **Force password reset (recommended):** Set `require_password_reset: true`
   on all imported users. Send password reset emails.

2. **Lazy migration:** Store the Keycloak hash in a temporary field. On first
   login, attempt to verify against the old hash. If successful, hash the
   password with bcrypt and store it.

```go
// Lazy migration logic (in auth service)
func (s *AuthService) Login(ctx context.Context, username, password string) (*Token, error) {
    user, err := s.repo.GetByUsername(ctx, username)
    if err != nil {
        return nil, err
    }

    // Check if password is in old Keycloak format
    if user.PasswordHash == "" && user.LegacyPasswordHash != "" {
        if verifyKeycloakPassword(password, user.LegacyPasswordHash, user.LegacySalt) {
            // Migrate to bcrypt
            newHash, _ := bcrypt.GenerateFromPassword([]byte(password), 12)
            s.repo.UpdatePassword(ctx, user.ID, string(newHash), "")
            // Login successful
            return s.issueToken(user)
        }
        return nil, ErrInvalidCredentials
    }

    // Normal bcrypt verification
    if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
        return nil, ErrInvalidCredentials
    }
    return s.issueToken(user)
}
```

### Role Mapping

| Keycloak | GGID |
|----------|------|
| `realmRoles` | Role assignments (via `POST /users/:id/roles`) |
| `clientRoles` | Mapped to scoped roles per OAuth client |
| `groups` | Organizations (via `POST /orgs/:id/members`) |
| `compositeRoles` | Role hierarchy (parent → child roles) |

---

## Migration Checklist

- [ ] Export Keycloak realm JSON
- [ ] Run conversion script (`convert_keycloak.py`)
- [ ] Review converted output for completeness
- [ ] Create GGID tenant
- [ ] Import roles
- [ ] Import users (with password reset flag)
- [ ] Import OAuth clients (update secrets if possible)
- [ ] Import SAML service providers
- [ ] Import groups as organizations
- [ ] Configure GGID IdP metadata URL in applications
- [ ] Send password reset emails to all users
- [ ] Verify login flows (password, OAuth, SAML)
- [ ] Monitor for 24 hours
- [ ] Decommission Keycloak
