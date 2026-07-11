/**
 * GGID React SDK — Multi-Tenant Example
 *
 * Demonstrates tenant switching with the GGID React SDK:
 * - TenantSwitcher dropdown component
 * - Tenant-aware GGIDProvider reconfiguration
 * - useRoles for permission checks per tenant
 * - Session isolation between tenants
 */

import React, { useState, useEffect } from 'react';
import {
  GGIDProvider,
  useGGIDAuth,
  useRoles,
  ProtectedRoute,
  ErrorBoundary,
  type GGIDConfig,
} from '../src';

// ─── Tenant Registry ────────────────────────────────────────

interface Tenant {
  id: string;
  name: string;
  apiBaseUrl: string;
  color: string;
}

const TENANTS: Tenant[] = [
  { id: '00000000-0000-0000-0000-000000000001', name: 'Acme Corp', apiBaseUrl: 'http://localhost:8080', color: '#6366f1' },
  { id: '00000000-0000-0000-0000-000000000002', name: 'Globex Inc', apiBaseUrl: 'http://localhost:8080', color: '#10b981' },
  { id: '00000000-0000-0000-0000-000000000003', name: 'Initech LLC', apiBaseUrl: 'http://localhost:8080', color: '#f59e0b' },
];

const TENANT_STORAGE_KEY = 'ggid_selected_tenant';

// ─── Tenant Switcher ───────────────────────────────────────

function TenantSwitcher({
  tenants,
  currentTenantId,
  onSelect,
}: {
  tenants: Tenant[];
  currentTenantId: string;
  onSelect: (tenant: Tenant) => void;
}) {
  const [open, setOpen] = useState(false);
  const current = tenants.find((t) => t.id === currentTenantId);

  return (
    <div style={{ position: 'relative', display: 'inline-block' }}>
      <button
        onClick={() => setOpen(!open)}
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 8,
          padding: '8px 16px',
          borderRadius: 6,
          border: '1px solid #ddd',
          background: '#fff',
          cursor: 'pointer',
        }}
      >
        <span
          style={{
            width: 10,
            height: 10,
            borderRadius: '50%',
            background: current?.color ?? '#999',
          }}
        />
        {current?.name ?? 'Select Tenant'}
        <span style={{ fontSize: 10 }}>▾</span>
      </button>
      {open && (
        <div
          style={{
            position: 'absolute',
            top: '100%',
            left: 0,
            marginTop: 4,
            minWidth: 200,
            borderRadius: 6,
            border: '1px solid #ddd',
            background: '#fff',
            boxShadow: '0 4px 12px rgba(0,0,0,0.1)',
            zIndex: 100,
          }}
        >
          {tenants.map((tenant) => (
            <div
n              key={tenant.id}
              onClick={() => {
                onSelect(tenant);
                setOpen(false);
              }}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 8,
                padding: '10px 16px',
                cursor: 'pointer',
                borderBottom: '1px solid #f0f0f0',
                background: tenant.id === currentTenantId ? '#f5f3ff' : 'transparent',
              }}
            >
              <span
                style={{
                  width: 10,
                  height: 10,
                  borderRadius: '50%',
                  background: tenant.color,
                }}
              />
              <div>
                <div style={{ fontWeight: 500, fontSize: 14 }}>{tenant.name}</div>
                <div style={{ fontSize: 11, color: '#999' }}>{tenant.id.slice(0, 8)}...</div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

// ─── Tenant Dashboard ───────────────────────────────────────

function TenantDashboard() {
  const { user, logout } = useGGIDAuth();
  const { roles, scopes, isAdmin, hasAnyRole } = useRoles();

  return (
    <div style={{ maxWidth: 600, margin: '40px auto' }}>
      <h2>Tenant Dashboard</h2>
      <p>Logged in as: {user?.username} ({user?.email})</p>

      <div style={{ marginTop: 20, padding: 16, border: '1px solid #eee', borderRadius: 8 }}>
        <h3>Roles & Scopes</h3>
        <p>Roles: {roles.length > 0 ? roles.join(', ') : 'None'}</p>
        <p>Scopes: {scopes.length > 0 ? scopes.join(', ') : 'None'}</p>
        <p>Is Admin: {isAdmin ? 'Yes' : 'No'}</p>
        <p>Can manage users: {hasAnyRole('admin', 'user-manager') ? 'Yes' : 'No'}</p>
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

// ─── Login Form ─────────────────────────────────────────────

function LoginForm() {
  const { login, isLoading, error } = useGGIDAuth();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    await login(username, password);
  };

  return (
    <div style={{ maxWidth: 400, margin: '80px auto' }}>
      <h2>Sign in</h2>
      {error && <p style={{ color: 'red' }}>{error}</p>}
      <form onSubmit={handleSubmit}>
        <input
          type="text"
          value={username}
          onChange={(e) => setUsername(e.target.value)}
          placeholder="Username"
          style={{ width: '100%', padding: 8, marginBottom: 8, borderRadius: 4 }}
        />
        <input
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          placeholder="Password"
          style={{ width: '100%', padding: 8, marginBottom: 8, borderRadius: 4 }}
        />
        <button type="submit" disabled={isLoading}>
          {isLoading ? 'Signing in...' : 'Sign In'}
        </button>
      </form>
    </div>
  );
}

// ─── App Root with Tenant Switching ─────────────────────────

function AppContent() {
  const { isAuthenticated } = useGGIDAuth();

  if (!isAuthenticated) {
    return <LoginForm />;
  }

  return (
    <ProtectedRoute>
      <TenantDashboard />
    </ProtectedRoute>
  );
}

export default function App() {
  // Selected tenant — persisted in localStorage
  const [selectedTenantId, setSelectedTenantId] = useState<string>(
    typeof window !== 'undefined'
      ? localStorage.getItem(TENANT_STORAGE_KEY) || TENANTS[0].id
      : TENANTS[0].id
  );

  const selectedTenant = TENANTS.find((t) => t.id === selectedTenantId) ?? TENANTS[0];

  // Reconfigure provider when tenant changes
  const ggidConfig: GGIDConfig = {
    apiBaseUrl: selectedTenant.apiBaseUrl,
    tenantId: selectedTenant.id,
    scopes: ['openid', 'profile', 'email', 'offline_access'],
  };

  // Save tenant selection
  useEffect(() => {
    localStorage.setItem(TENANT_STORAGE_KEY, selectedTenantId);
  }, [selectedTenantId]);

  // Unique key forces GGIDProvider re-mount on tenant change
  const providerKey = `tenant-${selectedTenantId}`;

  return (
    <ErrorBoundary fallback={<div>Something went wrong. Please refresh.</div>}>
      <div style={{ minHeight: '100vh', background: '#f9fafb' }}>
        {/* Top bar with tenant switcher */}
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
          <h1 style={{ fontSize: 18, fontWeight: 600 }}>GGID Multi-Tenant Demo</h1>
          <TenantSwitcher
            tenants={TENANTS}
            currentTenantId={selectedTenantId}
            onSelect={(t) => setSelectedTenantId(t.id)}
          />
        </div>

        {/* Provider re-mounts when tenant changes → clean session isolation */}
        <GGIDProvider key={providerKey} config={ggidConfig}>
          <AppContent />
        </GGIDProvider>
      </div>
    </ErrorBoundary>
  );
}
