# GGID Admin Console

Web-based management interface for the GGID IAM platform, built with Next.js 15,
React 19, and Tailwind CSS.

## Quick Start

```bash
cd console
npm install
npm run dev
```

Open http://localhost:3000

The Console connects to the API Gateway at `http://localhost:8080` by default.
Override with `NEXT_PUBLIC_API_URL`:

```bash
NEXT_PUBLIC_API_URL=https://api.iam.example.com npm run dev
```

---

## Page Navigation

| Page | Route | Description |
|------|-------|-------------|
| Dashboard | `/` | Overview stats: user count, active sessions, recent audit events |
| Login | `/login` | Authenticate with username/password + optional MFA |
| Users | `/users` | User list with search, filter, pagination |
| User Detail | `/users/[id]` | Profile editor, role assignment, lock/unlock, sessions |
| Roles | `/roles` | Role management with tabs: Roles, Permissions, Policy Templates |
| Organizations | `/organizations` | Org tree view (LTREE hierarchy), departments, teams |
| Audit | `/audit` | Audit event table with filtering by action, actor, date range |
| Settings | `/settings` | Tenant configuration: password policy, branding, webhooks |

### Navigation Layout

```
+----------------------------------------------------------+
|  [Logo]  GGID Console          [User] [Notifications]    |
+----------+-----------------------------------------------+
|          |                                               |
| Dashboard|                                               |
| Users    |            Main Content Area                  |
| Roles    |                                               |
| Orgs     |                                               |
| Audit    |                                               |
| Settings |                                               |
|          |                                               |
+----------+-----------------------------------------------+
```

---

## User Management

### Create a User

1. Navigate to **Users** page
2. Click **"+ New User"** button
3. Fill in the form:
   - **Username** (required) — unique identifier
   - **Email** (required) — for notifications and password reset
   - **First Name / Last Name** (optional)
   - **Password** (required) — must meet password policy
   - **Send welcome email** (checkbox)
4. Click **Create**

The user appears in the list and can immediately log in.

### Edit a User

1. Click on a user row in the list
2. Edit fields on the detail page
3. Changes save on **blur** (auto-save) or click **Save Changes**

### Lock / Unlock a User

1. Navigate to user detail page
2. Click **Lock Account** (or **Unlock Account**)
3. Locked users cannot authenticate but their data is preserved

### Delete a User

1. Navigate to user detail page
2. Click **Delete** → confirm in dialog
3. Default: soft-delete (data preserved, `deleted_at` set)
4. Hard-delete available via API: `DELETE /api/v1/users/:id?hard=true`

---

## Role Assignment

### Assign a Role

1. Navigate to **User Detail** page (`/users/[id]`)
2. Scroll to **Roles** section
3. Click **"+ Assign Role"**
4. Select role from dropdown
5. Click **Assign**

The user immediately gains the permissions defined by that role.

### Revoke a Role

1. In the user's Roles section, find the role
2. Click the **trash icon** next to the role
3. Confirm revocation

### Create a Custom Role

1. Navigate to **Roles** page
2. Click **"+ New Role"**
3. Enter:
   - **Key** (required) — unique identifier (e.g., `viewer`, `billing-admin`)
   - **Name** (required) — display name
   - **Description** (optional)
   - **Permissions** — multi-select from available permissions
4. Click **Create**

### Permission Format

Permissions follow `{action}:{resource}` format:

```
read:users      — View user list and details
write:users     — Create, edit, delete users
read:roles      — View roles
write:roles     — Create, edit, delete roles
read:orgs       — View organizations
write:orgs      — Create, edit organizations
read:audit      — View audit events
admin:*         — Full admin access
```

---

## Audit Query

### View Audit Events

1. Navigate to **Audit** page
2. Events load with the most recent first (paginated, 20 per page)

### Filter Audit Events

Use the filter bar at the top:

- **Action** — dropdown: `user.login`, `user.create`, `role.assign`, etc.
- **Actor** — search by user ID or username
- **Date Range** — from/to date pickers
- **Resource** — filter by resource type or ID

### Export Audit Events

1. Apply desired filters
2. Click **"Export CSV"**
3. Downloads filtered events as a CSV file

### Real-Time Audit Stream

The Audit page connects to the SSE stream (`/api/v1/audit/stream`) for
real-time event updates. New events appear automatically without page refresh.

---

## Development

### Tech Stack

- **Framework:** Next.js 15 (App Router)
- **UI:** React 19 + Tailwind CSS
- **State:** React hooks (useState, useContext)
- **HTTP:** Native `fetch` with JWT bearer auth
- **Charts:** Recharts (dashboard stats)

### Project Structure

```
console/
├── src/
│   ├── app/
│   │   ├── layout.tsx          # Root layout with auth provider
│   │   ├── page.tsx            # Dashboard
│   │   ├── login/page.tsx      # Login page
│   │   ├── users/
│   │   │   ├── page.tsx        # User list
│   │   │   └── [id]/page.tsx   # User detail
│   │   ├── roles/page.tsx      # Roles management
│   │   ├── organizations/      # Org tree
│   │   ├── audit/page.tsx      # Audit log
│   │   └── settings/           # Settings
│   ├── components/              # Reusable UI components
│   ├── lib/
│   │   ├── api.ts              # API client
│   │   ├── auth.ts             # Auth context + JWT
│   │   └── theme.tsx           # Theme provider
│   └── globals.css
├── package.json
├── next.config.js
└── tailwind.config.ts
```

### Build for Production

```bash
npm run build
npm start
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `NEXT_PUBLIC_API_URL` | `http://localhost:8080` | API Gateway URL |
| `NEXT_PUBLIC_TENANT_ID` | — | Default tenant for Console |

### Mock vs Real Backend

The Console always connects to the real API Gateway. For development:

```bash
# Start full stack via Docker
cd deploy && docker compose up -d

# Run Console in dev mode
cd console && npm run dev
```

---

## Brand Customization

See [Brand Customization Guide](../docs/brand-customization.md) for logo,
colors, fonts, CSS, email templates, and i18n configuration.
