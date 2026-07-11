# Next.js 15 Integration Tutorial

This tutorial covers integrating GGID authentication into a Next.js 15 application with App Router, SSR authentication, middleware-based route protection, and server components.

## Prerequisites

- Node.js 18+
- Next.js 15+
- GGID Gateway running at `https://api.ggid.example.com`

## Step 1: Install Dependencies

```bash
npm install jose @ggid/node
```

> `jose` is used for JWT verification on the server side (Edge runtime compatible).

## Step 2: Environment Variables

Create `.env.local`:

```bash
GGID_GATEWAY=https://api.ggid.example.com
GGID_TENANT_ID=00000000-0000-0000-0000-000000000001
GGID_JWKS_URL=https://api.ggid.example.com/.well-known/jwks.json
NEXT_PUBLIC_GGID_GATEWAY=https://api.ggid.example.com
NEXT_PUBLIC_GGID_TENANT_ID=00000000-0000-0000-0000-000000000001
```

## Step 3: JWT Verification Utility

Create `lib/jwt.ts`:

```typescript
import { createRemoteJWKSet, jwtVerify } from 'jose';

const JWKS = createRemoteJWKSet(
  new URL(process.env.GGID_JWKS_URL!)
);

export interface GGIDClaims {
  sub: string;
  tenant_id: string;
  scope: string;
  exp: number;
  iat: number;
  jti: string;
  email?: string;
  roles?: string[];
}

export async function verifyJWT(token: string): Promise<GGIDClaims | null> {
  try {
    const { payload } = await jwtVerify(token, JWKS, {
      issuer: process.env.GGID_GATEWAY,
      algorithms: ['RS256'],
    });
    return payload as unknown as GGIDClaims;
  } catch {
    return null;
  }
}
```

## Step 4: Middleware for SSR Auth

Create `middleware.ts` (project root):

```typescript
import { NextRequest, NextResponse } from 'next/server';
import { verifyJWT } from './lib/jwt';

const PUBLIC_ROUTES = ['/login', '/register', '/api/auth/login', '/api/auth/register'];

export async function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;

  // Allow public routes
  if (PUBLIC_ROUTES.some(r => pathname.startsWith(r))) {
    return NextResponse.next();
  }

  // Check for access token in cookie
  const accessToken = request.cookies.get('ggid_access_token')?.value;

  if (!accessToken) {
    const loginUrl = new URL('/login', request.url);
    loginUrl.searchParams.set('redirect', pathname);
    return NextResponse.redirect(loginUrl);
  }

  // Verify JWT on the server
  const claims = await verifyJWT(accessToken);

  if (!claims) {
    const loginUrl = new URL('/login', request.url);
    loginUrl.searchParams.set('redirect', pathname);
    return NextResponse.redirect(loginUrl);
  }

  // Check admin routes
  if (pathname.startsWith('/admin') && !claims.scope?.includes('admin')) {
    return NextResponse.redirect(new URL('/dashboard', request.url));
  }

  // Inject user info into request headers for server components
  const response = NextResponse.next();
  response.headers.set('x-user-id', claims.sub);
  response.headers.set('x-user-tenant', claims.tenant_id);
  response.headers.set('x-user-scopes', claims.scope);

  return response;
}

export const config = {
  matcher: ['/((?!_next/static|_next/image|favicon.ico).*)'],
};
```

## Step 5: Server-Side Auth Actions

Create `app/actions/auth.ts`:

```typescript
'use server';

import { cookies } from 'next/headers';
import { redirect } from 'next/navigation';

const GGID_GATEWAY = process.env.GGID_GATEWAY!;
const TENANT_ID = process.env.GGID_TENANT_ID!;

export async function loginAction(formData: FormData) {
  const username = formData.get('username') as string;
  const password = formData.get('password') as string;

  const resp = await fetch(`${GGID_GATEWAY}/api/v1/auth/login`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-Tenant-ID': TENANT_ID,
    },
    body: JSON.stringify({ username, password }),
  });

  if (!resp.ok) {
    return { error: 'Invalid credentials' };
  }

  const tokens = await resp.json();

  // Store tokens in HTTP-only cookies
  cookies().set('ggid_access_token', tokens.access_token, {
    httpOnly: true,
    secure: process.env.NODE_ENV === 'production',
    sameSite: 'lax',
    maxAge: tokens.expires_in,
    path: '/',
  });

  cookies().set('ggid_refresh_token', tokens.refresh_token, {
    httpOnly: true,
    secure: process.env.NODE_ENV === 'production',
    sameSite: 'lax',
    maxAge: 7 * 24 * 60 * 60, // 7 days
    path: '/',
  });

  redirect('/dashboard');
}

export async function logoutAction() {
  cookies().delete('ggid_access_token');
  cookies().delete('ggid_refresh_token');
  redirect('/login');
}

export async function refreshTokenAction(): Promise<string | null> {
  const refreshToken = cookies().get('ggid_refresh_token')?.value;
  if (!refreshToken) return null;

  const resp = await fetch(`${GGID_GATEWAY}/api/v1/auth/refresh`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-Tenant-ID': TENANT_ID,
    },
    body: JSON.stringify({ refresh_token: refreshToken }),
  });

  if (!resp.ok) return null;

  const tokens = await resp.json();
  cookies().set('ggid_access_token', tokens.access_token, {
    httpOnly: true,
    secure: process.env.NODE_ENV === 'production',
    sameSite: 'lax',
    maxAge: tokens.expires_in,
    path: '/',
  });

  return tokens.access_token;
}
```

## Step 6: Server Component — Get Current User

Create `lib/session.ts`:

```typescript
import { cookies } from 'next/headers';
import { verifyJWT, GGIDClaims } from './jwt';

export async function getCurrentUser(): Promise<GGIDClaims | null> {
  const token = cookies().get('ggid_access_token')?.value;
  if (!token) return null;
  return verifyJWT(token);
}

export async function requireAuth(): Promise<GGIDClaims> {
  const user = await getCurrentUser();
  if (!user) {
    throw new Error('Unauthorized');
  }
  return user;
}

export async function requireAdmin(): Promise<GGIDClaims> {
  const user = await requireAuth();
  if (!user.scope?.includes('admin')) {
    throw new Error('Forbidden');
  }
  return user;
}
```

## Step 7: Dashboard (Server Component)

Create `app/dashboard/page.tsx`:

```tsx
import { getCurrentUser } from '@/lib/session';
import { redirect } from 'next/navigation';
import { LogoutButton } from './LogoutButton';

export default async function Dashboard() {
  const user = await getCurrentUser();
  if (!user) redirect('/login');

  // Fetch users server-side
  const resp = await fetch(`${process.env.GGID_GATEWAY}/api/v1/users`, {
    headers: {
      'Authorization': `Bearer ${cookies().get('ggid_access_token')?.value}`,
      'X-Tenant-ID': process.env.GGID_TENANT_ID!,
    },
    cache: 'no-store',
  });
  const users = await resp.json();

  return (
    <div className="min-h-screen bg-gray-50">
      <nav className="bg-white shadow px-6 py-4 flex justify-between items-center">
        <h1 className="text-xl font-bold">Dashboard</h1>
        <div className="flex items-center gap-4">
          <span className="text-sm text-gray-600">User: {user.sub}</span>
          <LogoutButton />
        </div>
      </nav>

      <main className="max-w-7xl mx-auto py-6 px-4">
        <h2 className="text-lg font-semibold mb-4">Users ({users.items?.length || 0})</h2>
        <div className="bg-white shadow rounded-lg overflow-hidden">
          <table className="w-full">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-4 py-2 text-left">Username</th>
                <th className="px-4 py-2 text-left">Email</th>
              </tr>
            </thead>
            <tbody>
              {users.items?.map((u: any) => (
                <tr key={u.id} className="border-t">
                  <td className="px-4 py-2">{u.username}</td>
                  <td className="px-4 py-2">{u.email}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </main>
    </div>
  );
}

import { cookies } from 'next/headers';
```

Create `app/dashboard/LogoutButton.tsx`:

```tsx
'use client';

import { logoutAction } from '@/app/actions/auth';

export function LogoutButton() {
  return (
    <form action={logoutAction}>
      <button type="submit" className="text-sm text-blue-600 hover:underline">
        Logout
      </button>
    </form>
  );
}
```

## Step 8: Login Page (Client Component)

Create `app/login/page.tsx`:

```tsx
'use client';

import { useState } from 'react';
import { loginAction } from '@/app/actions/auth';
import { useSearchParams } from 'next/navigation';

export default function LoginPage() {
  const [error, setError] = useState<string | null>(null);
  const searchParams = useSearchParams();
  const redirect = searchParams.get('redirect') || '/dashboard';

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <div className="max-w-md w-full space-y-8">
        <h2 className="text-center text-3xl font-bold">Sign in</h2>
        <form
          action={async (formData) => {
            const result = await loginAction(formData);
            if (result?.error) setError(result.error);
          }}
          className="space-y-4"
        >
          <input
            type="text"
            name="username"
            placeholder="Username"
            className="w-full px-3 py-2 border rounded"
            required
          />
          <input
            type="password"
            name="password"
            placeholder="Password"
            className="w-full px-3 py-2 border rounded"
            required
          />
          {error && <p className="text-red-500 text-sm">{error}</p>}
          <button
            type="submit"
            className="w-full py-2 bg-blue-600 text-white rounded hover:bg-blue-700"
          >
            Sign in
          </button>
        </form>
      </div>
    </div>
  );
}
```

## Step 9: API Route Handler

Create `app/api/users/route.ts`:

```typescript
import { NextRequest, NextResponse } from 'next/server';
import { getCurrentUser } from '@/lib/session';

const GGID_GATEWAY = process.env.GGID_GATEWAY!;
const TENANT_ID = process.env.GGID_TENANT_ID!;

export async function GET(request: NextRequest) {
  const user = await getCurrentUser();
  if (!user) {
    return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });
  }

  const token = request.cookies.get('ggid_access_token')?.value;
  const resp = await fetch(`${GGID_GATEWAY}/api/v1/users`, {
    headers: {
      'Authorization': `Bearer ${token}`,
      'X-Tenant-ID': TENANT_ID,
    },
  });

  const data = await resp.json();
  return NextResponse.json(data);
}

export async function POST(request: NextRequest) {
  const user = await getCurrentUser();
  if (!user?.scope?.includes('users:write')) {
    return NextResponse.json({ error: 'Forbidden' }, { status: 403 });
  }

  const body = await request.json();
  const token = request.cookies.get('ggid_access_token')?.value;

  const resp = await fetch(`${GGID_GATEWAY}/api/v1/users`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`,
      'X-Tenant-ID': TENANT_ID,
    },
    body: JSON.stringify(body),
  });

  const data = await resp.json();
  return NextResponse.json(data, { status: resp.status });
}
```

## Complete File Structure

```
project/
├── middleware.ts                    # Edge middleware for SSR auth
├── .env.local                       # Environment variables
├── lib/
│   ├── jwt.ts                       # JWT verification (jose)
│   └── session.ts                   # Server-side session helpers
├── app/
│   ├── actions/
│   │   └── auth.ts                  # Server actions (login, logout)
│   ├── api/
│   │   └── users/
│   │       └── route.ts             # API route handler
│   ├── login/
│   │   └── page.tsx                 # Login page (client)
│   └── dashboard/
│       ├── page.tsx                 # Dashboard (server component)
│       └── LogoutButton.tsx         # Logout button (client)
```

## Key Differences from React SPA

| Aspect | React SPA | Next.js 15 |
|--------|-----------|------------|
| Token storage | sessionStorage | HTTP-only cookie |
| Auth check | Client-side | Middleware (edge) |
| API calls | Client → Gateway | Server → Gateway (SSR) |
| Refresh | Client timer | Server action |
| SEO | Poor (CSR) | Good (SSR) |

## See Also

- [React SPA Integration](react-integration.md)
- [Node.js SDK Guide](../guides/node-sdk-guide.md)
- [API Reference](../api/rest-api.md)
