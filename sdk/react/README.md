# @ggid/react

React SDK for GGID Identity & Access Management.

## Install

```bash
npm install @ggid/react
```

## Quick Start

```tsx
import { GGIDProvider, useGGIDAuth } from '@ggid/react';

// 1. Wrap your app
<GGIDProvider config={{ apiBaseUrl: 'https://api.ggid.dev', tenantId: 'your-tenant-id' }}>
  <App />
</GGIDProvider>

// 2. Use the hook
const { isAuthenticated, user, login, logout } = useGGIDAuth();
```

## Table of Contents

- [Components](#components)
  - [GGIDProvider](#ggidprovider)
  - [ProtectedRoute](#protectedroute)
  - [RequireScope](#requirescope)
  - [LogoutButton](#logoutbutton)
  - [ErrorBoundary](#errorboundary)
- [Hooks](#hooks)
  - [useGGIDAuth](#useggidauth)
  - [useUser](#useuser)
  - [useRoles](#useroles)
  - [usePermissions](#usepermissions)
  - [useTokenRefresh](#usetokenrefresh)
  - [Data Hooks](#data-hooks)
    - [useUsers](#useusers)
    - [useAuditEvents](#useauditevents)
    - [useAuditStats](#useauditstats)
    - [useAccessRequests](#useaccessrequests)
    - [useAlerts](#usealerts)
    - [useSessions](#usesessions)
    - [useCompliance](#usecompliance)
    - [useOAuthClients](#useoauthclients)
    - [useBranding](#usebranding)
    - [useRetention](#useretention)
- [Types](#types)
- [Examples](#examples)

---

## Components

### `<GGIDProvider>`

Wraps your application to provide auth context.

| Prop | Type | Required | Description |
|------|------|----------|-------------|
| `config.apiBaseUrl` | `string` | Yes | Gateway URL |
| `config.tenantId` | `string` | Yes | Tenant ID |
| `config.clientId` | `string` | No | OAuth client ID |
| `config.redirectUri` | `string` | No | Post-login redirect |
| `config.scopes` | `string[]` | No | Requested scopes |
| `config.storageKey` | `string` | No | Token storage key (default: `ggid_token`) |

```tsx
<GGIDProvider config={{
  apiBaseUrl: 'https://api.ggid.dev',
  tenantId: '00000000-0000-0000-0000-000000000001',
  scopes: ['openid', 'profile', 'email'],
}}>
  <App />
</GGIDProvider>
```

### `<ProtectedRoute>`

Redirects to `/login` if not authenticated.

```tsx
import { ProtectedRoute } from '@ggid/react';

<ProtectedRoute>
  <Dashboard />
</ProtectedRoute>
```

### `<RequireScope>`

Conditionally renders children based on scope/permission checks.

| Prop | Type | Description |
|------|------|-------------|
| `scope` | `string` | Single required scope |
| `anyOf` | `string[]` | User must have ANY of these scopes |
| `allOf` | `string[]` | User must have ALL of these scopes |
| `fallback` | `ReactNode` | Content when unauthorized (default: `null`) |
| `loadingFallback` | `ReactNode` | Content while auth state loads |

```tsx
import { RequireScope } from '@ggid/react';

// Single scope
<RequireScope scope="admin">
  <AdminPanel />
</RequireScope>

// Any of multiple scopes
<RequireScope anyOf={['admin', 'user-manager']}>
  <ManageUsers />
</RequireScope>

// All of multiple scopes
<RequireScope allOf={['users:read', 'users:write']}>
  <EditUsers />
</RequireScope>

// With fallback
<RequireScope scope="admin" fallback={<AccessDenied />}>
  <AdminPanel />
</RequireScope>
```

### `<LogoutButton>`

Pre-built logout button with optional redirect.

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `label` | `string` | `"Logout"` | Button label text |
| `redirectAfterLogout` | `string` | `"/login"` | Redirect path after logout |
| `variant` | `"default" \| "danger" \| "ghost"` | `"default"` | Visual style |
| `className` | `string` | — | Custom CSS class |
| `showIcon` | `boolean` | `true` | Show logout icon |
| `disabled` | `boolean` | `false` | Disable button |

```tsx
import { LogoutButton } from '@ggid/react';

// Basic
<LogoutButton />

// Custom label + redirect
<LogoutButton label="Sign Out" redirectAfterLogout="/goodbye" />

// Danger variant
<LogoutButton variant="danger" label="Exit" />
```

### `<ErrorBoundary>`

Catches render errors in the auth tree and displays a fallback.

```tsx
import { ErrorBoundary } from '@ggid/react';

<ErrorBoundary fallback={<div>Something went wrong.</div>}>
  <App />
</ErrorBoundary>
```

---

## Hooks

### `useGGIDAuth()`

Returns auth state and actions.

```tsx
const {
  user,             // GGIDUser | null
  tokenSet,         // GGIDTokenSet | null
  isAuthenticated,  // boolean
  isLoading,        // boolean
  error,            // string | null
  login,            // (username: string, password: string) => Promise<void>
  logout,           // () => void
  getAccessToken,   // () => string | null
  hasRole,          // (role: string) => boolean
  hasScope,         // (scope: string) => boolean
} = useGGIDAuth();
```

### `useUser()`

Auto-fetches the current user profile from `GET /api/v1/users/me`.

```tsx
const { user, isLoading, error, refresh } = useUser();
```

### `useRoles()`

Provides role and scope helpers built on top of `useGGIDAuth`.

```tsx
import { useRoles } from '@ggid/react';

const {
  roles,          // string[]
  scopes,         // string[]
  hasRole,        // (role: string) => boolean
  hasScope,       // (scope: string) => boolean
  hasAnyRole,     // (...roles: string[]) => boolean
  hasAllRoles,    // (...roles: string[]) => boolean
  hasAnyScope,    // (...scopes: string[]) => boolean
  isAdmin,        // boolean — shortcut for hasRole('admin') || hasScope('admin')
} = useRoles();

// Example: conditional rendering
{isAdmin && <AdminSettings />}
{hasAnyRole('manager', 'supervisor') && <TeamDashboard />}
```

### `usePermissions()`

Fine-grained permission checking with wildcard support.

Permissions are derived from the user's scopes and normalized to `resource:action` format. Supports wildcard matching (`users:*` satisfies `users:read`).

```tsx
import { usePermissions } from '@ggid/react';

const {
  permissions,         // string[] — normalized permissions
  hasPermission,       // (permission: string) => boolean
  hasAnyPermission,    // (...permissions: string[]) => boolean
  hasAllPermissions,   // (...permissions: string[]) => boolean
  loading,             // boolean
} = usePermissions();

// Example: check single permission
if (hasPermission('users:read')) { fetchUsers(); }

// Example: check multiple
if (hasAllPermissions('users:read', 'users:write')) {
  enableUserEditing();
}

// Wildcard: 'users:*' in scopes satisfies 'users:read'
// Admin: 'admin' or '*' scope satisfies everything
```

**Permission normalization rules:**
| Scope format | Normalized permission |
|---|---|
| `users:read` | `users:read` (unchanged) |
| `user.read` | `user:read` (dot → colon) |
| `users:*` | Matches any `users:<action>` |
| `admin` or `*` | Matches all permissions |

### `useTokenRefresh()`

Automatically refreshes the access token before it expires (60s threshold). Parses JWT `exp` claim or uses `expires_at`.

```tsx
import { useTokenRefresh } from '@ggid/react';

function App() {
  useTokenRefresh(); // Call once near the root
  return <Dashboard />;
}
```

### Data Hooks

All data hooks follow the same pattern: `isLoading`, `error`, and a `refetch()` function. Mutation methods auto-refetch the list after success.

---

#### `useUsers()`

User list + CRUD + role assignment.

```tsx
import { useUsers } from '@ggid/react';

const {
  users,         // GGIDUserRecord[]
  isLoading,
  error,
  createUser,    // (input: CreateUserInput) => Promise<GGIDUserRecord | null>
  updateUser,    // (id: string, input: UpdateUserInput) => Promise<boolean>
  deleteUser,    // (id: string) => Promise<boolean>
  assignRole,    // (userId: string, roleId: string) => Promise<boolean>
  removeRole,    // (userId: string, roleId: string) => Promise<boolean>
  refetch,
} = useUsers();
```

---

#### `useAuditEvents(filter)`

Fetches audit events with filtering and pagination.

```tsx
import { useAuditEvents } from '@ggid/react';

const { events, isLoading, error, pagination, refetch } = useAuditEvents({
  eventType: 'user.login',
  dateFrom: '2025-01-01',
  page: 1,
  pageSize: 20,
});
```

| Filter | Type | Description |
|--------|------|-------------|
| `eventType` | `string` | Filter by action type |
| `resourceType` | `string` | Filter by resource type |
| `actorId` | `string` | Filter by actor |
| `result` | `string` | Filter by result (success/failure/denied) |
| `dateFrom` / `dateTo` | `string` | Date range |
| `page` / `pageSize` | `number` | Pagination |

---

#### `useAuditStats(options)`

Aggregate audit statistics for dashboard charts.

```tsx
import { useAuditStats } from '@ggid/react';

const { stats, hourlyData, topActors, isLoading, refetch } = useAuditStats({ hours: 24 });
// stats: { total_events_24h, failed_logins_24h, events_by_action, ... }
// hourlyData: [{ hour, count, failed, succeeded }]
// topActors: [{ actor_id, actor_name, count }]
```

---

#### `useAccessRequests(statusFilter)`

IGA access request workflow: list, create, approve, reject.

```tsx
import { useAccessRequests } from '@ggid/react';

const { requests, createRequest, approveRequest, rejectRequest, refetch } =
  useAccessRequests('pending');
```

---

#### `useAlerts()`

CRUD for audit alerting rules.

```tsx
import { useAlerts } from '@ggid/react';

const { rules, createRule, updateRule, deleteRule, toggleRule, refetch } = useAlerts();
```

---

#### `useSessions()`

List and revoke user sessions.

```tsx
import { useSessions } from '@ggid/react';

const { sessions, revokeSession, revokeAllOthers, refetch } = useSessions();
// sessions: [{ id, device, browser, os, ip_address, location, last_active, current }]
```

---

#### `useCompliance(filter)`

Fetch SOC2/HIPAA/GDPR compliance reports with date range.

```tsx
import { useCompliance } from '@ggid/react';

const { reports, downloadReport, refetch } = useCompliance({
  framework: 'soc2',
  dateFrom: '2025-01-01',
});
// downloadReport(id, 'pdf' | 'csv')
```

---

#### `useOAuthClients()`

OAuth client CRUD + secret regeneration.

```tsx
import { useOAuthClients } from '@ggid/react';

const { clients, createClient, updateClient, deleteClient, regenerateSecret, refetch } =
  useOAuthClients();
// createClient returns { client, client_secret } — secret only shown once
// regenerateSecret(id) returns new secret string
```

---

#### `useBranding()`

Fetch and update per-tenant branding configuration.

```tsx
import { useBranding } from '@ggid/react';

const { branding, updateBranding, refetch } = useBranding();
// branding: { logo_url, primary_color, css_override, custom_domain, ... }
```

---

#### `useRetention()`

Fetch and update audit log retention policy.

```tsx
import { useRetention } from '@ggid/react';

const { policy, updatePolicy, refetch } = useRetention();
// policy: { max_age_days, max_events, archive_enabled, compliance_mode }
```

---

## Types

```typescript
interface GGIDConfig {
  apiBaseUrl: string;
  tenantId: string;
  clientId?: string;
  redirectUri?: string;
  scopes?: string[];
  storageKey?: string;
}

interface GGIDUser {
  id: string;
  username: string;
  email: string;
  tenant_id: string;
  roles?: string[];
  scopes?: string[];
}

interface GGIDTokenSet {
  access_token: string;
  refresh_token?: string;
  expires_at?: number;
  token_type?: string;
}
```

---

## Examples

### Complete Login Flow

See [`examples/login-example.tsx`](./examples/login-example.tsx) for a full login experience with GGIDProvider, ProtectedRoute, useTokenRefresh, and ErrorBoundary.

### Multi-Tenant App

See [`examples/multi-tenant-example.tsx`](./examples/multi-tenant-example.tsx) for tenant switching with session isolation and useRoles.

### Admin Dashboard

See [`examples/dashboard-example.tsx`](./examples/dashboard-example.tsx) for a complete admin panel combining useGGIDAuth, useUser, useRoles, usePermissions, useAuditEvents, RequireScope, and LogoutButton.

### Permission-Based Access Control

```tsx
import { GGIDProvider, RequireScope, usePermissions, LogoutButton } from '@ggid/react';

function App() {
  return (
    <GGIDProvider config={config}>
      <RequireScope scope="admin" fallback={<AccessDenied />}>
        <AdminPanel />
      </RequireScope>

      <RequireScope anyOf={['users:read', 'users:write']}>
        <UserManagement />
      </RequireScope>

      <LogoutButton variant="danger" redirectAfterLogout="/goodbye" />
    </GGIDProvider>
  );
}
```

## License

Apache-2.0
