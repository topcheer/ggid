/**
 * GGID React SDK — useFeatureFlags hook
 *
 * Feature flag CRUD + toggle + per-tenant override.
 */

import { useState, useEffect, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface FeatureFlag {
  id: string;
  key: string;
  name: string;
  description: string;
  enabled: boolean;
  type: 'boolean' | 'percentage' | 'variant';
  value: boolean | number | string;
  per_tenant: Record<string, boolean | number | string>;
  created_at: string;
  updated_at?: string;
}

export interface CreateFlagInput {
  key: string;
  name: string;
  description?: string;
  type?: 'boolean' | 'percentage' | 'variant';
  enabled?: boolean;
  value?: boolean | number | string;
}

export interface UseFeatureFlagsResult {
  flags: FeatureFlag[];
  isLoading: boolean;
  error: string | null;
  createFlag: (input: CreateFlagInput) => Promise<FeatureFlag | null>;
  updateFlag: (id: string, input: Partial<FeatureFlag>) => Promise<boolean>;
  deleteFlag: (id: string) => Promise<boolean>;
  toggleFlag: (id: string) => Promise<boolean>;
  setTenantOverride: (flagId: string, tenantId: string, value: boolean | number | string) => Promise<boolean>;
  isEnabled: (key: string) => boolean;
  refetch: () => Promise<void>;
}

export function useFeatureFlags(): UseFeatureFlagsResult {
  const { getAccessToken, isAuthenticated } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const currentTenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [flags, setFlags] = useState<FeatureFlag[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const makeHeaders = useCallback(() => {
    const tok = getAccessToken();
    return {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${tok}`,
      'X-Tenant-ID': currentTenantId,
    };
  }, [getAccessToken, currentTenantId]);

  const fetchFlags = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true);
    setError(null);
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/feature-flags`, { headers: makeHeaders() });
      if (!resp.ok) throw new Error(`Failed to fetch feature flags (${resp.status})`);
      const data = await resp.json();
      setFlags(data.flags ?? data.items ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      setFlags([]);
    } finally {
      setIsLoading(false);
    }
  }, [getAccessToken, apiBaseUrl, makeHeaders]);

  useEffect(() => {
    if (isAuthenticated) fetchFlags();
  }, [isAuthenticated, fetchFlags]);

  const createFlag = useCallback(async (input: CreateFlagInput): Promise<FeatureFlag | null> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/feature-flags`, {
        method: 'POST', headers: makeHeaders(), body: JSON.stringify(input),
      });
      if (!resp.ok) throw new Error(`Failed to create flag (${resp.status})`);
      const created = await resp.json();
      await fetchFlags();
      return created;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      return null;
    }
  }, [apiBaseUrl, makeHeaders, fetchFlags]);

  const updateFlag = useCallback(async (id: string, input: Partial<FeatureFlag>): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/feature-flags/${id}`, {
        method: 'PATCH', headers: makeHeaders(), body: JSON.stringify(input),
      });
      if (!resp.ok) throw new Error(`Failed to update flag (${resp.status})`);
      await fetchFlags();
      return true;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      return false;
    }
  }, [apiBaseUrl, makeHeaders, fetchFlags]);

  const deleteFlag = useCallback(async (id: string): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/feature-flags/${id}`, {
        method: 'DELETE', headers: makeHeaders(),
      });
      if (!resp.ok) throw new Error(`Failed to delete flag (${resp.status})`);
      await fetchFlags();
      return true;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      return false;
    }
  }, [apiBaseUrl, makeHeaders, fetchFlags]);

  const toggleFlag = useCallback(async (id: string): Promise<boolean> => {
    const flag = flags.find((f) => f.id === id);
    if (!flag) return false;
    return updateFlag(id, { enabled: !flag.enabled });
  }, [flags, updateFlag]);

  const setTenantOverride = useCallback(async (
    flagId: string, tenantId: string, value: boolean | number | string
  ): Promise<boolean> => {
    try {
      const resp = await fetch(`${apiBaseUrl}/api/v1/feature-flags/${flagId}/tenants/${tenantId}`, {
        method: 'PUT', headers: makeHeaders(), body: JSON.stringify({ value }),
      });
      if (!resp.ok) throw new Error(`Failed to set override (${resp.status})`);
      await fetchFlags();
      return true;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      return false;
    }
  }, [apiBaseUrl, makeHeaders, fetchFlags]);

  /** Check if a flag is enabled, respecting per-tenant overrides */
  const isEnabled = useCallback((key: string): boolean => {
    const flag = flags.find((f) => f.key === key);
    if (!flag) return false;
    // Check per-tenant override first
    const override = flag.per_tenant?.[currentTenantId];
    if (override !== undefined) return typeof override === 'boolean' ? override : Boolean(override);
    return flag.enabled;
  }, [flags, currentTenantId]);

  return {
    flags, isLoading, error,
    createFlag, updateFlag, deleteFlag, toggleFlag, setTenantOverride, isEnabled,
    refetch: fetchFlags,
  };
}
