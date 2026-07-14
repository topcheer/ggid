import { create } from 'zustand';
import { GGIDClient, GGIDSession, GGIDUser } from './client';

interface AuthState {
  client: GGIDClient | null;
  session: GGIDSession | null;
  loading: boolean;
  error: string | null;

  init: (config: Parameters<typeof GGIDClient.prototype.constructor>[0]) => void;
  login: () => Promise<void>;
  logout: () => Promise<void>;
  restore: () => Promise<void>;
  checkPermission: (resource: string, action: string) => Promise<boolean>;
}

export const useGGIDAuth = create<AuthState>((set, get) => ({
  client: null,
  session: null,
  loading: false,
  error: null,

  init: (config) => {
    const client = new GGIDClient(config);
    set({ client });
  },

  login: async () => {
    const { client } = get();
    if (!client) return;
    set({ loading: true, error: null });
    try {
      const session = await client.login();
      set({ session, loading: false });
    } catch (e: any) {
      set({ error: e.message, loading: false });
    }
  },

  logout: async () => {
    const { client } = get();
    if (!client) return;
    await client.logout();
    set({ session: null });
  },

  restore: async () => {
    const { client } = get();
    if (!client) return;
    set({ loading: true });
    const session = await client.getSession();
    set({ session, loading: false });
  },

  checkPermission: async (resource, action) => {
    const { client } = get();
    if (!client) return false;
    return client.checkPermission(resource, action);
  },
}));

// Convenience hooks
export function useUser(): GGIDUser | null {
  return useGGIDAuth((s) => s.session?.user ?? null);
}

export function useIsAuthenticated(): boolean {
  return useGGIDAuth((s) => s.session !== null);
}

export function useToken(): string | null {
  return useGGIDAuth((s) => s.session?.accessToken ?? null);
}

export { GGIDClient } from './client';
export type { GGIDConfig, GGIDSession, GGIDUser, GGIDClaims } from './types';
