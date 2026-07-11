# React/Frontend SDK Analysis

> Competitive analysis of Auth0, Clerk, and Logto React SDKs. API design recommendations for GGID React SDK.

---

## Competitor Analysis

### Auth0 SDK (`@auth0/auth0-react`)

**Key APIs:**
- `Auth0Provider` — wraps app, handles OAuth flow
- `useAuth0()` — returns `{ isAuthenticated, user, loginWithRedirect, logout, getAccessTokenSilently }`
- `withAuthenticationRequired(Component)` — HOC for protected routes
- JWT managed in memory or sessionStorage (configurable)

**Bundle size:** ~35KB minified

**SSR support:** `@auth0/nextjs-auth0` for Next.js (server-side sessions)

### Clerk SDK (`@clerk/react`)

**Key APIs:**
- `ClerkProvider` — wraps app
- `useUser()` — `{ user, isSignedIn }`
- `useAuth()` — `{ isSignedIn, userId, getToken }`
- `SignIn`, `SignUp`, `UserProfile` — pre-built UI components
- `SignedIn`/`SignedOut` — conditional rendering components

**Bundle size:** ~120KB minified (includes UI components)

**SSR support:** Built-in middleware for Next.js, edge-compatible

### Logto SDK (`@logto/react`)

**Key APIs:**
- `LogtoProvider` — wraps app with config
- `useLogto()` — `{ isAuthenticated, signIn, signOut, getAccessToken, fetchUserInfo }`
- OIDC-based, stores tokens in sessionStorage

**Bundle size:** ~25KB minified

**SSR support:** `@logto/next` for Next.js integration

---

## Comparison Matrix

| Feature | Auth0 | Clerk | Logto | GGID (proposed) |
|---------|-------|-------|-------|-----------------|
| Provider component | `Auth0Provider` | `ClerkProvider` | `LogtoProvider` | `GGIDProvider` |
| User hook | `useAuth0().user` | `useUser()` | `useLogto().fetchUserInfo` | `useUser()` |
| Auth hook | `useAuth0()` | `useAuth()` | `useLogto()` | `useAuth()` |
| Token access | `getAccessTokenSilently` | `getToken` | `getAccessToken` | `getAccessToken` |
| Pre-built UI | No | Yes | No | No (headless) |
| Protected route HOC | `withAuthenticationRequired` | `<SignedIn>` | Manual | `<RequireAuth>` |
| Bundle size | 35KB | 120KB | 25KB | ~20KB (target) |
| Token storage | Memory/Session | Memory/Cookie | Session | Memory (default) |
| Refresh | Hidden | Automatic | Manual | Automatic |
| Multi-tenant | No | No | No | Yes (tenant in hook) |

---

## GGID React SDK Design Proposal

### Package

```
@ggid/react — ~20KB minified, zero UI dependencies
```

### Core API

```typescript
import { GGIDProvider, useAuth, useUser, RequireAuth } from '@ggid/react';

// 1. Provider
function App() {
  return (
    <GGIDProvider
      domain="localhost:8080"
      tenantId="00000000-0000-0000-0000-000000000001"
      redirectUri={window.location.origin}
    >
      <Routes />
    </GGIDProvider>
  );
}

// 2. useAuth hook
function LoginButton() {
  const { isAuthenticated, login, logout } = useAuth();
  return isAuthenticated
    ? <button onClick={logout}>Sign Out</button>
    : <button onClick={login}>Sign In</button>;
}

// 3. useUser hook (with tenant awareness)
function Profile() {
  const { user, tenantId, roles, scopes } = useUser();
  return <div>{user.username} ({tenantId})</div>;
}

// 4. Token access
async function apiCall() {
  const { getAccessToken } = useAuth();
  const token = await getAccessToken();
  const resp = await fetch('/api/users', {
    headers: { Authorization: `Bearer ${token}` }
  });
}

// 5. Protected route
function ProtectedPage() {
  return (
    <RequireAuth fallback={<LoginScreen />}>
      <Dashboard />
    </RequireAuth>
  );
}

// 6. Scope-based guard
function AdminPanel() {
  return (
    <RequireScope scope="admin" fallback={<Forbidden />}>
      <AdminDashboard />
    </RequireScope>
  );
}
```

### Token Management

```typescript
// Default: in-memory (most secure)
// Configurable: sessionStorage, localStorage
<GGIDProvider tokenStorage="memory" autoRefresh={true}>
```

- **Auto-refresh:** SDK refreshes JWT 30s before expiry
- **Tenant switching:** `useAuth().switchTenant(tenantId)` for multi-tenant apps
- **Scope checking:** `useAuth().hasScope('read:users')` boolean helper

### SSR (Next.js)

```typescript
// app/providers.tsx
'use client';
import { GGIDProvider } from '@ggid/react';

export function Providers({ children }) {
  return (
    <GGIDProvider domain={process.env.GGID_URL} tenantId={process.env.TENANT_ID}>
      {children}
    </GGIDProvider>
  );
}
```

Server-side token verification via middleware:

```typescript
// middleware.ts
import { verifyGGIDToken } from '@ggid/react/server';

export async function middleware(request) {
  const token = request.cookies.get('ggid_token')?.value;
  if (!token || !(await verifyGGIDToken(token, process.env.JWKS_URL))) {
    return NextResponse.redirect(new URL('/login', request.url));
  }
}
```

---

## Implementation Priority

| Component | Effort | Priority |
|-----------|--------|----------|
| `GGIDProvider` + context | 2 days | P0 |
| `useAuth` hook | 2 days | P0 |
| `useUser` hook | 1 day | P0 |
| `RequireAuth` component | 0.5 days | P1 |
| `RequireScope` component | 0.5 days | P1 |
| Auto-refresh | 1 day | P1 |
| Tenant switching | 1 day | P2 |
| Next.js middleware | 1 day | P2 |
| **Total** | **~9 days** | |

---

## Key Differentiators vs Competitors

1. **Multi-tenant native** — `tenantId` in provider, `switchTenant` in hook (no competitor has this)
2. **Lightweight** — ~20KB target vs Clerk's 120KB
3. **Headless** — no pre-built UI (unlike Clerk), developers use their own components
4. **RBAC built-in** — `RequireScope` component, `hasScope()` helper
5. **Agent Identity** — `useAgent()` hook for AI agent token exchange

---

*See: [SDK Quickstart](../quickstart/sdk-quickstart.md) | [3-Line Integration](../quickstart/3-line-integration.md) | [Gap Closure Report](gap-closure-report.md)*

*Last updated: 2025-07-11*
