/**
 * GGID React SDK — Dashboard Example
 *
 * Complete admin panel demonstrating:
 * - useGGIDAuth for authentication state
 * - useUser for profile data
 * - useRoles for role/scope checking
 * - usePermissions for fine-grained access
 * - useAuditEvents for activity feed
 * - RequireScope for conditional sections
 * - LogoutButton for session end
 */

import React, { useState } from 'react';
import {
  GGIDProvider,
  useGGIDAuth,
  useUser,
  useRoles,
  usePermissions,
  useAuditEvents,
  ProtectedRoute,
  RequireScope,
  LogoutButton,
  ErrorBoundary,
  type GGIDConfig,
} from '../src';

// ─── Configuration ───────────────────────────────────────────

const config: GGIDConfig = {
  apiBaseUrl: process.env.NEXT_PUBLIC_GGID_API_URL || 'http://localhost:8080',
  tenantId: process.env.NEXT_PUBLIC_GGID_TENANT_ID || '00000000-0000-0000-0000-000000000001',
  scopes: ['openid', 'profile', 'email', 'offline_access'],
};

// ─── Login Screen ────────────────────────────────────────────

function LoginScreen() {
  const { login, isLoading, error } = useGGIDAuth();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    await login(username, password);
  };

  return (
    <div style={{ maxWidth: 360, margin: '80px auto', fontFamily: 'system-ui' }}>
      <h2 style={{ marginBottom: 24 }}>Sign in to GGID</h2>
      {error && <p style={{ color: '#ef4444', marginBottom: 12 }}>{error}</p>}
      <form onSubmit={submit}>
        <input
          type="text"
          placeholder="Username"
          value={username}
          onChange={(e) => setUsername(e.target.value)}
          style={{ width: '100%', padding: 10, marginBottom: 8, borderRadius: 6, border: '1px solid #ddd' }}
        />
        <input
          type="password"
          placeholder="Password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          style={{ width: '100%', padding: 10, marginBottom: 16, borderRadius: 6, border: '1px solid #ddd' }}
        />
        <button
          type="submit"
          disabled={isLoading}
          style={{
            width: '100%',
            padding: 10,
            borderRadius: 6,
            border: 'none',
            background: '#6366f1',
            color: '#fff',
            fontSize: 15,
            fontWeight: 600,
            cursor: isLoading ? 'not-allowed' : 'pointer',
          }}
        >
          {isLoading ? 'Signing in...' : 'Sign In'}
        </button>
      </form>
    </div>
  );
}

// ─── Stat Card ───────────────────────────────────────────────

function StatCard({ label, value, color }: { label: string; value: string | number; color: string }) {
  return (
    <div style={{
      padding: 16,
      borderRadius: 12,
      border: '1px solid #eee',
      background: '#fff',
    }}>
      <p style={{ fontSize: 12, color: '#888', marginBottom: 4 }}>{label}</p>
      <p style={{ fontSize: 28, fontWeight: 700, color }}>{value}</p>
    </div>
  );
}

// ─── Activity Feed ───────────────────────────────────────────

function ActivityFeed() {
  const { events, isLoading, error } = useAuditEvents({ pageSize: 8 });

  if (isLoading) return <p style={{ color: '#888' }}>Loading activity...</p>;
  if (error) return <p style={{ color: '#ef4444' }}>{error}</p>;
  if (events.length === 0) return <p style={{ color: '#aaa' }}>No recent activity.</p>;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
      {events.map((event) => (
        <div
          key={event.id}
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            padding: '8px 12px',
            borderRadius: 6,
            background: '#f9fafb',
          }}
        >
          <div>
            <span style={{ fontWeight: 500, fontSize: 13 }}>{event.action}</span>
            <span style={{ color: '#888', fontSize: 12, marginLeft: 8 }}>
              by {event.actor_name}
            </span>
          </div>
          <span
            style={{
              fontSize: 11,
              fontWeight: 600,
              padding: '2px 8px',
              borderRadius: 12,
              background: event.result === 'success' ? '#dcfce7' : '#fef2f2',
              color: event.result === 'success' ? '#16a34a' : '#ef4444',
            }}
          >
            {event.result}
          </span>
        </div>
      ))}
    </div>
  );
}

// ─── Dashboard ───────────────────────────────────────────────

function Dashboard() {
  const { user, isAuthenticated } = useGGIDAuth();
  const { user: profile, isLoading: profileLoading } = useUser();
  const { roles, scopes, isAdmin, hasAnyRole } = useRoles();
  const { permissions, hasPermission } = usePermissions();

  if (!isAuthenticated) return <LoginScreen />;

  return (
    <div style={{ minHeight: '100vh', background: '#f5f5f5', fontFamily: 'system-ui' }}>
      {/* Header */}
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          padding: '12px 24px',
          background: '#fff',
          borderBottom: '1px solid #eee',
        }}
      >
        <h1 style={{ fontSize: 18, fontWeight: 600 }}>GGID Admin Dashboard</h1>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          <span style={{ fontSize: 14, color: '#555' }}>
            {profileLoading ? 'Loading...' : (profile?.username ?? user?.username)}
          </span>
          <LogoutButton redirectAfterLogout="/login" />
        </div>
      </div>

      <div style={{ maxWidth: 800, margin: '0 auto', padding: 24 }}>
        {/* Profile card */}
        <div
          style={{
            padding: 20,
            borderRadius: 12,
            border: '1px solid #eee',
            background: '#fff',
            marginBottom: 20,
          }}
        >
          <h2 style={{ fontSize: 16, fontWeight: 600, marginBottom: 12 }}>Profile & Permissions</h2>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12, fontSize: 14 }}>
            <div>
              <strong>Email:</strong> {profile?.email ?? user?.email}
            </div>
            <div>
              <strong>Is Admin:</strong> {isAdmin ? 'Yes' : 'No'}
            </div>
            <div>
              <strong>Roles:</strong>{' '}
              {roles.length > 0 ? roles.join(', ') : 'None'}
            </div>
            <div>
              <strong>Scopes:</strong>{' '}
              {scopes.length > 0 ? scopes.join(', ') : 'None'}
            </div>
            <div>
              <strong>Permissions:</strong>{' '}
              {permissions.length > 0 ? permissions.slice(0, 5).join(', ') : 'None'}
              {permissions.length > 5 && ` (+${permissions.length - 5} more)`}
            </div>
            <div>
              <strong>Can manage users:</strong>{' '}
              {hasPermission('users:write') ? 'Yes' : 'No'}
            </div>
          </div>
        </div>

        {/* Stat cards */}
        <div
          style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(4, 1fr)',
            gap: 12,
            marginBottom: 20,
          }}
        >
          <StatCard label="Roles" value={roles.length} color="#8b5cf6" />
          <StatCard label="Scopes" value={scopes.length} color="#6366f1" />
          <StatCard label="Permissions" value={permissions.length} color="#10b981" />
          <StatCard
            label="Admin Access"
            value={isAdmin ? 'Yes' : 'No'}
            color={isAdmin ? '#16a34a' : '#94a3b8'}
          />
        </div>

        {/* Admin-only section */}
        <RequireScope
          scope="admin"
          fallback={
            <div
              style={{
                padding: 20,
                borderRadius: 12,
                border: '1px solid #fee2e2',
                background: '#fef2f2',
                color: '#991b1b',
                marginBottom: 20,
                fontSize: 14,
              }}
            >
              You need admin scope to view system settings.
            </div>
          }
        >
          <div
            style={{
              padding: 20,
              borderRadius: 12,
              border: '1px solid #ddd6fe',
              background: '#f5f3ff',
              marginBottom: 20,
            }}
          >
            <h3 style={{ fontSize: 15, fontWeight: 600, marginBottom: 8 }}>Admin Controls</h3>
            <p style={{ fontSize: 13, color: '#555' }}>
              Welcome, administrator. Full system controls are available below.
            </p>
          </div>
        </RequireScope>

        {/* Manager section */}
        <RequireScope anyOf={['admin', 'user-manager', 'manager']}>
          <div
            style={{
              padding: 20,
              borderRadius: 12,
              border: '1px solid #eee',
              background: '#fff',
              marginBottom: 20,
            }}
          >
            <h3 style={{ fontSize: 15, fontWeight: 600, marginBottom: 8 }}>User Management</h3>
            <p style={{ fontSize: 13, color: '#555' }}>
              You have user management access.
            </p>
          </div>
        </RequireScope>

        {/* Activity feed */}
        <div
          style={{
            padding: 20,
            borderRadius: 12,
            border: '1px solid #eee',
            background: '#fff',
          }}
        >
          <h3 style={{ fontSize: 15, fontWeight: 600, marginBottom: 12 }}>Recent Activity</h3>
          <ActivityFeed />
        </div>
      </div>
    </div>
  );
}

// ─── App Root ────────────────────────────────────────────────

export default function App() {
  return (
    <ErrorBoundary fallback={<div>Something went wrong. Please refresh.</div>}>
      <GGIDProvider config={config}>
        <Dashboard />
      </GGIDProvider>
    </ErrorBoundary>
  );
}
