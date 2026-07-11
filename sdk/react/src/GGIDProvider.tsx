/**
 * GGID React SDK — Provider Component
 *
 * Usage:
 *   <GGIDProvider config={{ apiBaseUrl: 'https://api.ggid.dev', tenantId: '...' }}>
 *     <App />
 *   </GGIDProvider>
 */

import { createContext, useState, useEffect, useCallback, type ReactNode } from 'react';
import type { GGIDConfig, GGIDUser, GGIDTokenSet, GGIDAuthState, GGIDAuthContextValue } from './types';

// --- Context ---

export const GGIDAuthContext = createContext<GGIDAuthContextValue | null>(null);

// --- Storage helpers ---

function loadTokenSet(storageKey: string): GGIDTokenSet | null {
  try {
    const raw = localStorage.getItem(storageKey);
    return raw ? JSON.parse(raw) : null;
  } catch {
    return null;
  }
}

function saveTokenSet(storageKey: string, ts: GGIDTokenSet | null) {
  if (ts) {
    localStorage.setItem(storageKey, JSON.stringify(ts));
  } else {
    localStorage.removeItem(storageKey);
  }
}

// --- Provider ---

export function GGIDProvider({
  config,
  children,
}: {
  config: GGIDConfig;
  children: ReactNode;
}) {
  const storageKey = config.storageKey || 'ggid_token';

  const [state, setState] = useState<GGIDAuthState>(() => {
    const tokenSet = typeof window !== 'undefined' ? loadTokenSet(storageKey) : null;
    return {
      user: null,
      tokenSet,
      isLoading: !!tokenSet,
      isAuthenticated: !!tokenSet,
      error: null,
    };
  });

  // Load user profile on mount if token exists
  useEffect(() => {
    if (!state.tokenSet) return;
    const token = state.tokenSet.access_token;
    fetch(`${config.apiBaseUrl}/api/v1/users/me`, {
      headers: {
        Authorization: `Bearer ${token}`,
        'X-Tenant-ID': config.tenantId,
      },
    })
      .then(async (r) => {
        if (!r.ok) throw new Error('Failed to load user');
        const user = await r.json();
        setState((prev) => ({ ...prev, user, isLoading: false }));
      })
      .catch(() => {
        // Token expired or invalid — clear and reset
        saveTokenSet(storageKey, null);
        setState({ user: null, tokenSet: null, isLoading: false, isAuthenticated: false, error: null });
      });
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  const login = useCallback(async (username: string, password: string) => {
    setState((prev) => ({ ...prev, isLoading: true, error: null }));
    try {
      const resp = await fetch(`${config.apiBaseUrl}/api/v1/auth/login`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': config.tenantId,
        },
        body: JSON.stringify({ username, password }),
      });
      if (!resp.ok) {
        const data = await resp.json().catch(() => ({}));
        throw new Error(data.error || data.message || 'Login failed');
      }
      const tokenSet: GGIDTokenSet = await resp.json();
      saveTokenSet(storageKey, tokenSet);
      setState({
        user: null, // Will be loaded by useEffect on next render cycle
        tokenSet,
        isLoading: false,
        isAuthenticated: true,
        error: null,
      });
    } catch (err) {
      setState((prev) => ({
        ...prev,
        isLoading: false,
        error: err instanceof Error ? err.message : 'Login failed',
      }));
      throw err;
    }
  }, [config.apiBaseUrl, config.tenantId, storageKey]);

  const logout = useCallback(() => {
    saveTokenSet(storageKey, null);
    setState({ user: null, tokenSet: null, isLoading: false, isAuthenticated: false, error: null });
  }, [storageKey]);

  const getAccessToken = useCallback(() => {
    return state.tokenSet?.access_token || null;
  }, [state.tokenSet]);

  const hasRole = useCallback((role: string) => {
    return state.user?.roles?.includes(role) ?? false;
  }, [state.user]);

  const hasScope = useCallback((scope: string) => {
    return state.user?.scopes?.includes(scope) ?? false;
  }, [state.user]);

  const ctx: GGIDAuthContextValue = {
    ...state,
    login,
    logout,
    getAccessToken,
    hasRole,
    hasScope,
  };

  return <GGIDAuthContext.Provider value={ctx}>{children}</GGIDAuthContext.Provider>;
}
