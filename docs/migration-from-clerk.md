# Migrating from Clerk to GGID

Complete guide for migrating applications from Clerk to GGID. Covers user
export, JWT key compatibility, NextAuth replacement, webhook migration, and
organization/role mapping.

---

## Table of Contents

- [Overview](#overview)
- [Clerk vs GGID Comparison](#clerk-vs-ggid-comparison)
- [User Export](#user-export)
- [JWT Key Compatibility](#jwt-key-compatibility)
- [NextAuth Replacement](#nextauth-replacement)
- [Webhook Migration](#webhook-migration)
- [Organization and Role Mapping](#organization-and-role-mapping)
- [Migration Script](#migration-script)
- [Migration Checklist](#migration-checklist)

---

## Overview

Clerk is a hosted authentication provider. GGID is a self-hosted IAM platform
offering the same features (passwordless, social, MFA, organizations, JWT)
with full data ownership and no per-user pricing.

### Key Differences

| Aspect | Clerk | GGID |
|--------|-------|------|
| Hosting | SaaS (managed) | Self-hosted (Docker/K8s) |
| Data residency | Clerk's servers | Your infrastructure |
| JWT signing | Clerk's keys | Your keys (RS256/EdDSA) |
| User API | `clerk.users.getUserList()` | `GET /api/v1/users` |
| Organizations | Clerk Organizations | GGID Organizations + RLS |
| Webhooks | Svix (Clerk Events) | NATS JetStream + Webhooks |
| SDK | `@clerk/nextjs` | `@ggid/sdk` (Go/Node/Java) |
| Pricing | Per MAU | Free (open source) |

---

## Clerk vs GGID Comparison

### Feature Mapping

| Clerk Feature | GGID Equivalent |
|---------------|-----------------|
| User Management API | Identity Service (`/api/v1/users`) |
| Clerk Sessions | Redis-backed sessions with JWT |
| Clerk JWT Templates | Custom JWT claims via policy engine |
| Clerk Organizations | GGID Organizations (multi-tenant) |
| Clerk Roles | RBAC + ABAC policy engine |
| Clerk Connect (OAuth) | Social connectors (Google, GitHub, etc.) |
| Clerk Webhooks | Webhook subscriptions + NATS events |
| Clerk Middleware | GGID JWT middleware |
| Allowlist / Blocklist | Rate limiting + IP filtering |

### API Endpoint Mapping

| Clerk API | GGID API |
|-----------|----------|
| `GET /v1/users` | `GET /api/v1/users` |
| `POST /v1/users` | `POST /api/v1/auth/register` |
| `GET /v1/users/{id}` | `GET /api/v1/users/{id}` |
| `PATCH /v1/users/{id}` | `PUT /api/v1/users/{id}` |
| `DELETE /v1/users/{id}` | `DELETE /api/v1/users/{id}` |
| `GET /v1/organizations` | `GET /api/v1/orgs` |
| `POST /v1/organizations` | `POST /api/v1/orgs` |
| `GET /v1/roles` | `GET /api/v1/roles` |
| `POST /v1/roles` | `POST /api/v1/roles` |

---

## User Export

### Export from Clerk API

```python
#!/usr/bin/env python3
"""
Export all users from Clerk via the Backend API.

Usage: python3 export_clerk.py --secret sk_test_xxx --output clerk-export.json
"""

import requests
import json
import argparse
import sys

def export_clerk_users(secret):
    """Export all users from Clerk."""
    headers = {"Authorization": f"Bearer {secret}"}
    users = []
    offset = 0
    limit = 100

    while True:
        resp = requests.get(
            f"https://api.clerk.com/v1/users",
            headers=headers,
            params={"limit": limit, "offset": offset}
        )

        if resp.status_code != 200:
            print(f"Clerk API error: {resp.status_code} {resp.text}", file=sys.stderr)
            sys.exit(1)

        data = resp.json()
        batch = data if isinstance(data, list) else data.get("data", [])

        if not batch:
            break

        users.extend(batch)
        print(f"Exported {len(users)} users...", file=sys.stderr)

        if len(batch) < limit:
            break
        offset += limit

    return users

def export_clerk_organizations(secret):
    """Export organizations from Clerk."""
    headers = {"Authorization": f"Bearer {secret}"}
    orgs = []
    offset = 0

    while True:
        resp = requests.get(
            "https://api.clerk.com/v1/organizations",
            headers=headers,
            params={"limit": 100, "offset": offset}
        )

        if resp.status_code != 200:
            break

        data = resp.json()
        batch = data if isinstance(data, list) else data.get("data", [])

        if not batch:
            break

        orgs.extend(batch)
        offset += 100

    return orgs

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--secret", required=True, help="Clerk Secret Key")
    parser.add_argument("--output", default="clerk-export.json")
    args = parser.parse_args()

    users = export_clerk_users(args.secret)
    orgs = export_clerk_organizations(args.secret)

    export = {"users": users, "organizations": orgs}

    with open(args.output, "w") as f:
        json.dump(export, f, indent=2)

    print(f"Exported {len(users)} users and {len(orgs)} organizations to {args.output}")
```

### Clerk User Structure

```json
{
  "id": "user_2abc123",
  "username": "johndoe",
  "first_name": "John",
  "last_name": "Doe",
  "email_addresses": [
    {
      "email_address": "john@example.com",
      "verification": { "status": "verified" }
    }
  ],
  "password_enabled": true,
  "has_password": true,
  "public_metadata": { "role": "admin" },
  "private_metadata": {},
  "created_at": 1700000000000,
  "last_sign_in_at": 1700000000000,
  "banned": false
}
```

---

## JWT Key Compatibility

Clerk signs JWTs with its own keys. GGID uses your keys (RS256 or EdDSA).

### Key Differences

| Aspect | Clerk | GGID |
|--------|-------|------|
| Algorithm | RS256 | RS256 or EdDSA (configurable) |
| Issuer | `https://clerk.example.com` | `https://iam.yourcompany.com` |
| JWKS URL | Clerk-hosted | `/.well-known/jwks.json` (self-hosted) |
| Claims | Clerk-specific (`__clerk_*`) | Standard OIDC + custom |

### Migration: Update JWKS URL

Applications that verify Clerk JWTs need to point to GGID's JWKS instead:

**Before (Clerk):**
```typescript
import Clerk from '@clerk/clerk-sdk-node';

// Clerk SDK verifies JWTs internally
const session = await clerk.verifyToken(token);
```

**After (GGID):**
```typescript
import jwt from 'jsonwebtoken';
import jwksClient from 'jwks-rsa';

const client = jwksClient({
    jwksUri: 'https://iam.yourcompany.com/.well-known/jwks.json',
    cache: true,
});

// Verify JWT against GGID's JWKS
async function verifyToken(token: string) {
    const decoded = jwt.decode(token, { complete: true });
    const key = await client.getSigningKey(decoded.header.kid);
    return jwt.verify(token, key.getPublicKey(), {
        algorithms: ['RS256'],
        issuer: 'https://iam.yourcompany.com',
    });
}
```

### Claims Mapping

| Clerk Claim | GGID Claim | Notes |
|------------|------------|-------|
| `sub` (Clerk user ID) | `sub` (GGID UUID) | IDs will change |
| `__clerk_db_jwt` | _(removed)_ | Clerk internal |
| `email_addresses[0]` | `email` | Standard OIDC claim |
| `public_metadata.role` | `roles` | Array of role keys |
| `org_id` | `tenant_id` | Tenant UUID |
| `act` (impersonation) | _(not supported)_ | Remove impersonation checks |
| `v` (Clerk version) | _(removed)_ | Clerk internal |

---

## NextAuth Replacement

Clerk's Next.js SDK (`@clerk/nextjs`) is replaced by GGID's JWT middleware
and SDK.

### Before: Clerk Middleware

```typescript
// middleware.ts (Clerk)
import { authMiddleware } from '@clerk/nextjs';

export default authMiddleware({
    publicRoutes: ['/login', '/register'],
});

export const config = {
    matcher: ['/((?!.*\\..*|_next).*)'],
};
```

### After: GGID Middleware

```typescript
// middleware.ts (GGID)
import { NextRequest, NextResponse } from 'next/server';
import jwt from 'jsonwebtoken';
import jwksClient from 'jwks-rsa';

const client = jwksClient({
    jwksUri: process.env.GGID_JWKS_URL || 'https://iam.yourcompany.com/.well-known/jwks.json',
    cache: true,
    cacheMaxAge: 600_000,
});

function getKey(header: jwt.JwtHeader): Promise<string> {
    return new Promise((resolve, reject) => {
        client.getSigningKey(header.kid, (err, key) => {
            if (err) reject(err);
            else resolve(key.getPublicKey());
        });
    });
}

export async function middleware(request: NextRequest) {
    // Skip public routes
    if (['/login', '/register', '/api/health'].some(p =>
        request.nextUrl.pathname.startsWith(p))) {
        return NextResponse.next();
    }

    const token = request.cookies.get('access_token')?.value ||
        request.headers.get('authorization')?.replace('Bearer ', '');

    if (!token) {
        return NextResponse.redirect(new URL('/login', request.url));
    }

    try {
        const decoded = jwt.decode(token, { complete: true });
        const key = await getKey(decoded.header);
        const payload = jwt.verify(token, key, {
            algorithms: ['RS256'],
            issuer: process.env.GGID_ISSUER,
        }) as any;

        // Inject user context for downstream handlers
        const headers = new Headers(request.headers);
        headers.set('x-user-id', payload.sub);
        headers.set('x-tenant-id', payload.tenant_id);
        headers.set('x-user-roles', payload.roles || '');

        return NextResponse.next({ request: { headers } });
    } catch {
        const refresh = new URL('/api/auth/refresh', request.url);
        refresh.searchParams.set('redirect', request.nextUrl.pathname);
        return NextResponse.redirect(refresh);
    }
}

export const config = {
    matcher: ['/((?!_next/static|_next/image|favicon.ico).*)'],
};
```

### Before: Clerk Server Component

```tsx
// app/dashboard/page.tsx (Clerk)
import { currentUser } from '@clerk/nextjs';

export default async function Dashboard() {
    const user = await currentUser();
    return <h1>Hello, {user?.firstName}</h1>;
}
```

### After: GGID Server Component

```tsx
// app/dashboard/page.tsx (GGID)
import { headers } from 'next/headers';

export default async function Dashboard() {
    const h = headers();
    const userId = h.get('x-user-id');
    const roles = h.get('x-user-roles') || '';

    // Fetch user from GGID API
    const res = await fetch(
        `${process.env.GGID_GATEWAY_URL}/api/v1/users/${userId}`,
        { headers: { 'Authorization': `Bearer ${process.env.GGID_SERVICE_TOKEN}` } }
    );
    const user = await res.json();

    return <h1>Hello, {user.name}</h1>;
}
```

### Clerk Provider → GGID Provider

**Before:**
```tsx
// app/layout.tsx (Clerk)
import { ClerkProvider } from '@clerk/nextjs';

export default function RootLayout({ children }) {
    return (
        <ClerkProvider>
            <html><body>{children}</body></html>
        </ClerkProvider>
    );
}
```

**After (no provider needed — GGID uses standard cookies/JWT):**
```tsx
// app/layout.tsx (GGID — no provider needed)
export default function RootLayout({ children }) {
    return (
        <html><body>{children}</body></html>
    );
}
```

---

## Webhook Migration

Clerk uses Svix for webhook delivery. GGID uses its own webhook system backed
by NATS JetStream.

### Clerk Event → GGID Event Mapping

| Clerk Event | GGID Event Type |
|-------------|-----------------|
| `user.created` | `user.created` |
| `user.updated` | `user.updated` |
| `user.deleted` | `user.deleted` |
| `session.created` | `auth.token_issued` |
| `session.ended` | `auth.token_revoked` |
| `organization.created` | `org.created` |
| `organizationMembership.created` | `org.member_added` |

### Before: Clerk Webhook (Svix)

```typescript
import { Webhook } from 'svix';

export async function POST(req: Request) {
    const payload = await req.text();
    const headers = Object.fromEntries(req.headers);

    const wh = new Webhook(process.env.CLERK_WEBHOOK_SECRET!);
    const event = wh.verify(payload, headers) as ClerkEvent;

    switch (event.type) {
        case 'user.created':
            await syncUserToCRM(event.data);
            break;
    }
}
```

### After: GGID Webhook

```typescript
import crypto from 'crypto';

export async function POST(req: Request) {
    const payload = await req.text();
    const signature = req.headers.get('x-ggid-signature') || '';
    const timestamp = req.headers.get('x-ggid-timestamp') || '';

    // Verify HMAC-SHA256 signature
    const expected = crypto
        .createHmac('sha256', process.env.GGID_WEBHOOK_SECRET!)
        .update(timestamp + '.' + payload)
        .digest('hex');

    if (!crypto.timingSafeEqual(
        Buffer.from(signature),
        Buffer.from(expected)
    )) {
        return new Response('Invalid signature', { status: 401 });
    }

    const event = JSON.parse(payload);

    switch (event.event_type) {
        case 'user.created':
            await syncUserToCRM(event.data);
            break;
    }

    return new Response('OK', { status: 200 });
}
```

### Register Webhook in GGID

```bash
curl -X POST $API/api/v1/webhooks \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -d '{
        "url": "https://app.example.com/api/webhooks/ggid",
        "events": [
            "user.created",
            "user.updated",
            "user.deleted",
            "auth.token_issued"
        ],
        "secret": "'$WEBHOOK_SECRET'"
    }'
```

---

## Organization and Role Mapping

### Clerk Organizations → GGID Tenants/Orgs

Clerk Organizations map to GGID's tenant + organization model:

| Clerk Concept | GGID Concept |
|---------------|-------------|
| Instance | Tenant |
| Organization | Organization (within tenant) |
| Organization Membership | Org membership |
| Organization Domains | Custom domain mapping |
| Organization Invitations | Org invitations API |

### Role Mapping

| Clerk Role | GGID Role Key |
|------------|--------------|
| `org:admin` | `admin` |
| `org:member` | `member` |
| Custom: `org:editor` | `editor` |
| Custom: `org:viewer` | `viewer` |

### Migrate Organizations

```bash
# Create GGID organization for each Clerk organization
for org in $(jq -c '.organizations[]' clerk-export.json); do
    name=$(echo $org | jq -r '.name')
    slug=$(echo $org | jq -r '.slug // .name | ascii_downcase')

    curl -X POST $API/api/v1/orgs \
        -H "Authorization: Bearer $ADMIN_TOKEN" \
        -H "X-Tenant-ID: $TENANT_ID" \
        -d "{
            \"name\": \"$name\",
            \"slug\": \"$slug\"
        }"
done
```

### Migrate User Memberships

```python
#!/usr/bin/env python3
"""Migrate Clerk organization memberships to GGID."""
import json
import requests

GGID_API = "https://iam.example.com"
ADMIN_TOKEN = "your-admin-token"
TENANT_ID = "your-tenant-id"

# Load export
with open("clerk-export.json") as f:
    data = json.load(f)

# Create user → GGID user ID mapping
user_map = {}  # clerk_user_id → ggid_user_id

for clerk_user in data["users"]:
    email = clerk_user["email_addresses"][0]["email_address"]
    # Look up GGID user by email
    resp = requests.get(
        f"{GGID_API}/api/v1/users",
        params={"filter": f'email eq "{email}"'},
        headers={"Authorization": f"Bearer {ADMIN_TOKEN}", "X-Tenant-ID": TENANT_ID}
    )
    ggid_user = resp.json()["data"][0]
    user_map[clerk_user["id"]] = ggid_user["id"]

# Migrate memberships
for org in data["organizations"]:
    # Find GGID org by name
    resp = requests.get(
        f"{GGID_API}/api/v1/orgs",
        params={"filter": f'name eq "{org["name"]}"'},
        headers={"Authorization": f"Bearer {ADMIN_TOKEN}", "X-Tenant-ID": TENANT_ID}
    )
    ggid_org = resp.json()["data"][0]

    # Add each member
    for membership in org.get("memberships", []):
        clerk_user_id = membership["user_id"]
        role = membership.get("role", "org:member").replace("org:", "")

        if clerk_user_id in user_map:
            requests.post(
                f"{GGID_API}/api/v1/orgs/{ggid_org['id']}/members",
                headers={"Authorization": f"Bearer {ADMIN_TOKEN}", "X-Tenant-ID": TENANT_ID},
                json={
                    "user_id": user_map[clerk_user_id],
                    "role": role
                }
            )
            print(f"Added {clerk_user_id} to {org['name']} as {role}")
```

---

## Migration Script

Complete Python script to convert Clerk export to GGID import format:

```python
#!/usr/bin/env python3
"""
Convert Clerk export to GGID import format.

Usage: python3 convert_clerk.py clerk-export.json > ggid-import.json
"""

import json
import sys

def convert(clerk_data):
    users = []
    for cu in clerk_data.get("users", []):
        emails = cu.get("email_addresses", [])
        primary_email = ""
        email_verified = False
        if emails:
            primary_email = emails[0].get("email_address", "")
            email_verified = emails[0].get("verification", {}).get("status") == "verified"

        # Determine status
        status = "active"
        if cu.get("banned"):
            status = "suspended"

        # Extract role from metadata
        roles = []
        meta_role = cu.get("public_metadata", {}).get("role")
        if meta_role:
            roles.append(meta_role)

        ggid_user = {
            "external_id": cu["id"],
            "username": cu.get("username") or primary_email,
            "email": primary_email,
            "name": f"{cu.get('first_name', '')} {cu.get('last_name', '')}".strip(),
            "status": status,
            "email_verified": email_verified,
            "require_password_reset": cu.get("password_enabled", False),
            "roles": roles,
            "created_at": cu.get("created_at"),
        }
        users.append(ggid_user)

    orgs = []
    for co in clerk_data.get("organizations", []):
        ggid_org = {
            "name": co["name"],
            "slug": co.get("slug", co["name"].lower().replace(" ", "-")),
            "members": [],
        }
        for m in co.get("memberships", []):
            role = m.get("role", "org:member").replace("org:", "")
            ggid_org["members"].append({
                "clerk_user_id": m["user_id"],
                "role": role,
            })
        orgs.append(ggid_org)

    return {
        "format": "ggid-import-v1",
        "users": users,
        "organizations": orgs,
    }

if __name__ == "__main__":
    with open(sys.argv[1]) as f:
        clerk_data = json.load(f)
    result = convert(clerk_data)
    json.dump(result, sys.stdout, indent=2)
```

---

## Migration Checklist

- [ ] Export users from Clerk Backend API
- [ ] Export organizations
- [ ] Run conversion script
- [ ] Create GGID tenant
- [ ] Import users (with password reset)
- [ ] Import organizations and memberships
- [ ] Create RBAC roles to match Clerk roles
- [ ] Configure social connections (Google, GitHub, etc.)
- [ ] Update JWKS URL in all consuming applications
- [ ] Replace `@clerk/nextjs` with GGID middleware
- [ ] Migrate webhooks from Svix to GGID format
- [ ] Update login/signup page to GGID hosted page or widget
- [ ] Update session token verification (Clerk → GGID JWKS)
- [ ] Send password reset emails
- [ ] Test all auth flows (password, social, MFA)
- [ ] Monitor for 24 hours
- [ ] Remove Clerk integration

---

## References

- [Migration from Auth0](./migration-from-auth0.md)
- [Migration from Keycloak](./migration-from-keycloak.md)
- [SDK Guide](./api-sdk-guide.md) — GGID SDKs
- [API Reference](./api-reference.md) — REST endpoints
