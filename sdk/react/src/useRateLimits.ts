/**
 * GGID React SDK — useRateLimits hook
 *
 * CRUD for per-endpoint rate limit configuration with tenant filter.
 *
 * Usage:
 *   const { limits, isLoading, updateLimit, createLimit, deleteLimit } = useRateLimits();
 */

import { useState, useEffect, useCallback } from 'react';
import { useGGIDAuth } from './useGGIDAuth';

export interface RateLimit {
  id: string;
  path_pattern: string;
  method: string;
  requests_per_minute: number;
  burst: number;
  per_tenant: boolean;
  tenant_id?: string;
  enabled: boolean;
}

export interface CreateRateLimitInput {
  path_pattern: string;
  method?: string;
  requests_per_minute: number;
  burst?: number;
  per_tenant?: boolean;
}

export interface UseRateLimitsResult {
  limits: RateLimit[];
  isLoading: boolean;
  error: string | null;
  createLimit: (input: CreateRateLimitInput) => Promise<RateLimit | null>;
  updateLimit: (id: string, input: Partial<RateLimit>) => Promise<boolean>;
  deleteLimit: (id: string) => Promise<boolean>;
  refetch: () => Promise<void>;
}

export function useRateLimits(tenantIdFilter?: string): UseRateLimitsResult {
  const { getAccessToken, isAuthenticated } = useGGIDAuth();
  const apiBaseUrl = typeof window !== 'undefined' ? localStorage.getItem('ggid_api_base') || '' : '';
  const tenantId = typeof window !== 'undefined' ? localStorage.getItem('ggid_tenant_id') || '' : '';

  const [limits, setLimits] = useState<RateLimit[]>([]);
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

  const fetchLimits = useCallback(async () => {
    const tok = getAccessToken();
    if (!tok) return;
    setIsLoading(true);
    setError(null);
    try {
      const params = new URLSearchParams();
      if (tenantIdFilter) params.set('tenant_id', tenantIdFilter);
      const resp = await fetch(
        `${apiBaseUrl}/api/v1/policy/rate-limits?${params.toString()}`,
        { headers: makeHeaders() },
      );
      if (!resp.ok) throw new Error(`Failed to fetch rate limits (${resp.status})`);
      const data = await resp.json();
      setLimits(data.limits ?? data.items ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      setLimits([]);
    } finally {
      setIsLoading(false);
    }
  }, [getAccessToken, apiBaseUrl, makeHeaders, tenantIdFilter]);

  useEffect(() => {
    if (isAuthenticated) fetchLimits();
  }, [isAuthenticated, fetchLimits]);

  const createLimit = useCallback(
    async (input: CreateRateLimitInput): Promise<RateLimit | null> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/policy/rate-limits`, {
          method: 'POST',
          headers: makeHeaders(),
          body: JSON.stringify(input),
        });
        if (!resp.ok) throw new Error(`Failed to create rate limit (${resp.status})`);
        const created = await resp.json();
        setLimits((prev: any) => [...prev, created]);
        return created;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return null;
      }
    },
    [apiBaseUrl, makeHeaders],
  );

  const updateLimit = useCallback(
    async (id: string, input: Partial<RateLimit>): Promise<boolean> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/policy/rate-limits/${id}`, {
          method: 'PATCH',
          headers: makeHeaders(),
          body: JSON.stringify(input),
        });
        if (!resp.ok) throw new Error(`Failed to update rate limit (${resp.status})`);
        const updated = await resp.json();
        setLimits((prev: any) => prev.map((l: any) => (l.id === id ? updated : l)));
        return true;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return false;
      }
    },
    [apiBaseUrl, makeHeaders],
  );

  const deleteLimit = useCallback(
    async (id: string): Promise<boolean> => {
      try {
        const resp = await fetch(`${apiBaseUrl}/api/v1/policy/rate-limits/${id}`, {
          method: 'DELETE',
          headers: makeHeaders(),
        });
        if (!resp.ok) throw new Error(`Failed to delete rate limit (${resp.status})`);
        setLimits((prev: any) => prev.filter((l: any) => l.id !== id));
        return true;
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
        return false;
      }
    },
    [apiBaseUrl, makeHeaders],
  );

  return {
    limits,
    isLoading,
    error,
    createLimit,
    updateLimit,
    deleteLimit,
    refetch: fetchLimits,
  };
}
