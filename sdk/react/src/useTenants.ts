/**
 * GGID React SDK — useTenants hook
 *
 * Tenant CRUD + branding management.
 */

import { useState, useEffect, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface Tenant {
  id: string;
  name: string;
  domain: string;
  status: 'active' | 'suspended' | 'provisioning';
  plan: 'free' | 'starter' | 'pro' | 'enterprise';
  branding?: {
    logo_url: string;
    primary_color: string;
    secondary_color: string;
    css_override: string;
    custom_domain: string;
  };
  default_roles: string[];
  mfa_required: boolean;
  created_at: string;
  updated_at?: string;
}

export interface CreateTenantInput {
  name: string;
  domain: string;
  plan?: string;
}

export interface UpdateTenantInput {
  name?: string;
  domain?: string;
  status?: string;
  plan?: string;
  default_roles?: string[];
  mfa_required?: boolean;
}

export interface UseTenantsResult {
  tenants: Tenant[];
  isLoading: boolean;
  error: string | null;
  createTenant: (input: CreateTenantInput) => Promise<Tenant | null>;
  updateTenant: (id: string, input: UpdateTenantInput) => Promise<boolean>;
  deleteTenant: (id: string) => Promise<boolean>;
  updateBranding: (id: string, branding: Partial<NonNullable<Tenant['branding']>>) => Promise<boolean>;
  refetch: () => Promise<void>;
}

export function useTenants(): UseTenantsResult {
  const { getAccessToken, isAuthenticated } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const makeHeaders = useCallback(() => {
    const tok = getAccessToken();
    return {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${tok}`,
      'X-Tenant-ID': tenantId,
    };
  }, [getAccessToken, tenantId]);

  const fetchTenants = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true);
    setError(null);
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/tenants`, {
        headers: makeHeaders(),
      });
      if (!resp.ok) throw new Error(`Failed to fetch tenants (${resp.status})`);
      const data = await resp.json();
      setTenants(data.tenants ?? data.items ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      setTenants([]);
    } finally {
      setIsLoading(false);
    }
  }, [getAccessToken, apiBaseUrl, makeHeaders]);

  useEffect(() => {
    if (isAuthenticated) fetchTenants();
  }, [isAuthenticated, fetchTenants]);

  const createTenant = useCallback(
    async (input: CreateTenantInput): Promise<Tenant | null> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/tenants`, {
          method: 'POST',
          headers: makeHeaders(),
          body: JSON.stringify(input),
        });
        if (!resp.ok) throw new Error(`Failed to create tenant (${resp.status})`);
        const created = await resp.json();
        await fetchTenants();
        return created;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return null;
      }
    },
    [apiBaseUrl, makeHeaders, fetchTenants]
  );

  const updateTenant = useCallback(
    async (id: string, input: UpdateTenantInput): Promise<boolean> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/tenants/${id}`, {
          method: 'PATCH',
          headers: makeHeaders(),
          body: JSON.stringify(input),
        });
        if (!resp.ok) throw new Error(`Failed to update tenant (${resp.status})`);
        await fetchTenants();
        return true;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return false;
      }
    },
    [apiBaseUrl, makeHeaders, fetchTenants]
  );

  const deleteTenant = useCallback(
    async (id: string): Promise<boolean> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/tenants/${id}`, {
          method: 'DELETE',
          headers: makeHeaders(),
        });
        if (!resp.ok) throw new Error(`Failed to delete tenant (${resp.status})`);
        await fetchTenants();
        return true;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return false;
      }
    },
    [apiBaseUrl, makeHeaders, fetchTenants]
  );

  const updateBranding = useCallback(
    async (id: string, branding: Partial<NonNullable<Tenant['branding']>>): Promise<boolean> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/tenants/${id}/branding`, {
          method: 'PUT',
          headers: makeHeaders(),
          body: JSON.stringify(branding),
        });
        if (!resp.ok) throw new Error(`Failed to update branding (${resp.status})`);
        await fetchTenants();
        return true;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return false;
      }
    },
    [apiBaseUrl, makeHeaders, fetchTenants]
  );

  return {
    tenants,
    isLoading,
    error,
    createTenant,
    updateTenant,
    deleteTenant,
    updateBranding,
    refetch: fetchTenants,
  };
}
