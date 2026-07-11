/**
 * GGID React SDK — Login Flow Example
 *
 * Demonstrates a complete login experience using @ggid/react:
 * - GGIDProvider wrapping the app
 * - ProtectedRoute for authenticated-only pages
 * - useGGIDAuth for login/logout
 * - useUser for user profile data
 * - useTokenRefresh for automatic token renewal
 * - ErrorBoundary for graceful error handling
 */

import React from 'react';
import {
  GGIDProvider,
  useGGIDAuth,
  useUser,
  ProtectedRoute,
  ErrorBoundary,
  useTokenRefresh,
  type GGIDConfig,
} from '../src';

// ─── Configuration ───────────────────────────────────────────

const ggidConfig: GGIDConfig = {
  apiBaseUrl: process.env.NEXT_PUBLIC_GGID_API_URL || 'http://localhost:8080',
  tenantId: process.env.NEXT_PUBLIC_GGID_TENANT_ID || '00000000-0000-0000-0000-000000000001',
  clientId: process.env.NEXT_PUBLIC_GGID_CLIENT_ID,
  redirectUri: typeof window !== 'undefined' ? window.location.origin : '',
  scopes: ['openid', 'profile', 'email', 'offline_access'],
};

// ─── Login Form Component ────────────────────────────────────

function LoginForm() {
  const { login, isLoading, error } = useGGIDAuth();
  const [username, setUsername] = React.useState('');
  const [password, setPassword] = React.useState('');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await login(username, password);
    } catch (err) {
      console.error('Login failed:', err);
    }
  };

  return (
    <div style={{ maxWidth: 400, margin: '80px auto' }}>
      <h2>Sign in to GGID</h2>
      {error && (
        <div style={{ color: 'red', marginBottom: 16 }}>
          {error}
        </div>
      )}
      <form onSubmit={handleSubmit}>
        <div style={{ marginBottom: 12 }}>
          <label>Username</label>
          <input
            type="text"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            placeholder="Enter username"
            style={{ width: '100%', padding: 8, borderRadius: 4 }}
            required
          />
        </div>
        <div style={{ marginBottom: 12 }}>
          <label>Password</label>
          <input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="Enter password"
            style={{ width: '100%', padding: 8, borderRadius: 4 }}
            required
          />
        </div>
        <button
          type="submit"
          disabled={isLoading}
          style={{
            width: '100%',
            padding: 10,
            borderRadius: 4,
            background: '#6366f1',
            color: 'white',
            border: 'none',
            cursor: isLoading ? 'not-allowed' : 'pointer',
          }}
        >
          {isLoading ? 'Signing in...' : 'Sign In'}
        </button>
      </form>
    </div>
  );
}

// ─── Dashboard (Protected) ───────────────────────────────────

function Dashboard() {
  const { user, tokenSet, logout, hasRole, hasScope } = useGGIDAuth();

  return (
    <div style={{ maxWidth: 600, margin: '40px auto' }}>
      <h2>Dashboard</h2>
      <p>Welcome, {user?.username}!</p>

      <div style={{ marginTop: 20 }}>
        <h3>Profile</h3>
        <pre style={{ background: '#f5f5f5', padding: 12, borderRadius: 4 }}>
          {JSON.stringify(user, null, 2)}
        </pre>
      </div>

      <div style={{ marginTop: 20 }}>
        <h3>Token</h3>
        <p style={{ fontSize: 12, color: '#666' }}>
          {tokenSet?.access_token.slice(0, 50)}...
        </p>
        <p>Expires at: {tokenSet?.expires_at ?? 'unknown'}</p>
      </div>

      <div style={{ marginTop: 20 }}>
        <h3>Permissions</h3>
        <p>Is admin: {hasRole('admin') ? 'Yes' : 'No'}</p>
        <p>Can write: {hasScope('write') ? 'Yes' : 'No'}</p>
      </div>

      <button
        onClick={logout}
        style={{
          marginTop: 20,
          padding: '8px 24px',
          borderRadius: 4,
          background: '#ef4444',
          color: 'white',
          border: 'none',
          cursor: 'pointer',
        }}
      >
        Sign Out
      </button>
    </div>
  );
}

// ─── User Profile Component ──────────────────────────────────

function UserProfile() {
  const { user, isLoading, error } = useUser();

  if (isLoading) return <p>Loading profile...</p>;
  if (error) return <p>Error loading profile: {error}</p>;
  if (!user) return <p>No user data</p>;

  return (
    <div>
      <img src={user.avatar_url} alt="avatar" width={48} height={48} />
      <span>{user.username} ({user.email})</span>
      <span>Roles: {user.roles?.join(', ')}</span>
    </div>
  );
}

// ─── App Root ────────────────────────────────────────────────

function AppContent() {
  const { isAuthenticated } = useGGIDAuth();

  // Automatically refresh tokens before expiry
  useTokenRefresh();

  if (!isAuthenticated) {
    return <LoginForm />;
  }

  return (
    <ProtectedRoute>
      <UserProfile />
      <Dashboard />
    </ProtectedRoute>
  );
}

export default function App() {
  return (
    <ErrorBoundary fallback={<div>Something went wrong. Please refresh.</div>}>
      <GGIDProvider config={ggidConfig}>
        <AppContent />
      </GGIDProvider>
    </ErrorBoundary>
  );
}
