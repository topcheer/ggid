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
<GgidProvider config={{ apiBaseUrl: 'https://api.ggid.dev', tenantId: 'your-tenant-id' }}>
  <App />
</GgidProvider>

// 2. Use the hook
function App() {
  const { isAuthenticated, login, user } = useGGIDAuth();
  if (!isAuthenticated) return <LoginForm onSubmit={login} />;
  return <Dashboard user={user} />;
}
```

## API Reference

### `<GGIDProvider>`

| Prop | Type | Required | Description |
|------|------|----------|-------------|
| `config.apiBaseUrl` | `string` | Yes | Gateway URL |
| `config.tenantId` | `string` | Yes | Tenant UUID |
| `config.clientId` | `string` | No | OAuth client ID |
| `config.redirectUri` | `string` | No | Post-login redirect |
| `config.scopes` | `string[]` | No | Token scopes |
| `config.storageKey` | `string` | No | localStorage key (default: `ggid_token`) |

### `useGGIDAuth()`

Returns `GGIDAuthContextValue`:

| Field | Type | Description |
|-------|------|-------------|
| `user` | `GGIDUser \| null` | Current user profile |
| `isAuthenticated` | `boolean` | Auth state |
| `isLoading` | `boolean` | Loading state |
| `error` | `string \| null` | Last error |
| `login(username, password)` | `Promise<void>` | Login |
| `logout()` | `void` | Clear session |
| `getAccessToken()` | `string \| null` | Raw JWT |
| `hasRole(role)` | `boolean` | Check role |
| `hasScope(scope)` | `boolean` | Check scope |

### `<ProtectedRoute>`

```tsx
import { ProtectedRoute } from '@ggid/react';

<ProtectedRoute loginPath="/login">
  <AdminPanel />
</ProtectedRoute>
```

Redirects to `loginPath` if not authenticated.

### `useUser()`

```tsx
import { useUser } from '@ggid/react';

function Profile() {
  const { user, isLoading } = useUser();
  if (isLoading) return <Spinner />;
  return <div>{user?.email}</div>;
}
```

Auto-fetches `GET /api/v1/users/me` on mount.

## License

Apache-2.0
