# Migrating from Auth0 to GGID

Step-by-step guide for migrating from Auth0 to the GGID IAM Platform.

---

## Overview

| Aspect | Auth0 | GGID |
|--------|-------|------|
| Deployment | SaaS only | Self-hosted (Docker/K8s) |
| Cost | Per-user pricing | Free (Apache 2.0) |
| Data residency | Auth0 servers | Your infrastructure |
| Authorization | RBAC only | RBAC + ABAC hybrid |
| Audit | proprietary | NATS JetStream (open) |
| SDK | Node.js, Python | Go, Node.js, Java, Python |

---

## Phase 1: Assessment (1-2 days)

### Export Auth0 Configuration

```bash
# Install Auth0 CLI
npm install -g auth0-cli

# Export tenants
auth0 tenants list

# Export applications
auth0 apps list --json > auth0_apps.json

# Export connections (social + DB)
auth0 connections list --json > auth0_connections.json

# Export rules / actions
auth0 actions list --json > auth0_actions.json

# Export roles
auth0 roles list --json > auth0_roles.json

# Export users (via Management API)
curl -H "Authorization: Bearer $AUTH0_MGMT_TOKEN" \
  "https://YOUR_DOMAIN/api/v2/users?per_page=100" > auth0_users.json
```

### Map Auth0 Concepts to GGID

| Auth0 | GGID | Notes |
|-------|------|-------|
| Tenant | Tenant | 1:1 mapping |
| Application | OAuth Client | Redirect URIs, grant types |
| Connection | Auth Provider | DB connection → Local; Social → Social config |
| Rule / Action | Auth Hook | Pre/post-registration, post-login hooks |
| Role | Role | Direct mapping (key = role name) |
| Permission | Role Permission | `resource:action` format |
| User Metadata | User Metadata | JSON key-values |
| Management API | Admin REST API | Different endpoints |
| Actions marketplace | Plugin system | Phase 12 |

---

## Phase 2: Export Auth0 Users

### Export Script

```python
import json, requests, os

AUTH0_DOMAIN = os.environ["AUTH0_DOMAIN"]
MGMT_TOKEN = os.environ["AUTH0_MGMT_TOKEN"]

def export_users():
    users = []
    page = 0
    while True:
        resp = requests.get(
            f"https://{AUTH0_DOMAIN}/api/v2/users",
            params={"per_page": 100, "page": page},
            headers={"Authorization": f"Bearer {MGMT_TOKEN}"},
        )
        batch = resp.json()
        if not batch:
            break
        users.extend(batch)
        page += 1
    return users

users = export_users()
with open("auth0_users_export.json", "w") as f:
    json.dump(users, f, indent=2)
print(f"Exported {len(users)} users")
```

### Transform to GGID Format

```python
import json, uuid

TENANT_ID = "00000000-0000-0000-0000-000000000001"

def transform_user(auth0_user):
    return {
        "username": auth0_user.get("nickname") or auth0_user["email"].split("@")[0],
        "email": auth0_user["email"],
        "email_verified": auth0_user.get("email_verified", False),
        "display_name": auth0_user.get("name", ""),
        "password_hash": auth0_user.get("_password_hash", ""),  # Auth0 doesn't export hashes
        "metadata": auth0_user.get("user_metadata", {}),
        "tenant_id": TENANT_ID,
        "auth0_id": auth0_user["user_id"],  # for reference
    }

with open("auth0_users_export.json") as f:
    auth0_users = json.load(f)

ggid_users = [transform_user(u) for u in auth0_users]
with open("ggid_users_import.json", "w") as f:
    json.dump(ggid_users, f, indent=2)
```

---

## Phase 3: Import Users to GGID

### Option A: Bulk Import via API

```bash
GGID_URL="http://localhost:8080"
TENANT_ID="00000000-0000-0000-0000-000000000001"

# Login as admin
TOKEN=$(curl -s -X POST "$GGID_URL/api/v1/auth/login" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"Admin@123456"}' | jq -r '.access_token')

# Import users
cat ggid_users_import.json | jq -c '.[]' | while read -r user; do
  curl -s -X POST "$GGID_URL/api/v1/auth/register" \
    -H "Authorization: Bearer $TOKEN" \
    -H "X-Tenant-ID: $TENANT_ID" \
    -H "Content-Type: application/json" \
    -d "$user"
done
```

### Option B: Direct Database Import

For large user bases (> 10,000), use direct SQL:

```sql
-- Insert users directly
INSERT INTO users (id, tenant_id, username, email, email_verified, display_name, created_at)
SELECT
  gen_random_uuid(),
  '00000000-0000-0000-0000-000000000001',
  nickname,
  email,
  email_verified,
  name,
  NOW()
FROM temp_auth0_users;

-- Insert credentials with temporary passwords
-- Users will be forced to reset on first login
INSERT INTO credentials (tenant_id, user_id, type, identifier, secret)
SELECT
  '00000000-0000-0000-0000-000000000001',
  u.id, 'password', u.username,
  '<argon2id-hash-of-temp-password>'
FROM users u
JOIN temp_auth0_users t ON u.email = t.email;
```

### Password Migration Strategy

Auth0 does **not** export password hashes. Two strategies:

1. **Forced Reset**: Import users with a temporary password, send "Reset your password" email
2. **Lazy Migration**: Deploy a bridge endpoint that intercepts login attempts — if GGID doesn't recognize the user, proxy to Auth0's API to verify the password, then create the user in GGID with the verified password

```python
# Lazy migration bridge (pseudo-code)
def login(username, password):
    # Try GGID first
    if ggid_login(username, password).ok:
        return ggid_tokens

    # Fall back to Auth0
    auth0_result = auth0_login(username, password)
    if auth0_result.ok:
        # Create user in GGID with this password
        ggid_register(username, password)
        return ggid_login(username, password)

    return unauthorized()
```

---

## Phase 4: Migrate Roles & Permissions

### Export Auth0 Roles

```bash
auth0 roles list --json > auth0_roles.json
```

### Transform and Import

```python
import json, requests

GGID_URL = "http://localhost:8080"
TENANT_ID = "00000000-0000-0000-0000-000000000001"
TOKEN = "..."

def transform_role(auth0_role):
    return {
        "key": auth0_role["name"].lower().replace(" ", "_"),
        "name": auth0_role["name"],
        "description": auth0_role.get("description", ""),
    }

def create_role(role_data):
    resp = requests.post(
        f"{GGID_URL}/api/v1/roles",
        headers={
            "Authorization": f"Bearer {TOKEN}",
            "X-Tenant-ID": TENANT_ID,
        },
        json=role_data,
    )
    return resp.json()

with open("auth0_roles.json") as f:
    auth0_roles = json.load(f)

for role in auth0_roles:
    ggid_role = transform_role(role)
    result = create_role(ggid_role)
    print(f"Created role: {result.get('key', 'FAILED')}")
```

---

## Phase 5: Migrate Social Connections

| Auth0 Connection | GGID Config |
|------------------|-------------|
| `google-oauth2` | `AUTH_GOOGLE_CLIENT_ID` + `AUTH_GOOGLE_CLIENT_SECRET` |
| `github` | `AUTH_GITHUB_CLIENT_ID` + `AUTH_GITHUB_CLIENT_SECRET` |
| `windowslive` | `AUTH_MICROSOFT_CLIENT_ID` + `AUTH_MICROSOFT_CLIENT_SECRET` |

```bash
# Set social login env vars in auth service
AUTH_GOOGLE_CLIENT_ID=your-google-client-id
AUTH_GOOGLE_CLIENT_SECRET=your-google-client-secret
AUTH_GOOGLE_REDIRECT_URL=https://iam.yourdomain.com/api/v1/auth/social/google/callback

AUTH_GITHUB_CLIENT_ID=your-github-client-id
AUTH_GITHUB_CLIENT_SECRET=your-github-client-secret
```

> **Important:** Update the redirect URI in Google/GitHub/Microsoft developer
> consoles to point to your GGID Gateway URL.

---

## Phase 6: Migrate Applications (OAuth Clients)

```python
def transform_app(auth0_app):
    return {
        "name": auth0_app["name"],
        "client_id": auth0_app["client_id"],
        "client_secret": auth0_app["client_secret"],
        "redirect_uris": auth0_app.get("callbacks", []),
        "grant_types": auth0_app.get("grant_types", ["authorization_code", "refresh_token"]),
        "scopes": ["openid", "profile", "email"],
    }
```

---

## Phase 7: Migrate Rules/Actions → Auth Hooks

| Auth0 Action Trigger | GGID Hook Event |
|----------------------|-----------------|
| `post-user-registration` | `post-registration` |
| `post-login` | `post-login` |
| `pre-user-registration` | `pre-registration` |
| `credentials-exchange` | `pre-token-issue` |

Migrate Auth0 Actions (JavaScript) to GGID webhook endpoints:

```python
# Flask endpoint replacing an Auth0 Action
from flask import Flask, request, jsonify
app = Flask(__name__)

@app.route("/hooks/post-login", methods=["POST"])
def post_login():
    data = request.json
    user_email = data["data"]["email"]

    # Custom logic (was Auth0 Action)
    if user_email.endswith("@contractor.company.com"):
        return jsonify({
            "action": "allow",
            "modify": {
                "claims": {"contractor": True},
                "roles": ["contractor"]
            }
        })

    return jsonify({"action": "allow"})
```

Register the hook:

```bash
curl -X POST "$GGID/api/v1/auth/hooks" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{
    "event": "post-login",
    "url": "https://yourapp.com/hooks/post-login",
    "secret": "your-hmac-secret"
  }'
```

---

## Phase 8: Cutover

### DNS Switch

1. Update your application's auth redirect URLs from Auth0 to GGID
2. Update DNS: `auth.yourdomain.com` → GGID Gateway IP
3. Keep Auth0 running for 7 days as fallback

### Dual-Run Period (Recommended)

```
Application → Auth0 (primary, with GGID as lazy migration bridge)
         ↓ (gradually shift)
Application → GGID (primary)
```

1. **Week 1**: Deploy GGID behind Auth0 bridge (lazy migration active)
2. **Week 2**: Switch primary auth to GGID, Auth0 as fallback
3. **Week 3**: Remove Auth0 fallback, decommission

### Post-Migration Checklist

- [ ] All users imported and can log in
- [ ] Roles and permissions mapped correctly
- [ ] Social login providers updated (redirect URIs)
- [ ] Auth hooks replacing Auth0 Actions
- [ ] Applications using GGID OAuth clients
- [ ] Audit logging verified
- [ ] Auth0 tenant deactivated
- [ ] DNS records updated

---

## API Endpoint Mapping

| Auth0 API | GGID API |
|-----------|----------|
| `POST /oauth/token` (password grant) | `POST /api/v1/auth/login` |
| `POST /api/v2/users` | `POST /api/v1/users` |
| `GET /api/v2/users` | `GET /api/v1/users` |
| `GET /api/v2/users/{id}` | `GET /api/v1/users/{id}` |
| `PATCH /api/v2/users/{id}` | `PUT /api/v1/users/{id}` |
| `DELETE /api/v2/users/{id}` | `DELETE /api/v1/users/{id}` |
| `POST /api/v2/roles` | `POST /api/v1/roles` |
| `GET /api/v2/roles` | `GET /api/v1/roles` |
| `POST /api/v2/users/{id}/roles` | `POST /api/v1/users/{id}/roles` |
| `GET /api/v2/logs` | `GET /api/v1/audit/events` |
| `POST /api/v2/jobs/users-exports` | `GET /api/v1/users?format=csv` |
