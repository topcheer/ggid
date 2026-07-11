# GGID Admin Console Guide

This guide provides a comprehensive tour of the GGID Admin Console — from dashboard to security center, covering all features and configuration options.

## Overview

The GGID Admin Console is a Next.js 15 web application providing a unified interface for managing users, roles, organizations, policies, authentication, audit logs, and security settings.

### Access

```
URL: https://console.ggid.example.com
Login: Admin credentials (requires 'admin' scope)
Default tenant: 00000000-0000-0000-0000-000000000001
```

## Console Pages

### Dashboard

**Path**: `/`

The dashboard provides a high-level overview of your IAM environment:

- **Active users** count and trend
- **Recent logins** (last 24 hours)
- **Failed login attempts** (security indicator)
- **Active sessions** count
- **MFA enrollment** rate
- **Quick actions**: Create user, Create role, View audit log

---

### Users

**Path**: `/users`

Manage user accounts across your tenant.

| Action         | Description                                      |
|----------------|--------------------------------------------------|
| Create User    | Register new user (username, email, password)    |
| Edit User      | Update email, phone, display name                |
| Delete User    | Soft-delete (marks as inactive)                  |
| Reset Password | Admin-initiated password reset                   |
| Assign Role    | Attach RBAC roles to user                        |
| View Sessions  | See active sessions for the user                 |
| Suspend        | Temporarily disable account                      |

**Filters**: Search by username, email, status (active/suspended), role.

---

### Groups

**Path**: `/groups`

Organize users into groups for batch operations and policy assignment.

- Create groups (e.g., "Engineering", "Sales", "Admins")
- Add/remove users from groups
- Assign group-level roles
- Nested group support

---

### Roles

**Path**: `/roles`

Manage RBAC roles and their associated permissions.

| Field     | Description                              |
|-----------|------------------------------------------|
| Key       | Unique identifier (e.g., "admin", "dev") |
| Name      | Display name                             |
| Permissions | List of resource:action pairs           |

**Built-in roles**:
- `admin` — Full access to all resources
- `user` — Self-service (profile, password change)
- `auditor` — Read-only access to audit logs

---

### Permissions

**Path**: `/permissions`

View and manage fine-grained permissions.

- Resource types: `users`, `roles`, `organizations`, `audit`, `policies`
- Actions: `read`, `write`, `delete`, `admin`
- Permission format: `resource:action` (e.g., `users:read`)

---

### Policies

**Path**: `/policies`

Define ABAC (Attribute-Based Access Control) policies.

- Create attribute-based policies
- Version policy changes
- Test policy evaluation
- View policy decision logs

---

### Organizations

**Path**: `/organizations`

Manage organizational hierarchy.

- Create top-level organizations
- Create nested sub-organizations
- Assign org-level roles
- View org member counts
- Transfer users between orgs

---

### Audit

**Path**: `/audit`

Search and analyze audit events.

| Feature          | Description                                |
|------------------|--------------------------------------------|
| Event Search     | Filter by actor, action, resource, time    |
| Event Details    | Full event metadata, diff, IP, user agent  |
| Export           | CSV/JSON export for compliance             |
| Hash Chain       | Verify audit log integrity                 |
| Real-time Stream | Live event feed (WebSocket)                |

**Event Types**: `user.login`, `user.logout`, `user.create`, `user.delete`, `role.assign`, `policy.change`, `org.create`, `agent.register`, `agent.token_exchange`

---

### Security Center

**Path**: `/security-center`

Centralized security monitoring and configuration.

| Section              | Description                                |
|----------------------|--------------------------------------------|
| Threat Overview      | Failed logins, rate-limited IPs, blocked   |
| MFA Statistics       | Enrollment rate, method breakdown          |
| Session Management   | Active sessions, session timeout policy    |
| Password Policy      | Min length, complexity, history, expiry    |
| IP Allowlist         | Restrict admin access to known IPs         |
| Account Lockout      | Threshold, lockout duration                |

---

### Sessions

**Path**: `/sessions`

View and manage active user sessions.

- List all active sessions (JWT-based)
- Revoke individual sessions
- View session metadata (IP, device, location, issued-at)
- Force logout all sessions for a user

---

### Activity

**Path**: `/activity`

Real-time activity feed showing system events.

- User registrations
- Login/logout events
- Role changes
- Policy updates
- Configuration changes
- Filterable by event type and time range

---

### AI Agents

**Path**: `/agents`

Manage AI agent identities for MCP and automated workflows.

| Action              | Description                              |
|---------------------|------------------------------------------|
| Register Agent      | Create new agent identity                |
| List Agents         | View all registered agents               |
| Exchange Token      | Obtain delegation token for agent        |
| Verify Agent Token  | Validate agent JWT claims                |
| Suspend Agent       | Temporarily disable agent                |
| View Delegation Chain | See full chain of delegated tokens     |

**Agent Types**: `service`, `mcp`, `workflow`

---

### Access Requests (IGA)

**Path**: `/access-requests`

Identity Governance & Administration workflows.

- Submit access request (user requests role/permission)
- Manager approval flow
- Auto-provisioning on approval
- Access certification campaigns
- Review and revoke stale access

---

### OAuth Clients

**Path**: `/oauth-clients`

Manage OAuth 2.0 / OIDC client applications.

| Field         | Description                              |
|---------------|------------------------------------------|
| Client ID     | Public identifier                        |
| Client Secret | Confidential (shown once on creation)    |
| Redirect URIs | Allowed callback URLs                    |
| Grant Types   | authorization_code, client_credentials   |
| Scopes        | Allowed scopes for this client           |

---

### SSO / SAML

**Path**: `/sso`

Configure Single Sign-On providers.

| Provider       | Protocol | Status     |
|----------------|----------|------------|
| SAML 2.0       | SAML     | Supported  |
| OIDC           | OIDC     | Supported  |
| Google         | OAuth    | Supported  |
| Microsoft      | OAuth    | Supported  |
| GitHub         | OAuth    | Supported  |
| LDAP           | LDAP     | Supported  |

**SAML Configuration**:
- IdP metadata XML upload
- SP metadata download
- Certificate management
- Attribute mapping
- Single Logout (SLO)

---

### MFA / Login Flows

**Path**: `/flows`

Configure authentication step sequences.

- Define login step sequences (password → MFA → device check)
- Configure step-up triggers
- Set fallback methods
- Passwordless flow configuration (WebAuthn)

---

### Settings

**Path**: `/settings`

System-wide configuration.

#### Sub-pages:

| Page              | Description                                |
|-------------------|--------------------------------------------|
| Tenant Config     | Tenant name, default roles, branding       |
| Branding          | Logo, colors, custom CSS, email templates  |
| Certificates      | TLS certs, signing keys, rotation          |
| API Keys          | Generate/revoke API keys                   |
| OAuth Clients     | (Alias to /oauth-clients)                  |
| Login Flows       | (Alias to /flows)                          |
| MFA Settings      | TOTP, WebAuthn, SMS configuration          |
| Tenant Config     | Multi-tenant settings, RLS policies        |

---

### SCIM Provisioning

**Path**: `/scim`

Configure SCIM 2.0 endpoint for automated user provisioning.

- SCIM endpoint URL: `https://api.ggid.example.com/scim/v2`
- Bearer token authentication
- Supported operations: Create, Read, Update (PATCH), Delete
- Sync from: Okta, Azure AD, JumpCloud, OneLogin

---

### Webhooks

**Path**: `/webhooks`

Configure outbound webhook integrations.

- Register webhook URL
- Select event types to receive
- HMAC-SHA256 signature verification
- Retry configuration (exponential backoff)
- Delivery log with status codes

---

### Monitoring

**Path**: `/monitoring`

System health and performance metrics.

- Service health status (all 7 microservices)
- Database connection pool status
- Redis connection status
- NATS JetStream metrics
- Request latency percentiles (p50, p95, p99)
- Error rate by service

---

### Notifications

**Path**: `/notifications`

Configure alerting and notification channels.

| Channel     | Use Case                          |
|-------------|-----------------------------------|
| Email       | Security alerts, summaries        |
| Slack       | Real-time operational alerts      |
| Webhook     | Custom integrations               |
| SIEM        | Compliance forwarding (CEF/LEEF)  |

---

### API Explorer

**Path**: `/api-explorer`

Interactive API documentation with try-it functionality.

- Browse all REST endpoints
- Send test requests directly from console
- View request/response examples
- Copy curl commands
- Swagger/OpenAPI integration

---

### Onboarding Wizard

**Path**: `/onboarding`

First-time setup wizard for new tenants.

1. Create admin user
2. Configure tenant branding
3. Set up authentication (password policy, MFA)
4. Create initial roles and permissions
5. Invite team members
6. Configure SSO (optional)
7. Review and launch

---

### Profile

**Path**: `/profile`

Self-service user profile management.

- Update display name, email, phone
- Change password
- Manage MFA devices (enroll/remove TOTP, WebAuthn)
- View active sessions
- Generate personal API keys

## Navigation

The console sidebar organizes pages into logical groups:

```
Dashboard
─────────
Users & Access
  Users
  Groups
  Roles
  Permissions
  Policies
─────────
Organizations
─────────
Security
  Security Center
  Sessions
  Activity
  Access Requests
─────────
Integrations
  OAuth Clients
  SSO / SAML
  SCIM Provisioning
  Webhooks
  AI Agents
─────────
Audit & Monitoring
  Audit Log
  Monitoring
  Notifications
─────────
Settings
  Tenant Config
  Branding
  Certificates
  API Keys
  Login Flows
  MFA Settings
─────────
API Explorer
```

## Permissions Required

| Console Area       | Required Scope              |
|--------------------|-----------------------------|
| Dashboard          | `dashboard:read`            |
| Users              | `users:read` / `users:write`|
| Roles              | `roles:read` / `roles:write`|
| Organizations      | `orgs:read` / `orgs:write`  |
| Audit              | `audit:read`                |
| Security Center    | `security:read`             |
| Settings           | `settings:write`            |
| AI Agents          | `agents:write`              |
| OAuth Clients      | `oauth:write`               |
| API Keys           | `apikeys:write`             |

## See Also

- [Quick Start](quick-start.md)
- [5-Minute Quickstart](5-minute-quickstart.md)
- [API Reference](api-reference.md)
- [Security Overview](../architecture/security-overview.md)
- [Multi-Tenant Guide](multi-tenant-guide.md)
