# GGID Admin Console — Screenshots Guide

Visual reference guide for the GGID Admin Console. Each section describes the
page layout, core features, and suggests screenshot locations for documentation.

> **Convention:** Replace `[SCREENSHOT: ...]` placeholders with actual screenshots.
> Recommended dimensions: 1280×720 (16:9), PNG format, optimized to < 500KB.

---

## Table of Contents

- [Login Page](#login-page)
- [Dashboard](#dashboard)
- [User Management](#user-management)
- [Role Management](#role-management)
- [Organization Management](#organization-management)
- [Audit Log Explorer](#audit-log-explorer)
- [Settings](#settings)
- [Monitoring](#monitoring)
- [OAuth Clients](#oauth-clients)
- [Webhooks](#webhooks)
- [User Profile](#user-profile)

---

## Login Page

**URL:** `/login`

### Layout

```
┌─────────────────────────────────────────┐
│            [Logo / Brand]               │
│                                         │
│     ┌───────────────────────────┐      │
│     │  Username                  │      │
│     ├───────────────────────────┤      │
│     │  Password                  │      │
│     ├───────────────────────────┤      │
│     │       [ Sign In ]          │      │
│     └───────────────────────────┘      │
│                                         │
│   [Google] [GitHub] [Microsoft]        │
│                                         │
│   ☐ Remember me    Forgot password?    │
└─────────────────────────────────────────┘
```

### Core Features

- **Username/Password form** — primary login
- **Social login buttons** — Google, GitHub, Microsoft, Discord (if configured)
- **Remember me** checkbox — extends session TTL
- **Forgot password link** — initiates password reset flow
- **Brand customization** — logo and colors from Settings

### Screenshot Suggestions

> **[SCREENSHOT: login-default]** — Default login page with username/password
>
> **[SCREENSHOT: login-social]** — Login page with social login buttons expanded
>
> **[SCREENSHOT: login-branded]** — Login page with custom brand colors and logo

---

## Dashboard

**URL:** `/` (after login)

### Layout

```
┌─────────────────────────────────────────────────────────┐
│ [Sidebar]  │  Dashboard                                 │
│            │                                             │
│ Dashboard  │  ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐    │
│ Users      │  │Users │ │Active│ │Failed│ │Events│    │
│ Roles      │  │ 142  │ │ 38   │ │  6   │ │ 1542 │    │
│ Orgs       │  └──────┘ └──────┘ └──────┘ └──────┘    │
│ Audit      │                                             │
│ Settings   │  Service Health                             │
│ Monitoring │  ┌────┐ ┌────┐ ┌────┐ ┌────┐ ┌────┐      │
│            │  │GW ✓│ │ID ✓│ │Auth✓│ │Pol ✓│ │Org ✓│      │
│            │  └────┘ └────┘ └────┘ └────┘ └────┘      │
└─────────────────────────────────────────────────────────┘
```

### Core Features

- **Metric cards** — Total Users, Active Sessions, Failed Logins (24h), Audit Events (24h)
- **Service health grid** — 7 service cards (Gateway, Identity, Auth, Policy, Org, Audit, OAuth)
  - Green check = healthy, Red = unhealthy
  - Shows response time
- **Recent activity feed** — last 5 audit events
- **Quick links** — shortcuts to common actions

### Screenshot Suggestions

> **[SCREENSHOT: dashboard-overview]** — Full dashboard with all metric cards and healthy services
>
> **[SCREENSHOT: dashboard-unhealthy]** — Dashboard with one service showing red (unhealthy) state

---

## User Management

**URL:** `/users`

### Layout

```
┌─────────────────────────────────────────────────────────┐
│ Users                                                   │
│ [Search...] [Add User] [Import CSV]                    │
├──────────┬───────────┬────────┬────────┬───────────────┤
│ Username │ Email      │ Status │ Created │ Actions       │
├──────────┼───────────┼────────┼────────┼───────────────┤
│ admin    │ a@exp.com │ active │ Jan 15  │ [Edit][Lock]  │
│ jane.doe │ j@exp.com │ active │ Jan 20  │ [Edit][Lock]  │
│ bob      │ b@exp.com │ locked │ Feb 01  │ [Edit][Unlock]│
└──────────┴───────────┴────────┴────────┴───────────────┘
```

### Core Features

- **Search bar** — filter by username or email
- **Add User button** — opens create user modal
- **Import CSV button** — bulk import from CSV file
- **User table columns**:
  - Username, Email, Status (active/locked/inactive), Created date
  - Action buttons: Edit, Lock/Unlock, Activate/Deactivate, Delete
- **Pagination** — 50 users per page

### User Edit Modal

When clicking **Edit**, a modal opens with tabs:
- **Profile** — edit email, phone, display name, locale, timezone
- **Roles** — checkbox list of assignable roles
- **Security** — reset password, force MFA, view sessions

### Screenshot Suggestions

> **[SCREENSHOT: users-list]** — User table with search and multiple status states
>
> **[SCREENSHOT: users-create-modal]** — Add User modal with all fields
>
> **[SCREENSHOT: users-edit-roles]** — User edit modal showing Roles tab
>
> **[SCREENSHOT: users-import-csv]** — CSV import dialog with results summary

---

## Role Management

**URL:** `/roles`

### Layout

```
┌─────────────────────────────────────────────────────────┐
│ Roles     [Roles] [Permissions] [Policies]             │
│           [Create Role]                                 │
│           ┌──────────┬──────────┬────────────┐         │
│           │ Key      │ Name     │ Permissions│         │
│           ├──────────┼──────────┼────────────┤         │
│           │ admin    │ Admin    │ 12 perms   │         │
│           │ editor   │ Editor   │ 5 perms    │         │
│           │ viewer   │ Viewer   │ 2 perms    │         │
│           └──────────┴──────────┴────────────┘         │
└─────────────────────────────────────────────────────────┘
```

### Core Features

- **Three tabs**: Roles, Permissions, Policies
- **Create Role** — modal with key, name, description, parent role
- **Role list** — shows key, display name, permission count
- **Role detail** — click to view/edit permissions, set parent

### Permissions Tab

- List of all permissions as `resource:action` pairs
- Add new permission to a role

### Policies Tab

- ABAC policy list with conditions
- Create policy with JSON condition editor

### Screenshot Suggestions

> **[SCREENSHOT: roles-list]** — Roles tab showing multiple roles
>
> **[SCREENSHOT: roles-create]** — Create Role modal
>
> **[SCREENSHOT: roles-permissions]** — Permissions tab with add/remove
>
> **[SCREENSHOT: roles-policies]** — Policies tab with ABAC conditions

---

## Organization Management

**URL:** `/organizations`

### Layout

```
┌─────────────────────────────────────────────────────────┐
│ Organizations                                           │
│ [Add Org] [Expand All] [Collapse All]                  │
│                                                         │
│  ▼ Engineering                           [Edit] [Add]  │
│    ▼ Backend                            [Edit] [Add]  │
│      • Go Team                          [Edit]         │
│      • DevOps                           [Edit]         │
│    ▶ Frontend                           [Edit] [Add]  │
│  ▶ Sales                                              │
│  ▶ Marketing                                          │
└─────────────────────────────────────────────────────────┘
```

### Core Features

- **Tree view** — hierarchical display of organizations and sub-organizations
- **Expand/Collapse** — navigate the tree
- **Add Org** — create root or sub-organization (with parent selection)
- **Members tab** — when an org is selected, view/add/remove members
- **Departments** — sub-units within an org
- **Teams** — cross-cutting groups

### Screenshot Suggestions

> **[SCREENSHOT: orgs-tree-expanded]** — Tree view with 3 levels expanded
>
> **[SCREENSHOT: orgs-members]** — Org detail showing members tab
>
> **[SCREENSHOT: orgs-create]** — Add Org modal with parent dropdown

---

## Audit Log Explorer

**URL:** `/audit`

### Layout

```
┌─────────────────────────────────────────────────────────┐
│ Audit Log                                               │
│ [Statistics] [Export CSV] [Live Stream]                 │
│                                                         │
│ Filters:                                                │
│ [Action ▼] [Result ▼] [Actor...] [Start...] [End...]  │
│ [Apply] [Clear]                                         │
│                                                         │
│ ┌──────────┬────────┬────────┬────────┬──────────┐    │
│ │ Time     │ Actor  │ Action │ Result │ Resource │    │
│ ├──────────┼────────┼────────┼────────┼──────────┤    │
│ │ 10:30:15 │ admin  │ login  │  ● OK  │ auth     │    │
│ │ 10:31:00 │ admin  │ create │  ● OK  │ role     │    │
│ │ 10:35:22 │ jane   │ login  │  ✗ FAIL│ auth     │    │
│ └──────────┴────────┴────────┴────────┴──────────┘    │
│ [Export CSV]              Page 1 of 32  [< >]         │
└─────────────────────────────────────────────────────────┘
```

### Core Features

- **Filter bar** — 5 filter dimensions (action, result, actor, resource, time range)
- **Color-coded results** — green dot (success), red dot (failure)
- **Event table** — timestamp, actor name, action, result, resource, IP
- **Click row** — expand to see full metadata (JSON)
- **Statistics** — charts of events by action, result, time, top actors
- **Export CSV** — download filtered events
- **Live Stream** — real-time SSE event feed

### Screenshot Suggestions

> **[SCREENSHOT: audit-events]** — Event table with mixed success/failure
>
> **[SCREENSHOT: audit-filters]** — Filter bar with multiple filters applied
>
> **[SCREENSHOT: audit-statistics]** — Statistics page with charts
>
> **[SCREENSHOT: audit-event-detail]** — Expanded event showing JSON metadata
>
> **[SCREENSHOT: audit-live-stream]** — Live stream view with real-time events

---

## Settings

**URL:** `/settings`

### Tabs

- **General** — platform name, support email, default locale, session timeout
- **SMTP** — email server configuration with test button
- **Branding** — logo, primary color, login background, custom CSS
- **Password Policy** — min length, complexity rules, history count
- **Security** — MFA required, rate limits, IP allowlist

### Screenshot Suggestions

> **[SCREENSHOT: settings-general]** — General settings tab
>
> **[SCREENSHOT: settings-smtp]** — SMTP configuration form with test button
>
> **[SCREENSHOT: settings-branding]** — Branding tab with color picker and logo upload
>
> **[SCREENSHOT: settings-password-policy]** — Password policy configuration
>
> **[SCREENSHOT: settings-security]** — Security settings with MFA toggle and rate limits

---

## Monitoring

**URL:** `/monitoring`

### Layout

```
┌─────────────────────────────────────────────────────────┐
│ Monitoring                                              │
│                                                         │
│  Request Rate (req/sec)        Error Rate (%)           │
│  ┌────────────────────┐  ┌────────────────────┐       │
│  │    ╱╲    ╱╲       │  │          _         │       │
│  │   ╱  ╲  ╱  ╲      │  │    ╱╲  ╱ ╲ ╲      │       │
│  │ _╱    ╲╱    ╲___  │  │ __╱  ╲╱   ╲ ╲___  │       │
│  └────────────────────┘  └────────────────────┘       │
│                                                         │
│  Latency p95 (ms)            Active Sessions           │
│  ┌────────────────────┐  ┌────────────────────┐       │
│  │  ╱╲   ╱╲           │  │  ╱╲    ╱╲    ╱╲   │       │
│  │ ╱  ╲ ╱  ╲ ___     │  │ ╱  ╲  ╱  ╲  ╱  ╲  │       │
│  └────────────────────┘  └────────────────────┘       │
│                                                         │
│  Backend Health                                        │
│  ● Gateway  OK  2ms    ● Auth     OK  5ms            │
│  ● Identity OK  3ms    ● Policy   OK  4ms            │
│  ● Org      OK  2ms    ● Audit    OK  8ms            │
└─────────────────────────────────────────────────────────┘
```

### Core Features

- **Real-time graphs** — request rate, error rate, p95 latency, active sessions
- **Backend health** — per-service status with response time
- **Time range selector** — 1h, 6h, 24h, 7d, 30d
- **Auto-refresh** — updates every 10 seconds

### Screenshot Suggestions

> **[SCREENSHOT: monitoring-dashboard]** — All 4 graphs with data
>
> **[SCREENSHOT: monitoring-backend]** — Backend health table
>
> **[SCREENSHOT: monitoring-time-range]** — With 24h time range selected

---

## OAuth Clients

**URL:** `/oauth-clients`

### Core Features

- **Client list** — shows name, client_id, grant types
- **Create Client** — name, redirect URIs, grant types, scopes
- **Rotate Secret** — generates new client_secret
- **Revoke** — disables client

### Screenshot Suggestions

> **[SCREENSHOT: oauth-clients-list]** — Client table
>
> **[SCREENSHOT: oauth-client-create]** — Create client modal with redirect URIs

---

## Webhooks

**URL:** `/webhooks`

### Core Features

- **Webhook list** — URL, events subscribed, last delivery status
- **Create Webhook** — URL, event selection, HMAC secret
- **Test** — send a test event to verify connectivity

### Screenshot Suggestions

> **[SCREENSHOT: webhooks-list]** — Webhook table with delivery status
>
> **[SCREENSHOT: webhook-create]** — Create webhook with event checkboxes

---

## User Profile

**URL:** `/profile`

### Tabs

- **Profile** — display name, email, phone, avatar
- **Security** — change password, enable/disable MFA, WebAuthn devices
- **Sessions** — list of active sessions with revoke buttons

### Screenshot Suggestions

> **[SCREENSHOT: profile-main]** — Profile tab with editable fields
>
> **[SCREENSHOT: profile-security-mfa]** — Security tab showing MFA QR code
>
> **[SCREENSHOT: profile-sessions]** — Sessions tab with revoke buttons

---

## Screenshot Style Guide

| Aspect | Recommendation |
|--------|----------------|
| Resolution | 1280x720 (16:9) or 1920x1080 |
| Format | PNG (lossless) |
| Max size | 500KB (optimize with TinyPNG) |
| Annotation | Use red boxes/arrows for callouts |
| Text | Ensure all text is readable at full zoom |
| Personal data | Use dummy data (john.doe@example.com) |
| Status variety | Include both healthy/success and error states |
| Dark mode | Capture in light mode (default) for documentation |

### File Naming Convention

```
img/dashboard-overview.png
img/users-list.png
img/users-create-modal.png
img/audit-events.png
img/settings-smtp.png
...
```

Store screenshots in `docs/img/` directory.
