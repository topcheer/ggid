# React SPA Integration Tutorial

This tutorial walks through integrating GGID authentication into a React single-page application using the `@ggid/react` SDK.

## Prerequisites

- Node.js 18+
- A GGID instance running (Gateway at `https://api.ggid.example.com`)
- A tenant ID (default: `00000000-0000-0000-0000-000000000001`)

## Step 1: Install Dependencies

```bash
npm install @ggid/react @ggid/node react-router-dom
```

## Step 2: Configure the Auth Provider

Create `src/auth/GGIDProvider.tsx`:

```tsx
import { createContext, useContext, useState, useCallback, ReactNode } from 'react';

const GGID_GATEWAY = 'https://api.ggid.example.com';
const TENANT_ID = '00000000-0000-0000-0000-000000000001';

interface User {
  id: string;
  username: string;
  email: string;
  roles: string[];
}

interface TokenSet {
  access_token: string;
  refresh_token: string;
  expires_in: number;
}

interface AuthContextType {
  user: User | null;
  tokens: TokenSet | null;
  login: (username: string, password: string) => Promise<void>;
  logout: () => void;
  isLoading: boolean;
  error: string | null;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function GGIDAuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [tokens, setTokens] = useState<TokenSet | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Load tokens from sessionStorage on mount
  useState(() => {
    const stored = sessionStorage.getItem('ggid_tokens');
    if (stored) {
      const parsed = JSON.parse(stored);
      setTokens(parsed);
      fetchUser(parsed.access_token).then(setUser).catch(() => {
        sessionStorage.removeItem('ggid_tokens');
      });
    }
  });

  const login = useCallback(async (username: string, password: string) => {
    setIsLoading(true);
    setError(null);
    try {
      const resp = await fetch(`${GGID_GATEWAY}/api/v1/auth/login`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': TENANT_ID,
        },
        body: JSON.stringify({ username, password }),
      });

      if (!resp.ok) {
        const data = await resp.json();
        throw new Error(data.error?.message || 'Login failed');
      }

      const tokenSet: TokenSet = await resp.json();
      setTokens(tokenSet);
      sessionStorage.setItem('ggid_tokens', JSON.stringify(tokenSet));

      const u = await fetchUser(tokenSet.access_token);
      setUser(u);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Login failed');
    } finally {
      setIsLoading(false);
    }
  }, []);

  const logout = useCallback(() => {
    setUser(null);
    setTokens(null);
    sessionStorage.removeItem('ggid_tokens');
  }, []);

  return (
    <AuthContext.Provider value={{ user, tokens, login, logout, isLoading, error }}>
      {children}
    </AuthContext.Provider>
  );
}

async function fetchUser(accessToken: string): Promise<User> {
  const resp = await fetch(`${GGID_GATEWAY}/api/v1/users/me`, {
    headers: {
      'Authorization': `Bearer ${accessToken}`,
      'X-Tenant-ID': TENANT_ID,
    },
  });
  if (!resp.ok) throw new Error('Failed to fetch user');
  return resp.json();
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used within GGIDAuthProvider');
  return ctx;
}

// Token refresh utility
export async function refreshAccessToken(refreshToken: string): Promise<TokenSet> {
  const resp = await fetch(`${GGID_GATEWAY}/api/v1/auth/refresh`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-Tenant-ID': TENANT_ID,
    },
    body: JSON.stringify({ refresh_token: refreshToken }),
  });
  if (!resp.ok) throw new Error('Token refresh failed');
  return resp.json();
}
```

## Step 3: Create Protected Route Component

Create `src/auth/ProtectedRoute.tsx`:

```tsx
import { Navigate, useLocation } from 'react-router-dom';
import { useAuth } from './GGIDProvider';

export function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { user, isLoading } = useAuth();
  const location = useLocation();

  if (isLoading) {
    return <div className="flex items-center justify-center min-h-screen">Loading...</div>;
  }

  if (!user) {
    return <Navigate to="/login" state={{ from: location }} replace />;
  }

  return <>{children}</>;
}

// Admin-only route
export function AdminRoute({ children }: { children: React.ReactNode }) {
  const { user } = useAuth();

  if (!user?.roles?.includes('admin')) {
    return <Navigate to="/dashboard" replace />;
  }

  return <>{children}</>;
}
```

## Step 4: Create API Client with Auto-Refresh

Create `src/api/client.ts`:

```tsx
import { useAuth, refreshAccessToken } from '../auth/GGIDProvider';

const GGID_GATEWAY = 'https://api.ggid.example.com';
const TENANT_ID = '00000000-0000-0000-0000-000000000001';

export class GGIDAPIClient {
  private tokens: any;
  private onTokenRefresh: (tokens: any) => void;

  constructor(tokens: any, onTokenRefresh: (tokens: any) => void) {
    this.tokens = tokens;
    this.onTokenRefresh = onTokenRefresh;
  }

  private async request(path: string, options: RequestInit = {}): Promise<Response> {
    let resp = await fetch(`${GGID_GATEWAY}${path}`, {
      ...options,
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${this.tokens.access_token}`,
        'X-Tenant-ID': TENANT_ID,
        ...options.headers,
      },
    });

    // Auto-refresh on 401
    if (resp.status === 401 && this.tokens.refresh_token) {
      const newTokens = await refreshAccessToken(this.tokens.refresh_token);
      this.tokens = newTokens;
      this.onTokenRefresh(newTokens);
      sessionStorage.setItem('ggid_tokens', JSON.stringify(newTokens));

      // Retry original request
      resp = await fetch(`${GGID_GATEWAY}${path}`, {
        ...options,
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${newTokens.access_token}`,
          'X-Tenant-ID': TENANT_ID,
          ...options.headers,
        },
      });
    }

    return resp;
  }

  async listUsers(page = 1, pageSize = 20) {
    const resp = await this.request(`/api/v1/users?page=${page}&page_size=${pageSize}`);
    return resp.json();
  }

  async createUser(username: string, email: string, password: string) {
    const resp = await this.request('/api/v1/users', {
      method: 'POST',
      body: JSON.stringify({ username, email, password }),
    });
    return resp.json();
  }

  async deleteUser(id: string) {
    const resp = await this.request(`/api/v1/users/${id}`, { method: 'DELETE' });
    return resp.ok;
  }

  async getAuditEvents(eventType?: string) {
    const params = eventType ? `?event_type=${eventType}` : '';
    const resp = await this.request(`/api/v1/audit/events${params}`);
    return resp.json();
  }
}
```

## Step 5: Login Page

Create `src/pages/Login.tsx`:

```tsx
import { useState } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import { useAuth } from '../auth/GGIDProvider';

export default function Login() {
  const { login, isLoading, error } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');

  const from = (location.state as any)?.from?.pathname || '/dashboard';

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    await login(username, password);
    navigate(from, { replace: true });
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <div className="max-w-md w-full space-y-8">
        <div>
          <h2 className="text-center text-3xl font-bold">Sign in to GGID</h2>
        </div>
        <form className="mt-8 space-y-6" onSubmit={handleSubmit}>
          <div>
            <input
              type="text"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              placeholder="Username"
              className="w-full px-3 py-2 border rounded"
              required
            />
          </div>
          <div>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="Password"
              className="w-full px-3 py-2 border rounded"
              required
            />
          </div>
          {error && <p className="text-red-500 text-sm">{error}</p>}
          <button
            type="submit"
            disabled={isLoading}
            className="w-full py-2 px-4 bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50"
          >
            {isLoading ? 'Signing in...' : 'Sign in'}
          </button>
        </form>
      </div>
    </div>
  );
}
```

## Step 6: Dashboard with Data Fetching

Create `src/pages/Dashboard.tsx`:

```tsx
import { useState, useEffect } from 'react';
import { useAuth } from '../auth/GGIDProvider';
import { GGIDAPIClient } from '../api/client';

export default function Dashboard() {
  const { user, tokens, logout, login } = useAuth();
  const [users, setUsers] = useState<any[]>([]);

  useEffect(() => {
    if (!tokens) return;
    const client = new GGIDAPIClient(tokens, (newTokens) => {
      sessionStorage.setItem('ggid_tokens', JSON.stringify(newTokens));
    });
    client.listUsers().then(data => setUsers(data.items || []));
  }, [tokens]);

  return (
    <div className="min-h-screen bg-gray-50">
      <nav className="bg-white shadow">
        <div className="max-w-7xl mx-auto px-4 py-4 flex justify-between">
          <h1 className="text-xl font-bold">GGID Dashboard</h1>
          <div className="flex items-center gap-4">
            <span className="text-sm text-gray-600">{user?.email}</span>
            <button onClick={logout} className="text-sm text-blue-600">Logout</button>
          </div>
        </div>
      </nav>

      <main className="max-w-7xl mx-auto py-6">
        <h2 className="text-lg font-semibold mb-4">Users ({users.length})</h2>
        <div className="bg-white shadow rounded-lg overflow-hidden">
          <table className="w-full">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-4 py-2 text-left">Username</th>
                <th className="px-4 py-2 text-left">Email</th>
                <th className="px-4 py-2 text-left">Status</th>
              </tr>
            </thead>
            <tbody>
              {users.map(u => (
                <tr key={u.id} className="border-t">
                  <td className="px-4 py-2">{u.username}</td>
                  <td className="px-4 py-2">{u.email}</td>
                  <td className="px-4 py-2">{u.status || 'active'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </main>
    </div>
  );
}
```

## Step 7: Wire Up Routes

Create `src/App.tsx`:

```tsx
import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { GGIDAuthProvider } from './auth/GGIDProvider';
import { ProtectedRoute, AdminRoute } from './auth/ProtectedRoute';
import Login from './pages/Login';
import Dashboard from './pages/Dashboard';

export default function App() {
  return (
    <GGIDAuthProvider>
      <BrowserRouter>
        <Routes>
          <Route path="/login" element={<Login />} />
          <Route path="/dashboard" element={
            <ProtectedRoute><Dashboard /></ProtectedRoute>
          } />
          <Route path="/admin" element={
            <ProtectedRoute>
              <AdminRoute>
                <div>Admin Panel</div>
              </AdminRoute>
            </ProtectedRoute>
          } />
          <Route path="*" element={<Login />} />
        </Routes>
      </BrowserRouter>
    </GGIDAuthProvider>
  );
}
```

## Step 8: Token Refresh Timer

Add automatic token refresh in the provider:

```tsx
// Inside GGIDAuthProvider component
useEffect(() => {
  if (!tokens) return;

  const refreshMs = (tokens.expires_in - 60) * 1000; // Refresh 60s before expiry
  const timer = setTimeout(async () => {
    try {
      const newTokens = await refreshAccessToken(tokens.refresh_token);
      setTokens(newTokens);
      sessionStorage.setItem('ggid_tokens', JSON.stringify(newTokens));
    } catch {
      logout();
    }
  }, refreshMs);

  return () => clearTimeout(timer);
}, [tokens]);
```

## MFA Flow (Step-Up Authentication)

```tsx
const [mfaRequired, setMfaRequired] = useState(false);
const [mfaToken, setMfaToken] = useState('');

// In login function, after 200 response:
if (data.mfa_required) {
  setMfaToken(data.mfa_token);
  setMfaRequired(true);
  return;
}

// MFA verification submit:
const verifyMFA = async (code: string) => {
  const resp = await fetch(`${GGID_GATEWAY}/api/v1/auth/mfa/verify`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'X-Tenant-ID': TENANT_ID },
    body: JSON.stringify({ mfa_token: mfaToken, code }),
  });
  if (resp.ok) {
    const tokenSet = await resp.json();
    setTokens(tokenSet);
    setMfaRequired(false);
  }
};
```

## Complete File Structure

```
src/
├── App.tsx
├── api/
│   └── client.ts
├── auth/
│   ├── GGIDProvider.tsx
│   └── ProtectedRoute.tsx
├── pages/
│   ├── Login.tsx
│   └── Dashboard.tsx
└── index.tsx
```

## See Also

- [Next.js Integration](nextjs-integration.md)
- [Node.js SDK Guide](../guides/node-sdk-guide.md)
- [API Reference](../api/rest-api.md)
- [Go Gin Integration](go-gin-integration.md)
