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
