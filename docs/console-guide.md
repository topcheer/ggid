# GGID Admin Console Guide

The GGID Admin Console is a Next.js 15 web application that provides a graphical
interface for managing users, roles, organizations, audit logs, and system settings.

**URL:** http://localhost:3000 (Docker Compose) or https://iam.example.com (production)

---

## Table of Contents

- [Accessing the Console](#accessing-the-console)
- [Dashboard](#dashboard)
- [User Management](#user-management)
- [Role Management](#role-management)
- [Organization Management](#organization-management)
- [Audit Log Explorer](#audit-log-explorer)
- [System Settings](#system-settings)
- [OAuth Clients](#oauth-clients)
- [Monitoring](#monitoring)
- [Webhooks](#webhooks)
- [User Profile](#user-profile)

---

## Accessing the Console

1. Navigate to the Console URL in your browser
2. You will be redirected to the **Login** page

> **[SCREENSHOT PLACEHOLDER: Login page with username/password form]**

3. Enter your credentials:
   - **Username:** Your registered username
   - **Password:** Your password

4. On successful login, you are redirected to the Dashboard

> The Console communicates with the Gateway at `:8080` via a server-side proxy.
> All API calls include your JWT automatically.

---

## Dashboard

The Dashboard provides an at-a-glance overview of your IAM environment.

> **[SCREENSHOT PLACEHOLDER: Dashboard with service health cards and metrics]**

### Service Health

Each microservice is displayed as a card showing:
- Service name (Gateway, Identity, Auth, Policy, Org, Audit, OAuth)
- Health status (green = healthy, red = unhealthy)
- Response time

### Key Metrics

- Total users
- Active sessions
- Failed login attempts (last 24h)
- Audit events (last 24h)

---

## User Management

Navigate to **Users** in the sidebar.

> **[SCREENSHOT PLACEHOLDER: Users page with table and search bar]**

### Listing Users

- The user table displays: username, email, status, created date
- Use the **search bar** to filter by username or email
- Pagination at the bottom (50 users per page)

### Creating a User

1. Click **"Add User"** button
2. Fill in the form:

> **[SCREENSHOT PLACEHOLDER: Create User modal with fields]**

| Field | Required | Description |
|-------|----------|-------------|
| Username | Yes | Unique within tenant |
| Email | Yes | User's email address |
| Password | Yes | Must meet password policy |
| Phone | No | Phone number |
| Display Name | No | Full name |
| Locale | No | e.g. `en-US` |
| Timezone | No | e.g. `America/New_York` |

3. Click **Create**

### User Actions

Each user row has action buttons:

| Action | Description |
|--------|-------------|
| **Edit** | Update email, phone, display name |
| **Lock** | Temporarily disable login (user sees "account locked") |
| **Unlock** | Re-enable login for a locked user |
| **Deactivate** | Mark user as inactive |
| **Activate** | Reactivate a deactivated user |
| **Delete** | Permanently remove user |

### Managing User Roles

1. Click **Edit** on a user
2. Navigate to the **Roles** tab
3. Check/uncheck roles to assign/remove
4. Changes are applied immediately

> **[SCREENSHOT PLACEHOLDER: User edit page with Roles tab]**

### Bulk Import

1. Click **"Import CSV"**
2. Upload a CSV file with columns: `username, email, password, phone, display_name`
3. Review the import summary (imported/failed counts)

---

## Role Management

Navigate to **Roles** in the sidebar.

> **[SCREENSHOT PLACEHOLDER: Roles page with tabs: Roles, Permissions, Policies]**

### Creating a Role

1. Go to the **Roles** tab
2. Click **"Create Role"**

> **[SCREENSHOT PLACEHOLDER: Create Role dialog]**

| Field | Required | Description |
|-------|----------|-------------|
| Key | Yes | Unique identifier (e.g. `editor`, `viewer`) |
| Name | Yes | Display name (e.g. "Content Editor") |
| Description | No | Human-readable description |
| Parent Role | No | Parent role for inheritance hierarchy |

3. Click **Create**

### Role Hierarchy

Roles can inherit permissions from a parent role:

- Create a parent role (e.g. `staff`)
- Create a child role (e.g. `admin`) and set its parent to `staff`
- The `admin` role inherits all permissions from `staff`

To set/change a parent:
1. Click **Edit** on the child role
2. Select a parent role from the dropdown
3. Save

### Managing Permissions

1. Go to the **Permissions** tab within a role
2. Add permissions as `resource:action` pairs:

| Resource | Action | Example |
|----------|--------|---------|
| `iam:users` | `read` | View users |
| `iam:users` | `write` | Create/edit users |
| `iam:roles` | `manage` | Full role management |
| `documents:*` | `*` | All document operations |

3. Click **Add Permission**

### Policy Configuration

1. Go to the **Policies** tab
2. Create ABAC policies with conditions:

```json
{
  "effect": "allow",
  "actions": ["read"],
  "resources": ["documents:*"],
  "conditions": {
    "department": "engineering",
    "clearance_level": ">= 3"
  }
}
```

---

## Organization Management

Navigate to **Organizations** in the sidebar.

> **[SCREENSHOT PLACEHOLDER: Organizations page with tree view]**

### Organization Tree

Organizations are displayed as a hierarchical tree:
- Top-level organizations (root nodes)
- Expandable sub-organizations
- Departments and teams nested within

### Creating an Organization

1. Click **"Add Organization"**
2. Fill in:
   - **Name** (required)
   - **Parent Organization** (optional — creates a sub-org)
   - **Description** (optional)

### Managing Members

1. Click on an organization to view details
2. Go to the **Members** tab

> **[SCREENSHOT PLACEHOLDER: Org members tab with add/remove buttons]**

- **Add member:** Select a user and assign a title (e.g. "Engineering Manager")
- **Remove member:** Click the remove button next to a member
- **View all members:** Lists all users in this org (including sub-orgs)

### Departments and Teams

- **Departments** are sub-units of an organization (e.g. "Backend", "Frontend")
- **Teams** are cross-cutting groups (e.g. "On-call", "Security")

Create via the respective tabs in the org detail view.

---

## Audit Log Explorer

Navigate to **Audit** in the sidebar.

> **[SCREENSHOT PLACEHOLDER: Audit page with filter bar and events table]**

### Filtering Events

Use the filter bar to narrow results:

| Filter | Description |
|--------|-------------|
| Action | Filter by event type (e.g. `user.login`, `role.create`) |
| Result | `success` or `failure` |
| Resource Type | e.g. `user`, `role`, `organization` |
| Actor | User UUID |
| Start Time | RFC3339 datetime |
| End Time | RFC3339 datetime |
| Page Size | 10–500 events per page |

### Event Details

Each event row shows:
- **Timestamp** — when the event occurred
- **Actor** — who performed the action
- **Action** — what was done (e.g. `user.login`)
- **Result** — success/failure (color-coded)
- **Resource** — what was affected
- **IP Address** — source IP

Click any row to see full event metadata (JSON).

### Audit Statistics

Click **"Statistics"** to view aggregated metrics:

> **[SCREENSHOT PLACEHOLDER: Audit stats page with charts]**

- Events by action type (pie chart)
- Events over time (hourly bar chart)
- Top actors by event count
- Success/failure ratio

### Exporting Events

1. Set your filters
2. Click **"Export CSV"**
3. The file downloads immediately with all matching events

### Real-Time Streaming

Click **"Live Stream"** to open a Server-Sent Events (SSE) connection that
pushes new audit events in real-time as they occur.

---

## System Settings

Navigate to **Settings** in the sidebar.

> **[SCREENSHOT PLACEHOLDER: Settings page with tabs]**

### General

- **Platform name** — displayed in the Console header and login page
- **Support email** — shown in error messages
- **Default locale** — language for new users
- **Session timeout** — idle timeout in minutes

### SMTP Configuration

Configure email sending for password reset, email verification, and notifications.

| Field | Description |
|-------|-------------|
| SMTP Host | Mail server hostname (e.g. `smtp.gmail.com`) |
| SMTP Port | Mail server port (587 for TLS, 465 for SSL) |
| Username | SMTP authentication username |
| Password | SMTP authentication password |
| From Email | Sender address (e.g. `noreply@example.com`) |
| From Name | Sender display name (e.g. `GGID Platform`) |

Click **"Send Test Email"** to verify the configuration.

> **[SCREENSHOT PLACEHOLDER: SMTP configuration form with test button]**

### Brand Customization

Customize the look of the Console and login page:

| Setting | Description |
|---------|-------------|
| Logo URL | Custom logo (shown in header) |
| Primary Color | Brand accent color (hex, e.g. `#3B82F6`) |
| Login Background | Background image URL for login page |
| Custom CSS | Additional CSS overrides |

> **[SCREENSHOT PLACEHOLDER: Brand customization panel with color picker]**

### Password Policy

Configure password requirements:

| Setting | Default | Description |
|---------|---------|-------------|
| Minimum Length | 8 | Minimum password length |
| Require Uppercase | On | At least one A-Z character |
| Require Lowercase | On | At least one a-z character |
| Require Digits | On | At least one 0-9 character |
| Require Special | On | At least one special character |
| History Count | 5 | Prevent reusing last N passwords |

### Security

| Setting | Description |
|---------|-------------|
| MFA Required | Force TOTP for all users |
| Session Duration | Access token TTL (default: 3600s) |
| Refresh Token TTL | Default: 30 days |
| Rate Limit Threshold | Max failed logins before lockout (default: 5) |
| IP Allowlist | Comma-separated CIDR ranges |

---

## OAuth Clients

Navigate to **OAuth Clients** in the sidebar.

> **[SCREENSHOT PLACEHOLDER: OAuth clients list]**

### Registering a Client

1. Click **"Add Client"**
2. Fill in:

| Field | Description |
|-------|-------------|
| Client Name | Display name |
| Client ID | Auto-generated (or custom) |
| Client Secret | Auto-generated (shown once) |
| Redirect URIs | Comma-separated allowed callback URLs |
| Grant Types | `authorization_code`, `client_credentials`, `refresh_token` |
| Scopes | `openid profile email` (space-separated) |

3. Click **Create**

> **[SCREENSHOT PLACEHOLDER: OAuth client creation form]**

### Managing Clients

- **View:** See client details and configured redirect URIs
- **Edit:** Update redirect URIs, grant types, scopes
- **Rotate Secret:** Generate a new client secret (old one invalidated)
- **Revoke:** Disable the client (tokens remain valid until expiry)

---

## Monitoring

Navigate to **Monitoring** in the sidebar.

> **[SCREENSHOT PLACEHOLDER: Monitoring page with graphs]**

### Metrics Displayed

- **Request Rate** — requests/sec through the Gateway
- **Latency** — p50/p95/p99 per service
- **Error Rate** — 4xx and 5xx counts
- **JWT Verifications** — success/failure counts
- **Active Sessions** — concurrent authenticated sessions
- **Backend Health** — per-service health status

### Alerts

Configure threshold-based alerts:
- Error rate > 5% for 5 minutes
- p95 latency > 500ms
- Backend unhealthy for > 1 minute

---

## Webhooks

Navigate to **Webhooks** in the sidebar.

> **[SCREENSHOT PLACEHOLDER: Webhooks configuration page]**

### Available Events

| Event | Triggered When |
|-------|---------------|
| `user.created` | A new user registers |
| `user.login` | A user logs in successfully |
| `user.login_failed` | A login attempt fails |
| `user.locked` | An account is locked |
| `role.created` | A new role is created |
| `role.assigned` | A role is assigned to a user |

### Configuring a Webhook

1. Click **"Add Webhook"**
2. Enter:
   - **URL** — your endpoint (must be HTTPS in production)
   - **Events** — select which events to subscribe to
   - **Secret** — used to sign the payload (HMAC-SHA256)

3. The webhook sends a POST with:
   ```json
   {
     "event": "user.login",
     "timestamp": "2024-01-15T10:30:00Z",
     "data": { "user_id": "...", "ip": "..." }
   }
   ```
   And header: `X-GGID-Signature: sha256=...`

---

## User Profile

Click your username in the top-right corner.

> **[SCREENSHOT PLACEHOLDER: Profile page]]

### Available Actions

- **Update display name, email, phone**
- **Change password** (requires current password)
- **Enable MFA** — scan QR code with Google Authenticator, enter 6-digit code
- **Disable MFA** — enter current TOTP code
- **View active sessions** — see all devices logged in
- **Revoke sessions** — logout from other devices

---

## Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| `?` | Show shortcut help |
| `g u` | Go to Users |
| `g r` | Go to Roles |
| `g o` | Go to Organizations |
| `g a` | Go to Audit |
| `g s` | Go to Settings |
| `/` | Focus search bar |
