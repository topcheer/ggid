# @ggid/react

React SDK for GGID Identity & Access Management.

## Install

```bash
npm install @ggid/react
```

## Quick Start (3 lines)

```tsx
import { GGIDProvider, useGGIDAuth } from '@ggid/react';

// 1. Wrap your app
<GgidProvider config={{ apiBaseUrl: 'https://api.ggid.dev', tenantId: 'your-tenant-id' }}>
  <App />
</GgidProvider>

// 2. Use the hook
const { isAuthenticated, user, login, logout } = useGGIDAuth();
```

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

### `<ProtectedRoute>`

Redirects to `/login` if not authenticated.

```tsx
import { ProtectedRoute } from '@ggid/react';

<ProtectedRoute><Dashboard /></ProtectedRoute>
```

## Hooks

### `useGGIDAuth()`

Returns auth state and actions.

```tsx
const {
  user,           // GGIDUser | null
  isAuthenticated, // boolean
  isLoading,       // boolean
  error,           // string | null
  login,           // (username, password) => Promise<void>
  logout,          // () => void
  getAccessToken,  // () => string | null
  hasRole,         // (role: string) => boolean
  hasScope,        // (scope: string) => boolean
} = useGGIDAuth();
```

### `useUser()`

Auto-fetches the current user profile from `GET /api/v1/users/me`.

```tsx
const { user, isLoading, error, refresh } = useUser();
```

## Types

```typescript
interface GGIDUser {
  id: string;
  username: string;
  email: string;
  tenant_id: string;
  roles?: string[];
  scopes?: string[];
}
```

## License

Apache-2.0
